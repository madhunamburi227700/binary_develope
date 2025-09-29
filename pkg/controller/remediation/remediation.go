package remediation

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/service"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type RemediationController struct {
	remediationService *service.RemediationService
	logger             *utils.ErrorLogger
}

func NewRemediationsController() *RemediationController {
	return &RemediationController{
		remediationService: service.NewRemediationService(),
		logger:             utils.NewErrorLogger("remediations_controller"),
	}
}

// SASTRemediation handles SAST (Static Application Security Testing) remediation
// @Summary Process SAST remediation
// @Description Processes SAST findings and provides remediation suggestions
// @Tags Remediation
// @Accept  json
// @Produce  json
// @Param   request body service.SASTRemediationRequest true "SAST findings data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/remediation/sast [post]
func (c *RemediationController) SASTRemediation(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	projectId := r.URL.Query().Get("project_id")
	if strings.TrimSpace(projectId) == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var payload service.SASTRemediationRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.remediationService.ValidateSASTRequest(&payload); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := c.remediationService.SAST(r.Context(), &payload, projectId, r.Header, r.URL.Query())
	if err != nil {
		c.logger.LogError(err, err.Error(), nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = client.FlushSSE(r.Context(), w, *resp); err != nil {
		c.logger.LogError(err, err.Error(), nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

}

// CVERemediation handles CVE (Common Vulnerabilities and Exposures) remediation
// @Summary Process CVE remediation
// @Description Processes CVE findings and provides remediation suggestions
// @Tags Remediation
// @Accept  json
// @Produce  json
// @Param   request body client.CVERemediationRequest true "CVE findings data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/remediation/cve [post]
func (c *RemediationController) CVERemediation(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	projectId := r.URL.Query().Get("project_id")
	if strings.TrimSpace(projectId) == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "project_id is required")
		return
	}

	var payload service.CVERemediationRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.remediationService.ValidateCVERequest(&payload); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := c.remediationService.CVE(r.Context(), &payload, projectId, r.Header, r.URL.Query())
	if err != nil {
		c.logger.LogError(err, err.Error(), nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = client.FlushSSE(r.Context(), w, *resp); err != nil {
		c.logger.LogError(err, err.Error(), nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
}
