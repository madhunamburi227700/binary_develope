package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/service"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type WebhookController struct {
	projectService  *service.ProjectService
	workflowService *service.WorkflowService
	logger          *utils.ErrorLogger
}

func NewWebhookController() *WebhookController {
	return &WebhookController{
		projectService:  service.NewProjectService(),
		workflowService: service.NewWorkflowService(),
		logger:          utils.NewErrorLogger("webhook_controller"),
	}
}

// SetupWorkflow creates GitHub Actions workflow in a repo.
// @Summary Setup GitHub workflow
// @Description Sets up GitHub Actions workflow for AI Guardian PR scanning.
// @Tags Integrations
// @Accept json
// @Produce json
// @Param request body service.SetupWorkflowRequest true "Workflow setup request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/integrations/github/setup/workflow [post]
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

// CheckWorkflowStatus checks workflow status in a repo.
// @Summary Check GitHub workflow status
// @Description Checks whether the GitHub Actions workflow is set up and returns status.
// @Tags Integrations
// @Accept json
// @Produce json
// @Param request body service.SetupWorkflowRequest true "Workflow status request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/integrations/github/check/workflow [post]
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

// HandleWebhook receives GitHub Actions webhooks.
// @Summary GitHub Actions webhook
// @Description Receives GitHub Actions webhook and starts PR scanning workflow.
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param request body models.WebhookRequest true "Webhook payload"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Security ApiKeyAuth
// @Router /webhooks/github/actions [post]
func (c *WebhookController) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	var payload models.WebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate request
	if err := c.projectService.ValidateWebhookRequest(&payload); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// TODO: Remove r.Host once the endpoint is deprecated
	url, err := c.workflowService.HandleWebhookRequest(r.Context(), payload, r.Host)
	if err != nil {
		sendWebhookResponse(w, url, "Please register repo and branch to scan and remediate vulnerabilities using AI Guardian PR scan feature.", "error")
		return
	}

	// send success response
	sendWebhookResponse(w, url, "PR scanning started. Any new vulnerability will be reported shortly in the PR comments with links to remediate it.", "success")
}

// webhook response format
func sendWebhookResponse(w http.ResponseWriter, url, message, status string) {
	utils.SendWebhookResponse(w, message, status, url)
}
