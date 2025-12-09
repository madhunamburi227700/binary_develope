package client

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSSDConfig(t *testing.T) {
	config := DefaultSSDConfig()

	assert.NotNil(t, config, "DefaultSSDConfig should not return nil")
	assert.Equal(t, "https://july-dev.aoa.oes.opsmx.org", config.BaseURL, "Default BaseURL should be set")
	assert.Equal(t, 30*time.Second, config.Timeout, "Default timeout should be 30 seconds")
	assert.Empty(t, config.OrgID, "Default OrgID should be empty")
	assert.Empty(t, config.SessionID, "Default SessionID should be empty")
}

func TestSSDConfigValidate_Success(t *testing.T) {
	config := &SSDConfig{
		BaseURL:   "https://example.com",
		OrgID:     "org-123",
		SessionID: "session-abc",
		Timeout:   30 * time.Second,
	}

	err := config.Validate()
	assert.NoError(t, err, "Valid configuration should not return an error")
}

func TestSSDConfigValidate_MissingBaseURL(t *testing.T) {
	config := &SSDConfig{
		OrgID:     "org-123",
		SessionID: "session-abc",
		Timeout:   30 * time.Second,
	}

	err := config.Validate()
	assert.Error(t, err, "Missing BaseURL should return an error")
	assert.Contains(t, err.Error(), "base URL is required", "Error message should mention base URL")
}

func TestSSDConfigValidate_MissingOrgID(t *testing.T) {
	config := &SSDConfig{
		BaseURL:   "https://example.com",
		SessionID: "session-abc",
		Timeout:   30 * time.Second,
	}

	err := config.Validate()
	assert.Error(t, err, "Missing OrgID should return an error")
	assert.Contains(t, err.Error(), "organization ID is required", "Error message should mention organization ID")
}

func TestSSDConfigValidate_MissingSessionID(t *testing.T) {
	config := &SSDConfig{
		BaseURL: "https://example.com",
		OrgID:   "org-123",
		Timeout: 30 * time.Second,
	}

	err := config.Validate()
	assert.Error(t, err, "Missing SessionID should return an error")
	assert.Contains(t, err.Error(), "session ID is required", "Error message should mention session ID")
}

func TestSSDConfigJSONMarshaling(t *testing.T) {
	config := &SSDConfig{
		BaseURL:   "https://example.com",
		OrgID:     "org-123",
		SessionID: "session-abc",
		Timeout:   45 * time.Second,
	}

	// Marshal to JSON
	data, err := json.Marshal(config)
	require.NoError(t, err, "Failed to marshal SSDConfig to JSON")
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var unmarshaled SSDConfig
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err, "Failed to unmarshal JSON to SSDConfig")

	// Verify fields
	assert.Equal(t, config.BaseURL, unmarshaled.BaseURL)
	assert.Equal(t, config.OrgID, unmarshaled.OrgID)
	assert.Equal(t, config.SessionID, unmarshaled.SessionID)
	assert.Equal(t, config.Timeout, unmarshaled.Timeout)
}

func TestRiskStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   RiskStatus
		expected RiskStatus
	}{
		{"Low risk", RiskStatusLowrisk, "lowrisk"},
		{"Medium risk", RiskStatusMediumrisk, "mediumrisk"},
		{"High risk", RiskStatusHighrisk, "highrisk"},
		{"Apocalypse risk", RiskStatusApocalypserisk, "apocalypserisk"},
		{"Scanning", RiskStatusScanning, "scanning"},
		{"Completed", RiskStatusCompleted, "completed"},
		{"Fail", RiskStatusFail, "fail"},
		{"Pending", RiskStatusPending, "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status)
		})
	}
}

func TestGraphQLRequestJSONMarshaling(t *testing.T) {
	request := GraphQLRequest{
		Query: "query { user { name email } }",
	}

	// Marshal to JSON
	data, err := json.Marshal(request)
	require.NoError(t, err)
	assert.Contains(t, string(data), "query")

	// Unmarshal from JSON
	var unmarshaled GraphQLRequest
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, request.Query, unmarshaled.Query)
}

func TestGraphQLResponseJSONMarshaling(t *testing.T) {
	response := GraphQLResponse{
		Data: map[string]interface{}{
			"user": map[string]string{
				"name":  "John Doe",
				"email": "john@example.com",
			},
		},
		Extensions: map[string]interface{}{
			"tracing": "enabled",
		},
		Errors: []struct {
			Message string `json:"message"`
		}{},
	}

	// Marshal to JSON
	data, err := json.Marshal(response)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var unmarshaled GraphQLResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.NotNil(t, unmarshaled.Data)
	assert.NotNil(t, unmarshaled.Extensions)
}

func TestOrganizationJSONMarshaling(t *testing.T) {
	org := Organization{
		ID:   "org-123",
		Name: "Test Organization",
		Roles: []struct {
			Permission string `json:"permission"`
		}{
			{Permission: "admin"},
			{Permission: "read"},
		},
		Teams: []Hub{
			{ID: "team-1", Name: "Team 1", Email: "team1@example.com"},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(org)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Test Organization")

	// Unmarshal from JSON
	var unmarshaled Organization
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, org.ID, unmarshaled.ID)
	assert.Equal(t, org.Name, unmarshaled.Name)
	assert.Equal(t, len(org.Roles), len(unmarshaled.Roles))
	assert.Equal(t, len(org.Teams), len(unmarshaled.Teams))
}

func TestCreateHubRequestJSONMarshaling(t *testing.T) {
	request := CreateHubRequest{
		Name:  "New Hub",
		Tag:   "production",
		Email: "hub@example.com",
	}

	// Marshal to JSON
	data, err := json.Marshal(request)
	require.NoError(t, err)
	assert.Contains(t, string(data), "New Hub")

	// Unmarshal from JSON
	var unmarshaled CreateHubRequest
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, request.Name, unmarshaled.Name)
	assert.Equal(t, request.Tag, unmarshaled.Tag)
	assert.Equal(t, request.Email, unmarshaled.Email)
}

func TestCreateHubResponseJSONMarshaling(t *testing.T) {
	response := CreateHubResponse{
		ID:    "hub-123",
		Name:  "Created Hub",
		Email: "created@example.com",
	}

	// Marshal to JSON
	data, err := json.Marshal(response)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshaled CreateHubResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, response.ID, unmarshaled.ID)
	assert.Equal(t, response.Name, unmarshaled.Name)
	assert.Equal(t, response.Email, unmarshaled.Email)
}

func TestIntegrationJSONMarshaling(t *testing.T) {
	integration := Integration{
		ID:             "int-123",
		Name:           "GitHub Integration",
		IntegratorType: "github",
		Category:       "source_control",
		Status:         "active",
		AuthType:       "oauth",
		URL:            "https://github.com",
		Team:           []string{"team-1", "team-2"},
		Environments:   []string{"dev", "prod"},
		FeatureConfigs: map[string]interface{}{
			"auto_scan": true,
		},
		IntegratorConfigs: map[string]interface{}{
			"api_key": "secret",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(integration)
	require.NoError(t, err)
	assert.Contains(t, string(data), "GitHub Integration")

	// Unmarshal from JSON
	var unmarshaled Integration
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, integration.ID, unmarshaled.ID)
	assert.Equal(t, integration.Name, unmarshaled.Name)
	assert.Equal(t, integration.IntegratorType, unmarshaled.IntegratorType)
}

func TestValidateIntegrationResponseJSONMarshaling(t *testing.T) {
	response := ValidateIntegrationResponse{
		Message: "Integration validated successfully",
	}

	// Marshal to JSON
	data, err := json.Marshal(response)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshaled ValidateIntegrationResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, response.Message, unmarshaled.Message)
}

func TestResourceResponseJSONMarshaling(t *testing.T) {
	response := ResourceResponse{
		Integrations: 5,
		Rules:        10,
	}

	// Marshal to JSON
	data, err := json.Marshal(response)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshaled ResourceResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, response.Integrations, unmarshaled.Integrations)
	assert.Equal(t, response.Rules, unmarshaled.Rules)
}

func TestProjectSummaryRequestJSONMarshaling(t *testing.T) {
	request := ProjectSummaryRequest{
		TeamIDs:     "team1,team2",
		PageNo:      1,
		PageLimit:   20,
		ProjectName: "Test Project",
		Platform:    "github",
		Status:      "active",
	}

	// Marshal to JSON
	data, err := json.Marshal(request)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshaled ProjectSummaryRequest
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, request.TeamIDs, unmarshaled.TeamIDs)
	assert.Equal(t, request.PageNo, unmarshaled.PageNo)
	assert.Equal(t, request.PageLimit, unmarshaled.PageLimit)
}

func TestVulnerabilityOptimizationJSONMarshaling(t *testing.T) {
	optimization := VulnerabilityOptimization{
		AllVulnerabilities:    100,
		UniqueVulnerabilities: 75,
		TopPriority:           10,
	}

	// Marshal to JSON
	data, err := json.Marshal(optimization)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshaled VulnerabilityOptimization
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, optimization.AllVulnerabilities, unmarshaled.AllVulnerabilities)
	assert.Equal(t, optimization.UniqueVulnerabilities, unmarshaled.UniqueVulnerabilities)
	assert.Equal(t, optimization.TopPriority, unmarshaled.TopPriority)
}

func TestRescanRequestJSONMarshaling(t *testing.T) {
	request := RescanRequest{
		ProjectID:   "proj-123",
		ProjectName: "Test Project",
		Platform:    "github",
		ScanID:      "scan-456",
		ScanType:    "SAST",
	}

	// Marshal to JSON
	data, err := json.Marshal(request)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshaled RescanRequest
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, request.ProjectID, unmarshaled.ProjectID)
	assert.Equal(t, request.ProjectName, unmarshaled.ProjectName)
	assert.Equal(t, request.ScanType, unmarshaled.ScanType)
}

func TestSummaryCountResponseJSONMarshaling(t *testing.T) {
	response := SummaryCountResponse{
		SourceScanSummaryCount: SourceScanSummaryCount{
			AutoScanEnabledRepos: 10,
			ReposRegistered:      25,
			TotalBranches:        50,
			TotalScans:           100,
			TotalProjects:        15,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(response)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshaled SummaryCountResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, response.SourceScanSummaryCount.AutoScanEnabledRepos, unmarshaled.SourceScanSummaryCount.AutoScanEnabledRepos)
	assert.Equal(t, response.SourceScanSummaryCount.TotalScans, unmarshaled.SourceScanSummaryCount.TotalScans)
}
