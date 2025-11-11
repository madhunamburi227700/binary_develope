package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// ProjectService handles project business logic
type ProjectService struct {
	ssdService  *SSDService
	projectRepo *repository.ProjectRepository
	scanRepo    *repository.ScanRepository
	logger      *utils.ErrorLogger
}

// NewProjectService creates a new project service
func NewProjectService() *ProjectService {
	return &ProjectService{
		projectRepo: repository.NewProjectRepository(),
		scanRepo:    repository.NewScanRepository(),
		logger:      utils.NewErrorLogger("project_service"),
	}
}

// CreateProjectRequest represents the request to create a project
type CreateProjectRequest struct {
	HubID         string `json:"hub_id" validate:"required"`
	IntegrationID string `json:"integration_id"`
	Name          string `json:"name" validate:"required,min=1,max=255"`
	RepoName      string `json:"repoName" validate:"required,min=1,max=255"`
	Branch        string `json:"branch"`
}

// UpdateProjectRequest represents the request to update a project
type UpdateProjectRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=255"`
	RepoURL     *string `json:"repo_url" validate:"omitempty,url"`
	Description *string `json:"description" validate:"omitempty,max=1000"`
}

// ProjectListRequest represents the request to list projects
type ProjectListRequest struct {
	HubID         *uuid.UUID `json:"hub_id"`
	IntegrationID *uuid.UUID `json:"integration_id"`
	Search        string     `json:"search"`
	Page          int        `json:"page" validate:"min=1"`
	PageSize      int        `json:"page_size" validate:"min=1,max=100"`
	OrderBy       string     `json:"order_by"`
	OrderDir      string     `json:"order_dir" validate:"omitempty,oneof=ASC DESC"`
}

type ProjectStats struct {
	ScanTimeFrequencies []*ScanTimeFrequency `json:"scans_time_frequencies"`
	SCAVulnerabilities  *VulnerabilityCounts `json:"sca_vulnerabilities"`
	SASTVulnerabilities *VulnerabilityCounts `json:"sast_vulnerabilities"`
	RecentScans         []*RecentScan        `json:"recent_scans"`
}

type VulnerabilityCounts struct {
	AllCount      *int `json:"all_count,omitempty"`
	UniqueCount   *int `json:"unique_count,omitempty"`
	CriticalCount *int `json:"critical_count"`
	HighCount     *int `json:"high_count"`
	MediumCount   *int `json:"medium_count"`
	LowCount      *int `json:"low_count"`
	UnknownCount  *int `json:"unknown_count"`
}

type ScanTimeFrequency struct {
	Date  *time.Time `json:"date"`
	Count *int       `json:"count"`
}

type RecentScan struct {
	Branch             string     `json:"branch"`
	CommitId           string     `json:"commit_id"`
	ScanTime           *time.Time `json:"scan_time"`
	IssueCriticalCount *int       `json:"issue_critical_count"`
	IssueHighCount     *int       `json:"issue_high_count"`
	IssueMediumCount   *int       `json:"issue_medium_count"`
	IssueLowCount      *int       `json:"issue_low_count"`
	IssueUnknownCount  *int       `json:"issue_unknown_count"`
}

// CreateProject creates a new project
func (s *ProjectService) CreateProject(ctx context.Context, req *CreateProjectRequest) (string, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}

	// same project name check would be auto applied via ssd
	// current they do not have any api to check project via name

	// getting username from ssd based on account id
	username, err := s.ssdService.GetGithubUsername(ctx, req.IntegrationID)
	if err != nil {
		err := s.ssdService.IntegratorHandler(ctx, err.Error(), req.IntegrationID, "", req.HubID)
		if err != nil {
			return "", err
		}
		return "", err
	}

	branch := "onlyMain"
	if strings.TrimSpace(req.Branch) != "" {
		branch = req.Branch
	}

	// Create project
	scheduleTime := 0
	scanUpto := 0
	project := &client.ProjectRef{
		Name:         req.Name,
		AccountID:    req.IntegrationID,
		Organisation: username,
		TeamID:       req.HubID,

		// TODO: automate below fields
		ProjectConfig: []client.ProjectConfigRef{{
			Repository:   req.RepoName,
			Branch:       []string{branch},
			ScheduleTime: &scheduleTime,
			ScanUpto:     &scanUpto,
		}},
		Type:      "user", // since we authenticating using github app token type can be any user/organisation
		ScanType:  "sourceScan",
		Platform:  "github",
		ScanLevel: "repoLevel",
	}

	projectId, err := s.ssdService.CreateProject(ctx, req.HubID, project)
	if err != nil {
		s.logger.LogError(err, "Failed to create project", map[string]interface{}{
			"request": project,
		})
		return "", fmt.Errorf("failed to create project: %w", err)
	}

	// Create a scan record for the project
	scan := &models.Scan{
		ProjectID:     projectId,
		Repository:    req.RepoName,
		Branch:        branch,
		CommitSHA:     "",
		PullRequestID: "",
		Tag:           "",
		Settings:      nil,
		HubID:         req.HubID,
		Status:        string(client.RiskStatusPending),
	}

	if err := s.scanRepo.Create(ctx, scan); err != nil {
		s.logger.LogError(err, "Failed to create scan record", map[string]interface{}{
			"project_id": projectId,
		})
		// Don't fail the entire operation if scan creation fails
	}

	s.logger.LogInfo("Project created successfully", map[string]interface{}{
		"project_id": projectId,
	})

	return projectId, nil
}

// GetProject retrieves a project by ID
func (s *ProjectService) GetProject(ctx context.Context, projectId string) (*client.ProjectRef, error) {
	project, err := s.ssdService.GetProjectDetails(ctx, projectId)
	if err != nil {
		s.logger.LogError(err, "Failed to get project", map[string]interface{}{
			"projectId": projectId,
		})
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// logic
	return project, nil
}

// UpdateProject updates a project
func (s *ProjectService) UpdateProject(ctx context.Context, id uuid.UUID, req *UpdateProjectRequest) (*models.Project, error) {
	// Validate request
	if err := s.validateUpdateRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if project exists
	exists, err := s.projectRepo.Exists(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check project existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("project not found")
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.RepoURL != nil {
		updates["repo_url"] = *req.RepoURL
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	// Update project
	err = s.projectRepo.Update(ctx, id, updates)
	if err != nil {
		s.logger.LogError(err, "Failed to update project", map[string]interface{}{
			"project_id": id,
			"updates":    updates,
		})
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	// Get updated project
	project, err := s.projectRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated project: %w", err)
	}

	s.logger.LogInfo("Project updated successfully", map[string]interface{}{
		"project_id": id,
		"updates":    updates,
	})

	return project, nil
}

// DeleteProject deletes a project
func (s *ProjectService) DeleteProject(ctx context.Context, teamIds, projectId string) error {
	// Delete project
	_, err := s.ssdService.DeleteProject(ctx, teamIds, projectId)
	if err != nil {
		s.logger.LogError(err, "Failed to delete project", map[string]interface{}{
			"project_id": projectId,
		})
		return fmt.Errorf("failed to delete project: %w", err)
	}

	s.logger.LogInfo("Project deleted successfully", map[string]interface{}{
		"project_id": projectId,
	})

	return nil
}

// ListProjects retrieves projects with pagination and filtering
func (s *ProjectService) ListProjects(ctx context.Context, req *ProjectListRequest) (*repository.QueryResult[models.Project], error) {
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	if req.OrderBy == "" {
		req.OrderBy = "created_at"
	}
	if req.OrderDir == "" {
		req.OrderDir = "DESC"
	}

	// Build query options
	options := &repository.QueryOptions{
		Limit:    req.PageSize,
		Offset:   (req.Page - 1) * req.PageSize,
		OrderBy:  req.OrderBy,
		OrderDir: req.OrderDir,
		Filters:  make(map[string]interface{}),
	}

	// Add filters
	if req.HubID != nil {
		options.Filters["hub_id"] = *req.HubID
	}
	if req.IntegrationID != nil {
		options.Filters["integration_id"] = *req.IntegrationID
	}

	// Execute query
	var result *repository.QueryResult[models.Project]
	var err error

	if req.Search != "" {
		result, err = s.projectRepo.SearchByName(ctx, req.Search, options)
	} else {
		result, err = s.projectRepo.List(ctx, options)
	}

	if err != nil {
		s.logger.LogError(err, "Failed to list projects", map[string]interface{}{
			"request": req,
		})
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return result, nil
}

// GetProjectStats retrieves a project stats by ID
func (s *ProjectService) GetProjectStats(ctx context.Context, projectId, db string) (*ProjectStats, error) {
	// send stats from postgres
	if db == "new" {
		return s.getProjectStats(ctx, projectId)
	}
	projectRef, err := s.ssdService.GetProjectDetailsCustom(ctx, projectId)
	if err != nil {
		s.logger.LogError(err, "Failed to get custom project details", map[string]interface{}{
			"projectId": projectId,
		})
		return nil, fmt.Errorf("failed to get project details: %w", err)
	}

	return s.calculateProjectStats(ctx, projectRef)
}

// GetProjectsByHub retrieves projects by hub ID
func (s *ProjectService) GetProjectsByHub(ctx context.Context, hubID uuid.UUID, req *ProjectListRequest) (*repository.QueryResult[models.Project], error) {
	req.HubID = &hubID
	return s.ListProjects(ctx, req)
}

// GetProjectsByIntegration retrieves projects by integration ID
func (s *ProjectService) GetProjectsByIntegration(ctx context.Context, integrationID uuid.UUID, req *ProjectListRequest) (*repository.QueryResult[models.Project], error) {
	req.IntegrationID = &integrationID
	return s.ListProjects(ctx, req)
}

// validateCreateRequest validates create project request
func (s *ProjectService) validateCreateRequest(req *CreateProjectRequest) error {
	if req.HubID == "" {
		return fmt.Errorf("hub_id is required")
	}
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.RepoName == "" {
		return fmt.Errorf("repoName is required user/organisation")
	}
	return nil
}

// validateUpdateRequest validates update project request
func (s *ProjectService) validateUpdateRequest(req *UpdateProjectRequest) error {
	if req.Name != nil && *req.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if req.Name != nil && len(*req.Name) > 255 {
		return fmt.Errorf("name must be less than 255 characters")
	}
	if req.Description != nil && len(*req.Description) > 1000 {
		return fmt.Errorf("description must be less than 1000 characters")
	}
	return nil
}

// /////////New//////////////
func (s *ProjectService) GetProjectSummariesForTeams(ctx context.Context, hubID string, pageNo int, pageLimit int) (*client.ProjectSummaryResponse, error) {
	return s.ssdService.GetProjectSummariesForTeams(ctx, &GetProjectSummariesForTeamsRequest{
		HubID:     hubID,
		PageNo:    pageNo,
		PageLimit: pageLimit,
	})
}

func (s *ProjectService) GetProjectSummaryCount(ctx context.Context, hubIDs []string) (*client.SourceScanSummaryCount, error) {
	return s.ssdService.GetProjectSummaryCount(ctx, hubIDs)
}

func (s *ProjectService) calculateProjectStats(ctx context.Context,
	projectRef *client.ProjectRef) (*ProjectStats, error) {
	pstats := &ProjectStats{}
	// picking up semgrep sast findings now
	sastTool := "semgrep"

	for _, scanTarget := range projectRef.Scans {
		var dayDateTime time.Time
		dayCountLastIdx := 0

		// expecting only one sast results for now
		if pstats.SASTVulnerabilities == nil {
			for _, scanResult := range scanTarget.ScanResults {
				if scanResult.ScanTool == sastTool {
					sastResults, err := s.ssdService.GetSASTScanResults(ctx,
						"sourceScan", projectRef.ID, *scanTarget.Id, &client.SASTScanRequest{
							Semgrep: client.SASTScanToolDetails{
								ScanName:   scanResult.ScanType,
								ScanTool:   scanResult.ScanTool,
								ResultFile: scanResult.ResultFile,
								Status:     string(scanResult.RiskStatus),
							},
						})
					if err != nil {
						s.logger.LogError(err, "Failed to get sast findings", map[string]interface{}{
							"projectId": projectRef.ID,
							"scanId":    *scanTarget.Id,
						})
						return nil, fmt.Errorf("failed to get sast findings: %w", err)
					}

					var criticalCount, highCount, mediumCount, lowCount, unknownCount int
					for _, sr := range sastResults {
						if sr.ScanName == sastTool {
							for _, srd := range sr.Data {
								switch srd.Severity {
								case "critical":
									criticalCount++
								case "high":
									highCount++
								case "medium":
									mediumCount++
								case "low":
									lowCount++
								case "undefined":
									unknownCount++
								}
							}
							break
						}
					}
					pstats.SASTVulnerabilities = &VulnerabilityCounts{
						CriticalCount: &criticalCount,
						HighCount:     &highCount,
						MediumCount:   &mediumCount,
						LowCount:      &lowCount,
						UnknownCount:  &unknownCount,
					}
				}
			}
		}

		var lastScannedTime time.Time
		// expecting scandata entries would come in asc order only
		if scanTarget.LastScannedTime != nil {
			lastScannedTime = *scanTarget.LastScannedTime
		}

		if scanTarget.Artifact != nil {
			sort.SliceStable(scanTarget.Artifact.ScanData, func(i, j int) bool {
				if scanTarget.Artifact.ScanData[i].LastScannedAt == nil &&
					scanTarget.Artifact.ScanData[j].LastScannedAt == nil {
					return false
				}
				return scanTarget.Artifact.ScanData[i].LastScannedAt.UnixNano() > scanTarget.Artifact.ScanData[j].LastScannedAt.UnixNano()
			})
			for _, scanData := range scanTarget.Artifact.ScanData {
				if scanData.Tool != "syft" {
					continue
				}
				var criticalVuln, highVuln, mediumVuln, lowVuln, unknownVuln int
				if scanData.VulnCriticalCount != nil {
					criticalVuln = *scanData.VulnCriticalCount
				}
				if scanData.VulnHighCount != nil {
					highVuln = *scanData.VulnHighCount
				}
				if scanData.VulnMediumCount != nil {
					mediumVuln = *scanData.VulnMediumCount
				}
				if scanData.VulnLowCount != nil {
					lowVuln = *scanData.VulnLowCount
				}
				if scanData.VulnUnknownCount != nil {
					unknownVuln = *scanData.VulnUnknownCount
				}

				if dayDateTime.IsZero() || dayDateTime.Format("2006-01-02") != lastScannedTime.Format("2006-01-02") {
					outDatetime := *scanTarget.LastScannedTime
					outCount := 1
					pstats.ScanTimeFrequencies = append(pstats.ScanTimeFrequencies, &ScanTimeFrequency{
						Date:  &outDatetime,
						Count: &outCount,
					})
					// dayDateTime = *scanTarget.LastScannedTime
					// dayCountLastIdx = len(pstats.ScanTimeFrequencies) - 1
				} else if dayDateTime.Format("2006-01-02") == lastScannedTime.Format("2006-01-02") {
					prevCount := *pstats.ScanTimeFrequencies[dayCountLastIdx].Count
					prevCount++
					pstats.ScanTimeFrequencies[dayCountLastIdx].Count = &prevCount
				}

				pstats.RecentScans = append(pstats.RecentScans, &RecentScan{
					Branch:             scanTarget.Branch,
					CommitId:           scanData.ArtifactSha[7:14],
					ScanTime:           &lastScannedTime,
					IssueCriticalCount: &criticalVuln,
					IssueHighCount:     &highVuln,
					IssueMediumCount:   &mediumVuln,
					IssueLowCount:      &lowVuln,
					IssueUnknownCount:  &unknownVuln,
				})
				// only keeping 1 entr for latest scan now
				break
			}
		}
		// only keeping 1 entr for latest scan now
		break
	}

	if len(pstats.RecentScans) > 0 {
		// picking up most recent scan vulns
		// here entries are coming in asc order date time wise
		mostRecentScan := pstats.RecentScans[len(pstats.RecentScans)-1]
		pstats.SCAVulnerabilities = &VulnerabilityCounts{
			CriticalCount: mostRecentScan.IssueCriticalCount,
			HighCount:     mostRecentScan.IssueHighCount,
			MediumCount:   mostRecentScan.IssueMediumCount,
			LowCount:      mostRecentScan.IssueLowCount,
			UnknownCount:  mostRecentScan.IssueUnknownCount,
		}
	}
	return pstats, nil
}

func (s *ProjectService) getGithubUsername(ctx context.Context, accountId string) (string, error) {
	userNames, err := s.ssdService.GetRepoBranchList(ctx, map[string]string{
		// automated param from UI
		"accountId": accountId,
		// default params
		// ssd will look for repos from installation id based token
		// if orgName is blank
		"platform":  "github", // automate platform in future release
		"scanLevel": "org",
		"type":      "user",
	})
	if err != nil {
		return "", err
	} else if len(userNames) == 0 {
		return "", fmt.Errorf("user not found")
	}
	// based on installation id token org should always be one
	return userNames[0], nil
}

// getProjectStats: gets project stats
func (s *ProjectService) getProjectStats(ctx context.Context, projectId string) (*ProjectStats, error) {
	pStats := &ProjectStats{}
	scansVulns, err := s.scanRepo.GetProjectScansVulns(ctx, projectId)
	if err != nil {
		s.logger.LogError(err, "Failed to get scans details", map[string]interface{}{
			"projectId": projectId,
		})
		return nil, fmt.Errorf("failed to get scans details: %w", err)
	}

	scanFreqIdx := map[string]int{}
	for _, entry := range scansVulns {
		if models.ScanStatus(entry.Status) != models.ScanStatusCompleted {
			continue
		}

		if !entry.EndTime.IsZero() {
			formattedDate := entry.EndTime.Format("2006-01-02")
			if idx, idxOk := scanFreqIdx[formattedDate]; idxOk {
				prevCount := *pStats.ScanTimeFrequencies[idx].Count
				prevCount++
				pStats.ScanTimeFrequencies[idx].Count = &prevCount
			} else {
				endTime := entry.EndTime
				defCount := 1
				pStats.ScanTimeFrequencies = append(pStats.ScanTimeFrequencies, &ScanTimeFrequency{
					Count: &defCount,
					Date:  &endTime,
				})
				scanFreqIdx[formattedDate] = len(pStats.ScanTimeFrequencies) - 1
			}
		}

		sastStats, scaStats := calculateProjectVulnStats(entry.Vulnerabilites)
		if pStats.SASTVulnerabilities == nil {
			pStats.SASTVulnerabilities = sastStats
		}
		if pStats.SCAVulnerabilities == nil {
			pStats.SCAVulnerabilities = scaStats
		}

		commitSha := entry.CommitSHA
		if len(entry.CommitSHA) >= 14 {
			commitSha = entry.CommitSHA[7:14]
		}

		scanTime := entry.EndTime
		totalCriticalCount := *sastStats.CriticalCount + *scaStats.CriticalCount
		totalHighCount := *sastStats.HighCount + *scaStats.HighCount
		totalMediumCount := *sastStats.MediumCount + *scaStats.MediumCount
		totalLowCount := *sastStats.LowCount + *scaStats.LowCount
		totalUnknownCount := *sastStats.UnknownCount + *scaStats.UnknownCount
		pStats.RecentScans = append(pStats.RecentScans, &RecentScan{
			Branch:             entry.Branch,
			CommitId:           commitSha,
			ScanTime:           &scanTime,
			IssueCriticalCount: &totalCriticalCount,
			IssueHighCount:     &totalHighCount,
			IssueMediumCount:   &totalMediumCount,
			IssueLowCount:      &totalLowCount,
			IssueUnknownCount:  &totalUnknownCount,
		})
	}
	return pStats, nil
}

func calculateProjectVulnStats(vulns []*models.Vulnerability) (*VulnerabilityCounts, *VulnerabilityCounts) {
	// sast tool
	sastTool := "semgrep"
	// sca tool
	scaTool := "syft"
	uniqueSCAVulns := map[string]bool{}
	var sastAllCounts, sastCriticalCount, sastHighCount, sastMediumCount, sastLowCount, sastUnknownCount int
	var scaAllCounts, scaCriticalCount, scaHighCount, scaMediumCount, scaLowCount, scaUnknownCount int
	for _, vuln := range vulns {
		if vuln.Tool.String == sastTool {
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
		if vuln.Tool.String == scaTool {
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
	}

	sastStats := VulnerabilityCounts{
		AllCount:      &sastAllCounts,
		CriticalCount: &sastCriticalCount,
		HighCount:     &sastHighCount,
		MediumCount:   &sastMediumCount,
		LowCount:      &sastLowCount,
		UnknownCount:  &sastUnknownCount,
	}

	scanUniqueCounts := len(uniqueSCAVulns)
	scaStats := VulnerabilityCounts{
		AllCount:      &scaAllCounts,
		UniqueCount:   &scanUniqueCounts,
		CriticalCount: &scaCriticalCount,
		HighCount:     &scaHighCount,
		MediumCount:   &scaMediumCount,
		LowCount:      &scaLowCount,
		UnknownCount:  &scaUnknownCount,
	}
	return &sastStats, &scaStats
}
