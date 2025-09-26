package handlers

import (
	"net/http"

	"github.com/opsmx/ai-guardian-api/pkg/middleware"
)

// SetupMiddleware configures all middleware for the HTTP server
func SetupMiddleware() []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.Logging,
		middleware.CORS,
	}
}

// ApplyMiddleware applies all middleware to a handler
func ApplyMiddleware(handler http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}
