package client

import (
	"context"
	"fmt"
	"time"

	"github.com/opsmx/ai-guardian-api/pkg/config"
)

// CspmMcpClient defines operations against the CSPM MCP service.
type CspmMcpClient interface {
	// Get network reachability map for an artifact by SHA digest or name+tag.
	//
	// GET /api/v1/networkmap
	GetNetworkMap(ctx context.Context, params GetNetworkMapParams) (*NetworkMapResponse, error)

	// Get a paginated list of CSPM resources, optionally filtered.
	//
	// GET /api/v1/cspm/resources
	GetCSPMResources(ctx context.Context, params GetCSPMResourcesParams) (*GetCSPMResourcesResponse, error)

	// Get an aggregate summary of CSPM resources.
	//
	// GET /api/v1/cspm/resources/summary
	GetCSPMResourcesSummary(ctx context.Context, params GetCSPMResourcesSummaryParams) (*GetCSPMResourcesSummaryResponse, error)

	// Get blast radius (BFS reachability) from a CSPM resource.
	//
	// GET /api/v1/cspm/resources/blast-radius
	GetCSPMResourceBlastRadius(ctx context.Context, params GetCSPMResourceBlastRadiusParams) (*BlastRadiusResponse, error)
}

type cspmMcpHTTPClient struct {
	restClient *RESTClient
}

// NewCspmMcpClient creates a new CSPM MCP client.
// Base address comes from config.
func NewCspmMcpClient() CspmMcpClient {
	baseURL := config.GetCSPMMCPBaseURL()

	restClient := NewRESTClient(RESTClientConfig{
		BaseURL: baseURL,
		Timeout: time.Duration(config.GetCSPMMCPTimeout()) * time.Second,
		Headers: map[string]string{
			"Accept":       "application/json",
			"Content-Type": "application/json",
		},
	})

	return &cspmMcpHTTPClient{
		restClient: restClient,
	}
}

// GetNetworkMap implements CspmMcpClient.GetNetworkMap.
//
// GET /api/v1/networkmap?name=...&tag=...   or
// GET /api/v1/networkmap?sha=...
func (c *cspmMcpHTTPClient) GetNetworkMap(
	ctx context.Context,
	params GetNetworkMapParams,
) (*NetworkMapResponse, error) {
	q := map[string]string{}

	if params.Sha != "" {
		q["sha"] = params.Sha
	} else {
		if params.Name == "" {
			return nil, fmt.Errorf("either Sha or Name must be provided")
		}
		q["name"] = params.Name
		if params.Tag != "" {
			q["tag"] = params.Tag
		}
	}

	resp, err := c.restClient.Get(ctx, "/api/v1/networkmap", &RequestOptions{
		Query: q,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call networkmap: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("networkmap request failed: status %d, body: %s",
			resp.StatusCode, resp.String())
	}

	var result NetworkMapResponse
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse networkmap response: %w", err)
	}

	return &result, nil
}

// GetCSPMResources implements CspmMcpClient.GetCSPMResources.
//
// GET /api/v1/cspm/resources?id=...&cloudProvider=...&cloudAccountName=...&resourceType=...&name=...&nameRegex=...&hasFindings=...&page=...&perPage=...
func (c *cspmMcpHTTPClient) GetCSPMResources(
	ctx context.Context,
	params GetCSPMResourcesParams,
) (*GetCSPMResourcesResponse, error) {
	q := map[string]string{}

	if params.ID != "" {
		q["id"] = params.ID
	}
	if params.CloudProvider != "" {
		q["cloudProvider"] = params.CloudProvider
	}
	if params.CloudAccountName != "" {
		q["cloudAccountName"] = params.CloudAccountName
	}
	if params.ResourceType != "" {
		q["resourceType"] = params.ResourceType
	}
	if params.Name != "" {
		q["name"] = params.Name
	}
	if params.NameRegex != "" {
		q["nameRegex"] = params.NameRegex
	}
	if params.HasFindings != nil {
		if *params.HasFindings {
			q["hasFindings"] = "true"
		} else {
			q["hasFindings"] = "false"
		}
	}
	if params.Page > 0 {
		q["page"] = fmt.Sprint(params.Page)
	}
	if params.PerPage > 0 {
		q["perPage"] = fmt.Sprint(params.PerPage)
	}

	resp, err := c.restClient.Get(ctx, "/api/v1/cspm/resources", &RequestOptions{
		Query: q,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call CSPM resources: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("CSPM resources request failed: status %d, body: %s",
			resp.StatusCode, resp.String())
	}

	var result GetCSPMResourcesResponse
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse CSPM resources response: %w", err)
	}

	return &result, nil
}

// GetCSPMResourcesSummary implements CspmMcpClient.GetCSPMResourcesSummary.
//
// GET /api/v1/cspm/resources/summary?cloudProvider=...&cloudAccountName=...&hasFindings=...
func (c *cspmMcpHTTPClient) GetCSPMResourcesSummary(
	ctx context.Context,
	params GetCSPMResourcesSummaryParams,
) (*GetCSPMResourcesSummaryResponse, error) {
	q := map[string]string{}

	if params.CloudProvider != "" {
		q["cloudProvider"] = params.CloudProvider
	}
	if params.CloudAccountName != "" {
		q["cloudAccountName"] = params.CloudAccountName
	}
	if params.HasFindings != nil {
		if *params.HasFindings {
			q["hasFindings"] = "true"
		} else {
			q["hasFindings"] = "false"
		}
	}

	resp, err := c.restClient.Get(ctx, "/api/v1/cspm/resources/summary", &RequestOptions{
		Query: q,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call CSPM resources summary: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("CSPM resources summary request failed: status %d, body: %s",
			resp.StatusCode, resp.String())
	}

	var result GetCSPMResourcesSummaryResponse
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse CSPM resources summary response: %w", err)
	}

	return &result, nil
}

// GetCSPMResourceBlastRadius implements CspmMcpClient.GetCSPMResourceBlastRadius.
//
// GET /api/v1/cspm/resources/blast-radius?id=...&maxDepth=...&cloudProvider=...&cloudAccountName=...
func (c *cspmMcpHTTPClient) GetCSPMResourceBlastRadius(
	ctx context.Context,
	params GetCSPMResourceBlastRadiusParams,
) (*BlastRadiusResponse, error) {
	q := map[string]string{}

	if params.ID != "" {
		q["id"] = params.ID
	}
	if params.MaxDepth > 0 {
		q["maxDepth"] = fmt.Sprint(params.MaxDepth)
	}
	if params.CloudProvider != "" {
		q["cloudProvider"] = params.CloudProvider
	}
	if params.CloudAccountName != "" {
		q["cloudAccountName"] = params.CloudAccountName
	}

	resp, err := c.restClient.Get(ctx, "/api/v1/cspm/resources/blast-radius", &RequestOptions{
		Query: q,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call CSPM blast-radius: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("CSPM blast-radius request failed: status %d, body: %s",
			resp.StatusCode, resp.String())
	}

	var result BlastRadiusResponse
	if err := resp.ParseJSON(&result); err != nil {
		return nil, fmt.Errorf("failed to parse CSPM blast-radius response: %w", err)
	}

	return &result, nil
}
