package repository

import (
	"context"

	"github.com/opsmx/ai-guardian-api/pkg/models"
)

// RemediationRepository handles remediation-related database operations
type RemediationRepository struct {
	*BaseRepository
}

// NewRemediationRepository creates a new project repository
func NewRemediationRepository() *RemediationRepository {
	return &RemediationRepository{
		BaseRepository: NewBaseRepository("remediations"),
	}
}

// List retrieves projects with pagination and filtering
func (r *RemediationRepository) List(ctx context.Context, options *QueryOptions) (*QueryResult[models.Remediation], error) {
	var remediations []models.Remediation

	// Add default ordering if not specified
	if options.OrderBy == "" {
		options.OrderBy = "created_at"
		options.OrderDir = "DESC"
	}

	pagination, err := r.BaseRepository.List(ctx, "remediations", options, &remediations)
	if err != nil {
		return nil, err
	}

	return &QueryResult[models.Remediation]{
		Data:       remediations,
		Pagination: pagination,
	}, nil
}
