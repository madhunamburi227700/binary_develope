package handlers

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/opsmx/ai-guardian-api/pkg/controller/audit"
	"github.com/opsmx/ai-guardian-api/pkg/controller/auth"
	"github.com/opsmx/ai-guardian-api/pkg/controller/features"
	"github.com/opsmx/ai-guardian-api/pkg/controller/feedback"
	"github.com/opsmx/ai-guardian-api/pkg/controller/hub"
	"github.com/opsmx/ai-guardian-api/pkg/controller/integrator"
	"github.com/opsmx/ai-guardian-api/pkg/controller/project"
	"github.com/opsmx/ai-guardian-api/pkg/controller/remediation"
	"github.com/opsmx/ai-guardian-api/pkg/controller/remediation_feedback"
	"github.com/opsmx/ai-guardian-api/pkg/controller/scan"
	vuln "github.com/opsmx/ai-guardian-api/pkg/controller/vulnerability"
	"github.com/opsmx/ai-guardian-api/pkg/controller/webhook"
	"github.com/opsmx/ai-guardian-api/pkg/middleware"
	"github.com/opsmx/ai-guardian-api/pkg/telemetry"
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
	remediationFeedbackController := remediation_feedback.NewRemediationFeedbackController()
	scanController := scan.NewScanController()
	feedbackController := feedback.NewFeedbackController()
	featuresController := features.NewFeaturesController()
	auditController := audit.NewAuditController()
	webhookController := webhook.NewWebhookController()
	// Public auth routes (no authentication required)
	authRouter := r.PathPrefix("/auth").Subrouter()
	{
		authRouter.HandleFunc("/google/login", authController.GoogleLogin).Methods(http.MethodGet)
		authRouter.HandleFunc("/google/callback", authController.GoogleCallback).Methods(http.MethodGet)
		authRouter.HandleFunc("/logout", authController.Logout).Methods(http.MethodPost)

		// install github app integrator
		// authRouter.HandleFunc("/github/install", integratorController.InstallGitHubAppIntegration).Methods(http.MethodGet)
		authRouter.HandleFunc("/github/install", authController.GithubUserAuth).Methods(http.MethodGet)
		authRouter.HandleFunc("/github/login", authController.GithubLogin).Methods(http.MethodPost)
		// GitHub OAuth
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
			projectRouter.HandleFunc("", projectController.CreateProject).Methods(http.MethodPost)
			// projectRouter.HandleFunc("/details", projectController.ListProjectsWithDetails).Methods(http.MethodGet)
			projectRouter.HandleFunc("/{id}", projectController.GetProject).Methods(http.MethodGet)
			projectRouter.HandleFunc("/{id}/stats", projectController.GetProjectStats).Methods(http.MethodGet)
			projectRouter.HandleFunc("/{id}", projectController.UpdateProject).Methods(http.MethodPut)
			projectRouter.HandleFunc("/{id}", projectController.DeleteProject).Methods(http.MethodDelete)

			// list projects by hub
			projectRouter.HandleFunc("/list/summary/{hub_id}", projectController.GetProjectSummariesForHub).Methods(http.MethodGet) //working
			// list all projects with repo/branch from latest scan
			projectRouter.HandleFunc("/list/all/{hub_id}", projectController.ListAllProjectsWithLatestScan).Methods(http.MethodGet)
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
			// get hub stats
			hubRouter.HandleFunc("/{id}/stats", hubController.GetHubStats).Methods(http.MethodGet) //working
			// list hubs by owner for left sidebar
			hubRouter.HandleFunc("/user/list", hubController.ListHubsByOwner).Methods(http.MethodGet) //working
			// get hub related remediations
			hubRouter.HandleFunc("/{id}/remediations/history", hubController.GetHubRemediations).Methods(http.MethodGet)
		}

		// vulnerabilities
		vulnerabilityRouter := apiRouter.PathPrefix("/vuln").Subrouter()
		{
			// list vulnerabilities by scan for sast ?hubname ?project name SAST
			vulnerabilityRouter.HandleFunc("/list/sast", vulnController.GetSastVulnerabilities).Methods(http.MethodGet)

			// list vulnerabilities by scan for sca ?hubname ?project name SCA
			vulnerabilityRouter.HandleFunc("/list/sca", vulnController.GetSCAVulnerabilityList).Methods(http.MethodGet)

			// Get vulnerability optimization data
			vulnerabilityRouter.HandleFunc("/optimisation", vulnController.GetVulnerabilityOptimization).Methods(http.MethodGet)

			// Get vulnerability prioritization data
			vulnerabilityRouter.HandleFunc("/prioritisation", vulnController.GetVulnerabilityPrioritization).Methods(http.MethodGet)

			// Download SAST report
			vulnerabilityRouter.HandleFunc("/sast/report", vulnController.DownloadSASTReport).Methods(http.MethodGet)

			// Download SCA report
			vulnerabilityRouter.HandleFunc("/sca/report", vulnController.DownloadSCAReport).Methods(http.MethodGet)
		}

		// integrations
		integrationsRouter := apiRouter.PathPrefix("/integrations").Subrouter()
		{
			// list Integrations
			integrationsRouter.HandleFunc("/", integratorController.ListIntegrations).Methods(http.MethodGet)
			// create integration
			integrationsRouter.HandleFunc("/github/create", integratorController.CreateGitHubIntegration).Methods(http.MethodPost)
			// update integration
			integrationsRouter.HandleFunc("/github/update", integratorController.UpdateGitHubIntegration).Methods(http.MethodPut)
			// validate integration
			integrationsRouter.HandleFunc("/github/validate", integratorController.ValidateGitHubIntegration).Methods(http.MethodPost)
			// integrations related details for github
			integrationsRouter.HandleFunc("/github/details", integratorController.GetIntegrationsGithubDetails).Methods(http.MethodGet)
			// delete integration
			integrationsRouter.HandleFunc("/github/delete", integratorController.DeleteIntegration).Methods(http.MethodDelete)
			// install github app integrator
			integrationsRouter.HandleFunc("/github/install", integratorController.InstallGitHubAppIntegration).Methods(http.MethodGet)
			// setup workflow
			integrationsRouter.HandleFunc("/github/setup/workflow", webhookController.SetupWorkflow).Methods(http.MethodPost)
			// check for workflow setup
			integrationsRouter.HandleFunc("/github/check/workflow", webhookController.CheckWorkflowStatus).Methods(http.MethodPost)
		}

		// remediations
		remediationsRouter := apiRouter.PathPrefix("/remediations").Subrouter()
		{
			remediationsRouter.HandleFunc("/sast", remediationController.SASTRemediation).Methods(http.MethodPost)
			remediationsRouter.HandleFunc("/cve", remediationController.CVERemediation).Methods(http.MethodPost)
			remediationsRouter.HandleFunc("/conversation/{id}", remediationController.Conversation).Methods(http.MethodGet)
		}

		// remediation feedback
		feedbackRouter := apiRouter.PathPrefix("/remediation-feedback").Subrouter()
		{
			feedbackRouter.HandleFunc("", remediationFeedbackController.CreateFeedback).Methods(http.MethodPost)

			// Developer endpoints
			// feedbackRouter.HandleFunc("", remediationFeedbackController.ListFeedbacks).Methods(http.MethodGet)

			// feedbackRouter.HandleFunc("/{id}", remediationFeedbackController.GetFeedback).Methods(http.MethodGet)

			// feedbackRouter.HandleFunc("/{id}", remediationFeedbackController.UpdateFeedback).Methods(http.MethodPut)

			// feedbackRouter.HandleFunc("/{id}", remediationFeedbackController.DeleteFeedback).Methods(http.MethodDelete)

			// feedbackRouter.HandleFunc("/remediation/{remediation_id}", remediationFeedbackController.GetFeedbacksByRemediationID).Methods(http.MethodGet)

			// feedbackRouter.HandleFunc("/vulnerability/{vulnerability_id}", remediationFeedbackController.GetFeedbacksByVulnerabilityID).Methods(http.MethodGet)

			// feedbackRouter.HandleFunc("/stats/{remediation_id}", remediationFeedbackController.GetFeedbackStats).Methods(http.MethodGet)
		}

		// Scans
		scansRouter := apiRouter.PathPrefix("/scans").Subrouter()
		{
			scansRouter.HandleFunc("/rescan", scanController.Rescan).Methods(http.MethodPost)
			scansRouter.HandleFunc("/file", scanController.ScanFile).Methods(http.MethodPost)
		}

		// Feedback (requires authentication)
		userFeedbackRouter := apiRouter.PathPrefix("/feedback").Subrouter()
		{
			userFeedbackRouter.HandleFunc("/send", feedbackController.SendFeedback).Methods(http.MethodPost)
		}

		// Features (requires authentication)
		featuresRouter := apiRouter.PathPrefix("/features").Subrouter()
		{
			featuresRouter.HandleFunc("", featuresController.GetUserFeatures).Methods(http.MethodGet)
		}

		// Audit (requires authentication)
		auditRouter := apiRouter.PathPrefix("/audit").Subrouter()
		{
			auditRouter.HandleFunc("/report", auditController.GetAuditReport).Methods(http.MethodGet)
		}

		// NLI stream (forwards request body to NLI_BASE_URL/stream, response as-is)
		nliRouter := apiRouter.PathPrefix("/nli").Subrouter()
		{
			nliRouter.HandleFunc("/stream", remediationController.NLI).Methods(http.MethodPost)
		}
	}

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods("GET")

	// Prometheus metrics endpoint
	r.Handle("/metrics", telemetry.MetricsHandler()).Methods("GET")

	// Root endpoint
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"AI Guardian API","version":"1.0.0"}`))
	}).Methods("GET")

	return r
}
