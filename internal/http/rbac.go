package http

import (
	nethttp "net/http"

	"extrusion-quality-system/internal/domain"
)

func RequireRoles(allowedRoles ...domain.UserRole) func(nethttp.Handler) nethttp.Handler {
	allowed := make(map[domain.UserRole]struct{}, len(allowedRoles))

	for _, role := range allowedRoles {
		allowed[role] = struct{}{}
	}

	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			user, ok := CurrentUser(r.Context())
			if !ok {
				writeError(w, nethttp.StatusUnauthorized, "unauthorized")
				return
			}

			if _, ok := allowed[user.Role]; !ok {
				writeError(w, nethttp.StatusForbidden, "forbidden")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
