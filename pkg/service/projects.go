package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// ProjectService handles project business logic
type ProjectService struct {
	ssdService  *SSDService
	projectRepo *repository.ProjectRepository
	logger      *utils.ErrorLogger
}

// NewProjectService creates a new project service
func NewProjectService() *ProjectService {
	return &ProjectService{
		projectRepo: repository.NewProjectRepository(),
		logger:      utils.NewErrorLogger("project_service"),
	}
}

// CreateProjectRequest represents the request to create a project
type CreateProjectRequest struct {
	HubID         *uuid.UUID `json:"hub_id" validate:"required"`
	IntegrationID *uuid.UUID `json:"integration_id"`
	Name          string     `json:"name" validate:"required,min=1,max=255"`
	RepoURL       *string    `json:"repo_url" validate:"omitempty,url"`
	Description   *string    `json:"description" validate:"omitempty,max=1000"`
}

// UpdateProjectRequest represents the request to update a project
type UpdateProjectRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=255"`
	RepoURL     *string `json:"repo_url" validate:"omitempty,url"`
	Description *string `json:"description" validate:"omitempty,max=1000"`
}

// ProjectListRequest represents the request to list projects
type ProjectListRequest struct {
	HubID         *uuid.UUID `json:"hub_id"`
	IntegrationID *uuid.UUID `json:"integration_id"`
	Search        string     `json:"search"`
	Page          int        `json:"page" validate:"min=1"`
	PageSize      int        `json:"page_size" validate:"min=1,max=100"`
	OrderBy       string     `json:"order_by"`
	OrderDir      string     `json:"order_dir" validate:"omitempty,oneof=ASC DESC"`
}

// CreateProject creates a new project
func (s *ProjectService) CreateProject(ctx context.Context, req *CreateProjectRequest) (*models.Project, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if project name already exists in the same hub
	exists, err := s.projectExistsInHub(ctx, req.HubID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check project existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("project with name '%s' already exists in this hub", req.Name)
	}

	// Create project
	project := &models.Project{
		HubID:         req.HubID,
		IntegrationID: req.IntegrationID,
		Name:          &req.Name,
		RepoURL:       req.RepoURL,
		Description:   req.Description,
	}

	err = s.projectRepo.Create(ctx, project)
	if err != nil {
		s.logger.LogError(err, "Failed to create project", map[string]interface{}{
			"request": req,
		})
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	s.logger.LogInfo("Project created successfully", map[string]interface{}{
		"project_id": project.ID,
		"name":       req.Name,
		"hub_id":     req.HubID,
	})

	return project, nil
}

// GetProject retrieves a project by ID
func (s *ProjectService) GetProject(ctx context.Context, id uuid.UUID) (*models.Project, error) {
	project, err := s.projectRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.LogError(err, "Failed to get project", map[string]interface{}{
			"project_id": id,
		})
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// logic
	return project, nil
}

// UpdateProject updates a project
func (s *ProjectService) UpdateProject(ctx context.Context, id uuid.UUID, req *UpdateProjectRequest) (*models.Project, error) {
	// Validate request
	if err := s.validateUpdateRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if project exists
	exists, err := s.projectRepo.Exists(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check project existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("project not found")
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.RepoURL != nil {
		updates["repo_url"] = *req.RepoURL
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	// Update project
	err = s.projectRepo.Update(ctx, id, updates)
	if err != nil {
		s.logger.LogError(err, "Failed to update project", map[string]interface{}{
			"project_id": id,
			"updates":    updates,
		})
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	// Get updated project
	project, err := s.projectRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated project: %w", err)
	}

	s.logger.LogInfo("Project updated successfully", map[string]interface{}{
		"project_id": id,
		"updates":    updates,
	})

	return project, nil
}

// DeleteProject deletes a project
func (s *ProjectService) DeleteProject(ctx context.Context, id uuid.UUID) error {
	// Check if project exists
	exists, err := s.projectRepo.Exists(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check project existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("project not found")
	}

	// Delete project
	err = s.projectRepo.Delete(ctx, id)
	if err != nil {
		s.logger.LogError(err, "Failed to delete project", map[string]interface{}{
			"project_id": id,
		})
		return fmt.Errorf("failed to delete project: %w", err)
	}

	s.logger.LogInfo("Project deleted successfully", map[string]interface{}{
		"project_id": id,
	})

	return nil
}

// ListProjects retrieves projects with pagination and filtering
func (s *ProjectService) ListProjects(ctx context.Context, req *ProjectListRequest) (*repository.QueryResult[models.Project], error) {
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	if req.OrderBy == "" {
		req.OrderBy = "created_at"
	}
	if req.OrderDir == "" {
		req.OrderDir = "DESC"
	}

	// Build query options
	options := &repository.QueryOptions{
		Limit:    req.PageSize,
		Offset:   (req.Page - 1) * req.PageSize,
		OrderBy:  req.OrderBy,
		OrderDir: req.OrderDir,
		Filters:  make(map[string]interface{}),
	}

	// Add filters
	if req.HubID != nil {
		options.Filters["hub_id"] = *req.HubID
	}
	if req.IntegrationID != nil {
		options.Filters["integration_id"] = *req.IntegrationID
	}

	// Execute query
	var result *repository.QueryResult[models.Project]
	var err error

	if req.Search != "" {
		result, err = s.projectRepo.SearchByName(ctx, req.Search, options)
	} else {
		result, err = s.projectRepo.List(ctx, options)
	}

	if err != nil {
		s.logger.LogError(err, "Failed to list projects", map[string]interface{}{
			"request": req,
		})
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return result, nil
}

// ListProjectsWithDetails retrieves projects with related data
func (s *ProjectService) ListProjectsWithDetails(ctx context.Context, req *ProjectListRequest) (*repository.QueryResult[repository.ProjectWithDetails], error) {
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	if req.OrderBy == "" {
		req.OrderBy = "created_at"
	}
	if req.OrderDir == "" {
		req.OrderDir = "DESC"
	}

	// Build query options
	options := &repository.QueryOptions{
		Limit:    req.PageSize,
		Offset:   (req.Page - 1) * req.PageSize,
		OrderBy:  req.OrderBy,
		OrderDir: req.OrderDir,
		Filters:  make(map[string]interface{}),
	}

	// Add filters
	if req.HubID != nil {
		options.Filters["hub_id"] = *req.HubID
	}
	if req.IntegrationID != nil {
		options.Filters["integration_id"] = *req.IntegrationID
	}

	// Execute query
	result, err := s.projectRepo.GetWithDetails(ctx, options)
	if err != nil {
		s.logger.LogError(err, "Failed to list projects with details", map[string]interface{}{
			"request": req,
		})
		return nil, fmt.Errorf("failed to list projects with details: %w", err)
	}

	return result, nil
}

// GetProjectsByHub retrieves projects by hub ID
func (s *ProjectService) GetProjectsByHub(ctx context.Context, hubID uuid.UUID, req *ProjectListRequest) (*repository.QueryResult[models.Project], error) {
	req.HubID = &hubID
	return s.ListProjects(ctx, req)
}

// GetProjectsByIntegration retrieves projects by integration ID
func (s *ProjectService) GetProjectsByIntegration(ctx context.Context, integrationID uuid.UUID, req *ProjectListRequest) (*repository.QueryResult[models.Project], error) {
	req.IntegrationID = &integrationID
	return s.ListProjects(ctx, req)
}

// validateCreateRequest validates create project request
func (s *ProjectService) validateCreateRequest(req *CreateProjectRequest) error {
	if req.HubID == nil {
		return fmt.Errorf("hub_id is required")
	}
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(req.Name) > 255 {
		return fmt.Errorf("name must be less than 255 characters")
	}
	if req.Description != nil && len(*req.Description) > 1000 {
		return fmt.Errorf("description must be less than 1000 characters")
	}
	return nil
}

// validateUpdateRequest validates update project request
func (s *ProjectService) validateUpdateRequest(req *UpdateProjectRequest) error {
	if req.Name != nil && *req.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if req.Name != nil && len(*req.Name) > 255 {
		return fmt.Errorf("name must be less than 255 characters")
	}
	if req.Description != nil && len(*req.Description) > 1000 {
		return fmt.Errorf("description must be less than 1000 characters")
	}
	return nil
}

// projectExistsInHub checks if a project with the given name exists in the hub
func (s *ProjectService) projectExistsInHub(ctx context.Context, hubID *uuid.UUID, name string) (bool, error) {
	options := &repository.QueryOptions{
		Filters: map[string]interface{}{
			"hub_id": hubID,
			"name":   name,
		},
		Limit: 1,
	}

	result, err := s.projectRepo.List(ctx, options)
	if err != nil {
		return false, err
	}

	return len(result.Data) > 0, nil
}

///////////New//////////////
func (s *ProjectService) GetProjectSummariesForTeams(ctx context.Context, req *GetProjectSummariesForTeamsRequest) (*client.ProjectSummaryResponse, error) {
	return s.ssdService.GetProjectSummariesForTeams(ctx, req)
}

// func (s *ProjectService) GetProjectDetails(ctx context.Context, req *client.ProjectDetailsRequest) (*client.ProjectDetailsResponse, error) {
// 	return s.ssdService.GetProjectDetails(ctx, req)
// }

func (s *ProjectService) GetProjectSummaryCount(ctx context.Context, hubIDs []string) (*client.SourceScanSummaryCount, error) {
	return s.ssdService.GetProjectSummaryCount(ctx, hubIDs)
}
