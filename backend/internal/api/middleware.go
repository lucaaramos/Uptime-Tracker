package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type contextKey string

const userContextKey contextKey = "user_id"

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
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// Also check for API key
				apiKey := r.Header.Get("X-API-Key")
				if apiKey == "" {
					http.Error(w, `{"error":"missing authorization"}`, http.StatusUnauthorized)
					return
				}
				// API key auth - store the key prefix as user
				ctx := context.WithValue(r.Context(), userContextKey, "apikey:"+apiKey[:8])
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader {
				http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error":"invalid claims"}`, http.StatusUnauthorized)
				return
			}

			userID, _ := claims["sub"].(string)
			ctx := context.WithValue(r.Context(), userContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
