package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// HubService handles hub operations using SSD APIs
type HubService struct {
	ssdService *SSDService
	logger     *utils.ErrorLogger
}

// NewHubService creates a new hub service
func NewHubService() *HubService {
	return &HubService{
		ssdService: NewSSDService(),
		logger:     utils.NewErrorLogger("hub_service"),
	}
}

// HubListRequest for hub operations
type HubListRequest struct {
	ID             *uuid.UUID `json:"id"`
	Name           *string    `json:"name"`
	Description    *string    `json:"description"`
	OwnerEmail     *string    `json:"owner_email"`
	CollaboratorID *uuid.UUID `json:"collaborator_id"`
	Search         string     `json:"search"`
	Page           int        `json:"page"`
	PageSize       int        `json:"page_size"`
	OrderBy        string     `json:"order_by"`
	OrderDir       string     `json:"order_dir"`
}

// CreateHubRequest represents a request to create a hub
type CreateHubRequest struct {
	Name  string `json:"name" validate:"required,min=1,max=255"`
	Tag   string `json:"tag"`
	Email string `json:"email" validate:"required,email"`
}

// CreateHubResponse represents the response from creating a hub
type CreateHubResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// HubListResponse represents the response for listing hubs
type HubListResponse struct {
	Hubs  []client.Hub `json:"hubs"`
	Total int          `json:"total"`
}

// CreateHub creates a new hub using SSD APIs
func (s *HubService) CreateHub(ctx context.Context, req *CreateHubRequest) (*CreateHubResponse, error) {
	// Validate request
	if err := s.validateCreateHubRequest(req); err != nil {
		return nil, err
	}

	// Check if hub already exists by name
	hub, err := s.ssdService.GetHubByName(ctx, req.Name)
	if err != nil {
		// If error is "not found", that's expected for new hubs - continue
		if !strings.Contains(err.Error(), "not found") {
			return nil, fmt.Errorf("failed to check if hub exists: %w", err)
		}
		// Hub doesn't exist, which is what we want for creation
	} else if hub != nil {
		// Hub exists, return error
		return nil, fmt.Errorf("hub with name '%s' already exists", req.Name)
	}

	// Convert to client request
	clientReq := &client.CreateHubRequest{
		Name:  req.Name,
		Tag:   req.Tag,
		Email: req.Email,
	}

	// Create hub using SSD service
	clientResp, err := s.ssdService.CreateHub(ctx, clientReq)
	if err != nil {
		s.logger.LogError(err, "Failed to create hub", map[string]interface{}{
			"name": req.Name,
		})
		return nil, fmt.Errorf("failed to create hub: %w", err)
	}

	// Check if clientResp is nil before accessing its fields
	if clientResp == nil {
		return nil, fmt.Errorf("received nil response from SSD service")
	}

	fmt.Println("clientResp ", clientResp)

	// Convert response
	response := &CreateHubResponse{
		ID:    clientResp.ID,
		Name:  clientResp.Name,
		Email: clientResp.Email,
	}

	s.logger.LogInfo("Hub created successfully via SSD", map[string]interface{}{
		"hub_id": response.ID,
		"name":   response.Name,
	})

	return response, nil
}

// List retrieves all hubs for an organization using SSD APIs
func (s *HubService) List(ctx context.Context, email string) (*HubListResponse, error) {
	// Get organizations and teams (hubs) from SSD
	orgs, err := s.ssdService.GetOrganizationsAndTeams(ctx)
	if err != nil {
		s.logger.LogError(err, "Failed to list hubs", map[string]interface{}{})
		return nil, fmt.Errorf("failed to list hubs: %w", err)
	}

	var hubs []client.Hub
	for _, org := range orgs.QueryOrganization {
		hubs = append(hubs, org.Teams...)
	}

	// Apply filtering if needed
	if email != "" {
		var filteredHubs []client.Hub
		for _, hub := range hubs {
			if hub.Email == email {
				filteredHubs = append(filteredHubs, hub)
			}
		}
		hubs = filteredHubs
	}

	response := &HubListResponse{
		Hubs:  hubs,
		Total: len(hubs),
	}

	return response, nil
}

// GetByID retrieves a hub by ID using SSD APIs
func (s *HubService) GetByID(ctx context.Context, hubID string) (*client.Hub, error) {
	hub, err := s.ssdService.GetHubByID(ctx, hubID)
	if err != nil {
		s.logger.LogError(err, "Failed to get hub by ID", map[string]interface{}{
			"hub_id": hubID,
		})
		return nil, fmt.Errorf("failed to get hub by ID: %w", err)
	}

	s.logger.LogInfo("Hub retrieved by ID successfully", map[string]interface{}{
		"hub_id":   hub.ID,
		"hub_name": hub.Name,
	})

	return hub, nil
}

// GetByName retrieves a hub by name using SSD APIs
func (s *HubService) GetByName(ctx context.Context, hubName string) (*client.Hub, error) {
	hub, err := s.ssdService.GetHubByName(ctx, hubName)
	if err != nil {
		s.logger.LogError(err, "Failed to get hub by name", map[string]interface{}{
			"hub_name": hubName,
		})
		return nil, fmt.Errorf("failed to get hub by name: %w", err)
	}

	s.logger.LogInfo("Hub retrieved by name successfully", map[string]interface{}{
		"hub_id":   hub.ID,
		"hub_name": hub.Name,
	})

	return hub, nil
}

// ValidateHubExists checks if a hub exists
func (s *HubService) ValidateHubExists(ctx context.Context, hubID string) (bool, error) {
	_, err := s.GetByID(ctx, hubID)
	if err != nil {
		if fmt.Sprintf("%v", err) == fmt.Sprintf("team with ID '%s' not found", hubID) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Validation methods
func (s *HubService) validateCreateHubRequest(req *CreateHubRequest) error {
	if req.Name == "" {
		return fmt.Errorf("hub name is required")
	}
	if len(req.Name) > 255 {
		return fmt.Errorf("hub name must be less than 255 characters")
	}
	if req.Email == "" {
		return fmt.Errorf("hub email is required")
	}
	// Basic email validation
	if !utils.IsValidEmail(req.Email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}
