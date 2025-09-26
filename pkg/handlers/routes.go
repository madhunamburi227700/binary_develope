package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/opsmx/ai-guardian-api/pkg/controller/auth"
	"github.com/opsmx/ai-guardian-api/pkg/controller/hub"
	"github.com/opsmx/ai-guardian-api/pkg/controller/integrator"
	"github.com/opsmx/ai-guardian-api/pkg/controller/project"
	"github.com/opsmx/ai-guardian-api/pkg/controller/remediation"
	"github.com/opsmx/ai-guardian-api/pkg/controller/scan"
	vuln "github.com/opsmx/ai-guardian-api/pkg/controller/vulnerability"
	"github.com/opsmx/ai-guardian-api/pkg/middleware"
)

// SetupRoutes configures all application routes
func SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Create controllers
	authController := auth.NewOAuthController()
	projectController := project.NewProjectController()
	hubController := hub.NewHubController()
	vulnController := vuln.NewVulnController()
	integratorController := integrator.NewIntegratorController()
	remediationController := remediation.NewRemediationsController()
	scanController := scan.NewScanController()

	// Public auth routes (no authentication required)
	authRouter := r.PathPrefix("/auth").Subrouter()
	{
		authRouter.HandleFunc("/google/login", authController.GoogleLogin).Methods(http.MethodGet)
		authRouter.HandleFunc("/google/callback", authController.GoogleCallback).Methods(http.MethodGet)
		authRouter.HandleFunc("/logout", authController.Logout).Methods(http.MethodPost)
	}

	// Protected routes (authentication required)
	apiRouter := r.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(middleware.RequireAuth)
	{
		// User profile
		apiRouter.HandleFunc("/profile", authController.GetProfile).Methods(http.MethodGet)

		// Project routes
		projectRouter := apiRouter.PathPrefix("/projects").Subrouter()
		{
			// Basic CRUD operations
			// projectRouter.HandleFunc("", projectController.ListProjects).Methods(http.MethodGet)
			// projectRouter.HandleFunc("", projectController.CreateProject).Methods(http.MethodPost)
			// projectRouter.HandleFunc("/details", projectController.ListProjectsWithDetails).Methods(http.MethodGet)
			// projectRouter.HandleFunc("/{id}", projectController.GetProject).Methods(http.MethodGet)
			// projectRouter.HandleFunc("/{id}", projectController.UpdateProject).Methods(http.MethodPut)
			// projectRouter.HandleFunc("/{id}", projectController.DeleteProject).Methods(http.MethodDelete)

			// list projects by hub
			projectRouter.HandleFunc("/list/summary/{hub_id}", projectController.GetProjectSummariesForHub).Methods(http.MethodGet) //working
			// projectRouter.HandleFunc("/details/{project_id}", projectController.GetProjectDetails).Methods(http.MethodGet) //working
			projectRouter.HandleFunc("/summaryCount/{hub_id}", projectController.GetProjectSummaryCount).Methods(http.MethodGet) //working
		}

		// Hub-specific project routes
		hubRouter := apiRouter.PathPrefix("/hubs").Subrouter()
		{
			// create hub during user login
			hubRouter.HandleFunc("", hubController.CreateHub).Methods(http.MethodPost) //working
			// get hub by id for settings
			hubRouter.HandleFunc("/{id}", hubController.GetHub).Methods(http.MethodGet) //working
			// list hubs by owner for left sidebar
			hubRouter.HandleFunc("/user/list", hubController.ListHubsByOwner).Methods(http.MethodGet) //working
		}

		// vulnerabilities
		vulnerabilityRouter := apiRouter.PathPrefix("/vuln").Subrouter()
		{
			// list vulnerabilities by scan for sast ?hubname ?project name SAST
			vulnerabilityRouter.HandleFunc("/list/sast", vulnController.GetSastVulnerabilities).Methods(http.MethodGet)

			// list vulnerabilities by scan for sca ?hubname ?project name SCA
			vulnerabilityRouter.HandleFunc("/list/sca", vulnController.GetSCAVulnerabilityList).Methods(http.MethodGet)
		}

		// integrations
		integrationsRouter := apiRouter.PathPrefix("/integrations").Subrouter()
		{
			// create integration
			integrationsRouter.HandleFunc("/github/create", integratorController.CreateGitHubIntegration).Methods(http.MethodPost)
			// validate integration
			integrationsRouter.HandleFunc("/github/validate", integratorController.ValidateGitHubIntegration).Methods(http.MethodPost)
			// install github app integrator
			integrationsRouter.HandleFunc("/github/install", integratorController.InstallGitHubAppIntegration).Methods(http.MethodGet)
		}

		// remediations
		remediationsRouter := apiRouter.PathPrefix("/remediations").Subrouter()
		{
			remediationsRouter.HandleFunc("/sast", remediationController.SASTRemediation).Methods(http.MethodPost)
			remediationsRouter.HandleFunc("/cve", remediationController.CVERemediation).Methods(http.MethodPost)
		}

		// Scans
		scansRouter := apiRouter.PathPrefix("/scans").Subrouter()
		{
			scansRouter.HandleFunc("/rescan", scanController.Rescan).Methods(http.MethodPost)
		}

		// Add other protected routes here
		// apiRouter.HandleFunc("/integrations", integrationController.GetIntegrations).Methods("GET")
		// apiRouter.HandleFunc("/scans", scanController.GetScans).Methods("GET")
		// apiRouter.HandleFunc("/vulnerabilities", vulnerabilityController.GetVulnerabilities).Methods("GET")
		// apiRouter.HandleFunc("/remediations", remediationController.GetRemediations).Methods("GET")
		// audit
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
