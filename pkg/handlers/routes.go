package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/opsmx/ai-gyardian-api/pkg/controller/auth"
	"github.com/opsmx/ai-gyardian-api/pkg/middleware"
)

// SetupRoutes configures all application routes
func SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Create auth controller
	authController := auth.NewOAuthController()

	// Public auth routes (no authentication required)
	authRouter := r.PathPrefix("/auth").Subrouter()
	{
		authRouter.HandleFunc("/google/login", authController.GoogleLogin).Methods("GET")
		authRouter.HandleFunc("/google/callback", authController.GoogleCallback).Methods("GET")
		authRouter.HandleFunc("/logout", authController.Logout).Methods("POST")
	}

	// Protected routes (authentication required)
	apiRouter := r.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(middleware.RequireAuth)
	{
		// User profile
		apiRouter.HandleFunc("/profile", authController.GetProfile).Methods("GET")

		// Add other protected routes here
		// apiRouter.HandleFunc("/hubs", hubController.GetHubs).Methods("GET")
		// apiRouter.HandleFunc("/projects", projectController.GetProjects).Methods("GET")
		// etc.
	}

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods("GET")

	// Root endpoint
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"AI Guardian API","version":"1.0.0"}`))
	}).Methods("GET")

	return r
}
