package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/config"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

const (
	// Workflow file path
	workflowFilePath = ".github/workflows/aiguardian-remediation.yml"
	// PR title
	prTitle = "Add AI Guardian PR Scan Workflow"
	// PR description
	prDescription = `This PR adds the AI Guardian PR Scan Remediation workflow to automatically scan pull requests for vulnerabilities.

The workflow will:
- Trigger on PR open, synchronize, and reopen events
- Call the AI Guardian API to scan for vulnerabilities
- Post scan results as PR comments

Once merged, this workflow will automatically scan all future pull requests.`
)

// getWorkflowBranchName returns the workflow branch name based on target base branch
func getWorkflowBranchName(baseBranch string) string {
	return fmt.Sprintf("aiguardian-workflow-setup-%s", baseBranch)
}

// SetupWorkflowRequest represents the request to setup workflow
type SetupWorkflowRequest struct {
	IntegrationID string `json:"integration_id" validate:"required"`
	Repository    string `json:"repository" validate:"required"`
	Branch        string `json:"branch" validate:"required"`
	HubID         string `json:"hub_id" validate:"required"`
}

// SetupWorkflowResponse represents the response from workflow setup
type SetupWorkflowResponse struct {
	PRURL    string `json:"pr_url"`
	PRNumber string `json:"pr_number"`
	Branch   string `json:"branch"`
}

// WorkflowSetupService handles workflow setup operations
type WorkflowSetupService struct {
	integrationService *IntegrationService
	ssdService         *SSDService
	githubClient       *client.GitHubClient
	logger             *utils.ErrorLogger
}

// NewWorkflowSetupService creates a new workflow setup service
func NewWorkflowSetupService() *WorkflowSetupService {
	return &WorkflowSetupService{
		integrationService: NewIntegrationService(),
		githubClient:       client.NewGitHubClient(),
		ssdService:         NewSSDService(),
		logger:             utils.NewErrorLogger("workflow_setup_service"),
	}
}

// SetupWorkflow sets up the AI Guardian workflow in a GitHub repository
func (s *WorkflowSetupService) SetupWorkflow(ctx context.Context, req SetupWorkflowRequest) (*SetupWorkflowResponse, error) {

	// Get the workflow branch name based on target branch
	workflowBranch := getWorkflowBranchName(req.Branch)

	owner, err := s.ssdService.GetGithubUsername(ctx, req.IntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub username: %w", err)
	}

	// Get GitHub token from integration ID
	githubToken, err := s.integrationService.GetGitHubTokenFromIntegrationID(ctx, req.HubID, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub token: %w", err)
	}

	// Check if there's already an open PR for workflow setup
	existingPR, err := s.githubClient.GetOpenPullRequest(ctx, githubToken, owner, req.Repository, workflowBranch, req.Branch)
	if err != nil {
		s.logger.LogError(err, "Failed to check for existing PR", map[string]interface{}{"error": err.Error()})
	}

	// If open PR exists, return it
	if existingPR != nil {
		s.logger.LogInfo("Found existing open PR for workflow setup", nil)
		return &SetupWorkflowResponse{
			PRURL:    existingPR.URL,
			PRNumber: fmt.Sprintf("%d", existingPR.Number),
			Branch:   workflowBranch,
		}, nil
	}

	// Get base branch SHA
	baseSHA, err := s.githubClient.GetBranchSHA(ctx, githubToken, owner, req.Repository, req.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to get base branch SHA: %w", err)
	}

	// Create branch (or reuse if it already exists)
	branchExisted, err := s.githubClient.CreateBranch(ctx, githubToken, owner, req.Repository, workflowBranch, baseSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	// If branch already existed, force update it to latest base SHA
	if branchExisted {
		err = s.githubClient.UpdateBranchRef(ctx, githubToken, owner, req.Repository, workflowBranch, baseSHA)
		if err != nil {
			return nil, fmt.Errorf("failed to update branch to latest: %w", err)
		}
		s.logger.LogInfo("Updated existing workflow branch to latest", map[string]interface{}{})
	}

	// Create or update workflow file on the branch
	commitMessage := "Add AI Guardian PR Scan Workflow"
	err = s.githubClient.CreateOrUpdateFile(ctx, githubToken, owner, req.Repository, workflowBranch, workflowFilePath, config.GetWorkflowContent(), commitMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow file: %w", err)
	}

	// Create pull request
	prResponse, err := s.githubClient.CreatePullRequest(ctx, githubToken, owner, req.Repository, prTitle, prDescription, workflowBranch, req.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	return &SetupWorkflowResponse{
		PRURL:    prResponse.URL,
		PRNumber: fmt.Sprintf("%d", prResponse.Number),
		Branch:   workflowBranch,
	}, nil
}

// CheckWorkflowStatus checks if the workflow is setup for the given repository and branch
func (s *WorkflowSetupService) CheckWorkflowStatus(ctx context.Context, req SetupWorkflowRequest) (bool, error) {
	// Get GitHub username from integration ID
	owner, err := s.ssdService.GetGithubUsername(ctx, req.IntegrationID)
	if err != nil {
		return false, fmt.Errorf("failed to get GitHub username: %w", err)
	}

	// Get GitHub token from integration ID
	githubToken, err := s.integrationService.GetGitHubTokenFromIntegrationID(ctx, req.HubID, owner)
	if err != nil {
		return false, fmt.Errorf("failed to get GitHub token: %w", err)
	}

	// Get the expected workflow content
	expectedContent := config.GetWorkflowContent()
	if expectedContent == "" {
		return false, fmt.Errorf("workflow content is not configured")
	}

	// Retrieve the actual workflow file content from GitHub
	actualContent, err := s.githubClient.GetFileContent(ctx, githubToken, owner, req.Repository, req.Branch, workflowFilePath)
	if err != nil {
		// If file doesn't exist, return false (workflow not set up)
		if strings.Contains(err.Error(), "file not found") {
			return false, nil
		}
		// For other errors, log and return error
		s.logger.LogError(err, "Failed to get workflow file content", map[string]interface{}{
			"repository": req.Repository,
			"branch":     req.Branch,
			"path":       workflowFilePath,
		})
		return false, fmt.Errorf("failed to get workflow file: %w", err)
	}

	// Compare the actual content with expected content (exact string match)
	if actualContent == expectedContent {
		return true, nil
	}

	// Content doesn't match
	return false, nil
}
