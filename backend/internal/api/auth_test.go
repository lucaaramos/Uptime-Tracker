package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lv99/uptime-tracker/internal/model"
	"github.com/lv99/uptime-tracker/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type memAuthStore struct {
	mu      sync.Mutex
	users   map[string]*model.User
	apiKeys map[string]*model.APIKey // keyed by KeyHash
}

func newMemAuthStore() *memAuthStore {
	return &memAuthStore{
		users:   make(map[string]*model.User),
		apiKeys: make(map[string]*model.APIKey),
	}
}

func (s *memAuthStore) CreateUser(_ context.Context, user *model.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.users[user.Username]; exists {
		return repository.ErrUserExists
	}
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	s.users[user.Username] = user
	return nil
}

func (s *memAuthStore) GetUserByUsername(_ context.Context, username string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[username]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (s *memAuthStore) CreateAPIKey(_ context.Context, key *model.APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key.ID = primitive.NewObjectID()
	key.CreatedAt = time.Now()
	s.apiKeys[key.KeyHash] = key
	return nil
}

func (s *memAuthStore) GetAPIKeyByHash(_ context.Context, keyHash string) (*model.APIKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k, ok := s.apiKeys[keyHash]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return k, nil
}

func (s *memAuthStore) ListAPIKeysByUser(_ context.Context, userID primitive.ObjectID) ([]model.APIKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var res []model.APIKey
	for _, k := range s.apiKeys {
		if k.UserID == userID {
			res = append(res, *k)
		}
	}
	return res, nil
}

func (s *memAuthStore) DeleteAPIKey(_ context.Context, id, userID primitive.ObjectID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for hash, k := range s.apiKeys {
		if k.ID == id && k.UserID == userID {
			delete(s.apiKeys, hash)
			return nil
		}
	}
	return repository.ErrNotFound
}

func (s *memAuthStore) TouchAPIKeyLastUsed(_ context.Context, id primitive.ObjectID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for _, k := range s.apiKeys {
		if k.ID == id {
			k.LastUsedAt = &now
			return nil
		}
	}
	return nil
}

func TestAuthRegisterLoginAndAPIKeyFlow(t *testing.T) {
	store := newMemAuthStore()
	secret := "test-jwt-secret"
	handler := NewAuthHandler(secret, store)

	// 1. Register with short password -> 400
	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{"username":"alice","password":"123"}`))
	w := httptest.NewRecorder()
	handler.Register(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for short password, got %d", w.Code)
	}

	// 2. Register valid user -> 201 + JWT
	req = httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{"username":"alice","password":"secret123"}`))
	w = httptest.NewRecorder()
	handler.Register(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var regResp LoginResponse
	if err := json.NewDecoder(w.Body).Decode(&regResp); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if regResp.Token == "" {
		t.Fatal("expected non-empty token")
	}

	// 3. Duplicate register -> 409
	req = httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{"username":"alice","password":"secret123"}`))
	w = httptest.NewRecorder()
	handler.Register(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate user, got %d", w.Code)
	}

	// 4. Login wrong password -> 401
	req = httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"username":"alice","password":"wrong"}`))
	w = httptest.NewRecorder()
	handler.Login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", w.Code)
	}

	// 5. Login valid credentials -> 200
	req = httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"username":"alice","password":"secret123"}`))
	w = httptest.NewRecorder()
	handler.Login(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid login, got %d", w.Code)
	}

	// 6. Generate API key via AuthMiddleware
	protected := chi.NewRouter()
	protected.Use(AuthMiddleware(secret, store))
	protected.Post("/apikey", handler.GenerateAPIKey)
	protected.Get("/apikeys", handler.ListAPIKeys)
	protected.Delete("/apikeys/{id}", handler.RevokeAPIKey)
	protected.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"user": UserIDFromContext(r.Context())})
	})

	req = httptest.NewRequest("POST", "/apikey", strings.NewReader(`{"name":"ci-key"}`))
	req.Header.Set("Authorization", "Bearer "+regResp.Token)
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for api key creation, got %d: %s", w.Code, w.Body.String())
	}

	var keyResp APIKeyResponse
	if err := json.NewDecoder(w.Body).Decode(&keyResp); err != nil {
		t.Fatalf("decode api key response: %v", err)
	}
	if !strings.HasPrefix(keyResp.APIKey, "ut_") {
		t.Fatalf("expected ut_ prefix, got %s", keyResp.APIKey)
	}

	// 7. Short / invalid API key must return 401 without panicking
	req = httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("X-API-Key", "abc")
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for short invalid key, got %d", w.Code)
	}

	// 8. Valid API key authenticates
	req = httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("X-API-Key", keyResp.APIKey)
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid api key, got %d", w.Code)
	}

	// 9. Query-param auth for WebSocket accepts valid token & api_key, rejects missing/invalid
	wsReq := httptest.NewRequest("GET", "/ws", nil)
	if _, err := authenticateRequest(wsReq, secret, store, true); err == nil {
		t.Fatal("expected error for unauthenticated ws request")
	}

	wsReq = httptest.NewRequest("GET", "/ws?token="+regResp.Token, nil)
	if _, err := authenticateRequest(wsReq, secret, store, true); err != nil {
		t.Fatalf("expected valid ws token auth, got %v", err)
	}

	wsReq = httptest.NewRequest("GET", "/ws?api_key="+keyResp.APIKey, nil)
	if _, err := authenticateRequest(wsReq, secret, store, true); err != nil {
		t.Fatalf("expected valid ws api_key auth, got %v", err)
	}

	// 10. Revoke API key -> subsequent use returns 401
	req = httptest.NewRequest("DELETE", "/apikeys/"+keyResp.ID, nil)
	req.Header.Set("Authorization", "Bearer "+regResp.Token)
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for revoke, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("X-API-Key", keyResp.APIKey)
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after revocation, got %d", w.Code)
	}
}
