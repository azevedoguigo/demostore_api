package middleware

import (
	"net/http"
	"strings"
	"sync"

	"github.com/azevedoguigo/demostore_api.git/pkg/utils"
	"github.com/go-chi/jwtauth/v5"
)

var (
	tokenAuth     *jwtauth.JWTAuth
	tokenAuthOnce sync.Once
)

func getTokenAuth() *jwtauth.JWTAuth {
	tokenAuthOnce.Do(func() {
		secret := utils.GetEnv("JWT_SECRET", "dev-secret-change-me")
		tokenAuth = jwtauth.New("HS256", []byte(secret), nil)
	})

	return tokenAuth
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		if strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		} else {
			http.Error(w, "Authorization header must be of type Bearer", http.StatusUnauthorized)
			return
		}

		token, err := getTokenAuth().Decode(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := jwtauth.NewContext(r.Context(), token, nil)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, claims, err := jwtauth.FromContext(r.Context())
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			roleClaim, ok := claims["role"].(string)
			if !ok {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			if _, ok := allowed[roleClaim]; !ok {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
