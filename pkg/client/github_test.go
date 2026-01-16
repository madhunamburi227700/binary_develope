package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestGitHubClient_GetBranchSHA_Success tests successful branch SHA retrieval
func TestGitHubClient_GetBranchSHA_Success(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	responseBody := `{"name": "main", "commit": {"sha": "abc123def456"}}`

	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.Path == "/repos/owner/repo/branches/main" &&
			req.Header.Get("Authorization") == "token test-token"
	})).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	sha, err := client.GetBranchSHA(context.Background(), "test-token", "owner", "repo", "main")

	assert.NoError(t, err)
	assert.Equal(t, "abc123def456", sha)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetBranchSHA_MissingToken tests error when token is missing
func TestGitHubClient_GetBranchSHA_MissingToken(t *testing.T) {
	client := NewGitHubClient()

	sha, err := client.GetBranchSHA(context.Background(), "", "owner", "repo", "main")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub token is required")
	assert.Empty(t, sha)
}

// TestGitHubClient_GetBranchSHA_MissingParams tests error when params are missing
func TestGitHubClient_GetBranchSHA_MissingParams(t *testing.T) {
	client := NewGitHubClient()

	sha, err := client.GetBranchSHA(context.Background(), "token", "", "repo", "main")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner, repo, and branch are required")
	assert.Empty(t, sha)
}

// TestGitHubClient_GetBranchSHA_BranchNotFound tests 404 response
func TestGitHubClient_GetBranchSHA_BranchNotFound(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 404,
		Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Branch not found"}`)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	sha, err := client.GetBranchSHA(context.Background(), "token", "owner", "repo", "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
	assert.Empty(t, sha)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_CreateBranch_Success tests successful branch creation
func TestGitHubClient_CreateBranch_Success(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	responseBody := `{"ref": "refs/heads/new-branch", "object": {"sha": "abc123"}}`

	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.Path == "/repos/owner/repo/git/refs" &&
			req.Method == "POST"
	})).Return(&http.Response{
		StatusCode: 201,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	exists, err := client.CreateBranch(context.Background(), "token", "owner", "repo", "new-branch", "abc123")

	assert.NoError(t, err)
	assert.False(t, exists)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_CreateBranch_AlreadyExists tests branch already exists scenario
func TestGitHubClient_CreateBranch_AlreadyExists(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 422,
		Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Reference already exists"}`)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	exists, err := client.CreateBranch(context.Background(), "token", "owner", "repo", "existing-branch", "abc123")

	assert.NoError(t, err)
	assert.True(t, exists) // Branch already existed
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_CreateBranch_MissingParams tests error when params are missing
func TestGitHubClient_CreateBranch_MissingParams(t *testing.T) {
	client := NewGitHubClient()

	exists, err := client.CreateBranch(context.Background(), "token", "owner", "repo", "", "abc123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner, repo, newBranch, and baseSHA are required")
	assert.False(t, exists)
}

// TestGitHubClient_GetFileSHA_Success tests successful file SHA retrieval
func TestGitHubClient_GetFileSHA_Success(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	responseBody := `{"sha": "file-sha-123", "name": "test.yml"}`

	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.Path == "/repos/owner/repo/contents/.github/workflows/test.yml" &&
			req.URL.Query().Get("ref") == "feature-branch"
	})).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	sha, err := client.GetFileSHA(context.Background(), "token", "owner", "repo", "feature-branch", ".github/workflows/test.yml")

	assert.NoError(t, err)
	assert.Equal(t, "file-sha-123", sha)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetFileSHA_FileNotFound tests 404 returns empty SHA (not error)
func TestGitHubClient_GetFileSHA_FileNotFound(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 404,
		Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Not Found"}`)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	sha, err := client.GetFileSHA(context.Background(), "token", "owner", "repo", "branch", "nonexistent.yml")

	assert.NoError(t, err) // 404 is not an error for this method
	assert.Empty(t, sha)   // Just returns empty SHA
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_CreateOrUpdateFile_CreateNew tests creating a new file
func TestGitHubClient_CreateOrUpdateFile_CreateNew(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	// First call: GetFileSHA returns 404 (file doesn't exist)
	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "GET"
	})).Return(&http.Response{
		StatusCode: 404,
		Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Not Found"}`)),
		Header:     http.Header{},
	}, nil).Once()

	// Second call: PUT to create the file
	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "PUT"
	})).Return(&http.Response{
		StatusCode: 201,
		Body:       io.NopCloser(bytes.NewBufferString(`{"content": {"sha": "new-sha"}}`)),
		Header:     http.Header{},
	}, nil).Once()

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	err := client.CreateOrUpdateFile(context.Background(), "token", "owner", "repo", "branch", "path/file.yml", "content", "commit message")

	assert.NoError(t, err)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_CreateOrUpdateFile_UpdateExisting tests updating an existing file
func TestGitHubClient_CreateOrUpdateFile_UpdateExisting(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	// First call: GetFileSHA returns existing SHA
	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "GET"
	})).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(`{"sha": "existing-sha-123"}`)),
		Header:     http.Header{},
	}, nil).Once()

	// Second call: PUT with SHA to update the file
	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "PUT"
	})).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(`{"content": {"sha": "updated-sha"}}`)),
		Header:     http.Header{},
	}, nil).Once()

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	err := client.CreateOrUpdateFile(context.Background(), "token", "owner", "repo", "branch", "path/file.yml", "new content", "update message")

	assert.NoError(t, err)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_CreatePullRequest_Success tests successful PR creation
func TestGitHubClient_CreatePullRequest_Success(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	responseBody := `{
		"number": 42,
		"html_url": "https://github.com/owner/repo/pull/42",
		"state": "open",
		"title": "Test PR",
		"head": {"ref": "feature"},
		"base": {"ref": "main"}
	}`

	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.Path == "/repos/owner/repo/pulls" &&
			req.Method == "POST"
	})).Return(&http.Response{
		StatusCode: 201,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	pr, err := client.CreatePullRequest(context.Background(), "token", "owner", "repo", "Test PR", "Description", "feature", "main")

	assert.NoError(t, err)
	assert.NotNil(t, pr)
	assert.Equal(t, 42, pr.Number)
	assert.Equal(t, "https://github.com/owner/repo/pull/42", pr.URL)
	assert.Equal(t, "open", pr.State)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_CreatePullRequest_MissingParams tests error when params are missing
func TestGitHubClient_CreatePullRequest_MissingParams(t *testing.T) {
	client := NewGitHubClient()

	pr, err := client.CreatePullRequest(context.Background(), "token", "owner", "repo", "", "body", "head", "base")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner, repo, title, head, and base are required")
	assert.Nil(t, pr)
}

// TestGitHubClient_GetOpenPullRequest_Found tests finding an existing open PR
func TestGitHubClient_GetOpenPullRequest_Found(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	responseBody := `[{
		"number": 10,
		"html_url": "https://github.com/owner/repo/pull/10",
		"state": "open",
		"title": "Existing PR",
		"head": {"ref": "feature"},
		"base": {"ref": "main"}
	}]`

	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.Path == "/repos/owner/repo/pulls" &&
			req.URL.Query().Get("state") == "open" &&
			req.Method == "GET"
	})).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	pr, err := client.GetOpenPullRequest(context.Background(), "token", "owner", "repo", "feature", "main")

	assert.NoError(t, err)
	assert.NotNil(t, pr)
	assert.Equal(t, 10, pr.Number)
	assert.Equal(t, "Existing PR", pr.Title)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetOpenPullRequest_NotFound tests when no open PR exists
func TestGitHubClient_GetOpenPullRequest_NotFound(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(`[]`)), // Empty array
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	pr, err := client.GetOpenPullRequest(context.Background(), "token", "owner", "repo", "feature", "main")

	assert.NoError(t, err)
	assert.Nil(t, pr) // No PR found, but no error
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_PostPRComment_Success tests successful PR comment posting
func TestGitHubClient_PostPRComment_Success(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	responseBody := `{
		"id": 12345,
		"body": "Test comment",
		"user": {"login": "testuser"},
		"created_at": "2024-01-01T00:00:00Z"
	}`

	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.Path == "/repos/owner/repo/issues/42/comments" &&
			req.Method == "POST"
	})).Return(&http.Response{
		StatusCode: 201,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	comment, err := client.PostPRComment(context.Background(), "token", "owner", "repo", "42", "Test comment")

	assert.NoError(t, err)
	assert.NotNil(t, comment)
	assert.Equal(t, int64(12345), comment.ID)
	assert.Equal(t, "Test comment", comment.Body)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetFileContent_Success tests successful file content retrieval
func TestGitHubClient_GetFileContent_Success(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	testContent := "name: Test Workflow\non: push"
	encodedContent := base64.StdEncoding.EncodeToString([]byte(testContent))
	responseBody := fmt.Sprintf(`{
		"sha": "file-sha-123",
		"content": "%s",
		"encoding": "base64"
	}`, encodedContent)

	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.Path == "/repos/owner/repo/contents/.github/workflows/test.yml" &&
			req.URL.Query().Get("ref") == "main"
	})).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	content, err := client.GetFileContent(context.Background(), "token", "owner", "repo", "main", ".github/workflows/test.yml")

	assert.NoError(t, err)
	assert.Equal(t, testContent, content)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetFileContent_FileNotFound tests 404 response
func TestGitHubClient_GetFileContent_FileNotFound(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 404,
		Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Not Found"}`)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	content, err := client.GetFileContent(context.Background(), "token", "owner", "repo", "main", ".github/workflows/test.yml")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
	assert.Empty(t, content)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetFileContent_MissingParams tests error when params are missing
func TestGitHubClient_GetFileContent_MissingParams(t *testing.T) {
	client := NewGitHubClient()

	content, err := client.GetFileContent(context.Background(), "token", "", "repo", "main", "path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner, repo, branch, and path are required")
	assert.Empty(t, content)
}

// TestGitHubClient_GetFileContent_MissingToken tests error when token is missing
func TestGitHubClient_GetFileContent_MissingToken(t *testing.T) {
	client := NewGitHubClient()

	content, err := client.GetFileContent(context.Background(), "", "owner", "repo", "main", "path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub token is required")
	assert.Empty(t, content)
}

// TestGitHubClient_GetBranchSHA_HTTPError tests error handling when HTTP request fails
func TestGitHubClient_GetBranchSHA_HTTPError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(nil, errors.New("network error"))

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	sha, err := client.GetBranchSHA(context.Background(), "token", "owner", "repo", "main")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get branch")
	assert.Empty(t, sha)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetBranchSHA_ParseError tests error handling when JSON parsing fails
func TestGitHubClient_GetBranchSHA_ParseError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(`invalid json`)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	sha, err := client.GetBranchSHA(context.Background(), "token", "owner", "repo", "main")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
	assert.Empty(t, sha)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_CreateBranch_MissingToken tests error when token is missing
func TestGitHubClient_CreateBranch_MissingToken(t *testing.T) {
	client := NewGitHubClient()

	exists, err := client.CreateBranch(context.Background(), "", "owner", "repo", "branch", "sha")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub token is required")
	assert.False(t, exists)
}

// TestGitHubClient_CreateBranch_HTTPError tests error handling when HTTP request fails
func TestGitHubClient_CreateBranch_HTTPError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(nil, errors.New("network error"))

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	exists, err := client.CreateBranch(context.Background(), "token", "owner", "repo", "branch", "sha")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create branch")
	assert.False(t, exists)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_CreateBranch_NonSuccessResponse tests error handling when API returns non-success
func TestGitHubClient_CreateBranch_NonSuccessResponse(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Internal Server Error"}`)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	exists, err := client.CreateBranch(context.Background(), "token", "owner", "repo", "branch", "sha")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub API returned status 500")
	assert.False(t, exists)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetFileSHA_MissingToken tests error when token is missing
func TestGitHubClient_GetFileSHA_MissingToken(t *testing.T) {
	client := NewGitHubClient()

	sha, err := client.GetFileSHA(context.Background(), "", "owner", "repo", "branch", "path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub token is required")
	assert.Empty(t, sha)
}

// TestGitHubClient_GetFileSHA_HTTPError tests error handling when HTTP request fails
func TestGitHubClient_GetFileSHA_HTTPError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(nil, errors.New("network error"))

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	sha, err := client.GetFileSHA(context.Background(), "token", "owner", "repo", "branch", "path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get file")
	assert.Empty(t, sha)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetFileSHA_NonSuccessResponse tests error handling when API returns non-success (non-404)
func TestGitHubClient_GetFileSHA_NonSuccessResponse(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Internal Server Error"}`)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	sha, err := client.GetFileSHA(context.Background(), "token", "owner", "repo", "branch", "path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub API returned status 500")
	assert.Empty(t, sha)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetFileSHA_ParseError tests error handling when JSON parsing fails
func TestGitHubClient_GetFileSHA_ParseError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(`invalid json`)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	sha, err := client.GetFileSHA(context.Background(), "token", "owner", "repo", "branch", "path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
	assert.Empty(t, sha)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetFileContent_HTTPError tests error handling when HTTP request fails
func TestGitHubClient_GetFileContent_HTTPError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(nil, errors.New("network error"))

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	content, err := client.GetFileContent(context.Background(), "token", "owner", "repo", "branch", "path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get file")
	assert.Empty(t, content)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetFileContent_NonSuccessResponse tests error handling when API returns non-success (non-404)
func TestGitHubClient_GetFileContent_NonSuccessResponse(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Internal Server Error"}`)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	content, err := client.GetFileContent(context.Background(), "token", "owner", "repo", "branch", "path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub API returned status 500")
	assert.Empty(t, content)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetFileContent_ParseError tests error handling when JSON parsing fails
func TestGitHubClient_GetFileContent_ParseError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(`invalid json`)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	content, err := client.GetFileContent(context.Background(), "token", "owner", "repo", "branch", "path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
	assert.Empty(t, content)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetFileContent_Base64DecodeError tests error handling when base64 decoding fails
func TestGitHubClient_GetFileContent_Base64DecodeError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	// Return invalid base64 content
	responseBody := `{
		"sha": "file-sha-123",
		"content": "invalid-base64!!!",
		"encoding": "base64"
	}`

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	content, err := client.GetFileContent(context.Background(), "token", "owner", "repo", "branch", "path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode base64 content")
	assert.Empty(t, content)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_CreateOrUpdateFile_MissingToken tests error when token is missing
func TestGitHubClient_CreateOrUpdateFile_MissingToken(t *testing.T) {
	client := NewGitHubClient()

	err := client.CreateOrUpdateFile(context.Background(), "", "owner", "repo", "branch", "path", "content", "message")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub token is required")
}

// TestGitHubClient_CreateOrUpdateFile_MissingParams tests error when parameters are missing
func TestGitHubClient_CreateOrUpdateFile_MissingParams(t *testing.T) {
	client := NewGitHubClient()

	err := client.CreateOrUpdateFile(context.Background(), "token", "", "repo", "branch", "path", "content", "message")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner, repo, branch, path, content, and message are required")
}

// TestGitHubClient_CreateOrUpdateFile_GetFileSHAError tests error handling when GetFileSHA fails
func TestGitHubClient_CreateOrUpdateFile_GetFileSHAError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	// First call to GetFileSHA fails
	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "GET"
	})).Return(nil, errors.New("network error")).Once()

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	err := client.CreateOrUpdateFile(context.Background(), "token", "owner", "repo", "branch", "path", "content", "message")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check existing file")
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_CreateOrUpdateFile_PutError tests error handling when PUT request fails
func TestGitHubClient_CreateOrUpdateFile_PutError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	// First call: GetFileSHA returns 404 (file doesn't exist)
	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "GET"
	})).Return(&http.Response{
		StatusCode: 404,
		Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Not Found"}`)),
		Header:     http.Header{},
	}, nil).Once()

	// Second call: PUT fails
	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "PUT"
	})).Return(nil, errors.New("network error")).Once()

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	err := client.CreateOrUpdateFile(context.Background(), "token", "owner", "repo", "branch", "path", "content", "message")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create/update file")
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_CreateOrUpdateFile_NonSuccessResponse tests error handling when PUT returns non-success
func TestGitHubClient_CreateOrUpdateFile_NonSuccessResponse(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	// First call: GetFileSHA returns 404
	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "GET"
	})).Return(&http.Response{
		StatusCode: 404,
		Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Not Found"}`)),
		Header:     http.Header{},
	}, nil).Once()

	// Second call: PUT returns error
	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "PUT"
	})).Return(&http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Internal Server Error"}`)),
		Header:     http.Header{},
	}, nil).Once()

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	err := client.CreateOrUpdateFile(context.Background(), "token", "owner", "repo", "branch", "path", "content", "message")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub API returned status 500")
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_CreatePullRequest_MissingToken tests error when token is missing
func TestGitHubClient_CreatePullRequest_MissingToken(t *testing.T) {
	client := NewGitHubClient()

	pr, err := client.CreatePullRequest(context.Background(), "", "owner", "repo", "title", "body", "head", "base")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub token is required")
	assert.Nil(t, pr)
}

// TestGitHubClient_CreatePullRequest_HTTPError tests error handling when HTTP request fails
func TestGitHubClient_CreatePullRequest_HTTPError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(nil, errors.New("network error"))

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	pr, err := client.CreatePullRequest(context.Background(), "token", "owner", "repo", "title", "body", "head", "base")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create pull request")
	assert.Nil(t, pr)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_CreatePullRequest_NonSuccessResponse tests error handling when API returns non-success
func TestGitHubClient_CreatePullRequest_NonSuccessResponse(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 400,
		Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Bad Request"}`)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	pr, err := client.CreatePullRequest(context.Background(), "token", "owner", "repo", "title", "body", "head", "base")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub API returned status 400")
	assert.Nil(t, pr)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_CreatePullRequest_ParseError tests error handling when JSON parsing fails
func TestGitHubClient_CreatePullRequest_ParseError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 201,
		Body:       io.NopCloser(bytes.NewBufferString(`invalid json`)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	pr, err := client.CreatePullRequest(context.Background(), "token", "owner", "repo", "title", "body", "head", "base")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
	assert.Nil(t, pr)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetOpenPullRequest_MissingToken tests error when token is missing
func TestGitHubClient_GetOpenPullRequest_MissingToken(t *testing.T) {
	client := NewGitHubClient()

	pr, err := client.GetOpenPullRequest(context.Background(), "", "owner", "repo", "head", "base")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub token is required")
	assert.Nil(t, pr)
}

// TestGitHubClient_GetOpenPullRequest_HTTPError tests error handling when HTTP request fails
func TestGitHubClient_GetOpenPullRequest_HTTPError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(nil, errors.New("network error"))

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	pr, err := client.GetOpenPullRequest(context.Background(), "token", "owner", "repo", "head", "base")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get pull requests")
	assert.Nil(t, pr)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetOpenPullRequest_NonSuccessResponse tests error handling when API returns non-success
func TestGitHubClient_GetOpenPullRequest_NonSuccessResponse(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(bytes.NewBufferString(`{"message": "Internal Server Error"}`)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	pr, err := client.GetOpenPullRequest(context.Background(), "token", "owner", "repo", "head", "base")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub API returned status 500")
	assert.Nil(t, pr)
	mockHTTPClient.AssertExpectations(t)
}

// TestGitHubClient_GetOpenPullRequest_ParseError tests error handling when JSON parsing fails
func TestGitHubClient_GetOpenPullRequest_ParseError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(`invalid json`)),
		Header:     http.Header{},
	}, nil)

	client := &GitHubClient{
		restClient: &RESTClient{
			baseURL:    "https://api.github.com",
			httpClient: mockHTTPClient,
			headers:    map[string]string{},
			cookies:    map[string]string{},
		},
	}

	pr, err := client.GetOpenPullRequest(context.Background(), "token", "owner", "repo", "head", "base")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
	assert.Nil(t, pr)
	mockHTTPClient.AssertExpectations(t)
}
