package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// IntegrationService handles integration-specific operations
type IntegrationService struct {
	ssdService *SSDService
	logger     *utils.ErrorLogger
}

const (
	// service name to app with oauth bridge
	serviceName = "ai-guardian-api"
)

// NewIntegrationService creates a new integration service
func NewIntegrationService() *IntegrationService {
	return &IntegrationService{
		ssdService: NewSSDService(),
		logger:     utils.NewErrorLogger("integration_service"),
	}
}

type CreateGitHubIntegrationRequest struct {
	Name    string   `json:"name"`
	Token   string   `json:"token"`
	TeamIDs []string `json:"team_ids"`

	// for app based access token decryption
	Timestamp      int64  `json:"timestamp"`
	InstallationId string `json:"installationId"`
}

type ValidateGitHubIntegrationRequest struct {
	Name    string   `json:"name"`
	Token   string   `json:"token"`
	TeamIDs []string `json:"team_ids"`
}

type InstallGitHubAppRequest struct {
	InstallationID int64    `json:"installation_id"`
	TeamIDs        []string `json:"team_ids"`
}

// GetIntegrationsByType retrieves integrations by type
func (s *IntegrationService) GetIntegrationsByType(ctx context.Context, integratorType, teamIDs string) ([]client.Integration, error) {
	ssdClient := client.NewSSDClient()

	integrations, err := ssdClient.GetIntegrations(ctx, integratorType, teamIDs)
	if err != nil {
		s.logger.LogError(err, "Failed to get integrations", map[string]interface{}{
			"integrator_type": integratorType,
			"team_ids":        teamIDs,
		})
		return nil, fmt.Errorf("failed to get integrations: %w", err)
	}

	s.logger.LogInfo("Integrations retrieved successfully", map[string]interface{}{
		"integration_count": len(integrations),
		"integrator_type":   integratorType,
	})

	return integrations, nil
}

// GetGitHubIntegrations retrieves all GitHub integrations
func (s *IntegrationService) GetGitHubIntegrations(ctx context.Context, teamIDs string) ([]client.Integration, error) {
	return s.GetIntegrationsByType(ctx, "github", teamIDs)
}

// ValidateGitHubIntegration validates a GitHub integration
func (s *IntegrationService) ValidateGitHubIntegration(ctx context.Context, req ValidateGitHubIntegrationRequest) (*client.ValidateIntegrationResponse, error) {
	encryptedToken, err := utils.EncryptToken(req.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token: %w", err)
	}

	reqClient := &client.ValidateIntegrationRequest{
		Name:           req.Name,
		IntegratorType: "github",
		Category:       "sourcetool",
		FeatureConfigs: map[string]interface{}{
			"authType": "token",
		},
		IntegratorConfigs: map[string]interface{}{
			"url":       "https://api.github.com",
			"token":     encryptedToken,
			"createdAt": fmt.Sprintf("%d", utils.GetCurrentTimestampMilliseconds()),
		},
		Team: s.createTeamAssignments(ctx, req.TeamIDs),
		ID:   utils.GenerateUUID(),
	}

	ssdClient := client.NewSSDClient()

	return ssdClient.ValidateIntegration(ctx, reqClient, req.TeamIDs)
}

// CreateIntegration creates a new integration
func (s *IntegrationService) CreateGitHubIntegration(ctx context.Context, req CreateGitHubIntegrationRequest) (string, error) {
	// Validate parameters
	if err := s.validateGitHubIntegrationParams(req); err != nil {
		return "", err
	}

	// bridgeClient, err := oauthBridge.NewClient(serviceName)
	// if err != nil {
	// 	s.logger.LogError(err, "failed to initialize oauth client", map[string]interface{}{
	// 		"name":     req.Name,
	// 		"team_ids": req.TeamIDs,
	// 	})
	// 	return "", fmt.Errorf("failed to initialize oauth client: %s", err.Error())
	// }

	// oauthDecryptedToken, err := bridgeClient.DecryptToken(req.Token, req.Timestamp)
	// if err != nil {
	// 	s.logger.LogError(err, "failed to decrypt oauth token", map[string]interface{}{
	// 		"name":      req.Name,
	// 		"team_ids":  req.TeamIDs,
	// 		"token":     req.Token,
	// 		"timestamp": req.Timestamp,
	// 	})
	// 	return "", fmt.Errorf("failed to decrypt oauth token: %s", err.Error())
	// }

	ssdClient := client.NewSSDClient()

	result, err := ssdClient.CreateGitHubIntegration(ctx, req.Name, req.Token, req.InstallationId, req.Timestamp, req.TeamIDs)
	if err != nil {
		s.logger.LogError(err, "Failed to create GitHub integration", map[string]interface{}{
			"name":     req.Name,
			"team_ids": req.TeamIDs,
		})
		return "", fmt.Errorf("failed to create GitHub integration: %w", err)
	}

	s.logger.LogInfo("GitHub integration created successfully", map[string]interface{}{
		"name":     req.Name,
		"team_ids": req.TeamIDs,
		"result":   result,
	})

	return result, nil
}

// CreateIntegration creates a new integration with full configuration
func (s *IntegrationService) CreateIntegration(ctx context.Context, req *client.CreateIntegrationRequest, teamIDs []string) (string, error) {
	// Validate request
	if err := s.validateCreateIntegrationRequest(req); err != nil {
		return "", err
	}

	ssdClient := client.NewSSDClient()

	result, err := ssdClient.CreateIntegration(ctx, req, teamIDs)
	if err != nil {
		s.logger.LogError(err, "Failed to create integration", map[string]interface{}{
			"integration_id": req.ID,
			"name":           req.Name,
		})
		return "", fmt.Errorf("failed to create integration: %w", err)
	}

	s.logger.LogInfo("Integration created successfully", map[string]interface{}{
		"integration_id": req.ID,
		"name":           req.Name,
		"result":         result,
	})

	return result, nil
}

// GetIntegrationStatus retrieves the status of integrations
func (s *IntegrationService) GetIntegrationStatus(ctx context.Context, integratorType, teamIDs string) (map[string]int, error) {
	integrations, err := s.GetIntegrationsByType(ctx, integratorType, teamIDs)
	if err != nil {
		return nil, err
	}

	statusCount := map[string]int{
		"active":   0,
		"inactive": 0,
		"pending":  0,
		"error":    0,
	}

	for _, integration := range integrations {
		status := strings.ToLower(integration.Status)
		if count, exists := statusCount[status]; exists {
			statusCount[status] = count + 1
		} else {
			statusCount["error"]++
		}
	}

	s.logger.LogInfo("Integration status retrieved", map[string]interface{}{
		"integrator_type": integratorType,
		"status_count":    statusCount,
	})

	return statusCount, nil
}

// ListActiveIntegrations retrieves only active integrations
func (s *IntegrationService) ListActiveIntegrations(ctx context.Context, integratorType, teamIDs string) ([]client.Integration, error) {
	integrations, err := s.GetIntegrationsByType(ctx, integratorType, teamIDs)
	if err != nil {
		return nil, err
	}

	var activeIntegrations []client.Integration
	for _, integration := range integrations {
		if strings.ToLower(integration.Status) == "active" {
			activeIntegrations = append(activeIntegrations, integration)
		}
	}

	s.logger.LogInfo("Active integrations retrieved", map[string]interface{}{
		"integrator_type": integratorType,
		"active_count":    len(activeIntegrations),
	})

	return activeIntegrations, nil
}

// GetResourceCounts retrieves resource counts from SSD
func (s *IntegrationService) GetResourceCounts(ctx context.Context) (*client.ResourceResponse, error) {
	ssdClient := client.NewSSDClient()

	resources, err := ssdClient.GetResources(ctx)
	if err != nil {
		s.logger.LogError(err, "Failed to get resources", map[string]interface{}{})
		return nil, fmt.Errorf("failed to get resources: %w", err)
	}

	s.logger.LogInfo("Resources retrieved successfully", map[string]interface{}{
		"integrations": resources.Integrations,
		"rules":        resources.Rules,
	})

	return resources, nil
}

func (s *IntegrationService) GetGithubAppInstallationURL(ctx context.Context) (string, error) {
	installurl, err := s.ssdService.GetGithubOauthUrl(ctx)
	if err != nil {
		return "", fmt.Errorf("error while initializing oauth client: %s", err.Error())
	}

	return installurl, nil
}

// Helper method to create team assignments
func (s *IntegrationService) createTeamAssignments(ctx context.Context, teamIDs []string) []client.TeamAssignment {
	var assignments []client.TeamAssignment

	for _, teamID := range teamIDs {
		// Get team name by ID
		team, err := s.ssdService.GetHubByID(ctx, teamID)

		if err != nil {
			s.logger.LogError(err, "Failed to get team for assignment", map[string]interface{}{
				"team_id": teamID,
			})
			continue
		}

		assignments = append(assignments, client.TeamAssignment{
			TeamName: team.Name,
			TeamID:   teamID,
		})
	}

	return assignments
}

// Validation methods
func (s *IntegrationService) validateCreateIntegrationRequest(req *client.CreateIntegrationRequest) error {
	if req.Name == "" {
		return fmt.Errorf("integration name is required")
	}
	if req.IntegratorType == "" {
		return fmt.Errorf("integrator type is required")
	}
	if req.Category == "" {
		return fmt.Errorf("category is required")
	}
	if req.ID == "" {
		return fmt.Errorf("integration ID is required")
	}
	if len(req.Team) == 0 {
		return fmt.Errorf("at least one team assignment is required")
	}
	return nil
}

func (s *IntegrationService) validateGitHubIntegrationParams(req CreateGitHubIntegrationRequest) error {
	if req.Name == "" {
		return fmt.Errorf("integration name is required")
	}
	if req.Token == "" {
		return fmt.Errorf("GitHub token is required")
	}
	if len(req.TeamIDs) == 0 {
		return fmt.Errorf("at least one team ID is required")
	}
	// required to decrypt token from oauth-bridge
	if req.Timestamp == 0 {
		return fmt.Errorf("auth generation timestamp required")
	}
	// TODO: enable for future short lived token cases
	// if req.InstallationId == "" {
	// 	return fmt.Errorf("auth generation InstallationId required")
	// }
	return nil
}
