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
