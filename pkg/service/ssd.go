package service

import (
	"context"
	"fmt"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// SSDService handles SSD (OpsMx) operations using the client layer
type SSDService struct {
	logger *utils.ErrorLogger
}

// NewSSDService creates a new SSD service
func NewSSDService() *SSDService {
	return &SSDService{
		logger: utils.NewErrorLogger("ssd_service"),
	}
}

// GetOrganizations retrieves organizations and hubs from SSD
func (s *SSDService) GetOrganizations(ctx context.Context) (*client.OrganizationResponse, error) {
	ssdClient := client.NewSSDClient()

	orgs, err := ssdClient.GetOrganizations(ctx)
	if err != nil {
		s.logger.LogError(err, "Failed to get organizations", map[string]interface{}{})
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}

	s.logger.LogInfo("Organizations retrieved successfully", map[string]interface{}{
		"org_count": len(orgs.QueryOrganization),
		"hub_count": len(orgs.QueryOrganization[0].Teams),
	})

	return orgs, nil
}

// GetOrganizationsAndTeams retrieves detailed organizations and hubs from SSD
func (s *SSDService) GetOrganizationsAndTeams(ctx context.Context) (*client.OrganizationResponse, error) {
	ssdClient := client.NewSSDClient()

	orgs, err := ssdClient.GetOrganizationsAndTeams(ctx)
	if err != nil {
		s.logger.LogError(err, "Failed to get organizations and hubs", map[string]interface{}{})
		return nil, fmt.Errorf("failed to get organizations and hubs: %w", err)
	}
	return orgs, nil
}

// GetHubByName retrieves a hub by name from SSD
func (s *SSDService) GetHubByName(ctx context.Context, hubName string) (*client.Hub, error) {
	ssdClient := client.NewSSDClient()

	hub, err := ssdClient.GetHubByName(ctx, hubName)
	if err != nil {
		s.logger.LogError(err, "Failed to get hub by name", map[string]interface{}{
			"hub_name": hubName,
		})
		return nil, fmt.Errorf("failed to get hub by name: %w", err)
	}

	s.logger.LogInfo("Hub retrieved by name successfully", map[string]interface{}{
		"hub_id":   hub.ID,
		"hub_name": hub.Name,
	})

	return hub, nil
}

// GetHubByID retrieves a hub by ID from SSD
func (s *SSDService) GetHubByID(ctx context.Context, hubID string) (*client.Hub, error) {
	ssdClient := client.NewSSDClient()

	hub, err := ssdClient.GetHubByID(ctx, hubID)
	if err != nil {
		s.logger.LogError(err, "Failed to get hub by ID", map[string]interface{}{
			"hub_id": hubID,
		})
		return nil, fmt.Errorf("failed to get hub by ID: %w", err)
	}

	s.logger.LogInfo("Hub retrieved by ID successfully", map[string]interface{}{
		"hub_id":   hub.ID,
		"hub_name": hub.Name,
	})

	return hub, nil
}

func (s *SSDService) CreateHub(ctx context.Context, req *client.CreateHubRequest) (*client.CreateHubResponse, error) {
	ssdClient := client.NewSSDClient()

	hub, err := ssdClient.CreateHub(ctx, req)
	if err != nil {
		s.logger.LogError(err, "Failed to create hub", map[string]interface{}{
			"hub_name": req.Name,
		})
	}

	return hub, nil
}

type GetProjectSummariesForTeamsRequest struct {
	HubID       string `json:"hub_id"`
	PageNo      int    `json:"page_no"`
	PageLimit   int    `json:"page_limit"`
	ProjectName string `json:"project_name"`
	Platform    string `json:"platform"`
	Status      string `json:"status"`
}

func (s *SSDService) GetProjectSummariesForTeams(ctx context.Context, req *GetProjectSummariesForTeamsRequest) (*client.ProjectSummaryResponse, error) {
	ssdClient := client.NewSSDClient()

	reqClient := &client.ProjectSummaryRequest{
		TeamIDs:     req.HubID,
		PageNo:      req.PageNo,
		PageLimit:   req.PageLimit,
		ProjectName: req.ProjectName,
		Platform:    req.Platform,
		Status:      req.Status,
	}

	return ssdClient.GetProjectSummaries(ctx, reqClient)
}

func (s *SSDService) GetProjectDetails(ctx context.Context, projectId string) (*client.ProjectRef, error) {
	ssdClient := client.NewSSDClient()

	return ssdClient.GetProjectDetails(ctx, projectId)
}

func (s *SSDService) GetProjectStatuses(ctx context.Context, projectIds []string) ([]client.ProjectDetailsResponse, error) {
	ssdClient := client.NewSSDClient()
	return ssdClient.GetProjectStatuses(ctx, projectIds)
}

// GetAllProjectStatuses fetches all projects from SSD without filtering by project IDs
func (s *SSDService) GetAllProjectStatuses(ctx context.Context) ([]client.ProjectRef, error) {
	ssdClient := client.NewSSDClient()
	return ssdClient.GetAllProjectStatuses(ctx)
}

func (s *SSDService) GetProjectDetailsCustom(ctx context.Context, projectId string) (*client.ProjectRef, error) {
	return client.NewSSDClient().GetProjectDetailsCustom(ctx, projectId)
}

func (s *SSDService) GetSASTScanResults(ctx context.Context, scanType, projectId, scanId string, sastReq *client.SASTScanRequest) ([]*client.SASTScanResult, error) {
	return client.NewSSDClient().GetSASTScanResults(ctx, scanType, projectId, scanId, sastReq)
}

func (s *SSDService) GetProjectSummaryCount(ctx context.Context, hubIDs []string) (*client.SourceScanSummaryCount, error) {
	ssdClient := client.NewSSDClient()

	result, err := ssdClient.GetProjectSummaryCount(ctx, hubIDs)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *SSDService) GetVulnerabilityData(ctx context.Context, req *client.VulnerabilityDataRequest, body interface{}) (*client.VulnerabilityDataResponse, error) {
	ssdClient := client.NewSSDClient()

	return ssdClient.GetVulnerabilityData(ctx, req, body)
}

func (s *SSDService) GetScanResultData(ctx context.Context, req *client.ScanResultDataRequest) (*client.ScanResultDataResponse, error) {
	ssdClient := client.NewSSDClient()

	result, err := ssdClient.GetScanResultData(ctx, req)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *SSDService) Rescan(ctx context.Context, req *RescanRequest, scanResult *client.ScanResultDataResponse) (string, error) {
	ssdClient := client.NewSSDClient()

	rescanReq := client.RescanRequest{
		ProjectID:   req.ProjectID,
		ProjectName: scanResult.ProjectName,
		Platform:    scanResult.Platform,
		ScanID:      scanResult.ScanID,
		ScanType:    "sourceScan",
	}

	resp, err := ssdClient.Rescan(ctx, &rescanReq)
	if err != nil {
		return "", err
	}

	return resp.Message, nil
}

func (s *SSDService) GetVulnerabilityList(ctx context.Context, req *client.VulnerabilityListRequest) (*client.VulnerabilityListResponse, error) {
	ssdClient := client.NewSSDClient()

	return ssdClient.GetVulnerabilityList(ctx, req)
}

// projects services below

func (s *SSDService) CreateProject(ctx context.Context, teamIds string, req *client.ProjectRef) (string, error) {
	ssdClient := client.NewSSDClient()

	result, err := ssdClient.CreateProject(ctx, teamIds, req)
	if err != nil {
		return "", err
	}
	return result, nil
}

func (s *SSDService) UpdateProject(ctx context.Context, teamIds string, req *client.ProjectRef) (string, error) {
	ssdClient := client.NewSSDClient()

	result, err := ssdClient.UpdateProject(ctx, teamIds, req)
	if err != nil {
		return "", err
	}
	return result, nil
}

func (s *SSDService) DeleteProject(ctx context.Context, teamIds, projectId string) (string, error) {
	ssdClient := client.NewSSDClient()

	result, err := ssdClient.DeleteProject(ctx, teamIds, projectId)
	if err != nil {
		return "", err
	}
	return result, nil
}

// integrations APIs
func (s *SSDService) GetGithubOauthUrl(ctx context.Context) (string, error) {
	ssdClient := client.NewSSDClient()

	installUrl, err := ssdClient.GetGithubOauthUrl(ctx)
	if err != nil {
		return "", err
	}
	return installUrl, nil
}

func (s *SSDService) CreateGitHubIntegration(ctx context.Context, name, token, installationId, githubIntegratorId string, timestamp int64, teamIDs []string) (string, error) {
	ssdClient := client.NewSSDClient()
	return ssdClient.CreateGitHubIntegration(ctx, name, token, installationId, githubIntegratorId, timestamp, teamIDs)
}

func (s *SSDService) UpdateGitHubIntegration(ctx context.Context, name, token, installationId, integrationId string, timestamp int64, teamIDs []string) (string, error) {
	ssdClient := client.NewSSDClient()
	return ssdClient.UpdateGitHubIntegration(ctx, name, token, installationId, integrationId, timestamp, teamIDs)
}

func (s *SSDService) GetRepoBranchList(ctx context.Context, params map[string]string) ([]string, error) {
	ssdClient := client.NewSSDClient()
	return ssdClient.GetRepoBranchList(ctx, params)
}

func (s *SSDService) GetSupportedIntegrators(ctx context.Context, level, teamIds string) ([]*client.IntegrationStatus, error) {
	ssdClient := client.NewSSDClient()
	return ssdClient.GetSupportedIntegrators(ctx, level, teamIds)
}

func (s *SSDService) getIntegratorToken(ctx context.Context, projectId string) (string, error) {
	// Installation ID
	ssdClient := client.NewSSDClient()

	integration, err := ssdClient.GetIntegratorConfigForProject(ctx, "github", projectId, "installationId")
	if err != nil {
		return "", err
	}

	if len(integration.QueryProject) == 0 {
		return "", fmt.Errorf("no installationId found for project %s", projectId)
	}
	// Fetch the token
	token, err := client.GetGithubTokenFromInstallationId(integration.QueryProject[0].IntegratorConfigs.Configs[0].Value)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *SSDService) GetGithubUsername(ctx context.Context, accountId string) (string, error) {
	userNames, err := s.GetRepoBranchList(ctx, map[string]string{
		"accountId": accountId,
		"platform":  "github",
		"scanLevel": "org",
		"type":      "user",
	})
	if err != nil {
		return "", err
	} else if len(userNames) == 0 {
		return "", fmt.Errorf("user not found")
	}
	return userNames[0], nil
}

func (s *SSDService) IntegratorHandler(ctx context.Context, err, integrationId, integrationName, hubID string) error {
	if utils.ContainsString(err, []string{"404 Not Found", "GITHUB_API_ERROR"}) {
		return fmt.Errorf("github app is not available, please reinstall the app")
	}
	return fmt.Errorf("internal server error")
}

func (s *SSDService) DeleteIntegration(ctx context.Context, integrationId, integrationName, hubID string) error {
	ssdClient := client.NewSSDClient()

	return ssdClient.DeleteIntegration(ctx, &client.DeleteIntegrationRequest{
		IntegrationID:   integrationId,
		IntegrationName: integrationName,
		IntegrationType: "github",
		Level:           "global",
		TeamID:          hubID,
	})
}
