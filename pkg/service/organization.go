package service

import (
	"context"

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
func (s *OrganizationService) GetOrganizationDetails(ctx context.Context, session string) (*client.OrganizationResponse, error) {
	return s.ssdService.GetOrganizationsAndTeams(ctx)
}

// GetOrganizationSummary retrieves basic organization information
func (s *OrganizationService) GetOrganizationSummary(ctx context.Context) (*client.OrganizationResponse, error) {
	return s.ssdService.GetOrganizations(ctx)
}
