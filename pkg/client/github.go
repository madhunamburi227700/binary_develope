package client

import (
	"context"
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
