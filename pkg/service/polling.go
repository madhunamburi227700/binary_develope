package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/config"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
	"github.com/rs/zerolog/log"
)

// PollingService handles polling of scan statuses from SSD API
type PollingService struct {
	ssdClient           *client.SSDClient
	ssdService          *SSDService
	vulnService         *VulnService
	scanRepository      *repository.ScanRepository
	vulnRepository      *repository.VulnerabilityRepository
	notificationService *NotificationService
	pollingInterval     time.Duration
	stopChan            chan struct{}
	logger              *utils.ErrorLogger
}

// NewPollingService creates a new polling service
func NewPollingService(ssdClient *client.SSDClient, pollingInterval time.Duration) *PollingService {
	ssdService := NewSSDService()
	vulnService := NewVulnService()
	vulnService.ssdService = ssdService

	return &PollingService{
		ssdClient:           ssdClient,
		ssdService:          ssdService,
		vulnService:         vulnService,
		scanRepository:      repository.NewScanRepository(),
		vulnRepository:      repository.NewVulnerabilityRepository(),
		notificationService: NewNotificationService(NewEmailNotifier()),
		pollingInterval:     pollingInterval,
		stopChan:            make(chan struct{}),
		logger:              utils.NewErrorLogger("polling_service"),
	}
}

// Start begins the polling process
func (ps *PollingService) Start(ctx context.Context) {
	log.Info().Msgf("Starting polling service with interval: %v", ps.pollingInterval)

	ticker := time.NewTicker(ps.pollingInterval)
	defer ticker.Stop()

	// Run immediately on start
	ps.pollScans(ctx)

	for {
		select {
		case <-ticker.C:
			ps.pollScans(ctx)
		case <-ps.stopChan:
			log.Info().Msg("Polling service stopped")
			return
		case <-ctx.Done():
			log.Info().Msg("Polling service context cancelled")
			return
		}
	}
}

// Stop stops the polling service
func (ps *PollingService) Stop() {
	close(ps.stopChan)
}

type ProjectStatus struct {
	ProjectID string
	Status    string
	TeamID    string
	Scan      repository.ScanRecord
	Branch    string
}

// pollScans queries all pending/in-progress scans and updates their status
func (ps *PollingService) pollScans(ctx context.Context) {

	// Query all scans with status QUEUED or RUNNING using repository
	scans, err := ps.scanRepository.GetPendingScans(ctx)
	if err != nil {
		ps.logger.LogError(err, "Failed to get pending scans", nil)
		return
	}

	if len(scans) == 0 {
		return
	}

	log.Info().Msgf("Found %d pending scans to check", len(scans))

	// get project status in one go
	projectIds := make([]string, len(scans))
	scansMap := make(map[string]repository.ScanRecord)
	for i, scan := range scans {
		projectIds[i] = scan.ProjectID
		scansMap[scan.ProjectID] = scan
	}

	projectStatuses, err := ps.getScanStatusFromSSD(ctx, projectIds)
	if err != nil {
		ps.logger.LogError(err, "Failed to get project statuses", map[string]interface{}{
			"project_ids": projectIds,
		})
		return
	}

	if len(projectStatuses) == 0 {
		log.Error().Msgf("No project statuses found for project ids: %v", projectIds)
		// mark scans as failed if not found in project statuses
		scanIDs := make([]string, len(scansMap))
		for _, scan := range scansMap {
			scanIDs = append(scanIDs, scan.ID)
		}
		if err := ps.updateScanStatusInBulk(ctx, scanIDs, string(client.RiskStatusFail)); err != nil {
			ps.logger.LogError(err, "Failed to update scan status in bulk", map[string]interface{}{
				"scan_ids": scanIDs,
			})
		}
		return
	}

	// projectStatusesMap := make(map[string]ProjectStatus)
	for _, projectStatus := range projectStatuses {
		scan, ok := scansMap[projectStatus.ID]
		if !ok {
			log.Error().Msgf("Scan not found for project %s", projectStatus.ID)
			continue
		}

		if len(projectStatus.Scans) > 0 {
			scan.Branch = projectStatus.Scans[0].Branch
			if err := ps.processScan(ctx, scan, string(projectStatus.RiskStatus), projectStatus.Team.ID, projectStatus.Team.Name); err != nil {
				ps.logger.LogError(err, fmt.Sprintf("Failed to process scan %s", scan.ID), map[string]interface{}{
					"scan_id": scan.ID,
				})
			}
		} else {
			log.Error().Msgf("No branches/scan data found for project %s", projectStatus.ID)
			continue
		}
	}
}

// processScan processes a single scan by querying SSD API and updating the database
func (ps *PollingService) processScan(ctx context.Context, scan repository.ScanRecord, projectStatus, teamID, projectName string) error {
	// Handle scan status based on completion state
	switch projectStatus {
	case string(client.RiskStatusCompleted):
		// Get scan result data for completed scans
		scanData, err := ps.ssdService.GetScanResultData(ctx, &client.ScanResultDataRequest{
			ProjectID:  scan.ProjectID,
			Repository: scan.Repository,
			Branch:     scan.Branch,
		})
		if err != nil {
			ps.logger.LogError(err, "Failed to get scan result data", map[string]interface{}{
				"scan_id": scan.ID,
			})
			if err := ps.updateScanStatus(ctx, scan.ID, &client.ScanResultDataResponse{
				Status: string(client.RiskStatusFail),
			}); err != nil {
				return fmt.Errorf("failed to update scan in database: %w", err)
			}
			return nil
		}

		scanData.Status = string(client.RiskStatusCompleted)

		// Update scan status using repository
		if err := ps.updateScanStatus(ctx, scan.ID, scanData); err != nil {
			return fmt.Errorf("failed to update scan in database: %w", err)
		}

		// Insert vulnerabilities for completed scans using repository
		if err := ps.insertVulnerabilities(ctx, scan, scanData, teamID); err != nil {
			log.Error().Err(err).Msgf("Failed to insert vulnerabilities for scan %s", scan.ID)
			// Don't fail the entire process if vulnerability insertion fails
		}

		if config.GetNotificationEnabled() {
			// Derive the email ID from here
			email := utils.ExtractEmail(projectName)

			if err := ps.notificationService.NotifyScanCompletion(ctx, email, scan.ProjectID, scan.Repository, scan.Branch, scan.CommitSHA); err != nil {
				log.Error().Err(err).Msgf("failed to notify user for a completed scan %s", scan.ID)
			}
		}

	case string(client.RiskStatusFail):
		// Failed: Update scans and scan_types with failed status
		log.Info().Msgf("Scan %s failed, updating database with failed status", scan.ID)

		// Update scan status using repository
		if err := ps.updateScanStatus(ctx, scan.ID, &client.ScanResultDataResponse{
			Status: string(client.RiskStatusFail),
		}); err != nil {
			return fmt.Errorf("failed to update scan in database: %w", err)
		}

	case string(client.RiskStatusScanning), string(client.RiskStatusPending):
		// In Progress/Pending: Keep polling in next cycle
		log.Debug().Msgf("Scan %s still %s, will check again in next cycle", scan.ID, projectStatus)

	default:
		// Unknown status: Log and continue
		log.Warn().Msgf("Unknown status '%s' for scan %s, will check again in next cycle", projectStatus, scan.ID)
	}

	return nil
}

// update scan status in database
func (ps *PollingService) updateScanStatus(ctx context.Context, scanID string, scanData *client.ScanResultDataResponse) error {
	if err := ps.scanRepository.UpdateScanStatus(ctx, scanID, scanData); err != nil {
		return fmt.Errorf("failed to update scan in database: %w", err)
	}
	return nil
}

// update scan status in bulk
func (ps *PollingService) updateScanStatusInBulk(ctx context.Context, scanIDs []string, status string) error {
	if err := ps.scanRepository.UpdateScanStatusInBulk(ctx, scanIDs, status, time.Now()); err != nil {
		return fmt.Errorf("failed to update scan status in bulk: %w", err)
	}
	return nil
}

// getScanStatusFromSSD queries the SSD API for scan status
func (ps *PollingService) getScanStatusFromSSD(ctx context.Context, projectIds []string) ([]client.ProjectDetailsResponse, error) {
	projectStatuses, err := ps.ssdService.GetProjectStatuses(ctx, projectIds)
	if err != nil {
		return nil, fmt.Errorf("failed to get project statuses: %w", err)
	}
	return projectStatuses, nil
}

// insertVulnerabilities fetches and inserts vulnerability data for a completed scan
func (ps *PollingService) insertVulnerabilities(ctx context.Context, scan repository.ScanRecord, scanData *client.ScanResultDataResponse, teamID string) error {
	log.Info().Msgf("Fetching vulnerabilities for scan %s", scan.ID)

	// Fetch SAST vulnerabilities if SAST scan is available
	sastVulnData := &client.VulnerabilityDataResponse{}
	if scanData.ScannedFiledData.SAST.Semgrep.ScanName != "" {
		scanData, err := ps.vulnService.GetSastVulnerabilities(ctx, scan.ProjectID, teamID, scan.Repository, scan.Branch)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to fetch SAST vulnerabilities for scan %s", scan.ID)
			return fmt.Errorf("failed to fetch SAST vulnerabilities: %w", err)
		}

		if scanData == nil || len(scanData.Results) == 0 {
			return fmt.Errorf("no SAST vulnerabilities found for scan %s", scan.ID)
		}

		sastVulnData.Results = scanData.Results
	}

	// Fetch SCA vulnerabilities if SCA scans are available
	var scaVulnData *client.VulnerabilityListResponse
	// if scanData.ScannedFiledData.SCA.CodeLicense.ScanName != "" ||
	// 	scanData.ScannedFiledData.SCA.CodeSecret.ScanName != "" ||
	// 	scanData.ScannedFiledData.SCA.Sbom.ScanName != "" {
	if scanData.ScannedFiledData.SBOM.SBOM.ScanName != "" {
		// Fetch all SCA vulnerabilities with pagination
		scaVulnData = &client.VulnerabilityListResponse{VulnerabilityList: []client.VulnerabilityItem{}}
		pageNo := 0
		pageLimit := 99 // API requires less than 100

		for {
			pageData, err := ps.vulnService.GetSCAVulnerabilityList(ctx, scan.ProjectID, teamID, scan.Repository, scan.Branch, pageNo, pageLimit)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to fetch SCA vulnerabilities page %d for scan %s", pageNo, scan.ID)
				break
			}

			if pageData == nil || len(pageData.VulnerabilityList) == 0 {
				break
			}

			scaVulnData.VulnerabilityList = append(scaVulnData.VulnerabilityList, pageData.VulnerabilityList...)
			log.Debug().Msgf("Fetched page %d with %d SCA vulnerabilities for scan %s", pageNo, len(pageData.VulnerabilityList), scan.ID)

			// If we got fewer results than the page limit, we've reached the end
			if len(pageData.VulnerabilityList) < pageLimit {
				break
			}

			pageNo++
		}
	}

	// Insert SAST vulnerabilities into database using repository
	if len(sastVulnData.Results) > 0 {
		if err := ps.vulnRepository.InsertVulnerabilities(ctx, scan.ID, sastVulnData); err != nil {
			log.Error().Err(err).Msgf("Failed to insert SAST vulnerabilities for scan %s", scan.ID)
		}
		// update scan type count for sast
		if err := ps.updateVulnSAST(ctx, scan.ID, sastVulnData); err != nil {
			log.Error().Err(err).Msgf("Failed to update scan type counts for sast")
		}
	}

	// Insert SCA vulnerabilities into database using repository
	if scaVulnData != nil && len(scaVulnData.VulnerabilityList) > 0 {
		// Expand items with comma-separated components into individual entries
		scaVulnData = ps.expandScaVulnerabilitiesByComponent(scaVulnData)

		if err := ps.vulnRepository.InsertScaVulnerabilities(ctx, scan.ID, scaVulnData); err != nil {
			log.Error().Err(err).Msgf("Failed to insert SCA vulnerabilities for scan %s", scan.ID)
		}
		// update scan type count for sca
		if err := ps.updateVulnSCA(ctx, scan.ID, scaVulnData); err != nil {
			log.Error().Err(err).Msgf("Failed to update scan type counts for sca")
		}
	}

	return nil
}

// expandScaVulnerabilitiesByComponent expands SCA vulnerability items with comma-separated
// components into individual entries, one per component
func (ps *PollingService) expandScaVulnerabilitiesByComponent(scaData *client.VulnerabilityListResponse) *client.VulnerabilityListResponse {
	expandedVulnList := []client.VulnerabilityItem{}

	for _, item := range scaData.VulnerabilityList {
		// Collect all individual components (handle both array and comma-separated strings)
		var components []string
		for _, comp := range item.Component {
			// Split by comma in case any component string contains comma-separated values
			splitComps := strings.Split(comp, ",")
			for _, splitComp := range splitComps {
				trimmed := strings.TrimSpace(splitComp)
				if trimmed != "" {
					components = append(components, trimmed)
				}
			}
		}

		// If no components found, create one entry with empty component
		if len(components) == 0 {
			components = []string{""}
		}

		// Create a separate VulnerabilityItem for each component
		for _, component := range components {
			newItem := item
			// Update Component to contain only this single component
			newItem.Component = []string{component}
			expandedVulnList = append(expandedVulnList, newItem)
		}
	}

	// Return new response with expanded list
	return &client.VulnerabilityListResponse{
		VulnerabilityList: expandedVulnList,
		ScanID:            scaData.ScanID,
		Platform:          scaData.Platform,
		TotalSize:         len(expandedVulnList),
	}
}

// insertScanTypeEntryForSast inserts scan type entry for SAST vulnerabilities
func (ps *PollingService) updateVulnSAST(ctx context.Context, scanID string, vulnData *client.VulnerabilityDataResponse) error {

	findingsCount := 0
	criticalCount := 0
	highCount := 0
	mediumCount := 0
	lowCount := 0
	unknownCount := 0

	for _, result := range vulnData.Results {
		for _, finding := range result.Data {
			findingsCount++
			severity := strings.ToLower(finding.Severity)
			switch severity {
			case "critical":
				criticalCount++
			case "high":
				highCount++
			case "medium":
				mediumCount++
			case "low":
				lowCount++
			case "unknown":
				unknownCount++
			}
		}
	}

	if err := ps.scanRepository.UpdateScanTypeCountsForType(ctx, scanID, "sast", map[string]int{
		"findings_count": findingsCount,
		"critical_count": criticalCount,
		"high_count":     highCount,
		"medium_count":   mediumCount,
		"low_count":      lowCount,
		"unknown_count":  unknownCount,
	}); err != nil {
		return fmt.Errorf("failed to update scan type counts for sast: %w", err)
	}

	return nil
}

// insertScanTypeEntryForSca inserts scan type entry for SCA vulnerabilities
func (ps *PollingService) updateVulnSCA(ctx context.Context, scanID string, scaData *client.VulnerabilityListResponse) error {

	findingsCount := 0
	criticalCount := 0
	highCount := 0
	mediumCount := 0
	lowCount := 0
	unknownCount := 0

	for _, item := range scaData.VulnerabilityList {
		findingsCount++
		severity := strings.ToLower(item.Severity)
		switch severity {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		case "medium":
			mediumCount++
		case "low":
			lowCount++
		case "unknown":
			unknownCount++
		}
	}

	if err := ps.scanRepository.UpdateScanTypeCountsForType(ctx, scanID, "sca", map[string]int{
		"findings_count": findingsCount,
		"critical_count": criticalCount,
		"high_count":     highCount,
		"medium_count":   mediumCount,
		"low_count":      lowCount,
		"unknown_count":  unknownCount,
	}); err != nil {
		return fmt.Errorf("failed to update scan type counts for sca: %w", err)
	}

	return nil
}
