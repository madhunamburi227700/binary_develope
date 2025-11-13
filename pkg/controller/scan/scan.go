package scan

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/opsmx/ai-guardian-api/pkg/service"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type ScanController struct {
	scanService    service.ScanService
	semgrepService service.SemgrepService
	logger         *utils.ErrorLogger
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

func (c *ScanController) ScanFile(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form with max memory of 64MB
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		c.logger.LogError(err, "Failed to parse multipart form", nil)
		utils.SendErrorResponse(w, http.StatusBadRequest, "Failed to parse form data")
		return
	}

	// Check if file exists in form
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		c.logger.LogError(err, "Failed to get file from form", nil)
		utils.SendErrorResponse(w, http.StatusBadRequest, "File is required")
		return
	}
	defer file.Close()

	// Get optional config parameter
	config := r.FormValue("config")

	// Initialize semgrep service
	semgrepService := service.NewSemgrepService("", 10) // 10 concurrent scans max

	// Scan file - service handles everything else
	result, err := semgrepService.ScanFileFromRequest(r.Context(), file, fileHeader, config)
	if err != nil {
		c.logger.LogError(err, "Semgrep scan failed", map[string]interface{}{
			"filename": fileHeader.Filename,
		})
		utils.SendErrorResponse(w, http.StatusInternalServerError,
			fmt.Sprintf("Scan failed: %v", err))
		return
	}

	// Return results
	utils.SendSuccessResponse(w, result, "Scan completed successfully")
}
