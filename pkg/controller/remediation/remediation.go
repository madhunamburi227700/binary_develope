package remediation

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/opsmx/ai-guardian-api/pkg/auth/session"
	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/config"
	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
	"github.com/opsmx/ai-guardian-api/pkg/service"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type RemediationController struct {
	remediationService *service.RemediationService
	userRepo           *repository.UserRepository
	logger             *utils.ErrorLogger
}

func NewRemediationsController() *RemediationController {
	return &RemediationController{
		remediationService: service.NewRemediationService(),
		userRepo:           repository.NewUserRepository(),
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

	projectId := r.URL.Query().Get("projectId")
	if strings.TrimSpace(projectId) == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "projectId cannot be blank")
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

	if config.GetNotificationEnabled() {
		userEmail, err := fetchUserEmail(r, c.userRepo, c.logger)
		if err != nil {
			c.logger.LogError(err, "Failed to fetch email for the user", nil)
		}
		payload.UserEmail = userEmail
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
// @Param   request body service.CVERemediationRequest true "CVE findings data"
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

	projectId := r.URL.Query().Get("projectId")
	if strings.TrimSpace(projectId) == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "projectId cannot be blank")
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

	if config.GetNotificationEnabled() {
		userEmail, err := fetchUserEmail(r, c.userRepo, c.logger)
		if err != nil {
			// Just log here
			c.logger.LogError(err, "Failed to fetch email for the user", nil)
		}
		payload.UserEmail = userEmail
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

func fetchUserEmail(r *http.Request, userRepo *repository.UserRepository, logger *utils.ErrorLogger) (string, error) {
	// Get user from session
	sessionUser := session.GetSessionExists(r)
	if sessionUser == nil {
		return "", errors.New("Authentication required")
	}

	// Fetch user email from database
	dbUser, err := userRepo.GetByProviderUserID(r.Context(), sessionUser.Username)
	if err != nil {
		logger.LogError(err, "Failed to fetch user information", nil)
		return "", errors.New("Failed to fetch user information")
	}

	// Get email from database
	userEmail := dbUser.Email.String
	if userEmail == "" {
		return "", errors.New("User email not found")
	}

	return userEmail, nil
}

func (c *RemediationController) Conversation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	remId, ok := vars["id"]
	if !ok {
		http.Error(w, "Remediation ID is required", http.StatusBadRequest)
		return
	}

	resp, err := c.remediationService.Conversation(r.Context(), remId)
	if err != nil {
		c.logger.LogError(err, err.Error(), nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    resp,
	})

}

func (c *RemediationController) NLI(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var payload map[string]interface{}
	err = json.Unmarshal(body, &payload)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := c.remediationService.NLI(r.Context(), payload, r.Header, r.URL.Query())
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

// CSPMRemediation handles CSPM (Cloud Security Posture Management) remediation
// @Summary Process CSPM remediation
// @Description Processes CSPM findings and provides remediation suggestions
// @Tags Remediation
// @Accept  json
// @Produce  json
// @Param   request body service.CSPMRemediationRequest true "CSPM remediation request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/remediation/cspm [post]
func (c *RemediationController) CSPMRemediation(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		c.logger.LogError(err, "failed to parse request body", nil)
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	queryContext := r.URL.Query().Get("context")
	projectId := r.URL.Query().Get("projectId")
	commitsha := r.URL.Query().Get("commitsha")
	if queryContext != models.RemediationContextCloud && (strings.TrimSpace(commitsha) == "" || strings.TrimSpace(projectId) == "") {
		utils.SendErrorResponse(w, http.StatusBadRequest, "commitsha and projectId cannot be blank")
		return
	}

	var payload service.CSPMRemediationRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if config.GetNotificationEnabled() {
		userEmail, err := fetchUserEmail(r, c.userRepo, c.logger)
		if err != nil {
			c.logger.LogError(err, "Failed to fetch email for the user", nil)
		}
		payload.UserEmail = userEmail
	}

	resp, err := c.remediationService.CSPM(r.Context(), &payload, projectId, r.Header, r.URL.Query(), commitsha, queryContext)
	if err != nil {
		c.logger.LogError(err, err.Error(), nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if resp == nil {
		c.logger.LogError(errors.New("failed to get remediation response"), "failed to get remediation response", nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to get remediation response")
		return
	}

	if err = client.FlushSSE(r.Context(), w, *resp); err != nil {
		c.logger.LogError(err, err.Error(), nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
}
