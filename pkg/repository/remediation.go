package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

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
func (r *RemediationRepository) List(ctx context.Context, options *QueryOptions) (*QueryResult[*models.Remediation], error) {
	var remediations []*models.Remediation

	// Add default ordering if not specified
	if options.OrderBy == "" {
		options.OrderBy = "created_at"
		options.OrderDir = "DESC"
	}

	pagination, err := r.BaseRepository.List(ctx, "remediations", options, &remediations)
	if err != nil {
		return nil, err
	}

	return &QueryResult[*models.Remediation]{
		Data:       remediations,
		Pagination: pagination,
	}, nil
}

// HubRemediation represents a remediation record joined with vulnerability/scan/project details
type HubRemediation struct {
	// Remediation Data
	RemediationID   string    `json:"remediation_id"`
	VulnerabilityID string    `json:"vulnerability_id"`
	Status          string    `json:"status"`
	PRLink          *string   `json:"pr_link,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Vulnerability Data
	VulnerabilityName string  `json:"vulnerability_name"`
	VulnerabilityType string  `json:"vulnerability_type"`
	Package           *string `json:"package,omitempty"`
	Description       *string `json:"description"`
	Severity          string  `json:"severity"`

	// Scan & Project Data
	HubID        string  `json:"hub_id"`
	ProjectID    string  `json:"project_id"`
	ScanID       string  `json:"scan_id"`
	Organisation *string `json:"organisation"`
	Repository   *string `json:"repository"`
	Branch       *string `json:"branch"`

	// Metadata
	TotalCount int `json:"total_count"`
}

func (r *RemediationRepository) GetRemediationsForHub(ctx context.Context, hubID string, page, pageSize int) ([]HubRemediation, int, error) {

	query, args := r.buildRemediationsQuery(hubID, page, pageSize)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var (
		results    = []HubRemediation{}
		totalCount = 0
	)

	for rows.Next() {
		var rm HubRemediation
		var prLink, pkg, description, organisation, repository, branch sql.NullString

		err := rows.Scan(
			&rm.RemediationID,
			&rm.VulnerabilityID,
			&rm.Status,
			&prLink,
			&rm.CreatedAt,
			&rm.UpdatedAt,
			&rm.VulnerabilityName,
			&rm.VulnerabilityType,
			&pkg,
			&description,
			&rm.Severity,
			&rm.HubID,
			&rm.ProjectID,
			&rm.ScanID,
			&repository,
			&branch,
			&organisation,
			&rm.TotalCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("row scan failed: %w", err)
		}

		// Map Nullable types to pointers
		if prLink.Valid {
			rm.PRLink = &prLink.String
		}
		if pkg.Valid {
			rm.Package = &pkg.String
		}
		if description.Valid {
			rm.Description = &description.String
		}
		if organisation.Valid {
			rm.Organisation = &organisation.String
		}
		if repository.Valid {
			rm.Repository = &repository.String
		}
		if branch.Valid {
			rm.Branch = &branch.String
		}

		totalCount = rm.TotalCount
		results = append(results, rm)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	return results, totalCount, nil
}

func (r *RemediationRepository) buildRemediationsQuery(
	hubID string,
	page, pageSize int,
) (string, []interface{}) {

	limit := pageSize
	offset := page * pageSize

	query := `
		SELECT
			-- Remediation Data
			r.id AS remediation_id,
			r.vulnerability_id,
			r.status,
			r.pr_link,
			r.created_at,
			r.updated_at,
			
			-- Vulnerability Data
			v.name,
			v.scan_type,
			v.package,
			v.description,
			v.severity,
			
			-- Scan & Project Data
			s.hub_id,
			s.project_id,
			s.id AS scan_id,
			s.repository,
			s.branch,
			p.organisation,
			COUNT(*) OVER() AS total_count
		FROM scans s
		JOIN vulnerabilities v 
			ON v.scan_id = s.id
		JOIN remediations r 
			ON r.vulnerability_id = v.id
		JOIN projects p 
			ON p.id = s.project_id
		WHERE s.hub_id = $1
		AND r.is_active = TRUE
		ORDER BY r.updated_at DESC
		LIMIT $2 OFFSET $3;
	`

	args := []interface{}{
		hubID,
		limit,
		offset,
	}

	return query, args
}

func (r *RemediationRepository) GetConversation(ctx context.Context, remId string) ([]interface{}, error) {

	tx, err := r.NewTransaction(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var rawStrings []string

	err = tx.QueryRow(ctx, `
        SELECT conversation
        FROM remediations
        WHERE id = $1
    `, remId).Scan(&rawStrings)
	if err != nil {
		return nil, err
	}

	var unmarshalledData []interface{}

	// Maybe rectify this from the source
	for _, s := range rawStrings {
		var item interface{}
		// Unmarshal each JSON string into an interface{}
		if err := json.Unmarshal([]byte(s), &item); err != nil {
			return nil, fmt.Errorf("failed to parse conversation: %w", err)
		}
		unmarshalledData = append(unmarshalledData, item)
	}

	return unmarshalledData, nil
}
