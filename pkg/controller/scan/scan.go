package scan

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/opsmx/ai-guardian-api/pkg/service"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type ScanController struct {
	scanService service.ScanService
	logger      *utils.ErrorLogger
}

func NewScanController() *ScanController {
	return &ScanController{
		scanService: *service.NewScanService(),
		logger:      utils.NewErrorLogger("scan_controller"),
	}
}

// Rescan triggers a rescan of the specified repository
// @Summary Trigger a rescan
// @Description Initiates a rescan of the specified repository for vulnerabilities
// @Tags Scans
// @Accept  json
// @Produce  json
// @Param   request body service.RescanRequest true "Rescan configuration"
// @Success 200 {object} map[string]interface{} "Rescan initiated successfully"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/scans/rescan [post]
func (c *ScanController) Rescan(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var payload service.RescanRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		c.logger.LogError(err, err.Error(), nil)
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.scanService.ValidateRescanRequest(&payload); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := c.scanService.Rescan(r.Context(), &payload)
	if err != nil {
		c.logger.LogError(err, err.Error(), nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SendSuccessResponseWithNoData(w, resp)
}

// ScanFile scans an uploaded file using semgrep CLI
// @Summary Scan a file with semgrep
// @Description Accepts a file via multipart form and scans it using semgrep CLI, returning JSON results in the format: { "data": { "results": [...], "version": "..." }, "message": "...", "success": true }
// @Tags Scans
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to scan (max 10MB, must have allowed extension)"
// @Success 200 {object} service.ScanFileResponse "Scan results with findings in data.results array"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 413 {object} map[string]string "File too large"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/scans/file [post]
func (c *ScanController) ScanFile(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form with max memory of 32MB
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		c.logger.LogWarning("Failed to parse multipart form", map[string]interface{}{
			"error": err.Error(),
		})
		utils.SendErrorResponse(w, http.StatusBadRequest, "Failed to parse form data")
		return
	}

	// Get file from form
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		c.logger.LogWarning("File not found in form", map[string]interface{}{
			"error": err.Error(),
		})
		utils.SendErrorResponse(w, http.StatusBadRequest, "File is required")
		return
	}
	defer file.Close()

	// Read file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		c.logger.LogError(err, "Failed to read uploaded file", map[string]interface{}{
			"filename": fileHeader.Filename,
		})
		utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to read uploaded file")
		return
	}

	// Create scan request
	scanReq := &service.ScanFileRequest{
		FileContent: fileContent,
		Filename:    fileHeader.Filename,
		FileSize:    fileHeader.Size,
	}

	// Scan file
	scanResp, err := c.scanService.ScanFileWithSemgrep(r.Context(), scanReq)
	if err != nil {
		c.logger.LogError(err, "Failed to scan file", map[string]interface{}{
			"filename": fileHeader.Filename,
		})
		utils.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Return the scan response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(scanResp)
}
