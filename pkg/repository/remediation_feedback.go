package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/opsmx/ai-guardian-api/pkg/models"
)

// RemediationFeedbackRepository handles database operations for remediation feedback
type RemediationFeedbackRepository struct {
	*BaseRepository
}

// NewRemediationFeedbackRepository creates a new remediation feedback repository
func NewRemediationFeedbackRepository() *RemediationFeedbackRepository {
	return &RemediationFeedbackRepository{
		BaseRepository: NewBaseRepository("remediation_feedback"),
	}
}

// Create creates a new remediation feedback record
func (r *RemediationFeedbackRepository) Create(ctx context.Context, feedback *models.RemediationFeedback) (*models.RemediationFeedback, error) {
	data := map[string]interface{}{
		"id":               uuid.New().String(),
		"remediation_id":   feedback.RemediationID,
		"vulnerability_id": feedback.VulnerabilityID,
	}

	if feedback.Comments != nil {
		data["comments"] = *feedback.Comments
	}
	if feedback.Rating != nil {
		data["rating"] = *feedback.Rating
	}

	id, err := r.BaseRepository.Create(ctx, "remediation_feedback", data)
	if err != nil {
		return nil, err
	}

	// Retrieve the created feedback
	return r.GetByID(ctx, id)
}

// GetByID retrieves a remediation feedback by ID
func (r *RemediationFeedbackRepository) GetByID(ctx context.Context, id string) (*models.RemediationFeedback, error) {
	var feedback models.RemediationFeedback
	err := r.BaseRepository.GetByID(ctx, "remediation_feedback", id, &feedback)
	if err != nil {
		return nil, err
	}
	return &feedback, nil
}

// Update updates a remediation feedback record
func (r *RemediationFeedbackRepository) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.BaseRepository.Update(ctx, "remediation_feedback", id, updates)
}

// Delete deletes a remediation feedback record
func (r *RemediationFeedbackRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.BaseRepository.Delete(ctx, "remediation_feedback", id)
}

// List retrieves remediation feedback records with pagination and filtering
func (r *RemediationFeedbackRepository) List(ctx context.Context, options *QueryOptions) ([]*models.RemediationFeedback, *PaginationResult, error) {
	var feedbacks []*models.RemediationFeedback
	pagination, err := r.BaseRepository.List(ctx, "remediation_feedback", options, &feedbacks)
	if err != nil {
		return nil, nil, err
	}
	return feedbacks, pagination, nil
}

// GetByRemediationID retrieves all feedback for a specific remediation
func (r *RemediationFeedbackRepository) GetByRemediationID(ctx context.Context, remediationID uuid.UUID) ([]*models.RemediationFeedback, error) {
	options := &QueryOptions{
		Filters: map[string]interface{}{
			"remediation_id": remediationID,
		},
		OrderBy:  "created_at",
		OrderDir: "DESC",
	}

	var feedbacks []*models.RemediationFeedback
	_, err := r.BaseRepository.List(ctx, "remediation_feedback", options, &feedbacks)
	if err != nil {
		return nil, err
	}
	return feedbacks, nil
}

// GetByVulnerabilityID retrieves all feedback for a specific vulnerability
func (r *RemediationFeedbackRepository) GetByVulnerabilityID(ctx context.Context, vulnerabilityID uuid.UUID) ([]*models.RemediationFeedback, error) {
	options := &QueryOptions{
		Filters: map[string]interface{}{
			"vulnerability_id": vulnerabilityID,
		},
		OrderBy:  "created_at",
		OrderDir: "DESC",
	}

	var feedbacks []*models.RemediationFeedback
	_, err := r.BaseRepository.List(ctx, "remediation_feedback", options, &feedbacks)
	if err != nil {
		return nil, err
	}
	return feedbacks, nil
}

// Exists checks if a feedback record exists
func (r *RemediationFeedbackRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.BaseRepository.Exists(ctx, "remediation_feedback", id)
}

// Count counts feedback records with filters
func (r *RemediationFeedbackRepository) Count(ctx context.Context, filters map[string]interface{}) (int64, error) {
	return r.BaseRepository.Count(ctx, "remediation_feedback", filters)
}

// GetAverageRatingByRemediationID calculates the average rating for a remediation
func (r *RemediationFeedbackRepository) GetAverageRatingByRemediationID(ctx context.Context, remediationID uuid.UUID) (*float64, error) {
	query := `
		SELECT AVG(rating) 
		FROM remediation_feedback 
		WHERE remediation_id = $1 AND rating IS NOT NULL
	`

	var avgRating *float64
	err := r.db.QueryRow(ctx, query, remediationID).Scan(&avgRating)
	if err != nil {
		r.logger.LogError(err, "Failed to get average rating", map[string]interface{}{
			"remediation_id": remediationID,
		})
		return nil, fmt.Errorf("failed to get average rating: %w", err)
	}

	return avgRating, nil
}

// ValidateRemediationExists checks if a remediation exists
func (r *RemediationFeedbackRepository) ValidateRemediationExists(ctx context.Context, remediationID uuid.UUID) (bool, error) {
	exists, err := r.BaseRepository.Exists(ctx, "remediations", remediationID)
	if err != nil {
		r.logger.LogError(err, "Failed to validate remediation exists", map[string]interface{}{
			"remediation_id": remediationID,
		})
		return false, err
	}
	return exists, nil
}

// ValidateVulnerabilityExists checks if a vulnerability exists
func (r *RemediationFeedbackRepository) ValidateVulnerabilityExists(ctx context.Context, vulnerabilityID uuid.UUID) (bool, error) {
	exists, err := r.BaseRepository.Exists(ctx, "vulnerabilities", vulnerabilityID)
	if err != nil {
		r.logger.LogError(err, "Failed to validate vulnerability exists", map[string]interface{}{
			"vulnerability_id": vulnerabilityID,
		})
		return false, err
	}
	return exists, nil
}
