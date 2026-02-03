package handlers

import (
	"net/http"

	"github.com/opsmx/ai-guardian-api/pkg/middleware"
	"github.com/opsmx/ai-guardian-api/pkg/telemetry"
)

// SetupMiddleware configures all middleware for the HTTP server
func SetupMiddleware() []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		telemetry.HTTPMiddleware, // First to measure all requests
		middleware.AuditLog,
		middleware.CORS,
		middleware.Logging,
	}
}

// ApplyMiddleware applies all middleware to a handler
func ApplyMiddleware(handler http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}
