package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lv99/uptime-tracker/internal/model"
	"github.com/lv99/uptime-tracker/internal/repository"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type AuthStore interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	CreateAPIKey(ctx context.Context, key *model.APIKey) error
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*model.APIKey, error)
	ListAPIKeysByUser(ctx context.Context, userID primitive.ObjectID) ([]model.APIKey, error)
	DeleteAPIKey(ctx context.Context, id, userID primitive.ObjectID) error
	TouchAPIKeyLastUsed(ctx context.Context, id primitive.ObjectID) error
}

type AuthHandler struct {
	jwtSecret string
	store     AuthStore
}

func NewAuthHandler(jwtSecret string, store AuthStore) *AuthHandler {
	return &AuthHandler{
		jwtSecret: jwtSecret,
		store:     store,
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

type CreateAPIKeyRequest struct {
	Name string `json:"name"`
}

type APIKeyResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	APIKey    string    `json:"api_key"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	username := strings.TrimSpace(req.Username)
	if username == "" || len(req.Password) < 6 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username required and password must be at least 6 characters"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Err(err).Msg("failed to hash password")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
		return
	}

	user := &model.User{
		Username:     username,
		PasswordHash: string(hash),
	}
	if err := h.store.CreateUser(r.Context(), user); err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "username already exists"})
			return
		}
		log.Error().Err(err).Msg("failed to create user")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
		return
	}

	h.issueToken(w, http.StatusCreated, user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	username := strings.TrimSpace(req.Username)
	if username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
		return
	}

	user, err := h.store.GetUserByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		log.Error().Err(err).Msg("failed to fetch user")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "authentication failed"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	h.issueToken(w, http.StatusOK, user)
}

func (h *AuthHandler) issueToken(w http.ResponseWriter, status int, user *model.User) {
	expiresAt := time.Now().Add(24 * time.Hour)
	claims := jwt.MapClaims{
		"sub":      user.ID.Hex(),
		"username": user.Username,
		"iat":      time.Now().Unix(),
		"exp":      expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		log.Error().Err(err).Msg("failed to sign token")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		return
	}

	writeJSON(w, status, LoginResponse{
		Token:     tokenStr,
		ExpiresAt: expiresAt.Unix(),
	})
}

func (h *AuthHandler) GenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	userIDHex := UserIDFromContext(r.Context())
	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid user context"})
		return
	}

	var req CreateAPIKeyRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "API Key"
	}

	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		log.Error().Err(err).Msg("failed to generate api key")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate key"})
		return
	}

	rawKey := "ut_" + hex.EncodeToString(rawBytes)
	apiKey := &model.APIKey{
		UserID:  userID,
		Name:    name,
		Prefix:  rawKey[:11],
		KeyHash: repository.HashAPIKey(rawKey),
	}

	if err := h.store.CreateAPIKey(r.Context(), apiKey); err != nil {
		log.Error().Err(err).Msg("failed to persist api key")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save key"})
		return
	}

	writeJSON(w, http.StatusCreated, APIKeyResponse{
		ID:        apiKey.ID.Hex(),
		Name:      apiKey.Name,
		Prefix:    apiKey.Prefix,
		APIKey:    rawKey,
		CreatedAt: apiKey.CreatedAt,
	})
}

func (h *AuthHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userIDHex := UserIDFromContext(r.Context())
	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid user context"})
		return
	}

	keys, err := h.store.ListAPIKeysByUser(r.Context(), userID)
	if err != nil {
		log.Error().Err(err).Msg("failed to list api keys")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list api keys"})
		return
	}
	if keys == nil {
		keys = []model.APIKey{}
	}

	writeJSON(w, http.StatusOK, keys)
}

func (h *AuthHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	userIDHex := UserIDFromContext(r.Context())
	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid user context"})
		return
	}

	keyID, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid api key id"})
		return
	}

	if err := h.store.DeleteAPIKey(r.Context(), keyID, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "api key not found"})
			return
		}
		log.Error().Err(err).Msg("failed to revoke api key")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to revoke api key"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}
