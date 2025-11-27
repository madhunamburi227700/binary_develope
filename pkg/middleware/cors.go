package middleware

import (
	"net/http"
	"strings"

	"github.com/opsmx/ai-guardian-api/pkg/config"
)

// CORS middleware to handle cross-origin requests
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowedOrigin := config.GetUIAddress()

		// Only allow the configured origin
		if origin != "" && origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		// Required for cookies or Authorization header
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Allowed methods
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		// Allowed request headers
		w.Header().Set("Access-Control-Allow-Headers",
			strings.Join([]string{
				"Content-Type",
				"Authorization",
				"X-Requested-With",
				"X-User",
				"X-Authenticated",
				"Cache-Control",
			}, ", "),
		)

		// Exposed response headers
		w.Header().Set("Access-Control-Expose-Headers",
			"Content-Type, Cache-Control, Connection",
		)

		// Cache the preflight for 24 hours (industry standard)
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle OPTIONS preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
