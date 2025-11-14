package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/models"
)

// ScanRepository handles scan-related database operations
type ScanRepository struct {
	*BaseRepository
}

// NewScanRepository creates a new scan repository
func NewScanRepository() *ScanRepository {
	return &ScanRepository{
		BaseRepository: NewBaseRepository("scans"),
	}
}

// Create creates a new scan record with status 'pending'
func (r *ScanRepository) Create(ctx context.Context, scan *models.Scan) error {
	// Prepare data for creation with only required fields
	data := map[string]interface{}{
		"id":         uuid.New().String(),
		"project_id": scan.ProjectID,
		"repository": scan.Repository,
		"branch":     scan.Branch,
		"commit_sha": scan.CommitSHA,
		"tag":        scan.Tag,
		"settings":   scan.Settings,
		"status":     scan.Status,
		"hub_id":     scan.HubID,
		"created_at": time.Now(),
	}

	id, err := r.BaseRepository.Create(ctx, r.table, data)
	if err != nil {
		return err
	}

	scan.ID = id
	return nil
}

// ScanRecord represents a scan record from the database for polling
type ScanRecord struct {
	ID         string
	ProjectID  string
	Status     string
	Repository string
	Branch     string
	CommitSHA  string
}

// GetPendingScans retrieves all scans with QUEUED or RUNNING status
func (r *ScanRepository) GetPendingScans(ctx context.Context) ([]ScanRecord, error) {
	query := `
		SELECT 
			id, 
			project_id, 
			status, 
			repository, 
			branch,
			commit_sha
		FROM scans 
		WHERE status IN ('pending', 'scanning')
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending scans: %w", err)
	}
	defer rows.Close()

	var scans []ScanRecord
	for rows.Next() {
		var scan ScanRecord

		err := rows.Scan(
			&scan.ID,
			&scan.ProjectID,
			&scan.Status,
			&scan.Repository,
			&scan.Branch,
			&scan.CommitSHA,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		scans = append(scans, scan)
	}

	return scans, rows.Err()
}

// UpdateScanStatus updates the scan status and related fields
func (r *ScanRepository) UpdateScanStatus(ctx context.Context, scanID string, scanData *client.ScanResultDataResponse) error {
	tx, err := r.NewTransaction(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update scans table
	now := time.Now()
	var endTime *time.Time
	if scanData.Status == "completed" || scanData.Status == "fail" {
		endTime = &now
	}

	updateScanQuery := `
		UPDATE scans 
		SET 
			status = $1,
			commit_sha = $2,
			end_time = $3,
			branch = $4
		WHERE id = $5
	`

	_, err = tx.Exec(ctx, updateScanQuery,
		scanData.Status,
		scanData.HeadCommit,
		endTime,
		scanData.Branch,
		scanID,
	)
	if err != nil {
		return fmt.Errorf("failed to update scans table: %w", err)
	}

	// Update or insert scan_type records for each scan type
	if err := r.updateScanTypes(ctx, tx, scanID, scanData); err != nil {
		return fmt.Errorf("failed to update scan types: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// update scan status in bulk
func (r *ScanRepository) UpdateScanStatusInBulk(ctx context.Context, scanIDs []string, status string, endTime time.Time) error {
	query := `
		UPDATE scans
		SET status = $1
		end_time = $2
		WHERE id = ANY($2)
	`
	_, err := r.db.Exec(ctx, query, status, endTime, scanIDs)
	if err != nil {
		return fmt.Errorf("failed to update scan status in bulk: %w", err)
	}
	return nil
}

// updateScanTypes updates the scan_type table with scan results
func (r *ScanRepository) updateScanTypes(ctx context.Context, tx *Transaction, scanID string, scanData *client.ScanResultDataResponse) error {
	// Process OpenSSF scans
	// if scanData.ScannedFiledData.OpenSSF.Openssf.ScanName != "" {
	// 	if err := r.upsertScanType(ctx, tx, scanID, "openssf", &scanData.ScannedFiledData.OpenSSF.Openssf); err != nil {
	// 		return err
	// 	}
	// }

	// Process SAST scans
	if scanData.ScannedFiledData.SAST.Semgrep.ScanName != "" {
		if err := r.upsertScanType(ctx, tx, scanID, "sast", &scanData.ScannedFiledData.SAST.Semgrep); err != nil {
			return err
		}
	}

	// Process SCA scans
	// if scanData.ScannedFiledData.SCA.CodeLicense.ScanName != "" {
	// 	if err := r.upsertScanType(ctx, tx, scanID, "codelicense", &scanData.ScannedFiledData.SCA.CodeLicense); err != nil {
	// 		return err
	// 	}
	// }

	// if scanData.ScannedFiledData.SCA.CodeSecret.ScanName != "" {
	// 	if err := r.upsertScanType(ctx, tx, scanID, "codesecret", &scanData.ScannedFiledData.SCA.CodeSecret); err != nil {
	// 		return err
	// 	}
	// }

	if scanData.ScannedFiledData.SBOM.SBOM.ScanName != "" {
		if err := r.upsertScanType(ctx, tx, scanID, "sca", &scanData.ScannedFiledData.SBOM.SBOM); err != nil {
			return err
		}
	}

	return nil
}

// upsertScanType inserts or updates a scan_type record
func (r *ScanRepository) upsertScanType(ctx context.Context, tx *Transaction, scanID, scanType string, scanFiles *client.ScanFiles) error {
	scanTypeID := fmt.Sprintf("%s-%s", scanID, scanType)

	// Convert scan files to JSON for raw_json field
	rawJSON, err := json.Marshal(scanFiles)
	if err != nil {
		return fmt.Errorf("failed to marshal scan files: %w", err)
	}

	query := `
		INSERT INTO scan_type (
			id, 
			scan_id, 
			scan_type, 
			tool, 
			file_url, 
			raw_json
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) 
		DO UPDATE SET
			tool = EXCLUDED.tool,
			file_url = EXCLUDED.file_url,
			raw_json = EXCLUDED.raw_json
	`

	_, err = tx.Exec(ctx, query,
		scanTypeID,
		scanID,
		scanType,
		scanFiles.ScanTool,
		scanFiles.ResultFile,
		rawJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert scan_type: %w", err)
	}

	return nil
}

// UpdateScanTypeCountsForType updates the count fields for a specific scan_type
// UpdateScanTypeCountsForType updates the count fields for a specific scan_type
func (r *ScanRepository) UpdateScanTypeCountsForType(ctx context.Context, scanID, scanType string, counts map[string]int) error {
	query := `
		UPDATE scan_type 
		SET 
			findings_count = $1,
			critical_count = $2,
			high_count = $3,
			medium_count = $4,
			low_count = $5,
			unknown_count = $6
		WHERE scan_id = $7 AND scan_type = $8
	`

	_, err := r.db.Exec(ctx, query,
		counts["findings_count"],
		counts["critical_count"],
		counts["high_count"],
		counts["medium_count"],
		counts["low_count"],
		counts["unknown_count"],
		scanID,
		scanType,
	)
	if err != nil {
		return fmt.Errorf("failed to update scan_type counts: %w", err)
	}
	return nil
}

func (s *ScanRepository) GetHubScansVulns(ctx context.Context, hubId string) ([]*models.ProjectExt, error) {
	var projects []*models.ProjectExt

	// Get scans with vulnerabilities and project info
	query := `SELECT s.id AS scan_id, s.project_id, p.name as project_name, s.status, s.branch, 
	s.commit_sha, s.end_time, v.id, v.scan_id, v.name, v.scan_type, v.tool, v.severity
	FROM scans s
	LEFT JOIN projects p ON s.project_id = p.id
	LEFT JOIN vulnerabilities v ON v.scan_id = s.id
	WHERE s.hub_id = $1
	ORDER BY s.end_time DESC, s.project_id DESC, s.id, v.name DESC`

	rows, err := s.db.Query(ctx, query, hubId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projectsIdx := map[string]int{}
	scansIdx := map[string]int{}
	for rows.Next() {
		var scan models.ScanExt
		var vuln models.Vulnerability
		var projectName string
		if err := rows.Scan(
			&scan.ScanId,
			&scan.ProjectId,
			&projectName,
			&scan.Status,
			&scan.Branch,
			&scan.CommitSHA,
			&scan.EndTime,
			&vuln.ID,
			&vuln.ScanID,
			&vuln.Name,
			&vuln.ScanType,
			&vuln.Tool,
			&vuln.Severity,
		); err != nil {
			return nil, err
		}
		if pIdx, pok := projectsIdx[scan.ProjectId]; pok {
			if sIdx, ok := scansIdx[scan.ScanId]; ok {
				projects[pIdx].Scans[sIdx].Vulnerabilites = append(projects[pIdx].Scans[sIdx].Vulnerabilites, &vuln)
			} else {
				scan.Vulnerabilites = append(scan.Vulnerabilites, &vuln)
				projects[pIdx].Scans = append(projects[pIdx].Scans, &scan)
				scansIdx[scan.ScanId] = len(projects[pIdx].Scans) - 1
			}
		} else {
			scan.Vulnerabilites = append(scan.Vulnerabilites, &vuln)
			projects = append(projects, &models.ProjectExt{
				ProjectId:   scan.ProjectId,
				ProjectName: projectName,
				Scans: []*models.ScanExt{
					&scan,
				},
			})
			projectsIdx[scan.ProjectId] = len(projects) - 1
			scansIdx[scan.ScanId] = 0
		}
	}
	return projects, nil
}

func (s *ScanRepository) GetProjectScansVulns(ctx context.Context, projectId string) ([]*models.ScanExt, error) {
	var scans []*models.ScanExt

	query := `SELECT s.id AS scan_id, s.project_id, s.status, s.branch, 
	s.commit_sha, s.end_time, v.scan_id, v.name, v.scan_type, v.tool, v.severity
	FROM scans s
	LEFT JOIN vulnerabilities v ON v.scan_id = s.id
	WHERE s.status = 'completed' AND s.project_id = $1
	ORDER BY s.end_time DESC, s.id, v.name DESC`

	rows, err := s.db.Query(ctx, query, projectId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	scansIdx := map[string]int{}
	for rows.Next() {
		var p models.ScanExt
		var v models.Vulnerability
		if err := rows.Scan(
			&p.ScanId,
			&p.ProjectId,
			&p.Status,
			&p.Branch,
			&p.CommitSHA,
			&p.EndTime,
			&v.ScanID,
			&v.Name,
			&v.ScanType,
			&v.Tool,
			&v.Severity,
		); err != nil {
			return nil, err
		}
		if idx, ok := scansIdx[p.ScanId]; ok {
			scans[idx].Vulnerabilites = append(scans[idx].Vulnerabilites, &v)
		} else {
			p.Vulnerabilites = append(p.Vulnerabilites, &v)
			scans = append(scans, &p)
			scansIdx[p.ScanId] = len(scans) - 1
		}
	}
	return scans, nil
}

// taking all data for audit
func (s *ScanRepository) GetScansVulns(ctx context.Context) ([]*models.Hub, error) {
	var hubs []*models.Hub
	query := `SELECT s.id AS scan_id, s.project_id, s.hub_id, s.status, 
	s.created_at, v.id, v.scan_id
	FROM scans s
	LEFT JOIN vulnerabilities v ON v.scan_id = s.id
	ORDER BY s.hub_id, s.project_id, s.end_time DESC, s.id DESC, v.name`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hubIdx := map[string]int{}
	projectsIdx := map[string]int{}
	scansIdx := map[string]int{}
	for rows.Next() {
		var scan models.ScanExt
		var vuln models.Vulnerability
		var hubId uuid.UUID
		if err := rows.Scan(
			&scan.ScanId,
			&scan.ProjectId,
			&hubId,
			&scan.Status,
			&scan.CreatedAt,
			&vuln.ID,
			&vuln.ScanID,
		); err != nil {
			return nil, err
		}
		pKey := hubId.String() + scan.ProjectId
		sKey := hubId.String() + scan.ProjectId + scan.ScanId
		if hIdx, hok := hubIdx[hubId.String()]; hok {
			if pIdx, pok := projectsIdx[pKey]; pok {
				if sIdx, sok := scansIdx[sKey]; sok {
					hubs[hIdx].Projects[pIdx].Scans[sIdx].Vulnerabilites = append(hubs[hIdx].Projects[pIdx].Scans[sIdx].Vulnerabilites, &vuln)
				} else {
					scan.Vulnerabilites = append(scan.Vulnerabilites, &vuln)
					hubs[hIdx].Projects[pIdx].Scans = append(hubs[hIdx].Projects[pIdx].Scans, &scan)
					scansIdx[sKey] = len(hubs[hIdx].Projects[pIdx].Scans) - 1
				}
			} else {
				scan.Vulnerabilites = append(scan.Vulnerabilites, &vuln)
				hubs[hIdx].Projects = append(hubs[hIdx].Projects, &models.ProjectExt{
					ProjectId: scan.ProjectId,
					Scans: []*models.ScanExt{
						&scan,
					},
				})
				projectsIdx[pKey] = len(hubs[hIdx].Projects) - 1
				scansIdx[sKey] = 0
			}
		} else {
			scan.Vulnerabilites = append(scan.Vulnerabilites, &vuln)
			hubs = append(hubs, &models.Hub{
				ID: hubId,
				Projects: []*models.ProjectExt{
					{
						ProjectId: scan.ProjectId,
						Scans: []*models.ScanExt{
							&scan,
						},
					},
				},
			})
			projectsIdx[pKey] = 0
			scansIdx[sKey] = 0
			hubIdx[hubId.String()] = len(hubs) - 1
		}
	}
	return hubs, nil
}
