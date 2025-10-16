package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

	scan.ID = id.String()
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
	if scanData.Status == "completed" || scanData.Status == "failed" {
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

	if scanData.ScannedFiledData.SCA.Sbom.ScanName != "" {
		if err := r.upsertScanType(ctx, tx, scanID, "sca", &scanData.ScannedFiledData.SCA.Sbom); err != nil {
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
