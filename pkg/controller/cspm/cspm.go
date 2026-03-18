package cspm

import (
	"net/http"
	"strconv"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/service"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// CSPMController handles CSPM HTTP requests.
type CSPMController struct {
	cspmService *service.CSPMService
	logger      *utils.ErrorLogger
}

// NewCSPMController creates a new CSPM controller.
func NewCSPMController() *CSPMController {
	return &CSPMController{
		cspmService: service.NewCSPMService(),
		logger:      utils.NewErrorLogger("cspm_controller"),
	}
}

// GET /api/v1/cspm/networkmap?name=...&tag=... or ?sha=...
func (c *CSPMController) GetNetworkMap(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	params := client.GetNetworkMapParams{
		Name: q.Get("name"),
		Tag:  q.Get("tag"),
		Sha:  q.Get("sha"),
	}

	// Basic validation: must have either sha or name.
	if params.Sha == "" && params.Name == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "either 'sha' or 'name' must be provided")
		return
	}

	result, err := c.cspmService.GetNetworkMap(r.Context(), params)
	if err != nil {
		c.logger.LogError(err, "failed to get network map", nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to get network map")
		return
	}

	utils.SendSuccessResponse(w, result, "Network map fetched successfully")
}

// GET /api/v1/cspm/resources?id=...&cloudProvider=...&...
func (c *CSPMController) GetResources(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var hasFindings *bool
	if v := q.Get("hasFindings"); v != "" {
		b := v == "true"
		hasFindings = &b
	}

	page := utils.StringToInt(q.Get("page"), 1)
	perPage := utils.StringToInt(q.Get("perPage"), 100)
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 100
	}

	params := client.GetCSPMResourcesParams{
		ID:               q.Get("id"),
		CloudProvider:    q.Get("cloudProvider"),
		CloudAccountName: q.Get("cloudAccountName"),
		ResourceType:     q.Get("resourceType"),
		Name:             q.Get("name"),
		NameRegex:        q.Get("nameRegex"),
		HasFindings:      hasFindings,
		Page:             page,
		PerPage:          perPage,
	}

	result, err := c.cspmService.GetResources(r.Context(), params)
	if err != nil {
		c.logger.LogError(err, "failed to get CSPM resources", nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to get CSPM resources")
		return
	}

	utils.SendSuccessResponse(w, result, "CSPM resources fetched successfully")
}

// GET /api/v1/cspm/resources/summary?cloudProvider=...&cloudAccountName=...&hasFindings=...
func (c *CSPMController) GetResourcesSummary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var hasFindings *bool
	if v := q.Get("hasFindings"); v != "" {
		b := v == "true"
		hasFindings = &b
	}

	params := client.GetCSPMResourcesSummaryParams{
		CloudProvider:    q.Get("cloudProvider"),
		CloudAccountName: q.Get("cloudAccountName"),
		HasFindings:      hasFindings,
	}

	result, err := c.cspmService.GetResourcesSummary(r.Context(), params)
	if err != nil {
		c.logger.LogError(err, "failed to get CSPM resources summary", nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to get CSPM resources summary")
		return
	}

	utils.SendSuccessResponse(w, result, "CSPM resources summary fetched successfully")
}

// GET /api/v1/cspm/resources/blast-radius?id=...&maxDepth=...&cloudProvider=...&cloudAccountName=...
func (c *CSPMController) GetBlastRadius(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	maxDepth := 0
	if v := q.Get("maxDepth"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxDepth = n
		}
	}

	params := client.GetCSPMResourceBlastRadiusParams{
		ID:               q.Get("id"),
		MaxDepth:         maxDepth,
		CloudProvider:    q.Get("cloudProvider"),
		CloudAccountName: q.Get("cloudAccountName"),
	}

	if params.ID == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "'id' is required")
		return
	}

	result, err := c.cspmService.GetBlastRadius(r.Context(), params)
	if err != nil {
		c.logger.LogError(err, "failed to get CSPM blast radius", nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to get CSPM blast radius")
		return
	}

	utils.SendSuccessResponse(w, result, "CSPM blast radius fetched successfully")
}

// GET /api/v1/cspm/deployments?commitsha=...&scanid=...
func (c *CSPMController) GetDeployments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	commitsha := q.Get("commitsha")
	scanid := q.Get("scanid")

	if commitsha == "" || scanid == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "commitsha and scanid are required")
		return
	}

	result, err := c.cspmService.GetDeployments(r.Context(), commitsha, scanid)
	if err != nil {
		c.logger.LogError(err, "failed to get CSPM deployments", nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to get CSPM deployments")
		return
	}

	utils.SendSuccessResponse(w, result, "CSPM deployments fetched successfully")
}
