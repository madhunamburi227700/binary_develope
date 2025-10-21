package service

import (
	"context"
	"fmt"

	"github.com/opsmx/ai-guardian-api/pkg/client"
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
	HubID         string `json:"hub_id" validate:"required"`
	IntegrationID string `json:"integration_id"`
	Name          string `json:"name" validate:"required,min=1,max=255"`
	RepoName      string `json:"repoName" validate:"required,min=1,max=255"`
}

// CreateProject creates a new project
func (s *ProjectService) CreateProject(ctx context.Context, req *CreateProjectRequest) (string, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}

	// same project name check would be auto applied via ssd
	// current they do not have any api to check project via name

	// getting username from ssd based on account id
	username, err := s.getGithubUsername(ctx, req.IntegrationID)
	if err != nil {
		s.logger.LogError(err, "Failed to get user", nil)
		return "", fmt.Errorf("failed to get user")
	}

	// {"name":"temp22","scanType":"sourceScan","platform":"github","accountId":"0x5f2f9","teamId":"fe2e8a09-a3f2-4263-b635-fa7e99f2d43b","scanLevel":"repoLevel","organisation":"arpit-jaswani","type":"user","projectConfigs":[{"repository":"python-app","scheduleTime":0,"branch":["onlyMain"],"branchPattern":"","scanUpto":0}]}
	// Create project
	scheduleTime := 0
	scanUpto := 0
	project := &client.ProjectRef{
		Name:         req.Name,
		AccountID:    req.IntegrationID,
		Organisation: username,
		TeamID:       req.HubID,

		// TODO: automate below fields
		ProjectConfig: []client.ProjectConfigRef{{
			Repository:   req.RepoName,
			Branch:       []string{"onlyMain"},
			ScheduleTime: &scheduleTime,
			ScanUpto:     &scanUpto,
		}},
		Type:      "user", // since we authenticating using github app token type can be any user/organisation
		ScanType:  "sourceScan",
		Platform:  "github",
		ScanLevel: "repoLevel",
	}

	projectId, err := s.ssdService.CreateProject(ctx, req.HubID, project)
	if err != nil {
		s.logger.LogError(err, "Failed to create project", map[string]interface{}{
			"request": project,
		})
		return "", fmt.Errorf("failed to create project: %w", err)
	}

	s.logger.LogInfo("Project created successfully", map[string]interface{}{
		"project_id": projectId,
		"name":       req.Name,
		"hub_id":     req.HubID,
	})

	return projectId, nil
}

// GetProject retrieves a project by ID
func (s *ProjectService) GetProject(ctx context.Context, projectId string) (*client.ProjectRef, error) {
	project, err := s.ssdService.GetProjectDetails(ctx, projectId)
	if err != nil {
		s.logger.LogError(err, "Failed to get project", map[string]interface{}{
			"projectId": projectId,
		})
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// logic
	return project, nil
}

// DeleteProject deletes a project
func (s *ProjectService) DeleteProject(ctx context.Context, teamIds, projectId string) error {
	// Delete project
	_, err := s.ssdService.DeleteProject(ctx, teamIds, projectId)
	if err != nil {
		s.logger.LogError(err, "Failed to delete project", map[string]interface{}{
			"project_id": projectId,
		})
		return fmt.Errorf("failed to delete project: %w", err)
	}

	s.logger.LogInfo("Project deleted successfully", map[string]interface{}{
		"project_id": projectId,
	})

	return nil
}

// validateCreateRequest validates create project request
func (s *ProjectService) validateCreateRequest(req *CreateProjectRequest) error {
	if req.HubID == "" {
		return fmt.Errorf("hub_id is required")
	}
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.RepoName == "" {
		return fmt.Errorf("repoName is required user/organisation")
	}
	return nil
}

func (s *ProjectService) GetProjectSummariesForTeams(ctx context.Context, req *GetProjectSummariesForTeamsRequest) (*client.ProjectSummaryResponse, error) {
	return s.ssdService.GetProjectSummariesForTeams(ctx, req)
}

func (s *ProjectService) GetProjectSummaryCount(ctx context.Context, hubIDs []string) (*client.SourceScanSummaryCount, error) {
	return s.ssdService.GetProjectSummaryCount(ctx, hubIDs)
}

func (s *ProjectService) getGithubUsername(ctx context.Context, accountId string) (string, error) {
	userNames, err := s.ssdService.GetRepoBranchList(ctx, map[string]string{
		// automated param from UI
		"accountId": accountId,
		// default params
		// ssd will look for repos from installation id based token
		// if orgName is blank
		"platform":  "github", // automate platform in future release
		"scanLevel": "org",
		"type":      "user",
	})
	if err != nil {
		return "", err
	} else if len(userNames) == 0 {
		return "", fmt.Errorf("user not found")
	}
	// based on installation id token org should always be one
	return userNames[0], nil
}
