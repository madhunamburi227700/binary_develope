package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/opsmx/ai-guardian-api/pkg/service"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type WebhookController struct {
	projectService  *service.ProjectService
	workflowService *service.WorkflowSetupService
	logger          *utils.ErrorLogger
}

func NewWebhookController() *WebhookController {
	return &WebhookController{
		projectService:  service.NewProjectService(),
		workflowService: service.NewWorkflowSetupService(),
		logger:          utils.NewErrorLogger("webhook_controller"),
	}
}

func (c *WebhookController) SetupWorkflow(w http.ResponseWriter, r *http.Request) {
	var req service.SetupWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.logger.LogError(err, "Failed to decode request", map[string]interface{}{
			"error": err.Error(), "request": req,
		})
		utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := validateWorkflowSetupRequest(req); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Setup workflow
	result, err := c.workflowService.SetupWorkflow(r.Context(), req)
	if err != nil {
		c.logger.LogError(err, "Failed to setup workflow", nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return success response
	utils.SendSuccessResponse(w, result, "Workflow setup successfully")
}

func (c *WebhookController) CheckWorkflowStatus(w http.ResponseWriter, r *http.Request) {
	var req service.SetupWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.logger.LogError(err, "Failed to decode request", map[string]interface{}{
			"error": err.Error(), "request": req,
		})
		utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := validateWorkflowSetupRequest(req); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check workflow status
	status, err := c.workflowService.CheckWorkflowStatus(r.Context(), req)
	if err != nil {
		c.logger.LogError(err, "Failed to check workflow status", nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return success response
	utils.SendSuccessResponse(w, status, "Workflow status fetched successfully")
}

// validate workflow setup request
func validateWorkflowSetupRequest(req service.SetupWorkflowRequest) error {
	if req.IntegrationID == "" {
		return fmt.Errorf("integration_id is required")
	}
	if req.Repository == "" {
		return fmt.Errorf("repository is required")
	}
	if req.Branch == "" {
		return fmt.Errorf("branch is required")
	}
	if req.HubID == "" {
		return fmt.Errorf("hub_id is required")
	}
	return nil
}
