package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"
)

// GitHubClient handles GitHub API operations
type GitHubClient struct {
	restClient *RESTClient
	baseURL    string
}

// NewGitHubClient creates a new GitHub client
func NewGitHubClient() *GitHubClient {
	restConfig := RESTClientConfig{
		BaseURL: "https://api.github.com",
		Timeout: 30 * time.Second,
		Headers: map[string]string{
			"Accept":               "application/vnd.github.v3+json",
			"Content-Type":         "application/json",
			"X-GitHub-Api-Version": "2022-11-28",
		},
	}

	restClient := NewRESTClient(restConfig)

	return &GitHubClient{
		restClient: restClient,
		baseURL:    "https://api.github.com",
	}
}

// PRCommentRequest represents a GitHub PR comment request
type PRCommentRequest struct {
	Body string `json:"body"`
}

// PRCommentResponse represents a GitHub PR comment response
type PRCommentResponse struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// PostPRComment posts a comment to a GitHub pull request
func (c *GitHubClient) PostPRComment(ctx context.Context, token, owner, repo, prNumber, comment string) (*PRCommentResponse, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}
	if owner == "" || repo == "" || prNumber == "" {
		return nil, fmt.Errorf("owner, repo, and pr_number are required")
	}

	// GitHub API endpoint for PR comments
	endpoint := fmt.Sprintf("/repos/%s/%s/issues/%s/comments", owner, repo, prNumber)

	reqBody := PRCommentRequest{
		Body: comment,
	}

	// Set authorization header
	options := &RequestOptions{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("token %s", token),
		},
	}

	resp, err := c.restClient.Post(ctx, endpoint, reqBody, options)
	if err != nil {
		return nil, fmt.Errorf("failed to post PR comment: %w", err)
	}

	if !resp.IsSuccess() {
		// Include request details in error for better debugging
		return nil, fmt.Errorf("GitHub API returned status %d for POST /repos/%s/%s/issues/%s/comments: %s", resp.StatusCode, owner, repo, prNumber, resp.String())
	}

	var commentResponse PRCommentResponse
	if err := resp.ParseJSON(&commentResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &commentResponse, nil
}

// BranchResponse represents a GitHub branch response
type BranchResponse struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

// GetBranchSHA gets the SHA of a branch
func (c *GitHubClient) GetBranchSHA(ctx context.Context, token, owner, repo, branch string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("GitHub token is required")
	}
	if owner == "" || repo == "" || branch == "" {
		return "", fmt.Errorf("owner, repo, and branch are required")
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/branches/%s", owner, repo, branch)

	options := &RequestOptions{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("token %s", token),
		},
	}

	resp, err := c.restClient.Get(ctx, endpoint, options)
	if err != nil {
		return "", fmt.Errorf("failed to get branch: %w", err)
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("GitHub API returned status %d for GET /repos/%s/%s/branches/%s: %s", resp.StatusCode, owner, repo, branch, resp.String())
	}

	var branchResponse BranchResponse
	if err := resp.ParseJSON(&branchResponse); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return branchResponse.Commit.SHA, nil
}

// CreateBranchRequest represents a GitHub create branch request
type CreateBranchRequest struct {
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}

// CreateBranchResponse represents a GitHub create branch response
type CreateBranchResponse struct {
	Ref    string `json:"ref"`
	NodeID string `json:"node_id"`
	URL    string `json:"url"`
	Object struct {
		SHA  string `json:"sha"`
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"object"`
}

// CreateBranch creates a new branch from a base SHA
// Returns (alreadyExists bool, error)
func (c *GitHubClient) CreateBranch(ctx context.Context, token, owner, repo, newBranch, baseSHA string) (bool, error) {
	if token == "" {
		return false, fmt.Errorf("GitHub token is required")
	}
	if owner == "" || repo == "" || newBranch == "" || baseSHA == "" {
		return false, fmt.Errorf("owner, repo, newBranch, and baseSHA are required")
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/git/refs", owner, repo)

	reqBody := CreateBranchRequest{
		Ref: fmt.Sprintf("refs/heads/%s", newBranch),
		SHA: baseSHA,
	}

	options := &RequestOptions{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("token %s", token),
		},
	}

	resp, err := c.restClient.Post(ctx, endpoint, reqBody, options)
	if err != nil {
		return false, fmt.Errorf("failed to create branch: %w", err)
	}

	// 422 with "Reference already exists" means branch exists - that's ok
	if resp.StatusCode == 422 {
		return true, nil // Branch already exists
	}

	if !resp.IsSuccess() {
		return false, fmt.Errorf("GitHub API returned status %d for POST /repos/%s/%s/git/refs: %s", resp.StatusCode, owner, repo, resp.String())
	}

	return false, nil // Branch created successfully
}

// CreateOrUpdateFileRequest represents a GitHub create/update file request
type CreateOrUpdateFileRequest struct {
	Message string `json:"message"`
	Content string `json:"content"`
	Branch  string `json:"branch,omitempty"`
	SHA     string `json:"sha,omitempty"` // Required when updating existing file
}

// FileContentResponse represents a GitHub file content response
type FileContentResponse struct {
	SHA string `json:"sha"`
}

// FileContentWithDataResponse represents a GitHub file content response with content
type FileContentWithDataResponse struct {
	SHA      string `json:"sha"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

// GetFileSHA gets the SHA of an existing file (returns empty string if file doesn't exist)
func (c *GitHubClient) GetFileSHA(ctx context.Context, token, owner, repo, branch, path string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("GitHub token is required")
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, branch)

	options := &RequestOptions{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("token %s", token),
		},
	}

	resp, err := c.restClient.Get(ctx, endpoint, options)
	if err != nil {
		return "", fmt.Errorf("failed to get file: %w", err)
	}

	// 404 means file doesn't exist - that's ok, return empty SHA
	if resp.StatusCode == 404 {
		return "", nil
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, resp.String())
	}

	var fileResponse FileContentResponse
	if err := resp.ParseJSON(&fileResponse); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return fileResponse.SHA, nil
}

// GetFileContent gets the content of an existing file from GitHub
// Returns the decoded file content as a string, or an error if the file doesn't exist or API call fails
func (c *GitHubClient) GetFileContent(ctx context.Context, token, owner, repo, branch, path string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("GitHub token is required")
	}
	if owner == "" || repo == "" || branch == "" || path == "" {
		return "", fmt.Errorf("owner, repo, branch, and path are required")
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, branch)

	options := &RequestOptions{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("token %s", token),
		},
	}

	resp, err := c.restClient.Get(ctx, endpoint, options)
	if err != nil {
		return "", fmt.Errorf("failed to get file: %w", err)
	}

	// 404 means file doesn't exist - return error
	if resp.StatusCode == 404 {
		return "", fmt.Errorf("file not found: %s", path)
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, resp.String())
	}

	var fileResponse FileContentWithDataResponse
	if err := resp.ParseJSON(&fileResponse); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Decode base64 content
	decodedContent, err := base64.StdEncoding.DecodeString(fileResponse.Content)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 content: %w", err)
	}

	return string(decodedContent), nil
}

// CreateOrUpdateFile creates or updates a file using GitHub Contents API
func (c *GitHubClient) CreateOrUpdateFile(ctx context.Context, token, owner, repo, branch, path, content, message string) error {
	if token == "" {
		return fmt.Errorf("GitHub token is required")
	}
	if owner == "" || repo == "" || branch == "" || path == "" || content == "" || message == "" {
		return fmt.Errorf("owner, repo, branch, path, content, and message are required")
	}

	// Check if file already exists to get its SHA
	existingSHA, err := c.GetFileSHA(ctx, token, owner, repo, branch, path)
	if err != nil {
		return fmt.Errorf("failed to check existing file: %w", err)
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)

	// Base64 encode the content
	encodedContent := base64.StdEncoding.EncodeToString([]byte(content))

	reqBody := CreateOrUpdateFileRequest{
		Message: message,
		Content: encodedContent,
		Branch:  branch,
	}

	// If file exists, include its SHA for update
	if existingSHA != "" {
		reqBody.SHA = existingSHA
	}

	options := &RequestOptions{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("token %s", token),
		},
	}

	resp, err := c.restClient.Put(ctx, endpoint, reqBody, options)
	if err != nil {
		return fmt.Errorf("failed to create/update file: %w", err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("GitHub API returned status %d for PUT /repos/%s/%s/contents/%s: %s", resp.StatusCode, owner, repo, path, resp.String())
	}

	return nil
}

// CreatePullRequestRequest represents a GitHub create PR request
type CreatePullRequestRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
}

// CreatePullRequestResponse represents a GitHub create PR response
type CreatePullRequestResponse struct {
	Number int    `json:"number"`
	URL    string `json:"html_url"`
	State  string `json:"state"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	Head   struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

// CreatePullRequest creates a pull request
func (c *GitHubClient) CreatePullRequest(ctx context.Context, token, owner, repo, title, body, head, base string) (*CreatePullRequestResponse, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}
	if owner == "" || repo == "" || title == "" || head == "" || base == "" {
		return nil, fmt.Errorf("owner, repo, title, head, and base are required")
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/pulls", owner, repo)

	reqBody := CreatePullRequestRequest{
		Title: title,
		Body:  body,
		Head:  head,
		Base:  base,
	}

	options := &RequestOptions{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("token %s", token),
		},
	}

	resp, err := c.restClient.Post(ctx, endpoint, reqBody, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("GitHub API returned status %d for POST /repos/%s/%s/pulls: %s", resp.StatusCode, owner, repo, resp.String())
	}

	var prResponse CreatePullRequestResponse
	if err := resp.ParseJSON(&prResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &prResponse, nil
}

// GetOpenPullRequest checks if there's an open PR from a specific head branch
func (c *GitHubClient) GetOpenPullRequest(ctx context.Context, token, owner, repo, headBranch, baseBranch string) (*CreatePullRequestResponse, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	// List PRs with head and base filters
	endpoint := fmt.Sprintf("/repos/%s/%s/pulls?state=open&head=%s:%s&base=%s", owner, repo, owner, headBranch, baseBranch)

	options := &RequestOptions{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("token %s", token),
		},
	}

	resp, err := c.restClient.Get(ctx, endpoint, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull requests: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, resp.String())
	}

	var prs []CreatePullRequestResponse
	if err := resp.ParseJSON(&prs); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(prs) > 0 {
		return &prs[0], nil // Return first matching open PR
	}

	return nil, nil // No open PR found
}

// UpdateBranchRef updates a branch to point to a new SHA (force update)
func (c *GitHubClient) UpdateBranchRef(ctx context.Context, token, owner, repo, branch, newSHA string) error {
	if token == "" {
		return fmt.Errorf("GitHub token is required")
	}
	if owner == "" || repo == "" || branch == "" || newSHA == "" {
		return fmt.Errorf("owner, repo, branch, and newSHA are required")
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/git/refs/heads/%s", owner, repo, branch)

	reqBody := map[string]interface{}{
		"sha":   newSHA,
		"force": true,
	}

	options := &RequestOptions{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("token %s", token),
		},
	}

	resp, err := c.restClient.Patch(ctx, endpoint, reqBody, options)
	if err != nil {
		return fmt.Errorf("failed to update branch ref: %w", err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, resp.String())
	}

	return nil
}
