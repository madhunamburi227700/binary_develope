package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/opsmx/ai-guardian-api/pkg/models"
)

// ProjectRepository handles project-related database operations
type ProjectRepository struct {
	*BaseRepository
}

// NewProjectRepository creates a new project repository
func NewProjectRepository() *ProjectRepository {
	return &ProjectRepository{
		BaseRepository: NewBaseRepository("projects"),
	}
}

// Create creates a new project
func (r *ProjectRepository) Create(ctx context.Context, project *models.Project) error {
	data := map[string]interface{}{
		"hub_id":         project.HubID,
		"integration_id": project.IntegrationID,
		"name":           project.Name,
		"repo_url":       project.RepoURL,
		"description":    project.Description,
	}

	id, err := r.BaseRepository.Create(ctx, "projects", data)
	if err != nil {
		return err
	}

	project.ID = id
	return nil
}

// GetByID retrieves a project by ID
func (r *ProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Project, error) {
	var project models.Project
	err := r.BaseRepository.GetByID(ctx, "projects", id, &project)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// Update updates a project
func (r *ProjectRepository) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.BaseRepository.Update(ctx, "projects", id, updates)
}

// Delete deletes a project
func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.BaseRepository.Delete(ctx, "projects", id)
}

// List retrieves projects with pagination and filtering
func (r *ProjectRepository) List(ctx context.Context, options *QueryOptions) (*QueryResult[models.Project], error) {
	var projects []models.Project

	// Add default ordering if not specified
	if options.OrderBy == "" {
		options.OrderBy = "created_at"
		options.OrderDir = "DESC"
	}

	pagination, err := r.BaseRepository.List(ctx, "projects", options, &projects)
	if err != nil {
		return nil, err
	}

	return &QueryResult[models.Project]{
		Data:       projects,
		Pagination: pagination,
	}, nil
}

// Count counts projects with filters
func (r *ProjectRepository) Count(ctx context.Context, filters map[string]interface{}) (int64, error) {
	return r.BaseRepository.Count(ctx, "projects", filters)
}

// Exists checks if a project exists
func (r *ProjectRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.BaseRepository.Exists(ctx, "projects", id)
}

// GetByHubID retrieves projects by hub ID
func (r *ProjectRepository) GetByHubID(ctx context.Context, hubID uuid.UUID, options *QueryOptions) (*QueryResult[models.Project], error) {
	if options == nil {
		options = &QueryOptions{}
	}

	if options.Filters == nil {
		options.Filters = make(map[string]interface{})
	}
	options.Filters["hub_id"] = hubID

	return r.List(ctx, options)
}

// GetByIntegrationID retrieves projects by integration ID
func (r *ProjectRepository) GetByIntegrationID(ctx context.Context, integrationID uuid.UUID, options *QueryOptions) (*QueryResult[models.Project], error) {
	if options == nil {
		options = &QueryOptions{}
	}

	if options.Filters == nil {
		options.Filters = make(map[string]interface{})
	}
	options.Filters["integration_id"] = integrationID

	return r.List(ctx, options)
}

// SearchByName searches projects by name
func (r *ProjectRepository) SearchByName(ctx context.Context, name string, options *QueryOptions) (*QueryResult[models.Project], error) {
	if options == nil {
		options = &QueryOptions{}
	}

	// Build search query
	whereClause := "WHERE name ILIKE $1"
	args := []interface{}{"%" + name + "%"}

	// Add additional filters
	argIndex := 2
	if options.Filters != nil {
		for column, value := range options.Filters {
			whereClause += fmt.Sprintf(" AND %s = $%d", column, argIndex)
			args = append(args, value)
			argIndex++
		}
	}

	// Build ORDER BY clause
	orderClause := ""
	if options.OrderBy != "" {
		orderDir := "ASC"
		if options.OrderDir == "DESC" {
			orderDir = "DESC"
		}
		orderClause = fmt.Sprintf(" ORDER BY %s %s", options.OrderBy, orderDir)
	}

	// Build LIMIT and OFFSET clause
	limitClause := ""
	if options.Limit > 0 {
		limitClause = fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, options.Limit)
		argIndex++
	}

	if options.Offset > 0 {
		limitClause += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, options.Offset)
		argIndex++
	}

	// Build main query
	query := fmt.Sprintf(`
		SELECT * FROM projects 
		%s%s%s`,
		whereClause,
		orderClause,
		limitClause,
	)

	// Execute query
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.logger.LogError(err, "Failed to search projects by name", map[string]interface{}{
			"name":    name,
			"options": options,
		})
		return nil, fmt.Errorf("failed to search projects: %w", err)
	}
	defer rows.Close()

	// Scan results
	var projects []models.Project
	err = r.scanRows(rows, &projects)
	if err != nil {
		return nil, fmt.Errorf("failed to scan results: %w", err)
	}

	// Get total count for pagination
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM projects %s", whereClause)
	var total int64
	err = r.db.QueryRow(ctx, countQuery, args[:len(args)-2]...).Scan(&total) // Exclude limit and offset
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Calculate pagination
	pageSize := options.Limit
	if pageSize <= 0 {
		pageSize = 10
	}
	page := (options.Offset / pageSize) + 1
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))

	pagination := &PaginationResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Pages:    pages,
	}

	return &QueryResult[models.Project]{
		Data:       projects,
		Pagination: pagination,
	}, nil
}

// GetWithDetails retrieves projects with related data
func (r *ProjectRepository) GetWithDetails(ctx context.Context, options *QueryOptions) (*QueryResult[ProjectWithDetails], error) {
	if options == nil {
		options = &QueryOptions{}
	}

	// Build SELECT clause
	selectClause := `
		p.id, p.hub_id, p.integration_id, p.name, p.repo_url, p.description, 
		p.created_at, p.updated_at,
		h.name as hub_name,
		i.name as integration_name, i.type as integration_type
	`

	// Build JOIN clause
	joins := []string{
		"LEFT JOIN hubs h ON p.hub_id = h.id",
		"LEFT JOIN integrations i ON p.integration_id = i.id",
	}

	// Build WHERE clause
	whereClause, whereArgs := r.buildWhereClause(options.Filters)

	// Build ORDER BY clause
	orderClause := ""
	if options.OrderBy != "" {
		orderDir := "ASC"
		if options.OrderDir == "DESC" {
			orderDir = "DESC"
		}
		orderClause = fmt.Sprintf(" ORDER BY p.%s %s", options.OrderBy, orderDir)
	} else {
		orderClause = " ORDER BY p.created_at DESC"
	}

	// Build LIMIT and OFFSET clause
	limitClause := ""
	args := whereArgs
	argIndex := len(whereArgs) + 1

	if options.Limit > 0 {
		limitClause = fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, options.Limit)
		argIndex++
	}

	if options.Offset > 0 {
		limitClause += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, options.Offset)
		argIndex++
	}

	// Build main query
	query := fmt.Sprintf(`
		SELECT %s 
		FROM projects p
		%s%s%s%s`,
		selectClause,
		strings.Join(joins, " "),
		whereClause,
		orderClause,
		limitClause,
	)

	// Execute query
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.logger.LogError(err, "Failed to get projects with details", map[string]interface{}{
			"options": options,
		})
		return nil, fmt.Errorf("failed to get projects with details: %w", err)
	}
	defer rows.Close()

	// Scan results
	var projects []ProjectWithDetails
	for rows.Next() {
		var p ProjectWithDetails
		err := rows.Scan(
			&p.ID, &p.HubID, &p.IntegrationID, &p.Name, &p.RepoURL, &p.Description,
			&p.CreatedAt, &p.UpdatedAt,
			&p.HubName, &p.IntegrationName, &p.IntegrationType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project details: %w", err)
		}
		projects = append(projects, p)
	}

	// Get total count for pagination
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM projects p
		%s%s`,
		strings.Join(joins, " "),
		whereClause,
	)

	var total int64
	err = r.db.QueryRow(ctx, countQuery, whereArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Calculate pagination
	pageSize := options.Limit
	if pageSize <= 0 {
		pageSize = 10
	}
	page := (options.Offset / pageSize) + 1
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))

	pagination := &PaginationResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Pages:    pages,
	}

	return &QueryResult[ProjectWithDetails]{
		Data:       projects,
		Pagination: pagination,
	}, nil
}

// ProjectWithDetails represents a project with related data
type ProjectWithDetails struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	HubID           *uuid.UUID `json:"hub_id" db:"hub_id"`
	IntegrationID   *uuid.UUID `json:"integration_id" db:"integration_id"`
	Name            *string    `json:"name" db:"name"`
	RepoURL         *string    `json:"repo_url" db:"repo_url"`
	Description     *string    `json:"description" db:"description"`
	CreatedAt       *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at" db:"updated_at"`
	HubName         *string    `json:"hub_name" db:"hub_name"`
	IntegrationName *string    `json:"integration_name" db:"integration_name"`
	IntegrationType *string    `json:"integration_type" db:"integration_type"`
}
