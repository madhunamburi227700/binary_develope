package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SSDClientInterface defines the interface for SSD operations
type SSDClientInterface interface {
	// Organization operations
	GetOrganizations(ctx context.Context) (*OrganizationResponse, error)
	GetOrganizationsAndTeams(ctx context.Context) (*OrganizationResponse, error)

	// Hub operations
	CreateHub(ctx context.Context, req *CreateHubRequest) (*CreateHubResponse, error)

	// Integration operations
	GetIntegrations(ctx context.Context, integratorType string, teamIDs []string) ([]Integration, error)
	ValidateIntegration(ctx context.Context, req *ValidateIntegrationRequest, teamIDs []string) (*ValidateIntegrationResponse, error)
	CreateIntegration(ctx context.Context, req *CreateIntegrationRequest, teamIDs []string) (string, error)

	// Resource operations
	GetResources(ctx context.Context) (*ResourceResponse, error)

	// GraphQL operations
	ExecuteGraphQL(ctx context.Context, query string) (*GraphQLResponse, error)

	// Project operations
	GetProjectSummariesForTeams(ctx context.Context, req *ProjectSummaryRequest) (*ProjectSummaryResponse, error)
	// GetProjectDetails(ctx context.Context, req *ProjectDetailsRequest) (*ProjectDetailsResponse, error)
}

// Organization operations
// Fix the GetOrganizations method in pkg/client/ssd.go
func (c *SSDClient) GetOrganizations(ctx context.Context) (*OrganizationResponse, error) {
	query := `query QueryOrganization {
		organizations: queryOrganization {
			id
			name
			roles(filter: { group: { in: ["admin","qa","dev"] } }) @cascade{
				permission
			}
			teams {
				id
				name
			}
		}
		teams: queryTeam @cascade {
			id
			name
			roles(filter: { group: { in: ["admin","qa","dev"] } }) {
				permission
			}
		}
	}`

	resp, err := c.ExecuteGraphQL(ctx, query)
	if err != nil {
		return nil, err
	}

	// Fix: Convert interface{} to []byte first, then unmarshal
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	var result OrganizationResponse
	if err := json.Unmarshal(dataBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal organization response: %w", err)
	}

	return &result, nil
}

// Fix the GetOrganizationsAndTeams method in pkg/client/ssd.go
func (c *SSDClient) GetOrganizationsAndTeams(ctx context.Context) (*OrganizationResponse, error) {
	query := `query QueryOrganization {
		queryOrganization {
			id
			name
			teams {
				id
				name
				email
				labels {
					name
					value
				}
			}
		}
		orgPermission: queryOrganization @cascade {
			id
			name
			roles(filter: { group: { in: ["admin","qa","dev"] } }) {
				permission
			}
		}
		teamPermission: queryTeam @cascade {
			id
			name
			roles(filter: { group: { in: ["admin","qa","dev"] } }) {
				permission
			}
		}
	}`

	resp, err := c.ExecuteGraphQL(ctx, query)
	if err != nil {
		return nil, err
	}

	// Fix: Convert interface{} to []byte first, then unmarshal
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	var result OrganizationResponse
	if err := json.Unmarshal(dataBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal organization response: %w", err)
	}

	return &result, nil
}

// Hub operations
func (c *SSDClient) CreateHub(ctx context.Context, req *CreateHubRequest) (*CreateHubResponse, error) {
	endpoint := fmt.Sprintf("/gate/ssdservice/v1/team?orgId=%s", c.orgID)

	resp, err := c.restClient.Post(ctx, endpoint, req, nil)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to create hub: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result CreateHubResponse
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse create hub response: %w", err)
	}

	return &result, nil
}

// Integration operations
func (c *SSDClient) GetIntegrations(ctx context.Context, integratorType, teamIDs string) ([]Integration, error) {
	endpoint := fmt.Sprintf("/gate/ssdservice/v1/integration?integratorType=%s&multiSupport=true&orgId=%s&level=global&teamId=%s",
		integratorType, c.orgID, teamIDs)

	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get integrations: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result []Integration
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse integrations response: %w", err)
	}

	return result, nil
}

func (c *SSDClient) ValidateIntegration(ctx context.Context, req *ValidateIntegrationRequest, teamIDs []string) (*ValidateIntegrationResponse, error) {
	teamIDsStr := strings.Join(teamIDs, ",")
	endpoint := fmt.Sprintf("/gate/ssdservice/v1/validateIntegration?orgId=%s&level=global&teamId=%s",
		c.orgID, teamIDsStr)

	resp, err := c.restClient.Post(ctx, endpoint, req, nil)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to validate integration: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result ValidateIntegrationResponse
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse validate integration response: %w", err)
	}

	return &result, nil
}

func (c *SSDClient) CreateIntegration(ctx context.Context, req *CreateIntegrationRequest, teamIDs []string) (string, error) {
	teamIDsStr := strings.Join(teamIDs, ",")
	endpoint := fmt.Sprintf("/gate/ssdservice/v1/integration?orgId=%s&level=global&teamId=%s",
		c.orgID, teamIDsStr)

	resp, err := c.restClient.Post(ctx, endpoint, req, nil)
	if err != nil {
		return "", err
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("failed to create integration: status %d, body: %s", resp.StatusCode, resp.String())
	}

	return resp.String(), nil
}

// Resource operations
func (c *SSDClient) GetResources(ctx context.Context) (*ResourceResponse, error) {
	endpoint := fmt.Sprintf("/gate/ssdservice/v1/resource?orgId=%s", c.orgID)

	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get resources: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result ResourceResponse
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse resources response: %w", err)
	}

	return &result, nil
}

// GraphQL operations
func (c *SSDClient) ExecuteGraphQL(ctx context.Context, query string) (*GraphQLResponse, error) {
	endpoint := "/graphql?req=get-Org-and-Teams-team-modal-popup"

	req := GraphQLRequest{Query: query}
	resp, err := c.restClient.Post(ctx, endpoint, req, nil)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to execute GraphQL query: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result GraphQLResponse
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse GraphQL response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL errors: %v", result.Errors)
	}

	return &result, nil
}

// Helper functions
func (c *SSDClient) SetSessionID(sessionID string) {
	c.sessionID = sessionID
	c.restClient.cookies["SESSION"] = sessionID
}

func (c *SSDClient) GetOrgID() string {
	return c.orgID
}

func (c *SSDClient) SetOrgID(orgID string) {
	c.orgID = orgID
}

// Fix the GetHubByName method in pkg/client/ssd.go
func (c *SSDClient) GetHubByName(ctx context.Context, hubName string) (*Hub, error) {
	orgs, err := c.GetOrganizationsAndTeams(ctx)
	if err != nil {
		return nil, err
	}

	// Search for hub in queryOrganization teams only
	for _, org := range orgs.QueryOrganization {
		for _, team := range org.Teams {
			if team.Name == hubName {
				return &team, nil
			}
		}
	}

	return nil, fmt.Errorf("team with name '%s' not found", hubName)
}

// Fix the GetHubByID method in pkg/client/ssd.go
func (c *SSDClient) GetHubByID(ctx context.Context, hubID string) (*Hub, error) {
	orgs, err := c.GetOrganizationsAndTeams(ctx)
	if err != nil {
		return nil, err
	}

	// Search for hub in queryOrganization teams only
	for _, org := range orgs.QueryOrganization {
		for _, team := range org.Teams {
			if team.ID == hubID {
				return &team, nil
			}
		}
	}

	return nil, fmt.Errorf("team with ID '%s' not found", hubID)
}

// CreateGitHubIntegration creates a GitHub integration with the given parameters
func (c *SSDClient) CreateGitHubIntegration(ctx context.Context, name, token, installationId string,
	timestamp int64, teamIDs []string) (string, error) {
	// encryptedToken, err := utils.EncryptToken(token)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to encrypt token: %w", err)
	// }

	req := &CreateIntegrationRequest{
		Name:           name,
		IntegratorType: "github",
		Category:       "sourcetool",
		FeatureConfigs: map[string]interface{}{
			"authType": "token",
		},
		IntegratorConfigs: map[string]interface{}{
			"url":       "https://api.github.com",
			"token":     token,
			"createdAt": fmt.Sprintf("%d", time.Now().Unix()),
		},
		Team: make([]TeamAssignment, len(teamIDs)),
		ID:   uuid.New().String(),
	}

	if installationId != "" {
		req.FeatureConfigs["authType"] = "app"
		req.IntegratorConfigs["installationId"] = installationId
		req.IntegratorConfigs["createdAt"] = fmt.Sprintf("%d", timestamp)
	}

	// Convert team IDs to team assignments
	for i, teamID := range teamIDs {
		team, err := c.GetHubByID(ctx, teamID)
		if err != nil {
			return "", fmt.Errorf("failed to get team by ID %s: %w", teamID, err)
		}
		req.Team[i] = TeamAssignment{
			TeamName: team.Name,
			TeamID:   teamID,
		}
	}

	return c.CreateIntegration(ctx, req, teamIDs)
}

// Projects API below

// project summaries for team
func (c *SSDClient) GetProjectSummaries(ctx context.Context, req *ProjectSummaryRequest) (*ProjectSummaryResponse, error) {
	// Build query parameters
	params := make([]string, 0)

	// Add team IDs
	if req.TeamIDs != "" {
		teamIDsStr := req.TeamIDs
		params = append(params, fmt.Sprintf("teamId=%s", teamIDsStr))
	}

	// Add pagination
	if req.PageNo > 0 {
		params = append(params, fmt.Sprintf("pageNo=%d", req.PageNo))
	}
	if req.PageLimit > 0 {
		params = append(params, fmt.Sprintf("pageLimit=%d", req.PageLimit))
	}

	// Add filters
	if req.ProjectName != "" {
		params = append(params, fmt.Sprintf("projectName=%s", req.ProjectName))
	}
	if req.Platform != "" {
		params = append(params, fmt.Sprintf("platform=%s", req.Platform))
	}
	if req.Status != "" {
		params = append(params, fmt.Sprintf("status=%s", req.Status))
	}

	// Build endpoint
	endpoint := "/gate/ssdservice/v1/sourceScan/summary"
	if len(params) > 0 {
		endpoint += "?" + strings.Join(params, "&")
	}

	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get project summaries: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result ProjectSummaryResponse
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse project summaries response: %w", err)
	}

	return &result, nil
}

func (c *SSDClient) GetProjectDetails(ctx context.Context, projectId string) (*ProjectRef, error) {
	// Build query parameters
	params := make([]string, 0)

	// Add orgId
	params = append(params, fmt.Sprintf("orgId=%s", c.orgID))
	// Add project id
	params = append(params, fmt.Sprintf("id=%s", projectId))

	// Build endpoint
	endpoint := "/gate/ssdservice/v1/scan/project"
	if len(params) > 0 {
		endpoint += "?" + strings.Join(params, "&")
	}

	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get project details: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result []*ProjectRef
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse project details response: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	return result[0], nil
}

// CreateProject creates a ssd project
// Add project
// Request Type POST
// https://july-dev.aoa.oes.opsmx.org/gate/ssdservice/v1/scan/project?orgId=c3b62ec6-42e2-43d4-beaf-5b2017ab5848
// {"name":"temp22","scanType":"sourceScan","platform":"github","accountId":"0x5f2f9","teamId":"fe2e8a09-a3f2-4263-b635-fa7e99f2d43b","scanLevel":"repoLevel","organisation":"arpit-jaswani","type":"user","projectConfigs":[{"repository":"python-app","scheduleTime":0,"branch":["onlyMain"],"branchPattern":"","scanUpto":0}]}
// returns 201 and json response {"id":"0x5f416"}
func (c *SSDClient) CreateProject(ctx context.Context, teamIds string, req *ProjectRef) (string, error) {
	// Build query parameters
	params := make([]string, 0)

	params = append(params, fmt.Sprintf("orgId=%s", c.orgID))

	// Add team IDs
	if teamIds != "" {
		params = append(params, fmt.Sprintf("teamId=%s", teamIds))
	}

	// Build endpoint
	endpoint := "/gate/ssdservice/v1/scan/project"
	if len(params) > 0 {
		endpoint += "?" + strings.Join(params, "&")
	}

	resp, err := c.restClient.Post(ctx, endpoint, req, nil)
	if err != nil {
		return "", err
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("failed to create project: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result struct {
		Id string `json:"id"`
	}
	if err := resp.ParseJSON(&result); err != nil {
		return "", fmt.Errorf("failed to parse project summaries response: %w", err)
	}

	return result.Id, nil
}

// Delete project
// Requst type DELETE
// https://july-dev.aoa.oes.opsmx.org/gate/ssdservice/v1/scan/project/0x5f2a6?orgId=c3b62ec6-42e2-43d4-beaf-5b2017ab5848&teamId=fe2e8a09-a3f2-4263-b635-fa7e99f2d43b
// returns string and success Response
func (c *SSDClient) DeleteProject(ctx context.Context, teamIds, projectId string) (string, error) {
	// Build query parameters
	params := make([]string, 0)

	// Add default org id
	params = append(params, fmt.Sprintf("orgId=%s", c.orgID))

	// Add team IDs
	params = append(params, fmt.Sprintf("teamId=%s", teamIds))

	// Build endpoint
	endpoint := fmt.Sprintf("/gate/ssdservice/v1/scan/project/%s", projectId)
	if len(params) > 0 {
		endpoint += "?" + strings.Join(params, "&")
	}

	resp, err := c.restClient.Delete(ctx, endpoint, nil)
	if err != nil {
		return "", err
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("failed to get project summaries: status %d, body: %s", resp.StatusCode, resp.String())
	}

	return "Project deleted", nil
}

func (c *SSDClient) GetProjectSummaryCount(ctx context.Context, hubIDs []string) (*SourceScanSummaryCount, error) {

	// Build endpoint
	endpoint := "/gate/ssdservice/v1/sourceScan/summaryCount"

	params := fmt.Sprintf("teamId=%s", strings.Join(hubIDs, ","))

	resp, err := c.restClient.Get(ctx, endpoint+"?"+params, nil)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get summary count: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result SummaryCountResponse
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse summary count response: %w", err)
	}

	return &result.SourceScanSummaryCount, nil
}

// GetScanSummaryData retrieves scan summary data for a specific project
func (c *SSDClient) GetScanResultData(ctx context.Context, req *ScanResultDataRequest) (*ScanResultDataResponse, error) {
	// Build query parameters
	params := make([]string, 0)

	// Required parameters
	if req.Repository != "" {
		params = append(params, fmt.Sprintf("repository=%s", req.Repository))
	}
	if req.TeamID != "" {
		params = append(params, fmt.Sprintf("teamId=%s", req.TeamID))
	}
	if req.ProjectID != "" {
		params = append(params, fmt.Sprintf("projectId=%s", req.ProjectID))
	}
	if req.Type != "" {
		params = append(params, fmt.Sprintf("type=%s", req.Type))
	}

	// Optional parameters
	if req.Branch != "" {
		params = append(params, fmt.Sprintf("branch=%s", req.Branch))
	}

	// Build endpoint
	endpoint := "/gate/ssdservice/v1/sourceScan/summarydata"
	if len(params) > 0 {
		endpoint += "?" + strings.Join(params, "&")
	}

	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get scan result data: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result ScanResultDataResponse
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse scan result data response: %w", err)
	}

	return &result, nil
}

// GetVulnerabilityData retrieves vulnerability data for a specific scan
func (c *SSDClient) GetVulnerabilityData(ctx context.Context, req *VulnerabilityDataRequest, body interface{}) (*VulnerabilityDataResponse, error) {
	// Build query parameters
	params := make([]string, 0)

	// Required parameters
	if req.Type != "" {
		params = append(params, fmt.Sprintf("type=%s", req.Type))
	}
	if req.ProjectID != "" {
		params = append(params, fmt.Sprintf("projectId=%s", req.ProjectID))
	}
	if req.ScanID != "" {
		params = append(params, fmt.Sprintf("scanId=%s", req.ScanID))
	}

	// Build endpoint
	endpoint := "/gate/ssdservice/v1/scan/filedata"
	if len(params) > 0 {
		endpoint += "?" + strings.Join(params, "&")
	}

	resp, err := c.restClient.Post(ctx, endpoint, body, nil)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get vulnerability data: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result []VulnerabilityScanResult
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse vulnerability data response: %w", err)
	}

	return &VulnerabilityDataResponse{
		Results: result,
	}, nil
}

func (c *SSDClient) Rescan(ctx context.Context, req *RescanRequest) (*RescanResponse, error) {

	endpoint := "/gate/ssdservice/v1/rescan"

	resp, err := c.restClient.Post(ctx, endpoint, req, nil)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to trigger rescan API: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result RescanResponse
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse rescan response: %w", err)
	}

	return &result, nil
}

// SCA API
// GetVulnerabilityList retrieves vulnerability list data with pagination and filtering
func (c *SSDClient) GetVulnerabilityList(ctx context.Context, req *VulnerabilityListRequest) (*VulnerabilityListResponse, error) {
	// Build query parameters
	params := make([]string, 0)

	// Required parameters
	req.OrgID = c.orgID

	if req.TeamID != "" {
		params = append(params, fmt.Sprintf("teamId=%s", req.TeamID))
	}

	// Pagination parameters
	if req.PageNo >= 0 {
		params = append(params, fmt.Sprintf("pageNo=%d", req.PageNo))
	}

	if req.PageLimit > 0 {
		params = append(params, fmt.Sprintf("pageLimit=%d", req.PageLimit))
	}

	// Sorting parameters
	if req.SortBy != "" {
		params = append(params, fmt.Sprintf("sortBy=%s", req.SortBy))
	}

	if req.SortOrder != "" {
		params = append(params, fmt.Sprintf("sortOrder=%s", req.SortOrder))
	}

	// Filter parameters
	if req.Artifacts != "" {
		params = append(params, fmt.Sprintf("Artifacts=%s", req.Artifacts))
	}

	if req.ArtifactSha != "" {
		params = append(params, fmt.Sprintf("ArtifactSha=%s", req.ArtifactSha))
	}

	if req.Tools != "" {
		params = append(params, fmt.Sprintf("Tools=%s", req.Tools))
	}

	params = append(params, fmt.Sprintf("orgId=%s", c.orgID))

	// Build endpoint
	endpoint := "/gate/ssdservice/v1/vulnerability/inpage"
	if len(params) > 0 {
		endpoint += "?" + strings.Join(params, "&")
	}

	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get vulnerability list: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result VulnerabilityListResponse
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse vulnerability list response: %w", err)
	}

	return &result, nil
}
