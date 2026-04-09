package http

import (
	"context"
	"net/http"
	"strings"

	"copasoftware/internal/modules/auth"
	"copasoftware/internal/shared"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func AuthMiddleware(authSvc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				shared.RespondError(w, shared.NewUnauthorizedError("token não fornecido", nil))
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				shared.RespondError(w, shared.NewUnauthorizedError("formato do token inválido", nil))
				return
			}

			adminID, err := authSvc.ValidateToken(parts[1])
			if err != nil {
				shared.RespondError(w, shared.NewUnauthorizedError("token inválido", err))
				return
			}

			ctx := context.WithValue(r.Context(), "adminID", adminID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
