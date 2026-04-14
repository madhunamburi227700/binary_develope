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

// GetNetworkMap returns an application/network map for an artifact.
// @Summary Get CSPM network map
// @Description Fetches the CSPM network map for a given artifact. Either `sha` or `name` must be provided.
// @Tags CSPM
// @Accept json
// @Produce json
// @Param sha query string false "Artifact SHA (alternative to name)"
// @Param name query string false "Artifact name (required if sha not provided)"
// @Param tag query string false "Artifact tag (optional when using name)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Missing required parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/cspm/networkmap [get]
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

// GetResources returns CSPM resources (paginated).
// @Summary List CSPM resources
// @Description Lists CSPM resources with optional filters and pagination.
// @Tags CSPM
// @Accept json
// @Produce json
// @Param id query string false "Resource ID"
// @Param cloudProvider query string false "Cloud provider (e.g. aws, azure, gcp)"
// @Param cloudAccountName query string false "Cloud account name"
// @Param resourceType query string false "Resource type"
// @Param name query string false "Resource name"
// @Param nameRegex query string false "Resource name regex"
// @Param hasFindings query bool false "Filter resources that have findings"
// @Param page query int false "Page number (default 1)"
// @Param perPage query int false "Results per page (default 100)"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/cspm/resources [get]
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

// GetAllResources returns CSPM resources without pagination (cached).
// @Summary List all CSPM resources (cached)
// @Description Lists all CSPM resources matching filters. Uses a cached backend call.
// @Tags CSPM
// @Accept json
// @Produce json
// @Param id query string false "Resource ID"
// @Param cloudProvider query string false "Cloud provider (e.g. aws, azure, gcp)"
// @Param cloudAccountName query string false "Cloud account name"
// @Param resourceType query string false "Resource type"
// @Param name query string false "Resource name"
// @Param nameRegex query string false "Resource name regex"
// @Param hasFindings query bool false "Filter resources that have findings"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/cspm/resources/all [get]
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

// GetResourcesSummary returns a summary for CSPM resources.
// @Summary Get CSPM resources summary
// @Description Returns an aggregated summary of resources for the given filters.
// @Tags CSPM
// @Accept json
// @Produce json
// @Param cloudProvider query string false "Cloud provider (e.g. aws, azure, gcp)"
// @Param cloudAccountName query string false "Cloud account name"
// @Param hasFindings query bool false "Filter resources that have findings"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/cspm/resources/summary [get]
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

// GetBlastRadius returns blast radius for a CSPM resource.
// @Summary Get CSPM resource blast radius
// @Description Returns blast radius (dependency graph) for a given resource ID.
// @Tags CSPM
// @Accept json
// @Produce json
// @Param id query string true "Resource ID"
// @Param maxDepth query int false "Maximum traversal depth (default 0)"
// @Param cloudProvider query string false "Cloud provider (e.g. aws, azure, gcp)"
// @Param cloudAccountName query string false "Cloud account name"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Missing required parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/cspm/resources/blast-radius [get]
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

// GetDeployments returns deployments for a given scan + commit.
// @Summary Get CSPM deployments
// @Description Returns deployments associated with the given `commitsha` and `scanid`.
// @Tags CSPM
// @Accept json
// @Produce json
// @Param commitsha query string true "Commit SHA"
// @Param scanid query string true "Scan ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Missing required parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/cspm/deployments [get]
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

// GetCSPMDashboard returns a CSPM dashboard summary for a scan.
// @Summary Get CSPM dashboard
// @Description Returns CSPM dashboard data for the given account and scan.
// @Tags CSPM
// @Accept json
// @Produce json
// @Param accountName query string true "Cloud account name"
// @Param scanId query string true "Scan ID"
// @Param accountType query string true "Account type (e.g. aws, azure, gcp)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Missing required parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/cspm/scan/dashboard [get]
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

// GetCSPMRulesStatusSummary returns rules status summary for a CSPM scan.
// @Summary Get CSPM rules status summary
// @Description Returns CSPM rules status summary for the provided account, scan, and service.
// @Tags CSPM
// @Accept json
// @Produce json
// @Param accountName query string true "Cloud account name"
// @Param scanId query string true "Scan ID"
// @Param accountType query string true "Account type (e.g. aws, azure, gcp)"
// @Param service query string true "Cloud service name"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Missing required parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/cspm/scan/rulesStatusSummary [get]
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

// GetCSPMPolicy returns a specific CSPM policy details.
// @Summary Get CSPM policy
// @Description Returns CSPM policy details for the given policy ID and scan context.
// @Tags CSPM
// @Accept json
// @Produce json
// @Param policyId path string true "Policy ID"
// @Param accountType query string true "Account type (e.g. aws, azure, gcp)"
// @Param accountName query string true "Cloud account name"
// @Param scanId query string true "Scan ID"
// @Param service query string true "Cloud service name"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Missing required parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/cspm/scan/policy/{policyId} [get]
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

// GetCSPMRegions returns regions for a CSPM policy.
// @Summary Get CSPM policy regions
// @Description Returns a list of regions for the given policy name and scan context.
// @Tags CSPM
// @Accept json
// @Produce json
// @Param policyName query string true "Policy name"
// @Param accountType query string true "Account type (e.g. aws, azure, gcp)"
// @Param accountName query string true "Cloud account name"
// @Param scanId query string true "Scan ID"
// @Param service query string true "Cloud service name"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Missing required parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/cspm/scan/regions [get]
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

// GetCSPMScanResult returns scan result for a CSPM scan input.
// @Summary Get CSPM scan result
// @Description Returns CSPM scan result for the provided file and cloud integration context.
// @Tags CSPM
// @Accept json
// @Produce json
// @Param fileName query string true "File name"
// @Param cloudServiceProvider query string true "Cloud service provider"
// @Param cloudAccountName query string true "Cloud account name"
// @Param scanOperation query string true "Scan operation (e.g. cspmscan)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Missing required parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/cspm/scan/scanResult [get]
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

// GetCloudSecurityIntegrationScan returns cloud security integration scan info.
// @Summary Get cloud integration scan info
// @Description Returns cloud security integration scan info (includes last scan ID) for the given team.
// @Tags CSPM
// @Accept json
// @Produce json
// @Param teamId query string true "Team ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Missing required parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/cspm/scan/cloudIntegration [get]
func (c *CSPMController) GetCloudSecurityIntegrationScan(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	teamId := q.Get("teamId")
	if teamId == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "teamId is required")
		return
	}

	result, err := c.cspmService.GetCloudSecurityIntegrationScan(r.Context(), teamId)
	if err != nil {
		c.logger.LogError(err, "failed to get cloud security integration scan", nil)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to get cloud security integration scan")
		return
	}

	utils.SendSuccessResponse(w, result, "Cloud security integration scan info fetched successfully")
}

// PostCSPMScan triggers a CSPM scan.
// @Summary Trigger a CSPM scan
// @Description Triggers a CSPM scan for the given hub and cloud account.
// @Tags CSPM
// @Accept json
// @Produce json
// @Param request body models.CSPMScanRequest true "CSPM scan request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Invalid request body or missing required fields"
// @Failure 502 {object} map[string]interface{} "Upstream error"
// @Security ApiKeyAuth
// @Router /api/v1/cspm/scan/trigger [post]
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
