package remediation_feedback

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/opsmx/ai-guardian-api/pkg/service"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// RemediationFeedbackController handles HTTP requests for remediation feedback
type RemediationFeedbackController struct {
	service *service.RemediationFeedbackService
	logger  *utils.ErrorLogger
}

// NewRemediationFeedbackController creates a new remediation feedback controller
func NewRemediationFeedbackController() *RemediationFeedbackController {
	return &RemediationFeedbackController{
		service: service.NewRemediationFeedbackService(),
		logger:  utils.NewErrorLogger("remediation_feedback_controller"),
	}
}

// CreateFeedback creates a new remediation feedback
// @Summary Create remediation feedback
// @Description Creates a new feedback entry for a remediation. Any authenticated user can create feedback.
// @Tags RemediationFeedback
// @Accept  json
// @Produce  json
// @Param   request body service.CreateFeedbackRequest true "Feedback creation details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/remediation-feedback [post]
func (c *RemediationFeedbackController) CreateFeedback(w http.ResponseWriter, r *http.Request) {
	var req service.CreateFeedbackRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.logger.LogWarning("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	feedback, err := c.service.CreateFeedback(r.Context(), &req)
	if err != nil {
		c.logger.LogError(err, "Failed to create feedback", map[string]interface{}{
			"request": req,
		})
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    feedback,
	})
}

// GetFeedback retrieves a feedback by ID
// @Summary Get feedback by ID (Internal)
// @Description Returns the feedback with the specified ID. Internal use only - requires authentication.
// @Tags RemediationFeedback-Internal
// @Accept  json
// @Produce  json
// @Param   id path string true "Feedback ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid feedback ID"
// @Failure 404 {object} map[string]string "Feedback not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/remediation-feedback/{id} [get]
func (c *RemediationFeedbackController) GetFeedback(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	feedbackIDStr, ok := vars["id"]
	if !ok {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Feedback ID is required")
		return
	}

	feedbackID, err := uuid.Parse(feedbackIDStr)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid feedback ID")
		return
	}

	feedback, err := c.service.GetFeedbackByID(r.Context(), feedbackID)
	if err != nil {
		c.logger.LogError(err, "Failed to get feedback", map[string]interface{}{
			"feedback_id": feedbackID,
		})
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    feedback,
	})
}

// UpdateFeedback updates an existing feedback
// @Summary Update feedback (Internal)
// @Description Updates an existing feedback entry. Internal use only - requires authentication.
// @Tags RemediationFeedback-Internal
// @Accept  json
// @Produce  json
// @Param   id path string true "Feedback ID"
// @Param   request body service.UpdateFeedbackRequest true "Feedback update details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 404 {object} map[string]string "Feedback not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/remediation-feedback/{id} [put]
func (c *RemediationFeedbackController) UpdateFeedback(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	feedbackIDStr, ok := vars["id"]
	if !ok {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Feedback ID is required")
		return
	}

	feedbackID, err := uuid.Parse(feedbackIDStr)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid feedback ID")
		return
	}

	var req service.UpdateFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.logger.LogWarning("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	feedback, err := c.service.UpdateFeedback(r.Context(), feedbackID, &req)
	if err != nil {
		c.logger.LogError(err, "Failed to update feedback", map[string]interface{}{
			"feedback_id": feedbackID,
			"request":     req,
		})
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    feedback,
	})
}

// DeleteFeedback deletes a feedback record
// @Summary Delete feedback (Internal)
// @Description Deletes a feedback entry. Internal use only - requires authentication.
// @Tags RemediationFeedback-Internal
// @Accept  json
// @Produce  json
// @Param   id path string true "Feedback ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid feedback ID"
// @Failure 404 {object} map[string]string "Feedback not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/remediation-feedback/{id} [delete]
func (c *RemediationFeedbackController) DeleteFeedback(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	feedbackIDStr, ok := vars["id"]
	if !ok {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Feedback ID is required")
		return
	}

	feedbackID, err := uuid.Parse(feedbackIDStr)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid feedback ID")
		return
	}

	err = c.service.DeleteFeedback(r.Context(), feedbackID)
	if err != nil {
		c.logger.LogError(err, "Failed to delete feedback", map[string]interface{}{
			"feedback_id": feedbackID,
		})
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Feedback deleted successfully",
	})
}

// ListFeedbacks retrieves feedback with pagination and filtering
// @Summary List feedbacks (Internal)
// @Description Returns a list of feedbacks with optional filtering. Internal use only - requires authentication.
// @Tags RemediationFeedback-Internal
// @Accept  json
// @Produce  json
// @Param   remediation_id query string false "Filter by remediation ID"
// @Param   vulnerability_id query string false "Filter by vulnerability ID"
// @Param   page query int false "Page number (default: 1)"
// @Param   page_size query int false "Page size (default: 10)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid query parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/remediation-feedback [get]
func (c *RemediationFeedbackController) ListFeedbacks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Parse optional filters
	var remediationID *uuid.UUID
	if remIDStr := query.Get("remediation_id"); remIDStr != "" {
		remID, err := uuid.Parse(remIDStr)
		if err != nil {
			utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid remediation_id")
			return
		}
		remediationID = &remID
	}

	var vulnerabilityID *uuid.UUID
	if vulnIDStr := query.Get("vulnerability_id"); vulnIDStr != "" {
		vulnID, err := uuid.Parse(vulnIDStr)
		if err != nil {
			utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid vulnerability_id")
			return
		}
		vulnerabilityID = &vulnID
	}

	// Parse pagination
	page := 1
	if pageStr := query.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 10
	if pageSizeStr := query.Get("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	result, err := c.service.ListFeedbacks(r.Context(), remediationID, vulnerabilityID, page, pageSize)
	if err != nil {
		c.logger.LogError(err, "Failed to list feedbacks", map[string]interface{}{
			"remediation_id":   remediationID,
			"vulnerability_id": vulnerabilityID,
		})
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"data":       result.Feedbacks,
		"pagination": result.Pagination,
	})
}

// GetFeedbacksByRemediationID retrieves all feedback for a remediation
// @Summary Get feedbacks by remediation ID (Internal)
// @Description Returns all feedbacks for a specific remediation. Internal use only - requires authentication.
// @Tags RemediationFeedback-Internal
// @Accept  json
// @Produce  json
// @Param   remediation_id path string true "Remediation ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid remediation ID"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/remediation-feedback/remediation/{remediation_id} [get]
func (c *RemediationFeedbackController) GetFeedbacksByRemediationID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	remediationIDStr, ok := vars["remediation_id"]
	if !ok {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Remediation ID is required")
		return
	}

	remediationID, err := uuid.Parse(remediationIDStr)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid remediation ID")
		return
	}

	feedbacks, err := c.service.GetFeedbacksByRemediationID(r.Context(), remediationID)
	if err != nil {
		c.logger.LogError(err, "Failed to get feedbacks by remediation ID", map[string]interface{}{
			"remediation_id": remediationID,
		})
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    feedbacks,
	})
}

// GetFeedbacksByVulnerabilityID retrieves all feedback for a vulnerability
// @Summary Get feedbacks by vulnerability ID (Internal)
// @Description Returns all feedbacks for a specific vulnerability. Internal use only - requires authentication.
// @Tags RemediationFeedback-Internal
// @Accept  json
// @Produce  json
// @Param   vulnerability_id path string true "Vulnerability ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid vulnerability ID"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/remediation-feedback/vulnerability/{vulnerability_id} [get]
func (c *RemediationFeedbackController) GetFeedbacksByVulnerabilityID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vulnerabilityIDStr, ok := vars["vulnerability_id"]
	if !ok {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Vulnerability ID is required")
		return
	}

	vulnerabilityID, err := uuid.Parse(vulnerabilityIDStr)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid vulnerability ID")
		return
	}

	feedbacks, err := c.service.GetFeedbacksByVulnerabilityID(r.Context(), vulnerabilityID)
	if err != nil {
		c.logger.LogError(err, "Failed to get feedbacks by vulnerability ID", map[string]interface{}{
			"vulnerability_id": vulnerabilityID,
		})
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    feedbacks,
	})
}

// GetFeedbackStats retrieves statistics for feedback
// @Summary Get feedback statistics (Internal)
// @Description Returns statistics for feedback of a specific remediation. Internal use only - requires authentication.
// @Tags RemediationFeedback-Internal
// @Accept  json
// @Produce  json
// @Param   remediation_id path string true "Remediation ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid remediation ID"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/remediation-feedback/stats/{remediation_id} [get]
func (c *RemediationFeedbackController) GetFeedbackStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	remediationIDStr, ok := vars["remediation_id"]
	if !ok {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Remediation ID is required")
		return
	}

	remediationID, err := uuid.Parse(remediationIDStr)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid remediation ID")
		return
	}

	stats, err := c.service.GetFeedbackStats(r.Context(), remediationID)
	if err != nil {
		c.logger.LogError(err, "Failed to get feedback stats", map[string]interface{}{
			"remediation_id": remediationID,
		})
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    stats,
	})
}
