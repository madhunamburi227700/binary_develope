package cspm

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/models"
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

// GET /api/v1/cspm/resources/all?id=...&cloudProvider=...&...
func (c *CSPMController) GetAllResources(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var hasFindings *bool
	if v := q.Get("hasFindings"); v != "" {
		b := v == "true"
		hasFindings = &b
	}

	params := client.GetCSPMResourcesParams{
		ID:               q.Get("id"),
		CloudProvider:    q.Get("cloudProvider"),
		CloudAccountName: q.Get("cloudAccountName"),
		ResourceType:     q.Get("resourceType"),
		Name:             q.Get("name"),
		NameRegex:        q.Get("nameRegex"),
		HasFindings:      hasFindings,
	}

	result, err := c.cspmService.GetAllResourcesCached(r.Context(), params)
	if err != nil {
		c.logger.LogError(err, "failed to get all CSPM resources", nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to get all CSPM resources")
		return
	}

	utils.SendSuccessResponse(w, result, "All CSPM resources fetched successfully")
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

// GET /api/v1/cspm/dashboard?accountName=...&scanId=...&accountType=...
func (c *CSPMController) GetCSPMDashboard(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	accountName := q.Get("accountName")
	scanID := q.Get("scanId")
	accountType := q.Get("accountType")
	if accountName == "" || scanID == "" || accountType == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "accountName, scanId, and accountType are required")
		return
	}

	result, err := c.cspmService.GetCSPMDashboard(r.Context(), accountName, scanID, accountType)
	if err != nil {
		c.logger.LogError(err, "failed to get CSPM dashboard", nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to get CSPM dashboard")
		return
	}

	utils.SendSuccessResponse(w, result, "CSPM dashboard fetched successfully")
}

// GET /api/v1/cspm/rulesStatusSummary?accountName=...&scanId=...&accountType=...&service=...
func (c *CSPMController) GetCSPMRulesStatusSummary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	accountName := q.Get("accountName")
	scanID := q.Get("scanId")
	accountType := q.Get("accountType")
	service := q.Get("service")
	if accountName == "" || scanID == "" || accountType == "" || service == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "accountName, scanId, accountType, and service are required")
		return
	}

	result, err := c.cspmService.GetCSPMRulesStatusSummary(r.Context(), accountName, scanID, accountType, service)
	if err != nil {
		c.logger.LogError(err, "failed to get CSPM rules status summary", nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to get CSPM rules status summary")
		return
	}

	utils.SendSuccessResponse(w, result, "CSPM rules status summary fetched successfully")
}

// GET /api/v1/cspm/policy/{policyId}?accountType=...&accountName=...&orgId=...&scanId=...&service=...
func (c *CSPMController) GetCSPMPolicy(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	vars := mux.Vars(r)
	policyID := vars["policyId"]
	accountType := q.Get("accountType")
	accountName := q.Get("accountName")
	scanID := q.Get("scanId")
	serviceName := q.Get("service")

	if policyID == "" || accountType == "" || accountName == "" || scanID == "" || serviceName == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "policyId, accountType, accountName, scanId, and service are required")
		return
	}

	result, err := c.cspmService.GetCSPMPolicy(r.Context(), policyID, accountType, accountName, scanID, serviceName)
	if err != nil {
		c.logger.LogError(err, "failed to get CSPM policy", nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to get CSPM policy")
		return
	}

	utils.SendSuccessResponse(w, result, "CSPM policy fetched successfully")
}

// GET /api/v1/cspm/regions?policyName=...&accountType=...&accountName=...&orgId=...&scanId=...&service=...
func (c *CSPMController) GetCSPMRegions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	policyName := q.Get("policyName")
	accountType := q.Get("accountType")
	accountName := q.Get("accountName")
	scanID := q.Get("scanId")
	serviceName := q.Get("service")

	if policyName == "" || accountType == "" || accountName == "" || scanID == "" || serviceName == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "policyName, accountType, accountName, scanId, and service are required")
		return
	}

	result, err := c.cspmService.GetCSPMRegions(r.Context(), policyName, accountType, accountName, scanID, serviceName)
	if err != nil {
		c.logger.LogError(err, "failed to get CSPM regions", nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to get CSPM regions")
		return
	}

	utils.SendSuccessResponse(w, result, "CSPM regions fetched successfully")
}

// GET /api/v1/cspm/scanResult?fileName=...&cloudServiceProvider=...&cloudAccountName=...&scanOperation=cspmscan
func (c *CSPMController) GetCSPMScanResult(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	fileName := q.Get("fileName")
	cloudServiceProvider := q.Get("cloudServiceProvider")
	cloudAccountName := q.Get("cloudAccountName")
	scanOperation := q.Get("scanOperation")

	if fileName == "" || cloudServiceProvider == "" || cloudAccountName == "" || scanOperation == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "fileName, cloudServiceProvider, cloudAccountName, and scanOperation are required")
		return
	}

	result, err := c.cspmService.GetCSPMScanResult(r.Context(), fileName, cloudServiceProvider, cloudAccountName, scanOperation)
	if err != nil {
		c.logger.LogError(err, "failed to get CSPM scanResult", nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to get CSPM scanResult")
		return
	}

	utils.SendSuccessResponse(w, result, "CSPM scanResult fetched successfully")
}

// GET /api/v1/cspm/scan/cloudIntegration?name=...&type=aws&orgId=optional
// Fetches SSD cloudSecurityIntegration list and returns only rows matching name and type (includes lastScanId).
func (c *CSPMController) GetCloudSecurityIntegrationScan(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	name := q.Get("name")
	accountType := q.Get("type")
	if name == "" || accountType == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "name and type are required")
		return
	}

	result, err := c.cspmService.GetCloudSecurityIntegrationScan(r.Context(), name, accountType)
	if err != nil {
		c.logger.LogError(err, "failed to get cloud security integration scan", nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to get cloud security integration scan")
		return
	}

	utils.SendSuccessResponse(w, result, "Cloud security integration scan info fetched successfully")
}

// POST /api/v1/cspm/scan/trigger — proxies POST /gate/ssd-opa/api/v1/cspmscan; forwards SSD status code and JSON body (e.g. 201 success, 400 validation).
func (c *CSPMController) PostCSPMScan(w http.ResponseWriter, r *http.Request) {
	var req models.CSPMScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.HubName == "" || req.CloudServiceProvider == "" || req.CloudAccountName == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "hubName, cloudServiceProvider, and cloudAccountName are required")
		return
	}

	resp, err := c.cspmService.TriggerCSPMScan(r.Context(), &req)
	if err != nil {
		c.logger.LogError(err, "failed to trigger CSPM scan", nil)
		utils.SendErrorResponse(w, http.StatusBadGateway, "failed to trigger CSPM scan")
		return
	}
	utils.SendSuccessResponse(w, resp, "CSPM scan triggered successfully")
}
