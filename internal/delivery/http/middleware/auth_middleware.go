package middleware

import (
	"context"
	"net/http"
	"rokomferi-backend/pkg/utils"
	"strings"
)

type contextKey string

const UserIDKey contextKey = "userID"
const UserRoleKey contextKey = "userRole"
const UserEmailKey contextKey = "userEmail"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Get Token from Header or Cookie
		tokenString := ""
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			cookie, err := r.Cookie("accessToken")
			if err == nil {
				tokenString = cookie.Value
			}
		}

		if tokenString == "" {
			http.Error(w, "Unauthorized: No token provided", http.StatusUnauthorized)
			return
		}

		// 2. Validate Token
		claims, err := utils.ValidateJWT(tokenString)
		if err != nil {
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		// 3. Set Context
		ctx := context.WithValue(r.Context(), UserIDKey, claims["sub"])
		ctx = context.WithValue(ctx, UserEmailKey, claims["email"])
		ctx = context.WithValue(ctx, UserRoleKey, claims["role"])

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
