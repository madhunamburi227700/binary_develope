package integrator

import (
	"encoding/json"
	"net/http"

	"github.com/opsmx/ai-guardian-api/pkg/service"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type IntegratorController struct {
	integratorService *service.IntegrationService
	logger            *utils.ErrorLogger
}

func NewIntegratorController() *IntegratorController {
	return &IntegratorController{
		integratorService: service.NewIntegrationService(),
		logger:            utils.NewErrorLogger("integrator_controller"),
	}
}

// CreateGitHubIntegration creates a new GitHub integration
// @Summary Create GitHub integration
// @Description Creates a new GitHub integration with the provided details
// @Tags Integrations
// @Accept  json
// @Produce  json
// @Param   request body service.CreateGitHubIntegrationRequest true "GitHub integration details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/integrations/github [post]
func (c *IntegratorController) CreateGitHubIntegration(w http.ResponseWriter, r *http.Request) {
	var req service.CreateGitHubIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.logger.LogError(err, "Failed to decode request", nil)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	integration, err := c.integratorService.CreateGitHubIntegration(r.Context(), req)
	if err != nil {
		c.logger.LogError(err, "Failed to create integration", nil)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    integration,
	})
}

// ValidateGitHubIntegration validates GitHub integration credentials
// @Summary Validate GitHub integration
// @Description Validates the provided GitHub integration credentials
// @Tags Integrations
// @Accept  json
// @Produce  json
// @Param   request body service.ValidateGitHubIntegrationRequest true "GitHub integration credentials"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/integrations/github/validate [post]
func (c *IntegratorController) ValidateGitHubIntegration(w http.ResponseWriter, r *http.Request) {
	var req service.ValidateGitHubIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.logger.LogError(err, "Failed to decode request", nil)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	validation, err := c.integratorService.ValidateGitHubIntegration(r.Context(), req)
	if err != nil {
		c.logger.LogError(err, "Failed to validate GitHub integration", nil)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    validation,
	})
}

// InstallGitHubAppIntegration installs the GitHub App integration
// @Summary Install GitHub App
// @Description Installs the GitHub App integration with the provided installation ID
// @Tags Integrations
// @Accept  json
// @Produce  json
// @Param   request body service.InstallGitHubAppRequest true "GitHub App installation details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/integrations/github/app/install [post]
func (c *IntegratorController) InstallGitHubAppIntegration(w http.ResponseWriter, r *http.Request) {

	installationUrl, err := c.integratorService.GetGithubAppInstallationURL(r.Context())
	if err != nil {
		c.logger.LogError(err, "Failed to get GitHub app install url", nil)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	redirectionURL := struct {
		Url string `json:"url"`
	}{
		Url: installationUrl,
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    redirectionURL,
	})
}

// GetIntegrationsGithubDetails get GitHub integration details based on various params
// @Summary Github Integrations details via Github APIs
// @Description Github Integrations details via Github APIs
// @Tags Integrations
// @Accept */*
// @Produce  json
// @Param platform query string true "Platform name (e.g., github)"
// @Param type query string true "Type of integration (e.g., organisation, user)"
// @Param accountId query string true "Account ID (e.g., 0x3b32)"
// @Param scanLevel query string true "Scan level (e.g., repository, organization)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/integrations/github/details [get]
func (c *IntegratorController) GetIntegrationsGithubDetails(w http.ResponseWriter, r *http.Request) {
	queryParams := map[string]string{
		// default scanLevel
		"scanLevel": "org",
	}
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}
	details, err := c.integratorService.GetGithubIntegrationsDetails(r.Context(), queryParams)
	if err != nil {
		c.logger.LogError(err, "Failed to get github integrations details", nil)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    details,
	})
}

func (c *IntegratorController) ListIntegrations(w http.ResponseWriter, r *http.Request) {
	teamIds := r.URL.Query().Get("teamIds")
	// TODO: manage integrator via provider in future release
	integratorType := "github"
	integrators, err := c.integratorService.ListActiveIntegrations(r.Context(), integratorType, teamIds)
	if err != nil {
		c.logger.LogError(err, "Failed to get GitHub app install url", nil)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    integrators,
	})
}
