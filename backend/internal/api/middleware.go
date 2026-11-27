package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lv99/uptime-tracker/internal/repository"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type contextKey string

const userContextKey contextKey = "user_id"

func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userContextKey).(string)
	return v
}

func RateLimiter(rdb *redis.Client, maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			key := "ratelimit:" + ip + ":" + r.URL.Path

			count, err := rdb.Incr(r.Context(), key).Result()
			if err != nil {
				log.Error().Err(err).Msg("rate limiter error")
				next.ServeHTTP(w, r)
				return
			}

			if count == 1 {
				rdb.Expire(r.Context(), key, window)
			}

			if count > int64(maxRequests) {
				writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func AuthMiddleware(jwtSecret string, store AuthStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := authenticateRequest(r, jwtSecret, store, false)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func authenticateRequest(r *http.Request, jwtSecret string, store AuthStore, allowQuery bool) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader || tokenStr == "" {
			return "", errors.New("invalid authorization format")
		}
		return validateJWT(tokenStr, jwtSecret)
	}

	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return validateAPIKey(r.Context(), apiKey, store)
	}

	if allowQuery {
		if tokenStr := r.URL.Query().Get("token"); tokenStr != "" {
			return validateJWT(tokenStr, jwtSecret)
		}
		if apiKey := r.URL.Query().Get("api_key"); apiKey != "" {
			return validateAPIKey(r.Context(), apiKey, store)
		}
	}

	return "", errors.New("missing authorization")
}

func validateJWT(tokenStr, jwtSecret string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}

	userID, _ := claims["sub"].(string)
	if userID == "" {
		return "", errors.New("invalid token subject")
	}

	return userID, nil
}

func validateAPIKey(ctx context.Context, rawKey string, store AuthStore) (string, error) {
	if store == nil || rawKey == "" {
		return "", errors.New("invalid api key")
	}

	keyHash := repository.HashAPIKey(rawKey)
	key, err := store.GetAPIKeyByHash(ctx, keyHash)
	if err != nil {
		return "", errors.New("invalid api key")
	}

	// Best-effort last_used_at update
	_ = store.TouchAPIKeyLastUsed(ctx, key.ID)

	return key.UserID.Hex(), nil
}
