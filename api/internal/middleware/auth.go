package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/MicahParks/keyfunc/v3"
)

type contextKey string

const userIDKey contextKey = "workos_user_id"

var clientID string

func SetClientID(id string) {
	clientID = id
}

func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

// RequireAuth rejects requests without a valid JWT.
func RequireAuth(jwks keyfunc.Keyfunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := validateToken(r, jwks)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth extracts user ID from JWT if present but doesn't reject unauthenticated requests.
func OptionalAuth(jwks keyfunc.Keyfunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if userID, ok := validateToken(r, jwks); ok {
				ctx := context.WithValue(r.Context(), userIDKey, userID)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func validateToken(r *http.Request, jwks keyfunc.Keyfunc) (string, bool) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", false
	}
	tokenStr := strings.TrimPrefix(auth, "Bearer ")

	token, err := jwt.Parse(tokenStr, jwks.KeyfuncCtx(r.Context()),
		jwt.WithIssuer(fmt.Sprintf("https://api.workos.com/user_management/%s", clientID)),
		jwt.WithIssuedAt(),
	)
	if err != nil {
		slog.Debug("jwt validation failed", "error", err)
		return "", false
	}
	if !token.Valid {
		slog.Debug("jwt token not valid")
		return "", false
	}

	sub, err := token.Claims.GetSubject()
	if err != nil || sub == "" {
		return "", false
	}

	return sub, true
}
