package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type ScanService struct {
	ssdService     *SSDService
	scanRepository *repository.ScanRepository
	logger         *utils.ErrorLogger
}

func NewScanService() *ScanService {
	return &ScanService{
		ssdService:     NewSSDService(),
		logger:         utils.NewErrorLogger("scan_service"),
		scanRepository: repository.NewScanRepository(),
	}
}

type RescanRequest struct {
	ProjectID  string `json:"projectId"`
	HubID      string `json:"hubId"`
	Repository string `json:"repository"`
	Branch     string `json:"branch"`
}

type SemgrepService struct {
	logger      *utils.ErrorLogger
	tempDir     string
	maxFileSize int64
	scanTimeout time.Duration
	workerPool  chan struct{} // Semaphore for concurrent scans
}

type RescanResponse struct {
	Message string `json:"message"`
}

func (s *ScanService) ValidateRescanRequest(req *RescanRequest) error {
	if req.ProjectID == "" || req.HubID == "" || req.Repository == "" || req.Branch == "" {
		return fmt.Errorf("invalid request")
	}
	return nil
}

func (s *ScanService) Rescan(ctx context.Context, req *RescanRequest) (string, error) {
	scanResult, err := s.ssdService.GetScanResultData(ctx, &client.ScanResultDataRequest{
		ProjectID:  req.ProjectID,
		TeamID:     req.HubID,
		Repository: req.Repository,
		Type:       "sourceScan",
		Branch:     req.Branch,
	})
	if err != nil {
		return "", err
	}

	if scanResult.ScanID == "" {
		return "", fmt.Errorf("scan result not found")
	}

	// CREATE SCANS ENTRY FOR POLLING
	scan := &models.Scan{
		ProjectID:  req.ProjectID,
		Branch:     req.Branch,
		Repository: req.Repository,
		CommitSHA:  scanResult.HeadCommit,
		Tag:        scanResult.ArtifactTag,
		Status:     string(client.RiskStatusPending),
		HubID:      req.HubID,
	}

	if err := s.scanRepository.Create(ctx, scan); err != nil {
		return "", err
	}

	return s.ssdService.Rescan(ctx, req, scanResult)
}

const (
	DefaultMaxFileSize = 50 * 1024 * 1024 // 50 MB
	DefaultScanTimeout = 5 * time.Minute
	DefaultMaxWorkers  = 10 // Max concurrent scans
)

func NewSemgrepService(tempDir string, maxWorkers int) *SemgrepService {
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers
	}

	return &SemgrepService{
		logger:      utils.NewErrorLogger("semgrep_service"),
		tempDir:     tempDir,
		maxFileSize: DefaultMaxFileSize,
		scanTimeout: DefaultScanTimeout,
		workerPool:  make(chan struct{}, maxWorkers),
	}
}

// ScanFile scans a file or directory using semgrep and returns results
func (s *SemgrepService) ScanFile(ctx context.Context, req *models.SemgrepScanRequest) (*models.SemgrepResult, error) {
	// Acquire worker slot (rate limiting)
	select {
	case s.workerPool <- struct{}{}:
		defer func() { <-s.workerPool }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Create context with timeout
	scanCtx, cancel := context.WithTimeout(ctx, s.scanTimeout)
	defer cancel()

	// Convert to absolute path
	absPath, err := filepath.Abs(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if path exists (file or directory)
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("path not found: %s, error: %w", absPath, err)
	}

	// If it's a file (not a directory), check size
	if !fileInfo.IsDir() {
		if fileInfo.Size() > s.maxFileSize {
			return nil, fmt.Errorf("file size %d exceeds maximum %d bytes", fileInfo.Size(), s.maxFileSize)
		}
	}

	// Verify semgrep is available
	semgrepPath, err := exec.LookPath("semgrep")
	if err != nil {
		return nil, fmt.Errorf("semgrep not found in PATH: %w. Install with: python3 -m pip install semgrep", err)
	}

	// Create temporary file for semgrep output (better for memory management)
	outputFile, err := os.CreateTemp(s.tempDir, "semgrep-output-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", err)
	}
	outputFilePath := outputFile.Name()
	outputFile.Close()
	defer os.Remove(outputFilePath) // Cleanup temp file

	// Build semgrep command arguments (mimicking the sample code)
	args := []string{
		"semgrep",
		"--json",
		"--quiet",                 // Reduce output
		"--disable-version-check", // Skip version check
		"--disable-nosem",
		"--no-git-ignore",
		"--no-rewrite-rule-ids",
		"-j", "1", // Single job to reduce memory
		"--max-memory", "3000", // Limit memory per file (MB)
		"--output", outputFilePath, // Write to file instead of stdout
	}

	// Add config if provided
	if req.Config != "" {
		args = append(args, "--config", req.Config)
		args = append(args, "--metrics=off")
	} else {
		// Use default ruleset - auto requires metrics to download config
		args = append(args, "--config", "auto")
		// Don't add --metrics=off here because auto config needs metrics
	}

	// Add custom rules if provided
	for _, rule := range req.Rules {
		args = append(args, "--config", rule)
	}

	// Add target file/directory (use absolute path)
	args = append(args, absPath)

	// Execute semgrep command
	cmd := exec.CommandContext(scanCtx, semgrepPath, args[1:]...) // Skip "semgrep" since semgrepPath already has it

	// Set environment variables - ONLY when using specific config (not "auto")
	envVars := os.Environ()
	if req.Config != "" {
		// Using specific config - can disable metrics (like air-gapped mode in sample)
		envVars = append(envVars,
			"SEMGREP_OFFLINE=1",
			"SEMGREP_SEND_METRICS=off",
		)
	}
	// If req.Config == "" (using "auto"), don't set these env vars
	cmd.Env = envVars

	// Set working directory to the file's parent directory
	if fileInfo.IsDir() {
		cmd.Dir = absPath
	} else {
		cmd.Dir = filepath.Dir(absPath)
	}

	// Capture stderr for error reporting
	var stderr strings.Builder
	cmd.Stderr = &stderr

	err = cmd.Run()
	errOutput := stderr.String()

	// Read the output file
	outputBytes, readErr := os.ReadFile(outputFilePath)
	if readErr != nil {
		// If file read fails, check if semgrep failed
		if err != nil {
			// Build comprehensive error message
			var errMsg strings.Builder
			errMsg.WriteString("semgrep execution failed")

			if exitErr, ok := err.(*exec.ExitError); ok {
				fmt.Fprintf(&errMsg, " (exit code %d)", exitErr.ExitCode())
			}
			fmt.Fprintf(&errMsg, ": %v", err)

			if errOutput != "" {
				fmt.Fprintf(&errMsg, "\nstderr: %s", strings.TrimSpace(errOutput))
			}
			errMsg.WriteString("\n(failed to read output file)")

			return nil, errors.New(errMsg.String())
		}
		return nil, fmt.Errorf("failed to read semgrep output file: %w", readErr)
	}

	// Check if we got valid JSON output (even if command failed)
	// Semgrep can return non-zero exit codes for valid reasons (findings found, warnings, etc.)
	if len(outputBytes) > 0 {
		var result models.SemgrepResult
		if jsonErr := json.Unmarshal(outputBytes, &result); jsonErr == nil {
			// Valid JSON - return it even if exit code was non-zero
			return &result, nil
		}
	}

	// If we got here, semgrep failed and didn't produce valid JSON
	if err != nil {
		// Check if it's an exec error
		if execErr, ok := err.(*exec.Error); ok {
			if execErr.Err == exec.ErrNotFound {
				return nil, fmt.Errorf("semgrep executable not found: %w. Install with: python3 -m pip install semgrep", err)
			}
		}

		// Build comprehensive error message
		var errMsg strings.Builder
		errMsg.WriteString("semgrep execution failed")

		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(&errMsg, " (exit code %d)", exitErr.ExitCode())
		}
		fmt.Fprintf(&errMsg, ": %v", err)

		if errOutput != "" {
			fmt.Fprintf(&errMsg, "\nstderr: %s", strings.TrimSpace(errOutput))
		}
		if len(outputBytes) > 0 {
			// Try to parse error from JSON output
			var result models.SemgrepResult
			if json.Unmarshal(outputBytes, &result) == nil && len(result.Errors) > 0 {
				fmt.Fprintf(&errMsg, "\nsemgrep errors: %+v", result.Errors)
			}
		}

		return nil, errors.New(errMsg.String())
	}

	// Parse JSON output if command succeeded
	if len(outputBytes) == 0 {
		return nil, fmt.Errorf("semgrep produced no output")
	}

	var result models.SemgrepResult
	if err := json.Unmarshal(outputBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse semgrep output: %w, output: %s", err, string(outputBytes))
	}

	return &result, nil
}

// ScanFileFromRequest handles file upload, validation, and scanning
func (s *SemgrepService) ScanFileFromRequest(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader, config string) (*models.SemgrepResult, error) {
	// Validate file size (50MB max)
	const maxFileSize = 50 * 1024 * 1024
	if fileHeader.Size > maxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum of %d MB", maxFileSize/(1024*1024))
	}

	// Process config parameter
	// If config is "auto" or empty, pass empty string to semgrep service
	// Empty string in semgrep.go will use "auto" without --metrics=off
	semgrepConfig := ""
	if config != "" && config != "auto" {
		semgrepConfig = config
	}

	// Create temporary directory for file upload
	tempDir, err := os.MkdirTemp(os.TempDir(), "semgrep-scan-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		// Cleanup: remove temp directory and all contents
		os.RemoveAll(tempDir)
	}()

	// Get absolute path of temp directory
	absTempDir, err := filepath.Abs(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create file in temp directory with original filename
	// This preserves the file extension so semgrep can detect the language
	tempFilePath := filepath.Join(absTempDir, fileHeader.Filename)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	// Copy uploaded file to temp file
	if _, err := io.Copy(tempFile, file); err != nil {
		tempFile.Close()
		return nil, fmt.Errorf("failed to copy file to temp: %w", err)
	}

	// Ensure file is fully written to disk
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return nil, fmt.Errorf("failed to sync file: %w", err)
	}
	tempFile.Close()

	// Verify file exists before scanning
	if _, err := os.Stat(tempFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("temp file does not exist after creation: %s", tempFilePath)
	}

	// Scan file - pass the specific file path (not directory) to semgrep
	scanReq := &models.SemgrepScanRequest{
		FilePath: tempFilePath, // Pass absolute file path
		FileName: fileHeader.Filename,
		Config:   semgrepConfig, // Empty string for "auto", specific config otherwise
	}

	return s.ScanFile(ctx, scanReq)
}
