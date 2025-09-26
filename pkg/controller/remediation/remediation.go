package remediation

import (
	"encoding/json"
	"io"
	"net/http"

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

func (c *RemediationController) SASTRemediation(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var payload service.SASTRemediationRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := c.remediationService.ValidateSASTRequest(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := c.remediationService.SAST(r.Context(), &payload, r.Header, r.URL.Query())
	if err != nil {
		c.logger.LogError(err, err.Error(), nil)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = client.FlushSSE(r.Context(), w, *resp); err != nil {
		c.logger.LogError(err, err.Error(), nil)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func (c *RemediationController) CVERemediation(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var payload service.CVERemediationRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := c.remediationService.ValidateCVERequest(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := c.remediationService.CVE(r.Context(), &payload, r.Header, r.URL.Query())
	if err != nil {
		c.logger.LogError(err, err.Error(), nil)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = client.FlushSSE(r.Context(), w, *resp); err != nil {
		c.logger.LogError(err, err.Error(), nil)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
