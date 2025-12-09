package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockHTTPClient is a mock implementation of HTTPClient
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

// TestNewRESTClient tests REST client creation
func TestNewRESTClient(t *testing.T) {
	config := RESTClientConfig{
		BaseURL: "https://example.com",
		Timeout: 30 * time.Second,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Cookies: map[string]string{
			"SESSION": "session-123",
		},
	}

	client := NewRESTClient(config)

	assert.NotNil(t, client, "Client should not be nil")
	assert.Equal(t, "https://example.com", client.baseURL)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, "application/json", client.headers["Content-Type"])
	assert.Equal(t, "session-123", client.cookies["SESSION"])
}

// TestNewRESTClient_WithCustomHTTPClient tests REST client with custom HTTP client
func TestNewRESTClient_WithCustomHTTPClient(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	config := RESTClientConfig{
		BaseURL:    "https://example.com",
		Timeout:    30 * time.Second,
		HTTPClient: mockHTTPClient,
	}

	client := NewRESTClient(config)

	assert.NotNil(t, client)
	assert.Equal(t, mockHTTPClient, client.httpClient)
}

// TestRESTClient_Get_Success tests successful GET request
func TestRESTClient_Get_Success(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	responseBody := `{"status": "success"}`

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	client := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{"Content-Type": "application/json"},
		cookies:    map[string]string{},
	}

	resp, err := client.Get(context.Background(), "/test", nil)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, responseBody, resp.String())
	mockHTTPClient.AssertExpectations(t)
}

// TestRESTClient_Get_WithQueryParams tests GET with query parameters
func TestRESTClient_Get_WithQueryParams(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.Query().Get("key1") == "value1" && req.URL.Query().Get("key2") == "value2"
	})).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
		Header:     http.Header{},
	}, nil)

	client := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	options := &RequestOptions{
		Query: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	resp, err := client.Get(context.Background(), "/test", options)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	mockHTTPClient.AssertExpectations(t)
}

// TestRESTClient_Post_Success tests successful POST request
func TestRESTClient_Post_Success(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	requestBody := map[string]string{"name": "test"}
	responseBody := `{"id": "123"}`

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 201,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	client := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	resp, err := client.Post(context.Background(), "/test", requestBody, nil)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 201, resp.StatusCode)
	mockHTTPClient.AssertExpectations(t)
}

// TestRESTClient_Put_Success tests successful PUT request
func TestRESTClient_Put_Success(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	requestBody := map[string]string{"name": "updated"}

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(`{"status": "updated"}`)),
		Header:     http.Header{},
	}, nil)

	client := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	resp, err := client.Put(context.Background(), "/test", requestBody, nil)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	mockHTTPClient.AssertExpectations(t)
}

// TestRESTClient_Delete_Success tests successful DELETE request
func TestRESTClient_Delete_Success(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 204,
		Body:       io.NopCloser(bytes.NewBufferString("")),
		Header:     http.Header{},
	}, nil)

	client := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	resp, err := client.Delete(context.Background(), "/test", nil)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 204, resp.StatusCode)
	mockHTTPClient.AssertExpectations(t)
}

// TestRESTClient_NetworkError tests network error handling
func TestRESTClient_NetworkError(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(nil, errors.New("network error"))

	client := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	resp, err := client.Get(context.Background(), "/test", nil)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "network error")
	mockHTTPClient.AssertExpectations(t)
}

// TestRESTClient_InvalidURL tests invalid URL handling
func TestRESTClient_InvalidURL(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	client := &RESTClient{
		baseURL:    "://invalid-url",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	resp, err := client.Get(context.Background(), "/test", nil)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to parse URL")
}

// TestRESTClient_InvalidJSON tests invalid JSON in request body
func TestRESTClient_InvalidJSON(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	client := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	// Channel cannot be marshaled to JSON
	invalidBody := make(chan int)

	resp, err := client.Post(context.Background(), "/test", invalidBody, nil)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to marshal request body")
}

// TestResponse_ParseJSON tests JSON parsing
func TestResponse_ParseJSON(t *testing.T) {
	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	responseBody := `{"name": "test", "value": 123}`
	resp := &Response{
		StatusCode: 200,
		Body:       []byte(responseBody),
		Headers:    http.Header{},
	}

	var data TestData
	err := resp.ParseJSON(&data)

	assert.NoError(t, err)
	assert.Equal(t, "test", data.Name)
	assert.Equal(t, 123, data.Value)
}

// TestResponse_ParseJSON_InvalidJSON tests invalid JSON parsing
func TestResponse_ParseJSON_InvalidJSON(t *testing.T) {
	resp := &Response{
		StatusCode: 200,
		Body:       []byte("invalid json"),
		Headers:    http.Header{},
	}

	var data map[string]interface{}
	err := resp.ParseJSON(&data)

	assert.Error(t, err)
}

// TestResponse_IsSuccess tests success status checking
func TestResponse_IsSuccess(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"200 OK", 200, true},
		{"201 Created", 201, true},
		{"204 No Content", 204, true},
		{"299 Custom Success", 299, true},
		{"199 Below Range", 199, false},
		{"300 Redirect", 300, false},
		{"400 Bad Request", 400, false},
		{"404 Not Found", 404, false},
		{"500 Server Error", 500, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &Response{StatusCode: tt.statusCode}
			assert.Equal(t, tt.expected, resp.IsSuccess())
		})
	}
}

// TestResponse_String tests string conversion
func TestResponse_String(t *testing.T) {
	responseBody := "test response body"
	resp := &Response{
		StatusCode: 200,
		Body:       []byte(responseBody),
		Headers:    http.Header{},
	}

	assert.Equal(t, responseBody, resp.String())
}

// TestMakeRequestOptions tests request options creation
func TestMakeRequestOptions(t *testing.T) {
	headers := map[string][]string{
		"Authorization": {"Bearer token123"},
		"Accept":        {"application/json"},
	}

	queryParams := map[string][]string{
		"page":  {"1"},
		"limit": {"10"},
	}

	options := MakeRequestOptions(headers, queryParams)

	assert.NotNil(t, options)
	assert.Equal(t, "Bearer token123", options.Headers["Authorization"])
	assert.Equal(t, "application/json", options.Headers["Accept"])
	assert.Equal(t, "1", options.Query["page"])
	assert.Equal(t, "10", options.Query["limit"])
}

// TestMakeRequestOptions_EmptyValues tests empty value handling
func TestMakeRequestOptions_EmptyValues(t *testing.T) {
	headers := map[string][]string{
		"Empty": {},
	}

	queryParams := map[string][]string{
		"EmptyQuery": {},
	}

	options := MakeRequestOptions(headers, queryParams)

	assert.NotNil(t, options)
	assert.Empty(t, options.Headers["Empty"])
	assert.Empty(t, options.Query["EmptyQuery"])
}

// TestRESTClient_WithHeaders tests header setting
func TestRESTClient_WithHeaders(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Header.Get("X-Custom-Header") == "custom-value" &&
			req.Header.Get("Content-Type") == "application/json"
	})).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
		Header:     http.Header{},
	}, nil)

	client := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers: map[string]string{
			"Content-Type": "application/json",
		},
		cookies: map[string]string{},
	}

	options := &RequestOptions{
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	}

	resp, err := client.Get(context.Background(), "/test", options)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	mockHTTPClient.AssertExpectations(t)
}

// TestRESTClient_WithCookies tests cookie setting
func TestRESTClient_WithCookies(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		cookies := req.Cookies()
		sessionFound := false
		authFound := false

		for _, cookie := range cookies {
			if cookie.Name == "SESSION" && cookie.Value == "session-123" {
				sessionFound = true
			}
			if cookie.Name == "AUTH" && cookie.Value == "auth-456" {
				authFound = true
			}
		}

		return sessionFound && authFound
	})).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
		Header:     http.Header{},
	}, nil)

	client := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies: map[string]string{
			"SESSION": "session-123",
		},
	}

	options := &RequestOptions{
		Cookies: map[string]string{
			"AUTH": "auth-456",
		},
	}

	resp, err := client.Get(context.Background(), "/test", options)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	mockHTTPClient.AssertExpectations(t)
}

// TestSSDClient_GetResources tests GetResources method
func TestSSDClient_GetResources(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	responseBody := `{"integrations": 5, "rules": 10}`

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	restClient := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	ssdClient := &SSDClient{
		restClient: restClient,
		orgID:      "org-123",
		sessionID:  "session-456",
	}

	result, err := ssdClient.GetResources(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 5, result.Integrations)
	assert.Equal(t, 10, result.Rules)
	mockHTTPClient.AssertExpectations(t)
}

// TestSSDClient_GetResources_Error tests error handling
func TestSSDClient_GetResources_Error(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(bytes.NewBufferString("Internal Server Error")),
		Header:     http.Header{},
	}, nil)

	restClient := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	ssdClient := &SSDClient{
		restClient: restClient,
		orgID:      "org-123",
		sessionID:  "session-456",
	}

	result, err := ssdClient.GetResources(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get resources")
	mockHTTPClient.AssertExpectations(t)
}

// TestSSDClient_CreateHub tests CreateHub method
func TestSSDClient_CreateHub(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	responseBody := `{"id": "hub-123", "name": "Test Hub", "email": "test@example.com"}`

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 201,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	restClient := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	ssdClient := &SSDClient{
		restClient: restClient,
		orgID:      "org-123",
		sessionID:  "session-456",
	}

	req := &CreateHubRequest{
		Name:  "Test Hub",
		Email: "test@example.com",
		Tag:   "production",
	}

	result, err := ssdClient.CreateHub(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "hub-123", result.ID)
	assert.Equal(t, "Test Hub", result.Name)
	assert.Equal(t, "test@example.com", result.Email)
	mockHTTPClient.AssertExpectations(t)
}

// TestSSDClient_GetIntegrations tests GetIntegrations method
func TestSSDClient_GetIntegrations(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	responseBody := `[
		{
			"id": "int-1",
			"name": "GitHub Integration",
			"integratorType": "github",
			"category": "source_control",
			"status": "active"
		}
	]`

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	restClient := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	ssdClient := &SSDClient{
		restClient: restClient,
		orgID:      "org-123",
		sessionID:  "session-456",
	}

	result, err := ssdClient.GetIntegrations(context.Background(), "github", "team-1,team-2")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, "int-1", result[0].ID)
	assert.Equal(t, "GitHub Integration", result[0].Name)
	mockHTTPClient.AssertExpectations(t)
}

// TestSSDClient_ExecuteGraphQL tests ExecuteGraphQL method
func TestSSDClient_ExecuteGraphQL(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	responseBody := `{
		"data": {
			"organizations": [
				{"id": "org-1", "name": "Test Org"}
			]
		},
		"errors": []
	}`

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	restClient := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	ssdClient := &SSDClient{
		restClient: restClient,
		orgID:      "org-123",
		sessionID:  "session-456",
	}

	query := `query { organizations { id name } }`
	result, err := ssdClient.ExecuteGraphQL(context.Background(), query, "test-request")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Data)
	assert.Empty(t, result.Errors)
	mockHTTPClient.AssertExpectations(t)
}

// TestSSDClient_ExecuteGraphQL_WithErrors tests GraphQL errors
func TestSSDClient_ExecuteGraphQL_WithErrors(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	responseBody := `{
		"data": null,
		"errors": [
			{"message": "Field 'invalid' not found"}
		]
	}`

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	restClient := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	ssdClient := &SSDClient{
		restClient: restClient,
		orgID:      "org-123",
		sessionID:  "session-456",
	}

	query := `query { invalid { field } }`
	result, err := ssdClient.ExecuteGraphQL(context.Background(), query, "test-request")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "GraphQL errors")
	mockHTTPClient.AssertExpectations(t)
}

// TestSSDClient_SetSessionID tests session ID setter
func TestSSDClient_SetSessionID(t *testing.T) {
	restClient := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: &http.Client{},
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	ssdClient := &SSDClient{
		restClient: restClient,
		orgID:      "org-123",
		sessionID:  "old-session",
	}

	ssdClient.SetSessionID("new-session-789")

	assert.Equal(t, "new-session-789", ssdClient.sessionID)
	assert.Equal(t, "new-session-789", ssdClient.restClient.cookies["SESSION"])
}

// TestSSDClient_GetOrgID tests org ID getter
func TestSSDClient_GetOrgID(t *testing.T) {
	ssdClient := &SSDClient{
		restClient: nil,
		orgID:      "org-123",
		sessionID:  "session-456",
	}

	assert.Equal(t, "org-123", ssdClient.GetOrgID())
}

// TestSSDClient_SetOrgID tests org ID setter
func TestSSDClient_SetOrgID(t *testing.T) {
	ssdClient := &SSDClient{
		restClient: nil,
		orgID:      "old-org",
		sessionID:  "session-456",
	}

	ssdClient.SetOrgID("new-org-789")

	assert.Equal(t, "new-org-789", ssdClient.orgID)
}

// TestSSDClient_Rescan tests Rescan method
func TestSSDClient_Rescan(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	responseBody := `{"message": "Rescan initiated"}`

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	restClient := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	ssdClient := &SSDClient{
		restClient: restClient,
		orgID:      "org-123",
		sessionID:  "session-456",
	}

	req := &RescanRequest{
		ProjectID:   "proj-123",
		ProjectName: "Test Project",
		Platform:    "github",
		ScanID:      "scan-456",
		ScanType:    "SAST",
	}

	result, err := ssdClient.Rescan(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Rescan initiated", result.Message)
	mockHTTPClient.AssertExpectations(t)
}

// TestSSDClient_DeleteProject tests DeleteProject method
func TestSSDClient_DeleteProject(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString("")),
		Header:     http.Header{},
	}, nil)

	restClient := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	ssdClient := &SSDClient{
		restClient: restClient,
		orgID:      "org-123",
		sessionID:  "session-456",
	}

	result, err := ssdClient.DeleteProject(context.Background(), "team-1,team-2", "proj-123")

	assert.NoError(t, err)
	assert.Equal(t, "Project deleted", result)
	mockHTTPClient.AssertExpectations(t)
}

// TestSSDClient_DeleteIntegration tests DeleteIntegration method
func TestSSDClient_DeleteIntegration(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString("")),
		Header:     http.Header{},
	}, nil)

	restClient := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	ssdClient := &SSDClient{
		restClient: restClient,
		orgID:      "org-123",
		sessionID:  "session-456",
	}

	req := &DeleteIntegrationRequest{
		IntegrationID:   "int-123",
		IntegrationName: "Test Integration",
		IntegrationType: "github",
		TeamID:          "team-1",
	}

	err := ssdClient.DeleteIntegration(context.Background(), req)

	assert.NoError(t, err)
	mockHTTPClient.AssertExpectations(t)
}

// TestSSDClient_GetGithubOauthUrl tests GetGithubOauthUrl method
func TestSSDClient_GetGithubOauthUrl(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	responseBody := `{"url": "https://github.com/apps/test-app/installations/new"}`

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{},
	}, nil)

	restClient := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	ssdClient := &SSDClient{
		restClient: restClient,
		orgID:      "org-123",
		sessionID:  "session-456",
	}

	result, err := ssdClient.GetGithubOauthUrl(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/apps/test-app/installations/new", result)
	mockHTTPClient.AssertExpectations(t)
}

// TestSSDClient_DownloadSBOMJSON tests DownloadSBOMJSON method
func TestSSDClient_DownloadSBOMJSON(t *testing.T) {
	mockHTTPClient := new(MockHTTPClient)
	sbomData := []byte(`{"packages": [{"name": "package1"}]}`)

	mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBuffer(sbomData)),
		Header:     http.Header{},
	}, nil)

	restClient := &RESTClient{
		baseURL:    "https://api.example.com",
		httpClient: mockHTTPClient,
		headers:    map[string]string{},
		cookies:    map[string]string{},
	}

	ssdClient := &SSDClient{
		restClient: restClient,
		orgID:      "org-123",
		sessionID:  "session-456",
	}

	result, err := ssdClient.DownloadSBOMJSON(context.Background(), "sbom-file.json")

	require.NoError(t, err)
	assert.Equal(t, sbomData, result)

	// Verify it's valid JSON
	var data map[string]interface{}
	err = json.Unmarshal(result, &data)
	assert.NoError(t, err)
	mockHTTPClient.AssertExpectations(t)
}
