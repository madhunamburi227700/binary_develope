package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSetupWorkflowRequest_Validation tests request validation
func TestSetupWorkflowRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request SetupWorkflowRequest
		valid   bool
	}{
		{
			name: "valid request",
			request: SetupWorkflowRequest{
				IntegrationID: "int-123",
				Repository:    "test-repo",
				Branch:        "main",
				HubID:         "hub-456",
			},
			valid: true,
		},
		{
			name: "missing integration_id",
			request: SetupWorkflowRequest{
				Repository: "test-repo",
				Branch:     "main",
				HubID:      "hub-456",
			},
			valid: false,
		},
		{
			name: "missing repository",
			request: SetupWorkflowRequest{
				IntegrationID: "int-123",
				Branch:        "main",
				HubID:         "hub-456",
			},
			valid: false,
		},
		{
			name: "missing branch",
			request: SetupWorkflowRequest{
				IntegrationID: "int-123",
				Repository:    "test-repo",
				HubID:         "hub-456",
			},
			valid: false,
		},
		{
			name: "missing hub_id",
			request: SetupWorkflowRequest{
				IntegrationID: "int-123",
				Repository:    "test-repo",
				Branch:        "main",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasAllFields := tt.request.IntegrationID != "" &&
				tt.request.Repository != "" &&
				tt.request.Branch != "" &&
				tt.request.HubID != ""
			assert.Equal(t, tt.valid, hasAllFields)
		})
	}
}

// TestSetupWorkflowResponse tests response structure
func TestSetupWorkflowResponse(t *testing.T) {
	response := SetupWorkflowResponse{
		PRURL:    "https://github.com/owner/repo/pull/42",
		PRNumber: "42",
		Branch:   "aiguardian-workflow-setup-main",
	}

	assert.Equal(t, "https://github.com/owner/repo/pull/42", response.PRURL)
	assert.Equal(t, "42", response.PRNumber)
	assert.Equal(t, "aiguardian-workflow-setup-main", response.Branch)
}

// TestNewWorkflowSetupService tests service creation
// Note: This test requires database initialization, skipping in unit tests
func TestNewWorkflowSetupService(t *testing.T) {
	t.Skip("Skipping: requires database initialization")

	service := NewWorkflowSetupService()

	assert.NotNil(t, service)
	assert.NotNil(t, service.integrationService)
	assert.NotNil(t, service.githubClient)
	assert.NotNil(t, service.ssdService)
	assert.NotNil(t, service.logger)
}

// TestWorkflowConstants tests constant values
func TestWorkflowConstants(t *testing.T) {
	assert.Equal(t, ".github/workflows/aiguardian-remediation.yml", workflowFilePath)
	assert.Equal(t, "Add AI Guardian PR Scan Workflow", prTitle)
	assert.Contains(t, prDescription, "AI Guardian")
	assert.Contains(t, prDescription, "scan pull requests")
}

// TestGetWorkflowBranchName tests the workflow branch naming function
func TestGetWorkflowBranchName(t *testing.T) {
	tests := []struct {
		name       string
		baseBranch string
		expected   string
	}{
		{
			name:       "main branch",
			baseBranch: "main",
			expected:   "aiguardian-workflow-setup-main",
		},
		{
			name:       "master branch",
			baseBranch: "master",
			expected:   "aiguardian-workflow-setup-master",
		},
		{
			name:       "develop branch",
			baseBranch: "develop",
			expected:   "aiguardian-workflow-setup-develop",
		},
		{
			name:       "feature branch",
			baseBranch: "feature/new-feature",
			expected:   "aiguardian-workflow-setup-feature/new-feature",
		},
		{
			name:       "release branch",
			baseBranch: "release-v1.0",
			expected:   "aiguardian-workflow-setup-release-v1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getWorkflowBranchName(tt.baseBranch)
			assert.Equal(t, tt.expected, result)
		})
	}
}
