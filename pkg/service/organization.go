package service

import (
	"context"
	"fmt"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// OrganizationService handles organization-specific operations
type OrganizationService struct {
	ssdService *SSDService
	logger     *utils.ErrorLogger
}

// NewOrganizationService creates a new organization service
func NewOrganizationService() *OrganizationService {
	return &OrganizationService{
		ssdService: NewSSDService(),
		logger:     utils.NewErrorLogger("organization_service"),
	}
}

// OrganizationServiceParams holds parameters for organization operations
type OrganizationServiceParams struct {
	SessionID string
	OrgID     string
}

// GetOrganizationDetails retrieves detailed organization information
func (s *OrganizationService) GetOrganizationDetails(ctx context.Context, params OrganizationServiceParams) (*client.OrganizationResponse, error) {
	return s.ssdService.GetOrganizationsAndTeams(ctx)
}

// GetOrganizationSummary retrieves basic organization information
func (s *OrganizationService) GetOrganizationSummary(ctx context.Context, params OrganizationServiceParams) (*client.OrganizationResponse, error) {
	return s.ssdService.GetOrganizations(ctx)
}

// GetTeamsByOrganization retrieves all teams for an organization
func (s *OrganizationService) GetTeamsByOrganization(ctx context.Context, params OrganizationServiceParams) ([]client.Hub, error) {
	orgs, err := s.GetOrganizationDetails(ctx, params)
	if err != nil {
		return nil, err
	}

	var teams []client.Hub
	for _, org := range orgs.QueryOrganization {
		teams = append(teams, org.Teams...)
	}

	s.logger.LogInfo("Teams retrieved for organization", map[string]interface{}{
		"org_id":     params.OrgID,
		"team_count": len(teams),
	})

	return teams, nil
}

// GetTeamByName retrieves a team by name within an organization
func (s *OrganizationService) GetTeamByName(ctx context.Context, params OrganizationServiceParams, teamName string) (*client.Hub, error) {
	return s.ssdService.GetHubByName(ctx, teamName)
}

// GetTeamByID retrieves a team by ID within an organization
func (s *OrganizationService) GetTeamByID(ctx context.Context, params OrganizationServiceParams, teamID string) (*client.Hub, error) {
	return s.ssdService.GetHubByID(ctx, teamID)
}

// ValidateTeamExists checks if a team exists in the organization
func (s *OrganizationService) ValidateTeamExists(ctx context.Context, params OrganizationServiceParams, teamID string) (bool, error) {
	_, err := s.GetTeamByID(ctx, params, teamID)
	if err != nil {
		if fmt.Sprintf("%v", err) == fmt.Sprintf("team with ID '%s' not found", teamID) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetTeamIDsByName retrieves team IDs by team names
func (s *OrganizationService) GetTeamIDsByName(ctx context.Context, params OrganizationServiceParams, teamNames []string) ([]string, error) {
	var teamIDs []string

	for _, teamName := range teamNames {
		team, err := s.GetTeamByName(ctx, params, teamName)
		if err != nil {
			s.logger.LogError(err, "Failed to get team by name", map[string]interface{}{
				"team_name": teamName,
				"org_id":    params.OrgID,
			})
			return nil, fmt.Errorf("failed to get team '%s': %w", teamName, err)
		}
		teamIDs = append(teamIDs, team.ID)
	}

	s.logger.LogInfo("Team IDs retrieved by names", map[string]interface{}{
		"team_names": teamNames,
		"team_ids":   teamIDs,
	})

	return teamIDs, nil
}
