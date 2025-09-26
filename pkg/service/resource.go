package service

import (
	"context"
	"fmt"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// ResourceService handles resource monitoring operations
type ResourceService struct {
	logger *utils.ErrorLogger
}

// NewResourceService creates a new resource service
func NewResourceService() *ResourceService {
	return &ResourceService{
		logger: utils.NewErrorLogger("resource_service"),
	}
}

// ResourceServiceParams holds parameters for resource operations
type ResourceServiceParams struct {
	SessionID string
	OrgID     string
}

// GetResourceCounts retrieves resource counts from SSD
func (s *ResourceService) GetResourceCounts(ctx context.Context, params ResourceServiceParams) (*client.ResourceResponse, error) {
	ssdClient := client.NewSSDClient()

	resources, err := ssdClient.GetResources(ctx)
	if err != nil {
		s.logger.LogError(err, "Failed to get resources", map[string]interface{}{
			"org_id": params.OrgID,
		})
		return nil, fmt.Errorf("failed to get resources: %w", err)
	}

	s.logger.LogInfo("Resources retrieved successfully", map[string]interface{}{
		"integrations": resources.Integrations,
		"rules":        resources.Rules,
	})

	return resources, nil
}

// GetResourceSummary retrieves a formatted resource summary
func (s *ResourceService) GetResourceSummary(ctx context.Context, params ResourceServiceParams) (*ResourceSummary, error) {
	resources, err := s.GetResourceCounts(ctx, params)
	if err != nil {
		return nil, err
	}

	summary := &ResourceSummary{
		Integrations: resources.Integrations,
		Rules:        resources.Rules,
		Total:        resources.Integrations + resources.Rules,
		Timestamp:    utils.GetCurrentTimestamp(),
	}

	s.logger.LogInfo("Resource summary retrieved", map[string]interface{}{
		"integrations": summary.Integrations,
		"rules":        summary.Rules,
		"total":        summary.Total,
	})

	return summary, nil
}

// ValidateResourceThresholds checks if resource counts exceed thresholds
func (s *ResourceService) ValidateResourceThresholds(ctx context.Context, params ResourceServiceParams, thresholds ResourceThresholds) (*ResourceValidationResult, error) {
	resources, err := s.GetResourceCounts(ctx, params)
	if err != nil {
		return nil, err
	}

	result := &ResourceValidationResult{
		IntegrationsExceeded: resources.Integrations > thresholds.MaxIntegrations,
		RulesExceeded:        resources.Rules > thresholds.MaxRules,
		CurrentIntegrations:  resources.Integrations,
		CurrentRules:         resources.Rules,
		MaxIntegrations:      thresholds.MaxIntegrations,
		MaxRules:             thresholds.MaxRules,
	}

	s.logger.LogInfo("Resource thresholds validated", map[string]interface{}{
		"integrations_exceeded": result.IntegrationsExceeded,
		"rules_exceeded":        result.RulesExceeded,
		"current_counts": map[string]int{
			"integrations": resources.Integrations,
			"rules":        resources.Rules,
		},
		"thresholds": thresholds,
	})

	return result, nil
}

// ResourceSummary represents a formatted resource summary
type ResourceSummary struct {
	Integrations int   `json:"integrations"`
	Rules        int   `json:"rules"`
	Total        int   `json:"total"`
	Timestamp    int64 `json:"timestamp"`
}

// ResourceThresholds defines resource limits
type ResourceThresholds struct {
	MaxIntegrations int `json:"max_integrations"`
	MaxRules        int `json:"max_rules"`
}

// ResourceValidationResult represents the result of threshold validation
type ResourceValidationResult struct {
	IntegrationsExceeded bool `json:"integrations_exceeded"`
	RulesExceeded        bool `json:"rules_exceeded"`
	CurrentIntegrations  int  `json:"current_integrations"`
	CurrentRules         int  `json:"current_rules"`
	MaxIntegrations      int  `json:"max_integrations"`
	MaxRules             int  `json:"max_rules"`
}
