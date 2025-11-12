package audit

import (
	"encoding/json"
	"net/http"

	"github.com/opsmx/ai-guardian-api/pkg/service"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type AuditsController interface {
	GetAuditReport(w http.ResponseWriter, r *http.Request)
}

type auditController struct {
	auditService service.AuditService
	logger       *utils.ErrorLogger
}

func NewAuditController() AuditsController {
	return &auditController{
		auditService: service.NewAuditService(),
		logger:       utils.NewErrorLogger("audit_controller"),
	}
}

// GetAuditReport retrieves users audit report
// @Summary Get users audit report
// @Description Returns the users audit report
// @Tags Audit
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 404 {object} map[string]string "No report found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/audit/report [get]
func (c *auditController) GetAuditReport(w http.ResponseWriter, r *http.Request) {
	auditReport, err := c.auditService.GetAuditReport(r.URL.Query().Get("from"))
	if err != nil {
		c.logger.LogError(err, "Failed to get audit report", nil)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    auditReport,
	})
}
