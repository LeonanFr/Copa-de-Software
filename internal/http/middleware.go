package http

import (
	"context"
	"net/http"
	"strings"

	"copasoftware/internal/modules/auth"
	"copasoftware/internal/shared"
)

func AuthMiddleware(authSvc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
