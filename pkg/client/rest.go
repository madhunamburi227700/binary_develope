package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// HTTPClient interface for making HTTP requests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// RESTClient provides a standard REST client for making HTTP requests
type RESTClient struct {
	baseURL    string
	httpClient HTTPClient
	headers    map[string]string
	cookies    map[string]string
}

// RESTClientConfig holds configuration for the REST client
type RESTClientConfig struct {
	BaseURL    string
	Timeout    time.Duration
	Headers    map[string]string
	Cookies    map[string]string
	HTTPClient HTTPClient
}

// NewRESTClient creates a new REST client with the given configuration
func NewRESTClient(config RESTClientConfig) *RESTClient {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.Timeout,
		}
	}

	return &RESTClient{
		baseURL:    config.BaseURL,
		httpClient: httpClient,
		headers:    config.Headers,
		cookies:    config.Cookies,
	}
}

// RequestOptions holds options for HTTP requests
type RequestOptions struct {
	Headers map[string]string
	Cookies map[string]string
	Query   map[string]string
}

// Response represents a generic HTTP response
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// Get makes a GET request to the specified endpoint
func (c *RESTClient) Get(ctx context.Context, endpoint string, options *RequestOptions) (*Response, error) {
	return c.request(ctx, http.MethodGet, endpoint, nil, options)
}

// Post makes a POST request to the specified endpoint
func (c *RESTClient) Post(ctx context.Context, endpoint string, body interface{}, options *RequestOptions) (*Response, error) {
	return c.request(ctx, http.MethodPost, endpoint, body, options)
}

// Put makes a PUT request to the specified endpoint
func (c *RESTClient) Put(ctx context.Context, endpoint string, body interface{}, options *RequestOptions) (*Response, error) {
	return c.request(ctx, http.MethodPut, endpoint, body, options)
}

// Delete makes a DELETE request to the specified endpoint
func (c *RESTClient) Delete(ctx context.Context, endpoint string, options *RequestOptions) (*Response, error) {
	return c.request(ctx, http.MethodDelete, endpoint, nil, options)
}

// Patch makes a PATCH request to the specified endpoint
func (c *RESTClient) Patch(ctx context.Context, endpoint string, body interface{}, options *RequestOptions) (*Response, error) {
	return c.request(ctx, http.MethodPatch, endpoint, body, options)
}

// request makes an HTTP request with the given method, endpoint, and body
func (c *RESTClient) request(ctx context.Context, method, endpoint string, body interface{}, options *RequestOptions) (*Response, error) {
	// Build URL
	requestURL, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add query parameters
	if options != nil && options.Query != nil {
		query := requestURL.Query()
		for key, value := range options.Query {
			query.Set(key, value)
		}
		requestURL.RawQuery = query.Encode()
	}

	// Prepare request body
	var requestBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		requestBody = bytes.NewBuffer(jsonBody)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}
	if options != nil && options.Headers != nil {
		for key, value := range options.Headers {
			req.Header.Set(key, value)
		}
	}

	// Set cookies
	for name, value := range c.cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	if options != nil && options.Cookies != nil {
		for name, value := range options.Cookies {
			req.AddCookie(&http.Cookie{Name: name, Value: value})
		}
	}

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       responseBody,
	}, nil
}

func (c *RESTClient) prepareRequest(ctx context.Context, method, endpoint string, body interface{}, options *RequestOptions) (*http.Request, error) {
	requestURL, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add query parameters
	if options != nil && options.Query != nil {
		query := requestURL.Query()
		for key, value := range options.Query {
			query.Set(key, value)
		}
		requestURL.RawQuery = query.Encode()
	}

	// Prepare request body
	var requestBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		requestBody = bytes.NewBuffer(jsonBody)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}
	if options != nil && options.Headers != nil {
		for key, value := range options.Headers {
			req.Header.Set(key, value)
		}
	}

	// Set cookies
	for name, value := range c.cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	if options != nil && options.Cookies != nil {
		for name, value := range options.Cookies {
			req.AddCookie(&http.Cookie{Name: name, Value: value})
		}
	}

	return req, nil
}

func MakeRequestOptions(headers, queryParams map[string][]string) *RequestOptions {
	options := &RequestOptions{
		Headers: make(map[string]string),
		Query:   make(map[string]string),
	}

	for key, values := range headers {
		if len(values) > 0 {
			options.Headers[key] = values[0]
		}
	}

	for key, values := range queryParams {
		if len(values) > 0 {
			options.Query[key] = values[0]
		}
	}
	return options
}

// ParseJSON parses the response body as JSON into the target interface
func (r *Response) ParseJSON(target interface{}) error {
	return json.Unmarshal(r.Body, target)
}

// IsSuccess returns true if the response status code indicates success
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// String returns the response body as a string
func (r *Response) String() string {
	return string(r.Body)
}
