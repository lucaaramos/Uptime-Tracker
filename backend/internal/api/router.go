package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/gorilla/websocket"
	"github.com/lv99/uptime-tracker/internal/config"
	"github.com/lv99/uptime-tracker/internal/repository"
	"github.com/lv99/uptime-tracker/internal/ws"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Server struct {
	cfg       *config.Config
	router    *chi.Mux
	hub       *ws.Hub
	monitor   *repository.MonitorRepo
	check     *repository.CheckRepo
	incidents IncidentLister
	auth      AuthStore
	rdb       *redis.Client
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in dev
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func NewServer(cfg *config.Config, hub *ws.Hub, monitor *repository.MonitorRepo, check *repository.CheckRepo, incidents IncidentLister, auth AuthStore, rdb *redis.Client) *Server {
	s := &Server{
		cfg:       cfg,
		hub:       hub,
		monitor:   monitor,
		check:     check,
		incidents: incidents,
		auth:      auth,
		rdb:       rdb,
	}
	s.buildRouter()
	return s
}

func (s *Server) buildRouter() {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	authHandler := NewAuthHandler(s.cfg.JWTSecret, s.auth)
	monitorHandler := NewMonitorHandler(s.monitor, s.check, s.rdb)
	incidentHandler := NewIncidentHandler(s.incidents)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", s.health)

		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(s.cfg.JWTSecret, s.auth))
			r.Use(RateLimiter(s.rdb, 60, 60*time.Second))

			r.Post("/auth/apikey", authHandler.GenerateAPIKey)
			r.Get("/auth/apikeys", authHandler.ListAPIKeys)
			r.Delete("/auth/apikeys/{id}", authHandler.RevokeAPIKey)

			r.Get("/incidents", incidentHandler.List)

			r.Post("/monitors", monitorHandler.Create)
			r.Get("/monitors", monitorHandler.List)
			r.Get("/monitors/{id}", monitorHandler.GetByID)
			r.Put("/monitors/{id}", monitorHandler.Update)
			r.Delete("/monitors/{id}", monitorHandler.Delete)
			r.Post("/monitors/{id}/pause", monitorHandler.Pause)
			r.Post("/monitors/{id}/resume", monitorHandler.Resume)
			r.Get("/monitors/{id}/history", monitorHandler.History)
			r.Get("/monitors/{id}/stats", monitorHandler.Stats)
			r.Get("/monitors/{id}/incidents", incidentHandler.ListByMonitor)
		})
	})

	// WebSocket endpoint (authenticated via ?token=... / ?api_key=... or headers)
	r.Get("/ws", s.handleWS)

	s.router = r
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	if _, err := authenticateRequest(r, s.cfg.JWTSecret, s.auth, true); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("websocket upgrade failed")
		return
	}

	s.hub.Register(conn)
}

func (s *Server) Handler() http.Handler {
	return s.router
}
