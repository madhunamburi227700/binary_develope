package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/config"
	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

const (
	// Webhook response messages
	webhookErrorMessage   = "Please register repo and branch to scan and remediate vulnerabilities using AI Guardian PR scan feature."
	webhookSuccessMessage = "PR scanning started. Any new vulnerability will be reported shortly in the PR comments with links to remediate it."

	// Webhook status values
	webhookStatusError   = "error"
	webhookStatusSuccess = "success"

	// Webhook URLs
	webhookErrorURL = "https://ai-rem-demo.remediation.opsmx.net/login"

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
// TODO: Remove requestHost once the endpoint is deprecated.
// requestHost is the host of the request coming to the endpoint.
// It is used to determine if the request is coming from the deprecated '-api' endpoint.
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

func (s *ProjectService) HandleWebhookRequest(ctx context.Context, payload models.WebhookRequest, requestHost string) (string, error) {

	headProjectID, url, err := s.HandleWebhook(ctx, payload)
	if err != nil {
		s.postPRCommentWithError(ctx, payload, "", requestHost)
		return webhookErrorURL, err
	}
	s.postPRComment(ctx, payload, headProjectID, url, webhookSuccessMessage, webhookStatusSuccess, requestHost)
	return url, nil
}

// handle webhook request
func (s *ProjectService) HandleWebhook(ctx context.Context, payload models.WebhookRequest) (string, string, error) {
	successUrl := config.GetUIAddress() + "/projects"

	// filter owner from repo url
	owner, repoName, err := utils.FilterOwnerAndRepoNameFromRepoURL(payload.RepoURL)
	if err != nil {
		return webhookErrorURL, "", fmt.Errorf("failed to filter owner and repo name from repo URL: %w", err)
	}

	projectExists, err := s.checkIfProjectExistsWithOwner(ctx, owner, repoName) //proiject id
	if err != nil {
		return webhookErrorURL, "", fmt.Errorf("failed to check if project exists: %w", err)
	}

	if !projectExists {
		return webhookErrorURL, "", fmt.Errorf("project does not exist for owner: %s", owner)
	}

	// Get HubID and IntegrationID from existing project with same owner
	hubID, integrationID, err := s.getHubIDAndIntegrationIDFromOwner(ctx, owner)
	if err != nil {
		return webhookErrorURL, "", fmt.Errorf("failed to get hubID and integrationID: %w", err)
	}

	// Process base branch
	baseProjectID, baseScanID, err := s.CheckAndScanOrCreate(ctx, owner, repoName, payload.BaseBranch, hubID, integrationID)
	if err != nil {
		s.logger.LogError(err, "Failed to process base branch", map[string]interface{}{
			"baseBranch": payload.BaseBranch,
			"repoURL":    payload.RepoURL,
		})
		// Continue with head branch even if base branch fails
		baseProjectID = ""
		baseScanID = ""
	}

	// Process head branch
	headProjectID, headScanID, err := s.CheckAndScanOrCreate(ctx, owner, repoName, payload.HeadBranch, hubID, integrationID)
	if err != nil {
		s.logger.LogError(err, "Failed to process head branch", map[string]interface{}{
			"headBranch": payload.HeadBranch,
			"repoURL":    payload.RepoURL,
		})
		return webhookErrorURL, "", fmt.Errorf("failed to process head branch: %w", err)
	}

	// Store project pair in Redis only if we have both project IDs
	if baseProjectID != "" && headProjectID != "" {
		scanPairService := NewWebhookScanPairService()
		if err := scanPairService.StoreProjectPair(
			ctx,
			payload.PRNumber,
			payload.RepoURL,
			baseProjectID,
			headProjectID,
			payload.BaseBranch,
			payload.HeadBranch,
		); err != nil {
			s.logger.LogError(err, "Failed to store project pair in Redis", map[string]interface{}{
				"pr_number": payload.PRNumber,
			})
			// Don't fail the request, just log - diff processing can still work
		} else {
			// Store scan ID mappings
			if baseScanID != "" {
				if err := scanPairService.StoreScanIDMapping(ctx, baseScanID, payload.PRNumber, true); err != nil {
					s.logger.LogError(err, "Failed to store base scan ID mapping", map[string]interface{}{
						"scan_id":   baseScanID,
						"pr_number": payload.PRNumber,
					})
				}
			}
			if headScanID != "" {
				if err := scanPairService.StoreScanIDMapping(ctx, headScanID, payload.PRNumber, false); err != nil {
					s.logger.LogError(err, "Failed to store head scan ID mapping", map[string]interface{}{
						"scan_id":   headScanID,
						"pr_number": payload.PRNumber,
					})
				}
			}
		}
	}

	return headProjectID, successUrl, nil
}

// postPRCommentWithError posts an error comment to the PR
func (s *ProjectService) postPRCommentWithError(ctx context.Context, payload models.WebhookRequest, headProjectID, requestHost string) {
	s.postPRComment(ctx, payload, headProjectID, webhookErrorURL, webhookErrorMessage, webhookStatusError, requestHost)
}

// postPRComment posts a comment to the PR with webhook response details
// Uses the same format as the GitHub Actions workflow
func (s *ProjectService) postPRComment(ctx context.Context, payload models.WebhookRequest, headProjectID, url, message, status, requestHost string) {
	owner, repo, err := utils.FilterOwnerAndRepoNameFromRepoURL(payload.RepoURL)
	if err != nil {
		s.logger.LogError(err, "Failed to parse repo URL for PR comment", map[string]interface{}{
			"repo_url": payload.RepoURL,
		})
		return
	}

	githubToken, err := s.getGitHubTokenForPRComment(ctx, owner, headProjectID)
	if err != nil {
		s.logger.LogError(err, "Failed to get GitHub token for PR comment", map[string]interface{}{
			"owner":           owner,
			"head_project_id": headProjectID,
		})
		return
	}

	commentBody := s.formatPRComment(payload.PRNumber, message, status, url)

	githubClient := client.NewGitHubClient()
	_, err = githubClient.PostPRComment(ctx, githubToken, owner, repo, payload.PRNumber, commentBody)
	if err != nil {
		s.logger.LogError(err, "Failed to post PR comment", map[string]interface{}{
			"owner":     owner,
			"repo":      repo,
			"pr_number": payload.PRNumber,
		})
		return
	}

	// Post endpoint deprecation comment on the new PR
	// TODO: Remove this once the endpoint is deprecated
	s.postEndpointDeprecationComment(
		ctx,
		githubToken,
		owner,
		repo,
		payload.PRNumber,
		requestHost,
	)
}

// getGitHubTokenForPRComment gets the GitHub token for posting PR comments
func (s *ProjectService) getGitHubTokenForPRComment(ctx context.Context, owner, headProjectID string) (string, error) {
	if headProjectID != "" {
		return s.ssdService.getIntegratorToken(ctx, headProjectID)
	}

	// Fallback: get token from owner's first project
	projects, err := s.projectRepo.GetProjectsByOwner(ctx, owner)
	if err != nil {
		return "", fmt.Errorf("failed to get projects by owner: %w", err)
	}
	if len(projects) == 0 {
		return "", fmt.Errorf("no projects found for owner: %s", owner)
	}

	return s.ssdService.getIntegratorToken(ctx, projects[0].ID)
}

// formatPRComment formats the PR comment message matching the GitHub Actions format
func (s *ProjectService) formatPRComment(prNumber, message, status, url string) string {
	return fmt.Sprintf(`### PR API Callback Details

**PR Number:** %s

#### Response from AI Guardian:
**Message:** %s

**Status:** %s

**URL:** %s
`, prNumber, message, status, url)
}

// postEndpointDeprecationComment posts a comment on the workflow setup PR
// informing the user about the endpoint change.
// TODO: Remove this once the endpoint is deprecated
func (s *ProjectService) postEndpointDeprecationComment(
	ctx context.Context,
	token, owner, repo, prNumber, requestHost string,
) {
	// Only post deprecation warning if request is coming from "-api" endpoint
	if !strings.Contains(requestHost, "-api") {
		s.logger.LogInfo("Skipping deprecation comment - request not from deprecated '-api' endpoint", map[string]interface{}{
			"request_host": requestHost,
		})
		return
	}
	oldEndpoint := "https://ai-rem-demo-api.remediation.opsmx.net"
	newEndpoint := config.GetApiAddr()

	comment := fmt.Sprintf(
		`AI Guardian notice:

The AI Guardian API endpoint %q is **deprecated**.
Please update your workflow configuration to use the new endpoint %q for API requests.`,
		oldEndpoint,
		newEndpoint,
	)

	githubClient := client.NewGitHubClient()
	if _, err := githubClient.PostPRComment(ctx, token, owner, repo, prNumber, comment); err != nil {
		s.logger.LogError(err, "Failed to post workflow endpoint deprecation comment", map[string]interface{}{
			"repo":      repo,
			"pr_number": prNumber,
		})
	}
}
