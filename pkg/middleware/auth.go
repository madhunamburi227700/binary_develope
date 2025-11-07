package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/opsmx/ai-guardian-api/pkg/auth/session"
)

const (
	HeaderXUser = "X-User"
)

// RequireAuth middleware checks if user is authenticated
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, err := session.GetSession(r)
		if err != nil {
			// Return JSON response with success false and 401 status
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "UNAUTHORIZED",
			})
			return
		}
		// Add user info to request context
		r.Header.Set(HeaderXUser, username)
		next.ServeHTTP(w, r)
	})
}

// OptionalAuth middleware sets user info if authenticated but doesn't require it
func OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, err := session.GetSession(r)
		if err == nil {
			r.Header.Set(HeaderXUser, username)
			r.Header.Set("X-Authenticated", "true")
		} else {
			r.Header.Set("X-Authenticated", "false")
		}
		next.ServeHTTP(w, r)
	})
}
