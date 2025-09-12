package middleware

import (
	"net/http"

	"github.com/opsmx/ai-gyardian-api/pkg/auth/session"
	"github.com/opsmx/ai-gyardian-api/pkg/utils"
)

// RequireAuth middleware checks if user is authenticated
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := utils.NewErrorLogger("auth_middleware")
		username, err := session.GetSession(r)
		if err != nil {
			logger.LogWarning("Unauthenticated request", map[string]interface{}{
				"path":   r.URL.Path,
				"method": r.Method,
				"ip":     r.RemoteAddr,
			})
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		// Add user info to request context
		r.Header.Set("X-User", username)
		next.ServeHTTP(w, r)
	})
}

// OptionalAuth middleware sets user info if authenticated but doesn't require it
func OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, err := session.GetSession(r)
		if err == nil {
			r.Header.Set("X-User", username)
			r.Header.Set("X-Authenticated", "true")
		} else {
			r.Header.Set("X-Authenticated", "false")
		}
		next.ServeHTTP(w, r)
	})
}
