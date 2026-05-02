package http

import (
	"context"
	"encoding/json"
	"log/slog"
	nethttp "net/http"
	"strings"

	authservice "extrusion-quality-system/internal/auth"
)

type contextKey string

const currentUserContextKey contextKey = "current_user"

func AuthMiddleware(
	logger *slog.Logger,
	tokenManager *authservice.TokenManager,
) func(nethttp.Handler) nethttp.Handler {
	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")

			token, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				w.WriteHeader(nethttp.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing authorization token"})
				return
			}

			claims, err := tokenManager.Parse(token)
			if err != nil {
				logger.Warn("invalid authorization token", "error", err)

				w.WriteHeader(nethttp.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid authorization token"})
				return
			}

			ctx := context.WithValue(r.Context(), currentUserContextKey, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func CurrentUser(ctx context.Context) (authservice.Claims, bool) {
	claims, ok := ctx.Value(currentUserContextKey).(authservice.Claims)

	return claims, ok
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "

	if !strings.HasPrefix(header, prefix) {
		return "", false
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))

	return token, token != ""
}
