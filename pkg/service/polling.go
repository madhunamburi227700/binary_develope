package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/config"
	"github.com/opsmx/ai-guardian-api/pkg/models"
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
	projectRepository   *repository.ProjectRepository
	userRepository      *repository.UserRepository
	notificationService *NotificationService
	pollingInterval     time.Duration
	pendingScanStopChan chan struct{}
	logger              *utils.ErrorLogger
	webhookScanPairRepo *repository.WebhookScanPairRepository
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
		projectRepository:   repository.NewProjectRepository(),
		userRepository:      repository.NewUserRepository(),
		notificationService: NewNotificationService(NewEmailNotifier()),
		pollingInterval:     pollingInterval,
		pendingScanStopChan: make(chan struct{}),
		logger:              utils.NewErrorLogger("polling_service"),
		webhookScanPairRepo: repository.NewWebhookScanPairRepository(),
	}
}

// Start begins the polling process with two separate pollers
func (ps *PollingService) Start(ctx context.Context) {
	// Start scheduled scan poller if enabled
	if config.GetScheduledScanPollingEnabled() {
		log.Info().Msgf("Starting scheduled scan poller with interval: %v", ps.pollingInterval)
		sleepInterval := config.GetScheduledScanPollingIntervalSeconds()
		go ps.startScheduledScanPoller(ctx, sleepInterval)
	} else {
		log.Info().Msg("Scheduled scan polling is disabled")
	}

	// Start pending scan poller
	log.Info().Msgf("Starting pending scan poller with interval: %v", ps.pollingInterval)
	ps.startPendingScanPoller(ctx, ps.pollingInterval)
}

// Stop stops the polling service
func (ps *PollingService) Stop() {
	close(ps.pendingScanStopChan)
}

type ProjectStatus struct {
	ProjectID string
	Status    string
	TeamID    string
	Scan      repository.ScanRecord
	Branch    string
}

// startScheduledScanPoller runs the scheduled scan poller
func (ps *PollingService) startScheduledScanPoller(ctx context.Context, sleepInterval int) {
	interval := time.Duration(sleepInterval) * time.Second
	for {
		// Run the polling function and wait for it to complete
		ps.pollScheduledScanProjects(ctx)
		time.Sleep(interval)
	}
}

// startPendingScanPoller runs the pending scan poller with the given interval
func (ps *PollingService) startPendingScanPoller(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on start
	ps.pollPendingScans(ctx)

	for {
		select {
		case <-ticker.C:
			ps.pollPendingScans(ctx)
		case <-ps.pendingScanStopChan:
			log.Info().Msg("Pending scan poller stopped")
			return
		case <-ctx.Done():
			log.Info().Msg("Pending scan poller context cancelled")
			return
		}
	}
}

// pollPendingScans polls for pending scan status updates
func (ps *PollingService) pollPendingScans(ctx context.Context) {
	// Query all scans with status QUEUED or RUNNING
	scans, err := ps.scanRepository.GetPendingScans(ctx)
	if err != nil {
		ps.logger.LogError(err, "Failed to get pending scans", nil)
		return
	}

	log.Debug().Msgf("Found %d pending scans to check", len(scans))

	if len(scans) == 0 {
		return
	}

	// Build project IDs list and scans map for efficient lookup
	projectIds := make([]string, len(scans))
	scansMap := make(map[string]repository.ScanRecord, len(scans))
	for i, scan := range scans {
		projectIds[i] = scan.ProjectID
		scansMap[scan.ProjectID] = scan
	}

	// Fetch project statuses from SSD in bulk
	projectStatuses, err := ps.getScanStatusFromSSD(ctx, projectIds)
	if err != nil {
		ps.logger.LogError(err, "Failed to get project statuses", map[string]interface{}{
			"project_ids": projectIds,
		})
		return
	}

	// If no statuses returned, mark all scans as failed
	if len(projectStatuses) == 0 {
		log.Error().Msgf("No project statuses found for project ids: %v", projectIds)
		scanIDs := make([]string, 0, len(scansMap))
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

	// Process each project status
	for _, projectStatus := range projectStatuses {
		scan, ok := scansMap[projectStatus.ID]
		if !ok {
			log.Error().Msgf("Scan not found for project %s", projectStatus.ID)
			continue
		}

		if len(projectStatus.Scans) == 0 {
			log.Error().Msgf("No branches/scan data found for project %s", projectStatus.ID)
			continue
		}

		scan.Branch = projectStatus.Scans[0].Branch

		// Extract lastScannedTime from SSD
		var lastScannedTime *time.Time
		if projectStatus.Scans[0].LastScannedTime != nil {
			lastScannedTime = projectStatus.Scans[0].LastScannedTime
		}

		if err := ps.processScan(ctx, scan, string(projectStatus.RiskStatus), projectStatus.Team.ID, projectStatus.Team.Email, lastScannedTime, scan.LastScannedTime); err != nil {
			ps.logger.LogError(err, fmt.Sprintf("Failed to process scan %s", scan.ID), map[string]interface{}{
				"scan_id": scan.ID,
			})
		}
	}
}

// processScan processes a single scan by querying SSD API and updating the database
func (ps *PollingService) processScan(ctx context.Context, scan repository.ScanRecord, projectStatus, teamID, email string, lastScannedTime *time.Time, dbLastScannedTime *time.Time) error {
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

			// If scan fails after completion check, delete webhook pair
			if err := ps.webhookScanPairRepo.DeleteProjectPairByScanID(ctx, scan.ID); err != nil {
				ps.logger.LogError(err, "Failed to delete webhook pair after scan failure", map[string]interface{}{
					"scan_id": scan.ID,
				})
				// Don't fail the entire process
			}

			return nil
		}

		scanData.Status = string(client.RiskStatusCompleted)

		// Update scan status using repository
		if err := ps.updateScanStatus(ctx, scan.ID, scanData); err != nil {
			return fmt.Errorf("failed to update scan in database: %w", err)
		}

		// Update last_scanned_time in projects table if it differs from PostgreSQL
		if lastScannedTime != nil {
			// Compare timestamps with microsecond precision
			ssdTimeUTC := lastScannedTime.UTC().Truncate(time.Microsecond)
			var shouldUpdate bool
			if dbLastScannedTime == nil {
				shouldUpdate = true
			} else {
				dbTimeUTC := dbLastScannedTime.UTC().Truncate(time.Microsecond)
				shouldUpdate = !ssdTimeUTC.Equal(dbTimeUTC)
			}

			if shouldUpdate {
				if err := ps.projectRepository.UpdateProject(ctx, scan.ProjectID, map[string]interface{}{
					"last_scanned_time": lastScannedTime,
				}); err != nil {
					ps.logger.LogError(err, "Failed to update last_scanned_time for completed scan", map[string]interface{}{
						"scan_id":    scan.ID,
						"project_id": scan.ProjectID,
					})
				}
			}
		}

		// Insert vulnerabilities for completed scans using repository
		if err := ps.insertVulnerabilities(ctx, scan, scanData, teamID); err != nil {
			log.Error().Err(err).Msgf("Failed to insert vulnerabilities for scan %s", scan.ID)
			// Don't fail the entire process if vulnerability insertion fails
		}

		// Check if this scan is part of a webhook pair
		isPairComplete, pairData, err := ps.webhookScanPairRepo.CheckAndUpdateProjectCompletion(ctx, scan.ID)
		if err != nil {
			ps.logger.LogError(err, "Failed to check project pair in Redis", map[string]interface{}{
				"scan_id": scan.ID,
			})
			// Continue with notification even if Redis check fails
		}

		// If both projects are completed, process vulnerability diff immediately
		if isPairComplete && pairData != nil {
			ps.logger.LogInfo("Both projects completed, processing vulnerability diff", map[string]interface{}{
				"pr_number":       pairData.PRNumber,
				"base_project_id": pairData.BaseProjectID,
				"head_project_id": pairData.HeadProjectID,
			})

			// Process vulnerability diff in background to not block notification
			go func() {
				if err := ps.processVulnerabilityDiff(context.Background(), pairData); err != nil {
					ps.logger.LogError(err, "Failed to process vulnerability diff", map[string]interface{}{
						"pr_number": pairData.PRNumber,
					})
				} else {
					// Mark diff as processed
					if err := ps.webhookScanPairRepo.MarkDiffProcessed(context.Background(), pairData.PRNumber); err != nil {
						ps.logger.LogError(err, "Failed to mark diff as processed", map[string]interface{}{
							"pr_number": pairData.PRNumber,
						})
					}
				}
			}()
		}

		// Send notification if enabled
		if config.GetNotificationEnabled() {
			if email != "" {
				dbUser, err := ps.userRepository.GetByProviderUserID(ctx, email)
				if err == nil {
					email = dbUser.Email.String
					project, commitSHA := scan.ProjectID, ""
					if scanData != nil {
						project = scanData.ProjectName
						commitSHA = scanData.HeadCommit
					}
					if err := ps.notificationService.NotifyScanCompletion(ctx, email, project, scan.Repository, scan.Branch, commitSHA); err != nil {
						ps.logger.LogError(err, "Failed to notify the user for a completed scan", map[string]interface{}{
							"scan": scan.ID, "teamID": teamID,
						})
					}

				} else {
					ps.logger.LogError(err, "Failed to notify the user for a completed scan", map[string]interface{}{
						"scan": scan.ID, "username": email, "teamID": teamID,
					})
				}
			} else {
				ps.logger.LogError(errors.New("Failed to notify the user for a completed scan"), "email has not been assigned or is blank for the team/hub", map[string]interface{}{
					"scan": scan.ID, "teamID": teamID,
				})
			}
		}

	case string(client.RiskStatusFail):
		// Failed: Update scans and scan_types with failed status
		log.Debug().Msgf("Scan %s failed, updating database with failed status", scan.ID)

		// Update scan status using repository
		if err := ps.updateScanStatus(ctx, scan.ID, &client.ScanResultDataResponse{
			Status: string(client.RiskStatusFail),
		}); err != nil {
			return fmt.Errorf("failed to update scan in database: %w", err)
		}

		// Update last_scanned_time in projects table if it differs from PostgreSQL
		if lastScannedTime != nil {
			// Compare timestamps with microsecond precision
			ssdTimeUTC := lastScannedTime.UTC().Truncate(time.Microsecond)
			var shouldUpdate bool
			if dbLastScannedTime == nil {
				shouldUpdate = true
			} else {
				dbTimeUTC := dbLastScannedTime.UTC().Truncate(time.Microsecond)
				shouldUpdate = !ssdTimeUTC.Equal(dbTimeUTC)
			}

			if shouldUpdate {
				if err := ps.projectRepository.UpdateProject(ctx, scan.ProjectID, map[string]interface{}{
					"last_scanned_time": lastScannedTime,
				}); err != nil {
					ps.logger.LogError(err, "Failed to update last_scanned_time for failed scan", map[string]interface{}{
						"scan_id":    scan.ID,
						"project_id": scan.ProjectID,
					})
				}
			}
		}

		// Delete webhook pair if this scan is part of one (since one scan failed, diff can't be calculated)
		if err := ps.webhookScanPairRepo.DeleteProjectPairByScanID(ctx, scan.ID); err != nil {
			ps.logger.LogError(err, "Failed to delete webhook pair after scan failure", map[string]interface{}{
				"scan_id": scan.ID,
			})
			// Don't fail the entire process if deletion fails
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

	log.Debug().Msgf("Fetching vulnerabilities for scan %s", scan.ID)

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

// processVulnerabilityDiff calculates and processes the diff between base and head projects
func (ps *PollingService) processVulnerabilityDiff(ctx context.Context, pairData *repository.ProjectPairData) error {
	// Get the latest completed scan IDs for both projects
	baseScanID, err := ps.scanRepository.GetLatestScanByProjectAndBranch(ctx, pairData.BaseProjectID, pairData.BaseBranch)
	if err != nil {
		return fmt.Errorf("failed to get base scan ID: %w", err)
	}

	headScanID, err := ps.scanRepository.GetLatestScanByProjectAndBranch(ctx, pairData.HeadProjectID, pairData.HeadBranch)
	if err != nil {
		return fmt.Errorf("failed to get head scan ID: %w", err)
	}

	// Use existing VulnService to get vulnerability diff
	// scanID1 = base (older), scanID2 = head (newer)
	// This returns vulnerabilities in head that are not in base (new vulnerabilities)
	diffResponse, err := ps.vulnService.GetVulnerabilityDiff(ctx, baseScanID, headScanID)
	if err != nil {
		return fmt.Errorf("failed to get vulnerability diff: %w", err)
	}

	// Log the diff results
	ps.logger.LogInfo("Vulnerability diff calculated successfully", map[string]interface{}{
		"pr_number":       pairData.PRNumber,
		"base_project_id": pairData.BaseProjectID,
		"head_project_id": pairData.HeadProjectID,
		"base_scan_id":    baseScanID,
		"head_scan_id":    headScanID,
		"base_branch":     pairData.BaseBranch,
		"head_branch":     pairData.HeadBranch,
		"new_sast_count":  len(diffResponse.SAST),
		"new_sca_count":   len(diffResponse.SCA),
		"total_new_vulns": len(diffResponse.SAST) + len(diffResponse.SCA),
	})

	// Post PR comment with diff results
	if err := ps.postPRCommentWithDiff(ctx, pairData, diffResponse); err != nil {
		ps.logger.LogError(err, "Failed to post PR comment", map[string]interface{}{
			"pr_number": pairData.PRNumber,
		})
	}
	return nil
}

// postPRCommentWithDiff posts a formatted comment to the PR with vulnerability diff results
func (ps *PollingService) postPRCommentWithDiff(ctx context.Context, pairData *repository.ProjectPairData, diffResponse *VulnerabilityDiffResponse) error {
	// Extract owner and repo from repo URL
	owner, repo, err := utils.FilterOwnerAndRepoNameFromRepoURL(pairData.RepoURL)
	if err != nil {
		return fmt.Errorf("failed to parse repo URL: %w", err)
	}

	// Get GitHub token for the project
	githubToken, err := ps.ssdService.getIntegratorToken(ctx, pairData.HeadProjectID)
	if err != nil {
		return fmt.Errorf("failed to get GitHub token: %w", err)
	}

	// Format the comment message
	comment := ps.formatDiffComment(pairData, diffResponse)

	// Post comment to GitHub using client
	githubClient := client.NewGitHubClient()
	_, err = githubClient.PostPRComment(ctx, githubToken, owner, repo, pairData.PRNumber, comment)
	if err != nil {
		// Log detailed error information for debugging
		ps.logger.LogError(err, "Failed to post PR comment", map[string]interface{}{
			"owner":     owner,
			"repo":      repo,
			"pr_number": pairData.PRNumber,
			"repo_url":  pairData.RepoURL,
			"token_set": githubToken != "",
		})
		return fmt.Errorf("failed to post PR comment: %w", err)
	}

	ps.logger.LogInfo("PR comment posted successfully", map[string]interface{}{
		"owner":     owner,
		"repo":      repo,
		"pr_number": pairData.PRNumber,
	})

	return nil
}

// formatDiffComment formats the vulnerability diff into a GitHub PR comment
func (ps *PollingService) formatDiffComment(pairData *repository.ProjectPairData, diffResponse *VulnerabilityDiffResponse) string {
	var comment strings.Builder

	// Header
	comment.WriteString("## Vulnerability Scan Results\n\n")
	comment.WriteString("Scan comparison completed for this pull request.\n\n")

	// Summary
	totalNewVulns := len(diffResponse.SAST) + len(diffResponse.SCA)
	if totalNewVulns == 0 {
		comment.WriteString("**No new vulnerabilities detected.**\n\n")
		comment.WriteString("The scan shows no new vulnerabilities compared to the base branch.\n\n")
	} else {
		comment.WriteString(fmt.Sprintf("**Found %d new vulnerability/vulnerabilities**\n\n", totalNewVulns))
		comment.WriteString(fmt.Sprintf("- SAST: %d new finding(s)\n", len(diffResponse.SAST)))
		comment.WriteString(fmt.Sprintf("- SCA: %d new finding(s)\n\n", len(diffResponse.SCA)))
	}

	// SAST Vulnerabilities
	if len(diffResponse.SAST) > 0 {
		comment.WriteString("### SAST Vulnerabilities\n\n")
		comment.WriteString("| Rule | Location | Severity |\n")
		comment.WriteString("|------|----------|----------|\n")

		for _, vuln := range diffResponse.SAST {
			severity := "Unknown"
			if vuln.Severity != "" {
				severity = vuln.Severity
			}
			// Normalize location to show only filename:line (remove temp directory prefix)
			location := utils.NormalizeSASTFilePath(vuln.Package)
			if location == "" {
				location = "N/A"
			}
			// Add leading slash for display: "main.go:179" -> "/main.go:179"
			if location != "N/A" && !strings.HasPrefix(location, "/") {
				location = "/" + location
			}
			comment.WriteString(fmt.Sprintf("| `%s` | `%s` | %s |\n",
				escapeMarkdown(vuln.Name),
				escapeMarkdown(location),
				severity))
		}
		comment.WriteString("\n")
	}

	// SCA Vulnerabilities
	if len(diffResponse.SCA) > 0 {
		comment.WriteString("### SCA Vulnerabilities\n\n")
		comment.WriteString("| CVE/Name | Package | Severity |\n")
		comment.WriteString("|----------|---------|----------|\n")

		for _, vuln := range diffResponse.SCA {
			severity := "Unknown"
			if vuln.Severity != "" {
				severity = vuln.Severity
			}
			packageName := vuln.Package
			if packageName == "" {
				packageName = "N/A"
			}
			comment.WriteString(fmt.Sprintf("| `%s` | `%s` | %s |\n",
				escapeMarkdown(vuln.Name),
				escapeMarkdown(packageName),
				severity))
		}
		comment.WriteString("\n")
	}

	// Extract owner and repo from RepoURL for the project URL
	owner, repo, err := utils.FilterOwnerAndRepoNameFromRepoURL(pairData.RepoURL)
	if err == nil {
		// Build project URL with HEAD_BRANCH
		projectURL := fmt.Sprintf("%s/projects/%s/%s/%s?branch=%s",
			config.GetUIAddress(),
			pairData.HeadProjectID,
			owner,
			repo,
			pairData.HeadBranch)

		// Footer with links
		comment.WriteString("---\n\n")
		comment.WriteString(fmt.Sprintf("[View project scan](%s)\n\n", projectURL))
	} else {
		// Fallback if URL parsing fails
		comment.WriteString("---\n\n")
	}

	comment.WriteString("_This comment was automatically generated by AI Guardian_\n")

	return comment.String()
}

// escapeMarkdown escapes special markdown characters
func escapeMarkdown(text string) string {
	// Replace backticks with code spans to prevent markdown issues
	text = strings.ReplaceAll(text, "`", "\\`")
	text = strings.ReplaceAll(text, "|", "\\|")
	text = strings.ReplaceAll(text, "\n", " ")
	return text
}

// pollScheduledScanProjects syncs all projects with SSD
// 1. Fetches all projects from SSD and PostgreSQL
// 2. Gets pending scan info for PostgreSQL projects
// 3. Compares and updates lastScannedTime and scheduleTime if they differ
// 4. Creates scans if lastScannedTime differs and no pending scan exists
func (ps *PollingService) pollScheduledScanProjects(ctx context.Context) {
	// Step 1: Get all projects from PostgreSQL
	dbProjects, err := ps.projectRepository.GetAll(ctx)
	if err != nil {
		ps.logger.LogError(err, "Failed to get projects from database", nil)
		return
	}

	if len(dbProjects) == 0 {
		return
	}

	dbProjectIDs := make([]string, 0, len(dbProjects))
	for _, project := range dbProjects {
		dbProjectIDs = append(dbProjectIDs, project.ID)
	}

	// Step 2: Get all projects from SSD
	ssdProjects, err := ps.ssdService.GetAllProjectStatuses(ctx)
	if err != nil {
		ps.logger.LogError(err, "Failed to get all project statuses from SSD", nil)
		return
	}

	if len(ssdProjects) == 0 {
		log.Warn().Msg("No projects found in SSD")
		return
	}

	// Create map for quick lookup
	ssdProjectsMap := make(map[string]*client.ProjectRef)
	for i := range ssdProjects {
		ssdProjectsMap[ssdProjects[i].ID] = &ssdProjects[i]
	}

	// Step 4: Process projects that exist in PostgreSQL
	scansCreated := 0
	projectsUpdated := 0

	for _, dbProject := range dbProjects {
		ssdProject, existsInSSD := ssdProjectsMap[dbProject.ID]
		if !existsInSSD {
			log.Debug().Msgf("Project %s exists in PostgreSQL but not in SSD, skipping", dbProject.ID)
			continue
		}

		// Collect updates for this project
		updates := make(map[string]interface{})

		// Check and collect scheduled_time update
		if len(ssdProject.ProjectConfig) > 0 && ssdProject.ProjectConfig[0].ScheduleTime != nil {
			ssdScheduledTime := *ssdProject.ProjectConfig[0].ScheduleTime
			if dbProject.ScheduledTime == nil || *dbProject.ScheduledTime != ssdScheduledTime {
				updates["scheduled_time"] = ssdScheduledTime
			}
		}

		// Get repository and branch info
		var ssdLastScannedTime *time.Time
		var repository, branch string

		if len(ssdProject.Scans) > 0 {
			if ssdProject.Scans[0].LastScannedTime != nil {
				ssdLastScannedTime = ssdProject.Scans[0].LastScannedTime
			}
			branch = ssdProject.Scans[0].Branch
		}

		if len(ssdProject.ProjectConfig) > 0 {
			repository = ssdProject.ProjectConfig[0].Repository
		}

		// Check if lastScannedTime differs
		if ssdLastScannedTime != nil {
			ssdTimeUTC := ssdLastScannedTime.UTC().Truncate(time.Microsecond)

			var lastScannedTimeDiffers bool
			if dbProject.LastScannedTime == nil {
				lastScannedTimeDiffers = false
			} else {
				dbTimeUTC := dbProject.LastScannedTime.UTC().Truncate(time.Microsecond)
				lastScannedTimeDiffers = !ssdTimeUTC.Equal(dbTimeUTC)
			}

			if lastScannedTimeDiffers {
				updates["last_scanned_time"] = *ssdLastScannedTime
				if repository == "" {
					log.Warn().Msgf("No repository found for project %s, skipping scan creation", dbProject.ID)
				} else {
					scan := &models.Scan{
						ProjectID:  dbProject.ID,
						Repository: repository,
						Branch:     branch,
						Status:     "pending",
						HubID:      dbProject.HubID,
					}

					if err := ps.scanRepository.Create(ctx, scan); err != nil {
						ps.logger.LogError(err, "Failed to create scan entry", map[string]interface{}{
							"project_id": dbProject.ID,
						})
						delete(updates, "last_scanned_time")
					} else {
						scansCreated++
					}
				}
			}
		}

		// Execute update if there are any changes
		if len(updates) > 0 {
			if err := ps.projectRepository.UpdateProject(ctx, dbProject.ID, updates); err != nil {
				ps.logger.LogError(err, "Failed to update project", map[string]interface{}{
					"project_id": dbProject.ID,
				})
			} else {
				projectsUpdated++
			}
		}
	}

	log.Debug().Msgf("Scheduled scan polling completed: %d scans created, %d projects updated",
		scansCreated, projectsUpdated)
}
