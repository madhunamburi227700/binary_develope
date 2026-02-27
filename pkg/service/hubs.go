package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type HubStats struct {
	TotalProjects       int                  `json:"total_projects"`
	TotalCompletedScans int                  `json:"total_completed_scans"`
	TotalPRRaised       int                  `json:"total_pr_raised"`
	TotalScanning       int                  `json:"total_scanning"`
	SCAVulnerabilities  *VulnerabilityCounts `json:"sca_vulnerabilities"`
	SASTVulnerabilities *VulnerabilityCounts `json:"sast_vulnerabilities"`
	ScanTimeFrequencies []ScanTimeFrequency  `json:"scan_time_frequencies"`
	RecentScans         []ProjectRecentScan  `json:"recent_scans"`
}

type ScanSummary struct {
	ScanID    string         `json:"scan_id"`
	Branch    string         `json:"branch"`
	CommitID  string         `json:"commit_id"`
	Timestamp time.Time      `json:"timestamp"`
	Issues    map[string]int `json:"issues"`
}

type ProjectRecentScan struct {
	ProjectID    string       `json:"project_id"`
	ProjectName  string       `json:"project_name"`
	Repository   string       `json:"repository"`
	Organisation string       `json:"organisation"`
	Scan         *ScanSummary `json:"scan"`
}

// HubService handles hub operations using SSD APIs
type HubService struct {
	ssdService      *SSDService
	logger          *utils.ErrorLogger
	scanRepo        *repository.ScanRepository
	remediationRepo *repository.RemediationRepository
}

// NewHubService creates a new hub service
func NewHubService() *HubService {
	return &HubService{
		ssdService:      NewSSDService(),
		logger:          utils.NewErrorLogger("hub_service"),
		scanRepo:        repository.NewScanRepository(),
		remediationRepo: repository.NewRemediationRepository(),
	}
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

// GetHubStats retrieves a hub stats by ID
func (s *HubService) GetHubStats(ctx context.Context, hubId string) (*HubStats, error) {
	projects, err := s.scanRepo.GetHubScansVulns(ctx, hubId)
	if err != nil {
		s.logger.LogError(err, "Failed to get hub scans details", map[string]interface{}{
			"hubId": hubId,
		})
		return nil, fmt.Errorf("failed to get hub scans details: %w", err)
	}

	// getting remediations
	remediations, err := s.remediationRepo.List(ctx, &repository.QueryOptions{
		OrderBy: "created_at"})
	if err != nil {
		return nil, err
	}

	remediationsMap := map[string][]models.Remediation{}
	for _, r := range remediations.Data {
		remediationsMap[r.VulnerabilityID.String()] = append(remediationsMap[r.VulnerabilityID.String()], *r)
	}

	var sastAllCount, sastCriticalCount, sastHighCount, sastMediumCount, sastLowCount, sastUnknownCount int
	var scaAllCount, scaCriticalCount, scaHighCount, scaMediumCount, scaLowCount, scaUnknownCount int
	var totalCompletedScans, totalPrRaised, totalScanning int
	var recentScans []ProjectRecentScan
	var scanTimeFrequencies []ScanTimeFrequency
	uniqueSCAVulns, scanFreqIdx := map[string]bool{}, map[string]int{}

	for _, project := range projects {
		totalCompletedScans += len(project.Scans)
		for j, entry := range project.Scans {
			if models.ScanStatus(entry.Status) != models.ScanStatusCompleted {
				totalCompletedScans--
				totalScanning++
				continue
			}
			if !entry.EndTime.IsZero() {
				formattedDate := entry.EndTime.Format("2006-01-02")
				if idx, idxOk := scanFreqIdx[formattedDate]; idxOk {
					prevCount := *scanTimeFrequencies[idx].Count
					prevCount++
					scanTimeFrequencies[idx].Count = &prevCount
				} else {
					endTime := *entry.EndTime
					defCount := 1
					scanTimeFrequencies = append(scanTimeFrequencies, ScanTimeFrequency{
						Count: &defCount,
						Date:  &endTime,
					})
					scanFreqIdx[formattedDate] = len(scanTimeFrequencies) - 1
				}
			}

			sastStats, scaStats, uniqueSCAVulnsFromStats, prRaisedCountFromStats := calculateHubVulnStats(entry.Vulnerabilites, uniqueSCAVulns, remediationsMap)
			if j == 0 {
				uniqueSCAVulns = uniqueSCAVulnsFromStats
				totalPrRaised += prRaisedCountFromStats

				sastAllCount += *sastStats.AllCount
				sastCriticalCount += *sastStats.CriticalCount
				sastMediumCount += *sastStats.MediumCount
				sastHighCount += *sastStats.HighCount
				sastLowCount += *sastStats.LowCount
				sastUnknownCount += *sastStats.UnknownCount

				scaAllCount += *scaStats.AllCount
				scaCriticalCount += *scaStats.CriticalCount
				scaMediumCount += *scaStats.MediumCount
				scaHighCount += *scaStats.HighCount
				scaLowCount += *scaStats.LowCount
				scaUnknownCount += *scaStats.UnknownCount

				// Count vulnerabilities by severity
				issues := make(map[string]int)
				issues["critical"] = *sastStats.CriticalCount + *scaStats.CriticalCount
				issues["high"] = *sastStats.HighCount + *scaStats.HighCount
				issues["medium"] = *sastStats.MediumCount + *scaStats.MediumCount
				issues["low"] = *sastStats.LowCount + *scaStats.LowCount
				issues["unknown"] = *sastStats.UnknownCount + *scaStats.UnknownCount

				// Create a summary of the scan
				scanSummary := &ScanSummary{
					ScanID:    entry.ScanId,
					Branch:    entry.Branch,
					CommitID:  entry.CommitSHA,
					Timestamp: *entry.EndTime,
					Issues:    issues,
				}

				// Extract 7 characters from index 0 to 6
				if len(scanSummary.CommitID) >= 7 {
					scanSummary.CommitID = scanSummary.CommitID[0:7]
				}

				// for hub stats would use only latest scan of each project
				// top 5 only
				// skip projects without a name only for recent scans;
				if project.ProjectName != "" && len(recentScans) != 5 {
					recentScans = append(recentScans, ProjectRecentScan{
						ProjectID:    project.ProjectId,
						ProjectName:  project.ProjectName,
						Repository:   entry.Repository,
						Organisation: project.Organisation,
						Scan:         scanSummary,
					})
				}
			} else {
				totalPrRaised += prRaisedCountFromStats
			}
		}
	}

	sastVuln := &VulnerabilityCounts{
		AllCount:      &sastAllCount,
		CriticalCount: &sastCriticalCount,
		HighCount:     &sastHighCount,
		MediumCount:   &sastMediumCount,
		LowCount:      &sastLowCount,
		UnknownCount:  &sastUnknownCount,
	}
	scaUniqueCount := len(uniqueSCAVulns)
	scaVulns := &VulnerabilityCounts{
		AllCount:      &scaAllCount,
		UniqueCount:   &scaUniqueCount,
		CriticalCount: &scaCriticalCount,
		HighCount:     &scaHighCount,
		MediumCount:   &scaMediumCount,
		LowCount:      &scaLowCount,
		UnknownCount:  &scaUnknownCount,
	}

	return &HubStats{
		TotalProjects:       len(projects),
		TotalCompletedScans: totalCompletedScans,
		TotalPRRaised:       totalPrRaised,
		TotalScanning:       totalScanning,
		SASTVulnerabilities: sastVuln,
		SCAVulnerabilities:  scaVulns,
		ScanTimeFrequencies: scanTimeFrequencies,
		RecentScans:         recentScans,
	}, nil
}

func calculateHubVulnStats(vulns []*models.Vulnerability,
	uniqueSCAVulns map[string]bool,
	remediationsMap map[string][]models.Remediation,
) (*VulnerabilityCounts, *VulnerabilityCounts, map[string]bool, int) {
	var sastStats, scaStats VulnerabilityCounts
	var prRaised int
	var sastAllCounts, sastCriticalCount, sastHighCount, sastMediumCount, sastLowCount, sastUnknownCount int
	var scaAllCounts, scaCriticalCount, scaHighCount, scaMediumCount, scaLowCount, scaUnknownCount int
	for _, vuln := range vulns {
		if vuln.ScanType.String == models.ScanTypeSAST {
			sastAllCounts++
			switch vuln.Severity.String {
			case "critical":
				sastCriticalCount++
			case "high":
				sastHighCount++
			case "medium":
				sastMediumCount++
			case "low":
				sastLowCount++
			default:
				sastUnknownCount++
			}
		}
		if vuln.ScanType.String == models.ScanTypeSCA {
			scaAllCounts++
			uniqueSCAVulns[vuln.Name.String] = true
			switch vuln.Severity.String {
			case "critical":
				scaCriticalCount++
			case "high":
				scaHighCount++
			case "medium":
				scaMediumCount++
			case "low":
				scaLowCount++
			default:
				scaUnknownCount++
			}
		}
		if rems, rok := remediationsMap[vuln.ID.String()]; rok {
			for _, rem := range rems {
				if rem.Status != nil && *rem.Status == "PR_RAISED" {
					prRaised++
				}
			}
		}
	}

	sastStats = VulnerabilityCounts{
		AllCount:      &sastAllCounts,
		CriticalCount: &sastCriticalCount,
		HighCount:     &sastHighCount,
		MediumCount:   &sastMediumCount,
		LowCount:      &sastLowCount,
		UnknownCount:  &sastUnknownCount,
	}

	scaStats = VulnerabilityCounts{
		AllCount:      &scaAllCounts,
		CriticalCount: &scaCriticalCount,
		HighCount:     &scaHighCount,
		MediumCount:   &scaMediumCount,
		LowCount:      &scaLowCount,
		UnknownCount:  &scaUnknownCount,
	}
	return &sastStats, &scaStats, uniqueSCAVulns, prRaised
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

type HubRemediationResponse struct {
	ID            string           `json:"id"`
	Project       string           `json:"project"`
	Platform      string           `json:"platform"`
	Organization  *string          `json:"organization"`
	Repository    *string          `json:"repository"`
	Branch        *string          `json:"branch"`
	ScanID        string           `json:"scan_id"`
	Status        string           `json:"status"`
	PRLink        *string          `json:"pr_link"`
	Vulnerability HubVulnerability `json:"vulnerability"`
	CreatedAt     *time.Time       `json:"created_at"`
	UpdatedAt     *time.Time       `json:"updated_at"`
}

type HubVulnerability struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // sast / sca
	Severity string                 `json:"severity"`
	Details  map[string]interface{} `json:"details"`
}

type HubRemediationsResult struct {
	Remediations []HubRemediationResponse `json:"remediations"`
	TotalSize    int                      `json:"totalSize"`
}

// GetHubRemediations returns paginated remediations for a hub
func (s *HubService) GetHubRemediations(ctx context.Context, hubId string, page, pageSize int) (*HubRemediationsResult, error) {

	result, count, err := s.remediationRepo.GetRemediationsForHub(ctx, hubId, page, pageSize)
	if err != nil {
		s.logger.LogError(err, "failed to get remediations by hub", map[string]interface{}{"hubId": hubId})
		return nil, err
	}

	remediations := make([]HubRemediationResponse, 0, len(result))

	for _, rem := range result {
		remParsed := HubRemediationResponse{
			ID:           rem.RemediationID,
			Project:      rem.ProjectID,
			Platform:     "github",
			Organization: rem.Organisation,
			Repository:   rem.Repository,
			Branch:       rem.Branch,
			ScanID:       rem.ScanID,
			Status:       rem.Status,
			PRLink:       rem.PRLink,
			Vulnerability: HubVulnerability{
				ID:       rem.VulnerabilityID,
				Type:     rem.VulnerabilityType,
				Severity: rem.Severity,
			},
			CreatedAt: &rem.CreatedAt,
			UpdatedAt: &rem.UpdatedAt,
		}

		switch strings.ToLower(rem.VulnerabilityType) {
		case "sca":
			remParsed.Vulnerability.Details = map[string]interface{}{
				"package": rem.Package,
				"cve_id":  rem.VulnerabilityName,
			}
		case "sast":
			// TODO: Fix the count mismatch in case of data issues
			if rem.Package == nil {
				s.logger.LogWarning("skipped remediation id as there were no packages found", map[string]interface{}{"hubId": hubId, "remId": rem.RemediationID})
				continue
			}

			idx := strings.LastIndex(*rem.Package, ":")
			if idx == -1 {
				s.logger.LogWarning("skipped remediation id as path/line could not be derived", map[string]interface{}{"hubId": hubId, "remId": rem.RemediationID})
				continue
			}
			path, line := (*rem.Package)[:idx], (*rem.Package)[idx+1:]

			remParsed.Vulnerability.Details = map[string]interface{}{
				"rule_name": rem.VulnerabilityName,
				"message":   rem.Description,
				"file_path": path,
				"line_no":   line,
			}
		}

		remediations = append(remediations, remParsed)
	}

	out := &HubRemediationsResult{
		Remediations: remediations,
		TotalSize:    count,
	}

	return out, nil
}
