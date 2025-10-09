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
// @Param accountId query string true "Account ID (e.g., 0x3b32)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/integrations/github/details [get]
func (c *IntegratorController) GetIntegrationsGithubDetails(w http.ResponseWriter, r *http.Request) {
	integrationId := r.URL.Query().Get("accountId")
	if integrationId == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Account ID is required")
		return
	}

	integrationName := r.URL.Query().Get("integrationName")
	if integrationName == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Integration name is required")
		return
	}

	hubID := r.URL.Query().Get("hubId")
	if hubID == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Hub ID is required")
		return
	}

	queryParams := map[string]string{
		// automated param from UI
		// "accountId":"account-id",

		// default params
		// ssd will look for repos from installation id based token
		// if orgName is blank
		"platform":  "github", // automate platform in future release
		"scanLevel": "repository",
		"type":      "user",
		"orgName":   "",
	}
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	details, err := c.integratorService.GetGithubIntegrationsDetails(r.Context(), queryParams, integrationId, integrationName, hubID)
	if err != nil {
		c.logger.LogError(err, "Failed to get github integrations details", nil)
		utils.SendErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    details,
	})
}

// failed to get repo branch list: failed to get repo branch list: status 500, body: {"status":"error","code":500,"error":"unable to fetch github token, error while receiving token response: HTTP 400: {\"success\":false,\"error\":\"failed to create installation token: POST https://api.github.com/app/installations/88096131/access_tokens: 404 Not Found []\",\"code\":\"GITHUB_API_ERROR\"}\n"}
// github app is not available, please reinstall the app

// ListIntegrations lists all active integrations
// @Summary List active integrations
// @Description Returns a list of all active integrations
// @Tags Integrations
// @Accept  json
// @Produce  json
// @Param   teamIds query string false "Comma-separated list of team IDs to filter by"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/integrations [get]
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

func (c *IntegratorController) DeleteIntegration(w http.ResponseWriter, r *http.Request) {
	integrationId := r.URL.Query().Get("integrationId")
	if integrationId == "" {
		http.Error(w, "Integration ID is required", http.StatusBadRequest)
		return
	}

	hubID := r.URL.Query().Get("hubID")
	if hubID == "" {
		http.Error(w, "Hub ID is required", http.StatusBadRequest)
		return
	}

	integrationName := r.URL.Query().Get("integrationName")
	if integrationName == "" {
		http.Error(w, "Integration name is required", http.StatusBadRequest)
		return
	}

	err := c.integratorService.DeleteIntegration(r.Context(), integrationId, integrationName, hubID)
	if err != nil {
		c.logger.LogError(err, "Failed to delete integration", nil)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	utils.SendSuccessResponseWithNoData(w, "Integration deleted successfully")
}
