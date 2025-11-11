package remediation

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/opsmx/ai-guardian-api/pkg/auth/session"
	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/config"
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
			c.logger.LogError(err, err.Error(), nil)
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
			c.logger.LogError(err, err.Error(), nil)
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
		return "", errors.New("Failed to fetch user information")
	}

	// Get email from database
	userEmail := dbUser.Email.String
	if userEmail == "" {
		return "", errors.New("User email not found")
	}

	return userEmail, nil
}
