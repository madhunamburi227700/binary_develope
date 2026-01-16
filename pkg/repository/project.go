package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
		"id":             project.ID,
		"name":           project.Name,
		"hub_id":         project.HubID,
		"organisation":   project.Organisation,
		"integration_id": project.IntegrationID,
	}

	// Add scheduled_time field if present
	if project.ScheduledTime != nil {
		data["scheduled_time"] = *project.ScheduledTime
	}

	_, err := r.BaseRepository.Create(ctx, "projects", data)
	if err != nil {
		return err
	}

	return nil
}

// GetByID retrieves a project by ID
func (r *ProjectRepository) GetByID(ctx context.Context, id string) (*models.Project, error) {
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

// UpdateProject updates a project by string ID with a map of field updates
func (r *ProjectRepository) UpdateProject(ctx context.Context, projectID string, updates map[string]interface{}) error {
	return r.BaseRepository.UpdateByStringID(ctx, "projects", projectID, updates)
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

// GetAll retrieves all projects without pagination
func (r *ProjectRepository) GetAll(ctx context.Context) ([]*models.Project, error) {
	var projects []*models.Project

	err := r.BaseRepository.ListAll(ctx, "projects", &QueryOptions{
		OrderBy:  "created_at",
		OrderDir: "DESC",
	}, &projects)
	if err != nil {
		return nil, err
	}

	return projects, nil
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

// GetProjectsByOwnerAndRepoName retrieves projects by owner (organisation), repository name, and branch name
// This joins with the scans table since repository and branch are stored there
func (r *ProjectRepository) GetProjectsByOwnerAndRepoName(ctx context.Context, owner, repoName, branchName string) ([]models.Project, error) {
	query := `
		SELECT DISTINCT p.id, p.name, p.hub_id, p.integration_id, p.organisation, p.created_at, p.updated_at
		FROM projects p
		INNER JOIN scans s ON p.id = s.project_id
		WHERE p.organisation = $1 AND s.repository = $2 AND s.branch = $3
		ORDER BY p.created_at DESC
	`
	rows, err := r.db.Query(ctx, query, owner, repoName, branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to query projects: %w", err)
	}
	defer rows.Close()

	var projects []models.Project
	err = r.scanRows(rows, &projects)
	if err != nil {
		return nil, fmt.Errorf("failed to scan results: %w", err)
	}

	return projects, nil
}

// CheckProjectByOwnerAndRepo checks if projects exist by owner (organisation) and repository name
// This joins with the scans table since repository is stored there
func (r *ProjectRepository) CheckProjectByOwnerAndRepo(ctx context.Context, owner, repoName string) (bool, error) {
	query := `
	SELECT p.id
	FROM projects p
	INNER JOIN scans s ON p.id = s.project_id
	WHERE p.organisation = $1 AND s.repository = $2 
	LIMIT 1;
	`
	var projectId string
	err := r.db.QueryRow(ctx, query, owner, repoName).Scan(&projectId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check project existence: %w", err)
	}
	return true, nil
}

// GetProjectsByOwner retrieves projects by owner (organisation)
func (r *ProjectRepository) GetProjectsByOwner(ctx context.Context, owner string) ([]*models.Project, error) {
	query := `
		SELECT * FROM projects WHERE organisation = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to query projects: %w", err)
	}
	defer rows.Close()

	var projects []*models.Project
	err = r.scanRows(rows, &projects)
	if err != nil {
		return nil, fmt.Errorf("failed to scan results: %w", err)
	}

	return projects, nil
}

// get one project by owner and repository
func (r *ProjectRepository) GetLatestByOwnerAndHubID(
	ctx context.Context,
	owner, hubID string,
) (*models.Project, error) {

	query := `
		SELECT
			id,
			name,
			hub_id,
			integration_id,
			organisation,
			created_at,
			updated_at
		FROM projects
		WHERE organisation = $1
		  AND hub_id = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	var project models.Project
	err := r.db.QueryRow(ctx, query, owner, hubID).Scan(
		&project.ID,
		&project.Name,
		&project.HubID,
		&project.IntegrationID,
		&project.Organisation,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // or custom NotFound error
		}
		return nil, fmt.Errorf("failed to query project: %w", err)
	}

	return &project, nil
}
