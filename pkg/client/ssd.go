package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// SSDClientInterface defines the interface for SSD operations
type SSDClientInterface interface {
	// Vulnerability operations
	GetVulnerabilityOptimization(ctx context.Context, orgID string, teamIDs []string, suppressedFlag bool, current bool) (*VulnerabilityOptimization, error)
	GetVulnerabilityPrioritization(ctx context.Context, orgID string, teamIDs []string, suppressedFlag bool, current bool) (*VulnerabilityPriority, error)
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

	resp, err := c.ExecuteGraphQL(ctx, query, "get-Org-and-Teams-team-modal-popup")
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

	resp, err := c.ExecuteGraphQL(ctx, query, "get-Org-and-Teams-team-modal-popup")
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

// CreateCloudIntegration creates or updates a cloud integration via POST /gate/ssdservice/v1/integration.
func (c *SSDClient) CreateCloudIntegration(ctx context.Context, teamID string, req *CreateIntegrationRequest) (string, error) {
	q := url.Values{}
	q.Set("orgId", c.orgID)
	q.Set("level", "global")
	q.Set("teamId", teamID)
	endpoint := "/gate/ssdservice/v1/integration?" + q.Encode()

	resp, err := c.restClient.Post(ctx, endpoint, req, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create cloud integration: %w", err)
	}
	if !resp.IsSuccess() {
		return "", fmt.Errorf("failed to create cloud integration: status %d, body: %s", resp.StatusCode, resp.String())
	}
	return resp.String(), nil
}

// UpdateCloudIntegration updates a cloud integration via PUT /gate/ssdservice/v1/integration/{id} (includes integratorType in query).
func (c *SSDClient) UpdateCloudIntegration(ctx context.Context, teamIDs string, req *CreateIntegrationRequest) (string, error) {
	q := url.Values{}
	q.Set("integratorType", req.IntegratorType)
	q.Set("orgId", c.orgID)
	q.Set("level", "global")
	q.Set("teamId", teamIDs)
	endpoint := fmt.Sprintf("/gate/ssdservice/v1/integration/%s?%s", req.ID, q.Encode())

	resp, err := c.restClient.Put(ctx, endpoint, req, nil)
	if err != nil {
		return "", fmt.Errorf("failed to update cloud integration: %w", err)
	}
	if !resp.IsSuccess() {
		return "", fmt.Errorf("failed to update cloud integration: status %d, body: %s", resp.StatusCode, resp.String())
	}
	return resp.String(), nil
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

func (c *SSDClient) UpdateIntegration(ctx context.Context, req *CreateIntegrationRequest, teamIDs []string) (string, error) {
	teamIDsStr := strings.Join(teamIDs, ",")
	endpoint := fmt.Sprintf("/gate/ssdservice/v1/integration/%s?orgId=%s&level=global&teamId=%s",
		req.ID, c.orgID, teamIDsStr)

	resp, err := c.restClient.Put(ctx, endpoint, req, nil)
	if err != nil {
		return "", err
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("failed to update integration: status %d, body: %s", resp.StatusCode, resp.String())
	}

	return resp.String(), nil
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

func (c *SSDClient) GetIntegratorConfigForProject(ctx context.Context, platform, projectId, key string) (*GetIntegratorConfigResponse, error) {
	query := `query GetIntegratorConfig{
    queryProject(filter: { platform: { eq: "%s" }, id: "%s"}) @cascade{
        id
        name
        platform
        integratorConfigs {
            name
            status
            configs(filter: { key: { in: ["%s"] } }) {
                id
                key
                value
                encrypt
            }
        }
    }
}`

	query = fmt.Sprintf(query, platform, projectId, key)

	resp, err := c.ExecuteGraphQL(ctx, query, "get-Integrator-Config-For-Project")
	if err != nil {
		return nil, err
	}

	// Fix: Convert interface{} to []byte first, then unmarshal
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	var result GetIntegratorConfigResponse
	if err := json.Unmarshal(dataBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal integrator config response: %w", err)
	}

	return &result, nil
}

func (c *SSDClient) GetProjectStatuses(ctx context.Context, projectIds []string) ([]ProjectDetailsResponse, error) {
	var results []ProjectDetailsResponse

	// Prepare IDs quoted properly for GraphQL
	var quotedIDs []string
	for _, id := range projectIds {
		quotedIDs = append(quotedIDs, fmt.Sprintf(`"%s"`, id))
	}

	query := fmt.Sprintf(`query QueryProject {
		queryProject(filter: { id: [%s] }) {
			id
			error
			riskStatus
			team {
				id
				name
				email
			}
			scans {
            	branch
				lastScannedTime
        	}
		}
	}`, strings.Join(quotedIDs, ","))

	resp, err := c.ExecuteGraphQL(ctx, query, "get-Project-Statuses")
	if err != nil {
		fmt.Printf("Error querying projects %v: %v\n", projectIds, err)
		return nil, err
	}

	// Adjust this based on actual response structure
	var responseWrapper struct {
		QueryProject []ProjectDetailsResponse `json:"queryProject"`
	}

	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		fmt.Printf("Error marshaling response: %v\n", err)
		return nil, err
	}

	if err := json.Unmarshal(dataBytes, &responseWrapper); err != nil {
		fmt.Printf("Error unmarshaling response: %v\n", err)
		return nil, err
	}

	results = append(results, responseWrapper.QueryProject...)

	return results, nil
}

// GetAllProjectStatuses fetches all projects from SSD without filtering by project IDs
func (c *SSDClient) GetAllProjectStatuses(ctx context.Context) ([]ProjectRef, error) {
	var results []ProjectRef

	query := `query QueryProject {
		queryProject {
			id
			error
			scans {
            	branch
            	lastScannedTime
        	}
			projectConfigs {
				repository
				scheduleTime
			}
		}
	}`

	resp, err := c.ExecuteGraphQL(ctx, query, "get-All-Project-Statuses")
	if err != nil {
		fmt.Printf("Error querying all projects: %v\n", err)
		return nil, err
	}

	// Adjust this based on actual response structure
	var responseWrapper struct {
		QueryProject []ProjectRef `json:"queryProject"`
	}

	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		fmt.Printf("Error marshaling response: %v\n", err)
		return nil, err
	}

	if err := json.Unmarshal(dataBytes, &responseWrapper); err != nil {
		fmt.Printf("Error unmarshaling response: %v\n", err)
		return nil, err
	}

	results = append(results, responseWrapper.QueryProject...)

	return results, nil
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
func (c *SSDClient) ExecuteGraphQL(ctx context.Context, query string, req string) (*GraphQLResponse, error) {
	endpoint := fmt.Sprintf("/graphql?req=%s", req)

	request := GraphQLRequest{Query: query}
	resp, err := c.restClient.Post(ctx, endpoint, request, nil)
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
func (c *SSDClient) CreateGitHubIntegration(ctx context.Context, name,
	token, installationId, githubIntegratorId string, timestamp int64,
	teamIDs []string) (string, error) {

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
		ID:   githubIntegratorId,
	}

	if installationId != "" {
		req.FeatureConfigs["authType"] = "app"
		req.IntegratorConfigs["installationId"] = installationId
		req.IntegratorConfigs["createdAt"] = fmt.Sprintf("%d", timestamp)
		delete(req.IntegratorConfigs, "token")
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

// UpdateGitHubIntegration updates a GitHub integration with the given parameters
func (c *SSDClient) UpdateGitHubIntegration(ctx context.Context, name,
	token, installationId, integrationId string, timestamp int64,
	teamIDs []string) (string, error) {

	req := &CreateIntegrationRequest{
		ID:             integrationId,
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
	}

	if installationId != "" {
		req.FeatureConfigs["authType"] = "app"
		req.IntegratorConfigs["installationId"] = installationId
		req.IntegratorConfigs["createdAt"] = fmt.Sprintf("%d", timestamp)
		delete(req.IntegratorConfigs, "token")
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

	return c.UpdateIntegration(ctx, req, teamIDs)
}

// project summaries for team
func (c *SSDClient) GetProjectSummaries(ctx context.Context, req *ProjectSummaryRequest) (*ProjectSummaryResponse, error) {
	// Build query parameters
	params := make([]string, 0)

	// Add team IDs
	if req.TeamIDs != "" {
		teamIDsStr := req.TeamIDs
		params = append(params, fmt.Sprintf("teamId=%s", teamIDsStr))
	}

	params = append(params, fmt.Sprintf("orgId=%s", c.orgID))

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

	result.ProjectSummaryResponse = filterByTeamID(result.ProjectSummaryResponse, req.TeamIDs)

	return &result, nil
}

// filter by team id
func filterByTeamID(projects []ProjectSummary, teamID string) []ProjectSummary {
	var filteredProjects []ProjectSummary
	for _, project := range projects {
		if project.SummaryMetaData.TeamID == teamID {
			filteredProjects = append(filteredProjects, project)
		}
	}
	return filteredProjects
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

// GetProjectDetailsCustom get project details based on custom dgraph query
func (c *SSDClient) GetProjectDetailsCustom(ctx context.Context, projectId string) (*ProjectRef, error) {
	query := `query GetProject {
		getProject(id: "` + projectId + `") {
			id
			name
			riskStatus
			scans {
				id
				branch
				lastScannedTime
				riskStatus
				scanResults {
					id
					scanType
					resultFile
					scanTool
					riskStatus
					error
				}
				artifact {
					id
					artifactName
					artifactTag
					artifactSha
					scanData {
						id
						artifactSha
						artifactNameTag
						tool
						lastScannedAt
						vulnScanState
						scanState
						vulnCriticalCount
						vulnHighCount
						vulnMediumCount
						vulnLowCount
						vulnInfoCount
						vulnUnknownCount
						vulnNoneCount
						vulnTotalCount
					}
				}
			}
		}
	}`

	resp, err := c.ExecuteGraphQL(ctx, query, "get-Project-Details-Custom")
	if err != nil {
		return nil, err
	}

	// Fix: Convert interface{} to []byte first, then unmarshal
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	type project struct {
		ProjectRef *ProjectRef `json:"getProject"`
	}

	result := &project{}
	if err := json.Unmarshal(dataBytes, result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal custom project details response: %w", err)
	}

	return result.ProjectRef, nil
}

func (c *SSDClient) GetSASTScanResults(ctx context.Context,
	scanType, projectId, scanId string, sastReq *SASTScanRequest,
) ([]*SASTScanResult, error) {
	// Build query parameters
	params := make([]string, 0)
	params = append(params, fmt.Sprintf("type=%s", scanType))
	params = append(params, fmt.Sprintf("projectId=%s", projectId))
	params = append(params, fmt.Sprintf("scanId=%s", scanId))

	// Build endpoint
	endpoint := "/gate/ssdservice/v1/scan/filedata"
	if len(params) > 0 {
		endpoint += "?" + strings.Join(params, "&")
	}

	resp, err := c.restClient.Post(ctx, endpoint, sastReq, nil)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get sast scan results: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result []*SASTScanResult
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse sast scan results response: %w", err)
	}

	return result, nil
}

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
		return "", fmt.Errorf("failed to parse create project response: %w", err)
	}

	return result.Id, nil
}

func (c *SSDClient) UpdateProject(ctx context.Context, teamIds string, req *ProjectRef) (string, error) {
	// Build query parameters
	params := make([]string, 0)

	params = append(params, fmt.Sprintf("orgId=%s", c.orgID))

	// Add team IDs
	if teamIds != "" {
		params = append(params, fmt.Sprintf("teamIds=%s", teamIds))
	}

	// Build endpoint
	endpoint := fmt.Sprintf("/gate/ssdservice/v1/scan/project/%s", req.ID)
	if len(params) > 0 {
		endpoint += "?" + strings.Join(params, "&")
	}

	resp, err := c.restClient.Put(ctx, endpoint, req, nil)
	if err != nil {
		return "", err
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("failed to update project: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result struct {
		Id string `json:"id"`
	}
	if err := resp.ParseJSON(&result); err != nil {
		return "", fmt.Errorf("failed to parse update project response: %w", err)
	}

	return result.Id, nil
}

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

// GetVulnerabilityOptimization retrieves vulnerability optimization data
func (c *SSDClient) GetVulnerabilityOptimization(ctx context.Context, teamIDs []string, suppressedFlag, current bool) (*VulnerabilityOptimization, error) {

	// Build query parameters
	params := make([]string, 0)
	params = append(params, fmt.Sprintf("orgId=%s", c.orgID))
	params = append(params, fmt.Sprintf("suppressedFlag=%t", suppressedFlag))
	params = append(params, fmt.Sprintf("current=%t", current))

	// Add team IDs if provided
	if len(teamIDs) > 0 {
		params = append(params, fmt.Sprintf("teamId=%s", strings.Join(teamIDs, ",")))
	}

	// Build endpoint
	endpoint := fmt.Sprintf("/gate/ssdservice/v1/vulnerability/optimisation?%s", strings.Join(params, "&"))

	// Make request
	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get vulnerability optimization data: %w", err)
	}

	var response VulnerabilityOptimization
	if err := resp.ParseJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to parse vulnerability optimization response: %w", err)
	}

	return &response, nil
}

// GetVulnerabilityPrioritization retrieves vulnerability prioritization data
func (c *SSDClient) GetVulnerabilityPrioritization(ctx context.Context, teamIDs []string, suppressedFlag, current bool) (*VulnerabilityPriority, error) {

	// Build query parameters
	params := make([]string, 0)
	params = append(params, fmt.Sprintf("orgId=%s", c.orgID))
	params = append(params, fmt.Sprintf("suppressedFlag=%t", suppressedFlag))
	params = append(params, fmt.Sprintf("current=%t", current))

	// Add team IDs if provided
	if len(teamIDs) > 0 {
		params = append(params, fmt.Sprintf("teamId=%s", strings.Join(teamIDs, ",")))
	}

	// Build endpoint
	endpoint := fmt.Sprintf("/gate/ssdservice/v1/vulnerability/prioritisation?%s", strings.Join(params, "&"))

	// Make request
	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get vulnerability prioritization data: %w", err)
	}

	var response VulnerabilityPriority
	if err := resp.ParseJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to parse vulnerability prioritization response: %w", err)
	}

	return &response, nil
}

// SCA API
// GetVulnerabilityList retrieves vulnerability list data with pagination and filtering
func (c *SSDClient) GetVulnerabilityList(ctx context.Context, req *VulnerabilityListRequest) (*VulnerabilityListResponse, error) {
	// Build query parameters
	params := make([]string, 0)

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

	if req.Artifacts != "" {
		params = append(params, fmt.Sprintf("artifact=%s", req.Artifacts))
	}

	if req.ArtifactSha != "" {
		params = append(params, fmt.Sprintf("artifactSha=%s", req.ArtifactSha))
	}

	if req.Tools != "" {
		params = append(params, fmt.Sprintf("tool=%s", strings.ToLower(req.Tools)))
	}

	params = append(params, fmt.Sprintf("orgId=%s", c.orgID))

	endpoint := "/gate/ssdservice/v1/vulnerability/artifact"
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

// Github OAuth API
// GetGithubOauthUrl retrieves oauth url
func (c *SSDClient) GetGithubOauthUrl(ctx context.Context) (string, error) {

	// endpoint
	endpoint := "/gate/ssdservice/v1/github/auth/installation"

	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return "", err
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("failed to get oauth url: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result struct {
		InstallUrl string `json:"url"`
	}
	if err := resp.ParseJSON(&result); err != nil {
		return "", fmt.Errorf("failed to parse oauth url response: %w", err)
	}

	return result.InstallUrl, nil
}

func (c *SSDClient) GetRepoBranchList(ctx context.Context, qparams map[string]string) ([]string, error) {
	// Build query parameters
	params := make([]string, 0)

	// Add params
	for key, value := range qparams {
		params = append(params, fmt.Sprintf("%s=%s", key, value))
	}

	// Build endpoint
	endpoint := "/gate/ssdservice/v1/sourceScan/repobranchList"
	if len(params) > 0 {
		endpoint += "?" + strings.Join(params, "&")
	}

	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get repo branch list: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result []string
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse repo branch list response: %w", err)
	}

	return result, nil
}

func (c *SSDClient) GetSupportedIntegrators(ctx context.Context, level, teamIds string) ([]*IntegrationStatus, error) {
	// Build query parameters
	params := make([]string, 0)

	// Add orgId, level, teamId
	params = append(params, fmt.Sprintf("orgId=%s", c.orgID))
	params = append(params, fmt.Sprintf("level=%s", level))
	params = append(params, fmt.Sprintf("teamId=%s", teamIds))

	// Build endpoint
	endpoint := "/ssdservice/v1/supportedIntegrations"
	if len(params) > 0 {
		endpoint += "?" + strings.Join(params, "&")
	}

	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get supported integrators: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result struct {
		IntegratorsList []*IntegrationStatus `json:"integrationsStatus"`
	}
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse supported integrators response: %w", err)
	}

	return result.IntegratorsList, nil
}

func (c *SSDClient) DeleteIntegration(ctx context.Context, req *DeleteIntegrationRequest) error {
	endpoint := fmt.Sprintf("/gate/ssdservice/v1/integration/delete?integrationId=%s&integrationName=%s&integrationType=%s&orgId=%s&level=global&teamId=%s",
		req.IntegrationID, req.IntegrationName, req.IntegrationType, c.orgID, req.TeamID)

	resp, err := c.restClient.Delete(ctx, endpoint, nil)
	if err != nil {
		fmt.Println("error while deleting integration", err)
		return err
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("failed to delete integration: status %d, body: %s", resp.StatusCode, resp.String())
	}

	return nil
}

// Integration operations
func (c *SSDClient) GetActiveIntegrationsByTeamID(ctx context.Context, integratorType, teamID string) ([]Integration, error) {
	endpoint := fmt.Sprintf("/gate/ssdservice/v1/team/active/integration?integratorType=%s&teamId=%s",
		integratorType, teamID)

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

type ArtifactResponse struct {
	ArtifactNodes ArtifactNode `json:"artifactNode"`
}

type ArtifactNode struct {
	ArtifactName string `json:"artifactName"`
	ArtifactTag  string `json:"artifactTag"`
	ArtifactSha  string `json:"artifactSha"`
}

func (c *SSDClient) GetArtifact(ctx context.Context, commitsha, githubUrl string) ([]*ArtifactResponse, error) {
	endpoint := fmt.Sprintf("/gate/ssdservice/v1/source/artifact?commit=%s&sourcerepo=%s", commitsha, githubUrl)

	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get artifact: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result []*ArtifactResponse
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse artifact response: %w", err)
	}

	return result, nil
}

// GetCSPMDashboard proxies GET /gate/ssdservice/v1/cspm/dashboard.
func (c *SSDClient) GetCSPMDashboard(ctx context.Context, accountName, scanID, accountType string) ([]CSPMDashboardServiceRow, error) {

	q := url.Values{}
	q.Set("orgId", c.orgID)
	q.Set("accountName", accountName)
	q.Set("scanId", scanID)
	q.Set("accountType", accountType)
	endpoint := "/gate/ssdservice/v1/cspm/dashboard?" + q.Encode()

	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSPM dashboard: %w", err)
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get CSPM dashboard: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result []CSPMDashboardServiceRow
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse CSPM dashboard response: %w", err)
	}
	return result, nil
}

// GetCSPMRulesStatusSummary proxies GET /gate/ssdservice/v1/cspm/rulesStatusSummary.
func (c *SSDClient) GetCSPMRulesStatusSummary(ctx context.Context, accountName, scanID, accountType, service string) (*CSPMRulesStatusSummaryResponse, error) {

	q := url.Values{}
	q.Set("accountType", accountType)
	q.Set("accountName", accountName)
	q.Set("orgId", c.orgID)
	q.Set("service", service)
	q.Set("scanId", scanID)
	endpoint := "/gate/ssdservice/v1/cspm/rulesStatusSummary?" + q.Encode()

	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSPM rules status summary: %w", err)
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get CSPM rules status summary: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result CSPMRulesStatusSummaryResponse
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse CSPM rules status summary response: %w", err)
	}
	return &result, nil
}

// GetCSPMPolicy proxies GET /gate/ssdservice/v1/cspm/policy/{policyId}.
func (c *SSDClient) GetCSPMPolicy(ctx context.Context, policyID, policyName, accountType, accountName, scanID, service string) ([]CSPMPolicyAffectedResource, error) {
	q := url.Values{}
	q.Set("accountType", accountType)
	q.Set("accountName", accountName)
	q.Set("orgId", c.orgID)
	q.Set("scanId", scanID)
	q.Set("service", service)
	if policyName != "" {
		q.Set("policyName", policyName)
	}

	endpoint := fmt.Sprintf("/gate/ssdservice/v1/cspm/policy/%s?%s", policyID, q.Encode())
	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSPM policy: %w", err)
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get CSPM policy: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result []CSPMPolicyAffectedResource
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse CSPM policy response: %w", err)
	}
	return result, nil
}

// GetCSPMRegions proxies GET /gate/ssdservice/v1/cspm/regions.
func (c *SSDClient) GetCSPMRegions(ctx context.Context, policyName, accountType, accountName, scanID, service string) (*CSPMRegionsResponse, error) {
	q := url.Values{}
	q.Set("policyName", policyName)
	q.Set("accountType", accountType)
	q.Set("accountName", accountName)
	q.Set("orgId", c.orgID)
	q.Set("scanId", scanID)
	q.Set("service", service)

	endpoint := "/gate/ssdservice/v1/cspm/regions?" + q.Encode()
	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSPM regions: %w", err)
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get CSPM regions: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result CSPMRegionsResponse
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse CSPM regions response: %w", err)
	}
	return &result, nil
}

// GetCSPMScanResult proxies GET /gate/tool-chain/api/v1/scanResult (tool-chain).
func (c *SSDClient) GetCSPMScanResult(ctx context.Context, fileName, cloudServiceProvider, cloudAccountName, scanOperation string) (map[string]interface{}, error) {
	q := url.Values{}
	q.Set("fileName", fileName)
	q.Set("cloudServiceProvider", cloudServiceProvider)
	q.Set("cloudAccountName", cloudAccountName)
	q.Set("scanOperation", scanOperation)

	endpoint := "/gate/tool-chain/api/v1/scanResult?" + q.Encode()
	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSPM scanResult: %w", err)
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get CSPM scanResult: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result map[string]interface{}
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse CSPM scanResult response: %w", err)
	}
	return result, nil
}

// PostCSPMScan proxies POST /gate/ssd-opa/api/v1/cspmscan (OPA gate).
func (c *SSDClient) PostCSPMScan(ctx context.Context, req *CSPMScanRequestBody) (*Response, error) {
	const endpoint = "/gate/ssd-opa/api/v1/cspmscan"
	resp, err := c.restClient.Post(ctx, endpoint, req, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger CSPM scan: %w", err)
	}
	return resp, nil
}

// GetCloudSecurityIntegrations proxies GET /gate/ssdservice/v1/cloudSecurityIntegration.
func (c *SSDClient) GetCloudSecurityIntegrations(ctx context.Context, teamId string) ([]CSPMCloudSecurityIntegration, error) {
	q := url.Values{}
	q.Set("orgId", c.orgID)
	q.Set("teamId", teamId)
	endpoint := "/gate/ssdservice/v1/cloudSecurityIntegration?" + q.Encode()

	resp, err := c.restClient.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get cloud security integrations: %w", err)
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get cloud security integrations: status %d, body: %s", resp.StatusCode, resp.String())
	}

	var result []CSPMCloudSecurityIntegration
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse cloud security integrations response: %w", err)
	}
	return result, nil
}
