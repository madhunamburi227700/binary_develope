package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// RemediationFeedbackService handles business logic for remediation feedback
type RemediationFeedbackService struct {
	repo   *repository.RemediationFeedbackRepository
	logger *utils.ErrorLogger
}

// NewRemediationFeedbackService creates a new remediation feedback service
func NewRemediationFeedbackService() *RemediationFeedbackService {
	return &RemediationFeedbackService{
		repo:   repository.NewRemediationFeedbackRepository(),
		logger: utils.NewErrorLogger("remediation_feedback_service"),
	}
}

// CreateFeedbackRequest represents a request to create feedback
type CreateFeedbackRequest struct {
	RemediationID   uuid.UUID `json:"remediation_id" validate:"required"`
	VulnerabilityID uuid.UUID `json:"vulnerability_id" validate:"required"`
	Comments        *string   `json:"comments"`
	Rating          *float64  `json:"rating" validate:"omitempty,min=0,max=5"`
}

// UpdateFeedbackRequest represents a request to update feedback
type UpdateFeedbackRequest struct {
	Comments *string  `json:"comments"`
	Rating   *float64 `json:"rating" validate:"omitempty,min=0,max=5"`
}

// FeedbackResponse represents the response for feedback operations
type FeedbackResponse struct {
	ID              uuid.UUID `json:"id"`
	RemediationID   uuid.UUID `json:"remediation_id"`
	VulnerabilityID uuid.UUID `json:"vulnerability_id"`
	Comments        *string   `json:"comments"`
	Rating          *float64  `json:"rating"`
	CreatedAt       *string   `json:"created_at"`
}

// FeedbackListResponse represents the response for listing feedback
type FeedbackListResponse struct {
	Feedbacks  []*FeedbackResponse          `json:"feedbacks"`
	Pagination *repository.PaginationResult `json:"pagination,omitempty"`
}

// FeedbackStatsResponse represents statistics for feedback
type FeedbackStatsResponse struct {
	TotalFeedbacks int      `json:"total_feedbacks"`
	AverageRating  *float64 `json:"average_rating"`
}

// CreateFeedback creates a new remediation feedback
func (s *RemediationFeedbackService) CreateFeedback(ctx context.Context, req *CreateFeedbackRequest) (*FeedbackResponse, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Validate that remediation exists
	remExists, err := s.repo.ValidateRemediationExists(ctx, req.RemediationID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate remediation: %w", err)
	}
	if !remExists {
		return nil, fmt.Errorf("remediation with ID %s not found", req.RemediationID)
	}

	// Validate that vulnerability exists
	vulnExists, err := s.repo.ValidateVulnerabilityExists(ctx, req.VulnerabilityID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate vulnerability: %w", err)
	}
	if !vulnExists {
		return nil, fmt.Errorf("vulnerability with ID %s not found", req.VulnerabilityID)
	}

	// Create feedback model
	feedback := &models.RemediationFeedback{
		RemediationID:   req.RemediationID,
		VulnerabilityID: req.VulnerabilityID,
		Comments:        req.Comments,
		Rating:          req.Rating,
	}

	// Create in database
	createdFeedback, err := s.repo.Create(ctx, feedback)
	if err != nil {
		s.logger.LogError(err, "Failed to create feedback", map[string]interface{}{
			"remediation_id":   req.RemediationID,
			"vulnerability_id": req.VulnerabilityID,
		})
		return nil, fmt.Errorf("failed to create feedback: %w", err)
	}

	s.logger.LogInfo("Feedback created successfully", map[string]interface{}{
		"feedback_id":      createdFeedback.ID,
		"remediation_id":   createdFeedback.RemediationID,
		"vulnerability_id": createdFeedback.VulnerabilityID,
	})

	return s.toFeedbackResponse(createdFeedback), nil
}

// GetFeedbackByID retrieves feedback by ID
func (s *RemediationFeedbackService) GetFeedbackByID(ctx context.Context, id uuid.UUID) (*FeedbackResponse, error) {
	feedback, err := s.repo.GetByID(ctx, id.String())
	if err != nil {
		s.logger.LogError(err, "Failed to get feedback by ID", map[string]interface{}{
			"feedback_id": id,
		})
		return nil, fmt.Errorf("failed to get feedback: %w", err)
	}

	return s.toFeedbackResponse(feedback), nil
}

// UpdateFeedback updates an existing feedback
func (s *RemediationFeedbackService) UpdateFeedback(ctx context.Context, id uuid.UUID, req *UpdateFeedbackRequest) (*FeedbackResponse, error) {
	// Validate request
	if err := s.validateUpdateRequest(req); err != nil {
		return nil, err
	}

	// Check if feedback exists
	exists, err := s.repo.Exists(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check if feedback exists: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("feedback with ID %s not found", id)
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.Comments != nil {
		updates["comments"] = *req.Comments
	}
	if req.Rating != nil {
		updates["rating"] = *req.Rating
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates provided")
	}

	// Update in database
	err = s.repo.Update(ctx, id, updates)
	if err != nil {
		s.logger.LogError(err, "Failed to update feedback", map[string]interface{}{
			"feedback_id": id,
			"updates":     updates,
		})
		return nil, fmt.Errorf("failed to update feedback: %w", err)
	}

	s.logger.LogInfo("Feedback updated successfully", map[string]interface{}{
		"feedback_id": id,
	})

	// Retrieve updated feedback
	return s.GetFeedbackByID(ctx, id)
}

// DeleteFeedback deletes a feedback record
func (s *RemediationFeedbackService) DeleteFeedback(ctx context.Context, id uuid.UUID) error {
	// Check if feedback exists
	exists, err := s.repo.Exists(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check if feedback exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("feedback with ID %s not found", id)
	}

	// Delete from database
	err = s.repo.Delete(ctx, id)
	if err != nil {
		s.logger.LogError(err, "Failed to delete feedback", map[string]interface{}{
			"feedback_id": id,
		})
		return fmt.Errorf("failed to delete feedback: %w", err)
	}

	s.logger.LogInfo("Feedback deleted successfully", map[string]interface{}{
		"feedback_id": id,
	})

	return nil
}

// ListFeedbacks retrieves feedback with pagination and filtering
func (s *RemediationFeedbackService) ListFeedbacks(ctx context.Context, remediationID *uuid.UUID, vulnerabilityID *uuid.UUID, page, pageSize int) (*FeedbackListResponse, error) {
	// Build filters
	filters := make(map[string]interface{})
	if remediationID != nil {
		filters["remediation_id"] = *remediationID
	}
	if vulnerabilityID != nil {
		filters["vulnerability_id"] = *vulnerabilityID
	}

	// Set defaults
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// Build query options
	options := &repository.QueryOptions{
		Filters:  filters,
		Limit:    pageSize,
		Offset:   (page - 1) * pageSize,
		OrderBy:  "created_at",
		OrderDir: "DESC",
	}

	// Get feedbacks
	feedbacks, pagination, err := s.repo.List(ctx, options)
	if err != nil {
		s.logger.LogError(err, "Failed to list feedbacks", map[string]interface{}{
			"filters": filters,
		})
		return nil, fmt.Errorf("failed to list feedbacks: %w", err)
	}

	// Convert to response
	responses := make([]*FeedbackResponse, 0, len(feedbacks))
	for _, feedback := range feedbacks {
		responses = append(responses, s.toFeedbackResponse(feedback))
	}

	return &FeedbackListResponse{
		Feedbacks:  responses,
		Pagination: pagination,
	}, nil
}

// GetFeedbacksByRemediationID retrieves all feedback for a remediation
func (s *RemediationFeedbackService) GetFeedbacksByRemediationID(ctx context.Context, remediationID uuid.UUID) ([]*FeedbackResponse, error) {
	feedbacks, err := s.repo.GetByRemediationID(ctx, remediationID)
	if err != nil {
		s.logger.LogError(err, "Failed to get feedbacks by remediation ID", map[string]interface{}{
			"remediation_id": remediationID,
		})
		return nil, fmt.Errorf("failed to get feedbacks: %w", err)
	}

	responses := make([]*FeedbackResponse, 0, len(feedbacks))
	for _, feedback := range feedbacks {
		responses = append(responses, s.toFeedbackResponse(feedback))
	}

	return responses, nil
}

// GetFeedbacksByVulnerabilityID retrieves all feedback for a vulnerability
func (s *RemediationFeedbackService) GetFeedbacksByVulnerabilityID(ctx context.Context, vulnerabilityID uuid.UUID) ([]*FeedbackResponse, error) {
	feedbacks, err := s.repo.GetByVulnerabilityID(ctx, vulnerabilityID)
	if err != nil {
		s.logger.LogError(err, "Failed to get feedbacks by vulnerability ID", map[string]interface{}{
			"vulnerability_id": vulnerabilityID,
		})
		return nil, fmt.Errorf("failed to get feedbacks: %w", err)
	}

	responses := make([]*FeedbackResponse, 0, len(feedbacks))
	for _, feedback := range feedbacks {
		responses = append(responses, s.toFeedbackResponse(feedback))
	}

	return responses, nil
}

// GetFeedbackStats retrieves statistics for feedback
func (s *RemediationFeedbackService) GetFeedbackStats(ctx context.Context, remediationID uuid.UUID) (*FeedbackStatsResponse, error) {
	// Get count
	count, err := s.repo.Count(ctx, map[string]interface{}{
		"remediation_id": remediationID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get feedback count: %w", err)
	}

	// Get average rating
	avgRating, err := s.repo.GetAverageRatingByRemediationID(ctx, remediationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get average rating: %w", err)
	}

	return &FeedbackStatsResponse{
		TotalFeedbacks: int(count),
		AverageRating:  avgRating,
	}, nil
}

// Validation methods
func (s *RemediationFeedbackService) validateCreateRequest(req *CreateFeedbackRequest) error {
	if req.RemediationID == uuid.Nil {
		return fmt.Errorf("remediation_id is required")
	}
	if req.VulnerabilityID == uuid.Nil {
		return fmt.Errorf("vulnerability_id is required")
	}
	if req.Rating != nil {
		if *req.Rating < 0 || *req.Rating > 5 {
			return fmt.Errorf("rating must be between 0 and 5")
		}
	}
	return nil
}

func (s *RemediationFeedbackService) validateUpdateRequest(req *UpdateFeedbackRequest) error {
	if req.Rating != nil {
		if *req.Rating < 0 || *req.Rating > 5 {
			return fmt.Errorf("rating must be between 0 and 5")
		}
	}
	return nil
}

// Helper methods
func (s *RemediationFeedbackService) toFeedbackResponse(feedback *models.RemediationFeedback) *FeedbackResponse {
	response := &FeedbackResponse{
		ID:              feedback.ID,
		RemediationID:   feedback.RemediationID,
		VulnerabilityID: feedback.VulnerabilityID,
		Comments:        feedback.Comments,
		Rating:          feedback.Rating,
	}

	if feedback.CreatedAt != nil {
		createdAt := feedback.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		response.CreatedAt = &createdAt
	}

	return response
}
