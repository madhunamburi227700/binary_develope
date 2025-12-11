package webhook

import (
	"encoding/json"
	"net/http"

	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/service"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type WebhookController struct {
	projectService *service.ProjectService
	logger         *utils.ErrorLogger
}

func NewWebhookController() *WebhookController {
	return &WebhookController{
		projectService: service.NewProjectService(),
		logger:         utils.NewErrorLogger("webhook_controller"),
	}
}

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

	c.logger.LogInfo("Webhook received", map[string]interface{}{
		"payload": payload,
	})

	url, err := c.projectService.HandleWebhookRequest(r.Context(), payload)
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
