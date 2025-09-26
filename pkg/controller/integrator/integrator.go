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

// CreateGitHubIntegration
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
