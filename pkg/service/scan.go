package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/config"
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

type RescanResponse struct {
	Message string `json:"message"`
}

func (s *ScanService) ValidateRescanRequest(req *RescanRequest) error {
	if req.ProjectID == "" || req.HubID == "" || req.Repository == "" || req.Branch == "" {
		return fmt.Errorf("invalid request")
	}
	return nil
}

func (s *ScanService) Rescan(ctx context.Context, req *RescanRequest) (string, string, error) {
	scanResult, err := s.ssdService.GetScanResultData(ctx, &client.ScanResultDataRequest{
		ProjectID:  req.ProjectID,
		TeamID:     req.HubID,
		Repository: req.Repository,
		Type:       "sourceScan",
		Branch:     req.Branch,
	})
	if err != nil {
		return "", "", err
	}

	if scanResult.ScanID == "" {
		return "", "", fmt.Errorf("scan result not found")
	}

	if strings.ToLower(scanResult.Status) == string(models.ScanStatusScanning) ||
		strings.ToLower(scanResult.Status) == string(models.ScanStatusPending) {
		return "", "", fmt.Errorf("scanning already in progress")
	}

	message, err := s.ssdService.Rescan(ctx, req, scanResult)
	if err != nil {
		return "", "", err
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
		return "", "", err
	}

	return message, scan.ID, nil
}

// ScanFileRequest represents a file scan request
type ScanFileRequest struct {
	FileContent []byte
	Filename    string
	FileSize    int64
}

// SemgrepData represents the data structure in the response
type SemgrepData struct {
	Results []interface{} `json:"results"`
	Version string        `json:"version"`
}

// ScanFileResponse represents the result of a file scan
type ScanFileResponse struct {
	Data    *SemgrepData `json:"data,omitempty"`
	Message string       `json:"message"`
	Success bool         `json:"success"`
}

// ValidateFile validates the uploaded file
func (s *ScanService) ValidateFile(req *ScanFileRequest) error {
	// Check file size
	maxSize := config.GetSemgrepMaxFileSize()
	if req.FileSize > maxSize {
		return fmt.Errorf("file size %d bytes exceeds maximum allowed size of %d bytes", req.FileSize, maxSize)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(req.Filename))
	if ext == "" {
		return fmt.Errorf("file must have an extension")
	}

	allowedExts := config.GetSemgrepAllowedExtensions()
	extAllowed := false
	for _, allowedExt := range allowedExts {
		if strings.ToLower(allowedExt) == ext {
			extAllowed = true
			break
		}
	}

	if !extAllowed {
		return fmt.Errorf("file extension '%s' is not allowed. Allowed extensions: %v", ext, allowedExts)
	}

	// Check if file content matches size
	if int64(len(req.FileContent)) != req.FileSize {
		return fmt.Errorf("file content size mismatch")
	}

	return nil
}

// ScanFileWithSemgrep scans a file using semgrep CLI and returns the results
func (s *ScanService) ScanFileWithSemgrep(ctx context.Context, req *ScanFileRequest) (*ScanFileResponse, error) {
	startTime := time.Now()

	// Validate file
	if err := s.ValidateFile(req); err != nil {
		return nil, fmt.Errorf("file validation failed: %w", err)
	}

	// Create temp file with proper extension
	tempFile, err := os.CreateTemp(os.TempDir(), "semgrep-scan-*"+filepath.Ext(req.Filename))
	if err != nil {
		s.logger.LogError(err, "Failed to create temporary file", map[string]interface{}{
			"filename": req.Filename,
		})
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempFilePath := tempFile.Name()

	// Ensure cleanup
	defer func() {
		if err := os.Remove(tempFilePath); err != nil {
			s.logger.LogWarning("Failed to remove temporary file", map[string]interface{}{
				"file_path": tempFilePath,
				"error":     err.Error(),
			})
		}
	}()

	// Write file content
	if _, err := tempFile.Write(req.FileContent); err != nil {
		s.logger.LogError(err, "Failed to write to temporary file", map[string]interface{}{
			"file_path": tempFilePath,
		})
		return nil, fmt.Errorf("failed to write file content: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		s.logger.LogError(err, "Failed to close temporary file", map[string]interface{}{
			"file_path": tempFilePath,
		})
		return nil, fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Execute semgrep with file-type-specific rules (much faster than all rules)
	semgrepData, err := s.executeSemgrep(ctx, tempFilePath, req.Filename)
	scanTime := time.Since(startTime)

	if err != nil {
		s.logger.LogError(err, "Semgrep execution failed", map[string]interface{}{
			"filename":  req.Filename,
			"scan_time": scanTime.String(),
		})
		return &ScanFileResponse{
			Success: false,
			Message: fmt.Sprintf("Scan failed: %s", err.Error()),
		}, nil // Return response with error, don't fail the request
	}

	return &ScanFileResponse{
		Data:    semgrepData,
		Message: "Scan completed successfully",
		Success: true,
	}, nil
}

// executeSemgrep executes semgrep CLI on the given file and returns parsed results
// Uses smart approach: file-type-specific rules (fastest) -> auto mode (online)
func (s *ScanService) executeSemgrep(ctx context.Context, filePath string, filename string) (*SemgrepData, error) {
	timeoutSeconds := config.GetSemgrepTimeoutSeconds()

	// Step 1: Try file-type-specific rule sets (fastest, works offline after first download)
	// This is much faster than using all rules or cache file
	ruleSet := s.getRuleSetForFile(filename)
	if ruleSet != "" {
		s.logger.LogInfo("Using file-type-specific semgrep rules", map[string]interface{}{
			"rule_set": ruleSet,
			"filename": filename,
		})
		result, err := s.runSemgrepWithConfig(ctx, "semgrep", timeoutSeconds, filePath, ruleSet)
		if err == nil {
			return result, nil
		}
		// If specific rule set fails, log and fall through to auto mode
		s.logger.LogWarning("File-type-specific rules failed, falling back to auto mode", map[string]interface{}{
			"rule_set": ruleSet,
			"error":    err.Error(),
		})
	}

	// Step 2: Fall back to auto mode (online, but semgrep caches rules automatically)
	s.logger.LogInfo("Using semgrep auto config (online mode)", map[string]interface{}{
		"filename": filename,
	})
	return s.runSemgrepWithConfig(ctx, "semgrep", timeoutSeconds, filePath, "auto")
}

// getRuleSetForFile returns the appropriate semgrep rule set based on file extension
// Returns empty string if no specific rule set matches
func (s *ScanService) getRuleSetForFile(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	// Map file extensions to semgrep rule sets
	ruleSetMap := map[string]string{
		".py":         "r/python",
		".js":         "r/javascript",
		".jsx":        "r/javascript",
		".ts":         "r/typescript",
		".tsx":        "r/typescript",
		".go":         "r/go",
		".java":       "r/java",
		".cpp":        "r/cpp",
		".c":          "r/c",
		".cs":         "r/csharp",
		".rb":         "r/ruby",
		".php":        "r/php",
		".rs":         "r/rust",
		".swift":      "r/swift",
		".kt":         "r/kotlin",
		".scala":      "r/scala",
		".sh":         "r/bash",
		".yaml":       "r/yaml",
		".yml":        "r/yaml",
		".json":       "r/json",
		".xml":        "r/xml",
		".html":       "r/html",
		".css":        "r/css",
		".sql":        "r/sql",
		".tf":         "r/terraform",
		".tfvars":     "r/terraform",
		".dockerfile": "r/docker",
	}

	if ruleSet, ok := ruleSetMap[ext]; ok {
		return ruleSet
	}

	return ""
}

// runSemgrepWithConfig executes semgrep with a specific config (cache path or "auto")
// Note: When using a local file path for --config, semgrep automatically works offline
// The offline parameter is kept for future compatibility but not used (semgrep doesn't support --offline flag)
func (s *ScanService) runSemgrepWithConfig(ctx context.Context, cliPath string, timeoutSeconds int, filePath string, configValue string) (*SemgrepData, error) {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	// Prepare semgrep command
	// When configValue is a file path (not "auto"), semgrep automatically works offline
	// --disable-version-check speeds up execution by skipping version checks
	args := []string{"--json", "--disable-version-check", "--no-git-ignore", "--config=" + configValue, filePath}

	cmd := exec.CommandContext(timeoutCtx, cliPath, args...)

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start semgrep: %w", err)
	}

	// Read stdout
	output, err := io.ReadAll(stdout)
	if err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to read semgrep output: %w", err)
	}

	// Read stderr for error messages
	stderrOutput, _ := io.ReadAll(stderr)

	// Wait for command to complete
	err = cmd.Wait()

	// Check for timeout
	if timeoutCtx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("semgrep execution timed out after %d seconds", timeoutSeconds)
	}

	// Check for execution errors
	if err != nil {
		errorMsg := string(stderrOutput)
		if errorMsg == "" {
			errorMsg = err.Error()
		}
		return nil, fmt.Errorf("semgrep execution failed: %s", errorMsg)
	}

	// Validate JSON output
	if len(output) == 0 {
		return &SemgrepData{
			Results: []interface{}{},
			Version: "",
		}, nil // Return empty results if no output
	}

	// Parse semgrep JSON output
	var semgrepOutput map[string]interface{}
	if err := json.Unmarshal(output, &semgrepOutput); err != nil {
		s.logger.LogError(err, "Invalid JSON output from semgrep", map[string]interface{}{
			"file_path": filePath,
			"output":    string(output),
		})
		return nil, fmt.Errorf("semgrep returned invalid JSON: %w", err)
	}

	// Extract results and version from semgrep output
	results, ok := semgrepOutput["results"]
	if !ok {
		results = []interface{}{}
	}

	// Convert results to slice if it's not already
	var resultsSlice []interface{}
	switch v := results.(type) {
	case []interface{}:
		resultsSlice = v
	case nil:
		resultsSlice = []interface{}{}
	default:
		// Try to convert to slice
		resultsSlice = []interface{}{v}
	}

	version, _ := semgrepOutput["version"].(string)
	if version == "" {
		version = "unknown"
	}

	return &SemgrepData{
		Results: resultsSlice,
		Version: version,
	}, nil
}
