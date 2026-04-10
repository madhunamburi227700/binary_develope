package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/config"
	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// CSPMService handles CSPM MCP operations using the client layer.
type CSPMService struct {
	cspmClient client.CspmMcpClient
	scanRepo   *repository.ScanRepository
	ssdClient  *client.SSDClient
	logger     *utils.ErrorLogger
	cacheMu    sync.RWMutex
	cache      map[string]cspmResourcesCacheEntry
}

type cspmResourcesCacheEntry struct {
	response  *client.GetCSPMResourcesResponse
	expiresAt time.Time
}

const (
	cspmAllResourcesPerPage         = 1000
	cspmResourcesCacheSweepInterval = 2 * time.Minute // periodic removal of expired cache keys
)

func (s *CSPMService) cspmResourcesCacheGet(key string) (cspmResourcesCacheEntry, bool) {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	e, ok := s.cache[key]
	return e, ok
}

func (s *CSPMService) cspmResourcesCacheSet(key string, entry cspmResourcesCacheEntry) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cache[key] = entry
}

func (s *CSPMService) cspmResourcesCacheDelete(key string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	delete(s.cache, key)
}

// NewCSPMService creates a new CSPM service.
func NewCSPMService() *CSPMService {
	s := &CSPMService{
		cspmClient: client.NewCspmMcpClient(),
		scanRepo:   repository.NewScanRepository(),
		ssdClient:  client.NewSSDClient(),
		logger:     utils.NewErrorLogger("cspm_service"),
		cache:      make(map[string]cspmResourcesCacheEntry),
	}
	go s.runCSPMResourcesCacheSweeper()
	return s
}

func (s *CSPMService) runCSPMResourcesCacheSweeper() {
	ticker := time.NewTicker(cspmResourcesCacheSweepInterval)
	defer ticker.Stop()
	for range ticker.C {
		s.sweepExpiredCSPMResourcesCache()
	}
}

func (s *CSPMService) sweepExpiredCSPMResourcesCache() {
	now := time.Now()
	s.cacheMu.Lock()
	removed := 0
	for key, entry := range s.cache {
		if !now.Before(entry.expiresAt) {
			delete(s.cache, key)
			removed++
		}
	}
	remaining := len(s.cache)
	s.cacheMu.Unlock()

	if removed > 0 {
		s.logger.LogInfo("CSPM resources cache sweep removed expired entries", map[string]interface{}{
			"removed":   removed,
			"remaining": remaining,
		})
	}
}

func (s *CSPMService) GetNetworkMap(ctx context.Context, params client.GetNetworkMapParams) (*client.NetworkMapResponse, error) {
	result, err := s.cspmClient.GetNetworkMap(ctx, params)
	if err != nil {
		s.logger.LogError(err, "failed to get network map", map[string]interface{}{
			"name": params.Name,
			"tag":  params.Tag,
			"sha":  params.Sha,
		})
		return nil, fmt.Errorf("failed to get network map: %w", err)
	}
	return result, nil
}

func (s *CSPMService) GetResources(ctx context.Context, params client.GetCSPMResourcesParams) (*client.GetCSPMResourcesResponse, error) {
	result, err := s.cspmClient.GetCSPMResources(ctx, params)
	if err != nil {
		s.logger.LogError(err, "failed to get CSPM resources", nil)
		return nil, fmt.Errorf("failed to get CSPM resources: %w", err)
	}
	return result, nil
}

func (s *CSPMService) GetAllResourcesCached(ctx context.Context, params client.GetCSPMResourcesParams) (*client.GetCSPMResourcesResponse, error) {
	params.Page = 0
	params.PerPage = 0

	cacheKey := s.cspmResourcesCacheKey(params)
	now := time.Now()

	if entry, ok := s.cspmResourcesCacheGet(cacheKey); ok {
		if now.Before(entry.expiresAt) {
			return entry.response, nil
		}
		s.cspmResourcesCacheDelete(cacheKey)
	}

	baseParams := params
	baseParams.Page = 1
	baseParams.PerPage = cspmAllResourcesPerPage

	firstPage, err := s.cspmClient.GetCSPMResources(ctx, baseParams)
	if err != nil {
		s.logger.LogError(err, "failed to get CSPM resources first page", nil)
		return nil, fmt.Errorf("failed to get CSPM resources first page: %w", err)
	}

	mergedGroups := make(map[string]*client.CSPMResourceGroup)
	groupOrder := make([]string, 0)
	mergePageGroups := func(groups []client.CSPMResourceGroup) {
		for _, g := range groups {
			key := g.CloudProvider + "|" + g.CloudAccountName
			existing, found := mergedGroups[key]
			if !found {
				groupCopy := client.CSPMResourceGroup{
					CloudProvider:    g.CloudProvider,
					CloudAccountName: g.CloudAccountName,
					Resources:        append([]client.CSPMResource{}, g.Resources...),
				}
				mergedGroups[key] = &groupCopy
				groupOrder = append(groupOrder, key)
				continue
			}

			existing.Resources = append(existing.Resources, g.Resources...)
		}
	}

	mergePageGroups(firstPage.Data)
	totalPages := firstPage.PageInfo.TotalPages

	if totalPages < 1 {
		if len(firstPage.Data) > 0 {
			totalPages = 1
		} else {
			totalPages = 0
		}
	}

	for page := 2; page <= totalPages; page++ {
		pageParams := baseParams
		pageParams.Page = page

		pageResult, pageErr := s.cspmClient.GetCSPMResources(ctx, pageParams)
		if pageErr != nil {
			s.logger.LogError(pageErr, "failed to get CSPM resources page", map[string]interface{}{
				"page": page,
			})
			return nil, fmt.Errorf("failed to get CSPM resources page %d: %w", page, pageErr)
		}

		mergePageGroups(pageResult.Data)
	}

	allData := make([]client.CSPMResourceGroup, 0, len(groupOrder))
	for _, key := range groupOrder {
		allData = append(allData, *mergedGroups[key])
	}

	totalItems := firstPage.PageInfo.TotalItems
	if totalItems == 0 {
		totalItems = 0
		for _, group := range allData {
			totalItems += len(group.Resources)
		}
	}

	aggregated := &client.GetCSPMResourcesResponse{
		Data: allData,
		PageInfo: client.PageInfo{
			Page:       1,
			PerPage:    totalItems,
			TotalItems: totalItems,
			TotalPages: 1,
		},
	}

	s.cspmResourcesCacheSet(cacheKey, cspmResourcesCacheEntry{
		response:  aggregated,
		expiresAt: now.Add(time.Duration(config.GetCSPMStaticResource()) * time.Hour),
	})

	return aggregated, nil
}

func (s *CSPMService) cspmResourcesCacheKey(params client.GetCSPMResourcesParams) string {
	var hasFindings string
	if params.HasFindings == nil {
		hasFindings = "nil"
	} else if *params.HasFindings {
		hasFindings = "true"
	} else {
		hasFindings = "false"
	}

	parts := []string{
		params.ID,
		params.CloudProvider,
		params.CloudAccountName,
		params.ResourceType,
		params.Name,
		params.NameRegex,
		hasFindings,
	}

	return strings.Join(parts, "|")
}

func (s *CSPMService) GetResourcesSummary(ctx context.Context, params client.GetCSPMResourcesSummaryParams) (*client.GetCSPMResourcesSummaryResponse, error) {
	result, err := s.cspmClient.GetCSPMResourcesSummary(ctx, params)
	if err != nil {
		s.logger.LogError(err, "failed to get CSPM resources summary", nil)
		return nil, fmt.Errorf("failed to get CSPM resources summary: %w", err)
	}
	return result, nil
}

func (s *CSPMService) GetBlastRadius(ctx context.Context, params client.GetCSPMResourceBlastRadiusParams) (*client.BlastRadiusResponse, error) {
	result, err := s.cspmClient.GetCSPMResourceBlastRadius(ctx, params)
	if err != nil {
		s.logger.LogError(err, "failed to get CSPM blast radius", map[string]interface{}{
			"id": params.ID,
		})
		return nil, fmt.Errorf("failed to get CSPM blast radius: %w", err)
	}
	return result, nil
}

func (s *CSPMService) GetDeployments(ctx context.Context, commitsha, scanid string) (interface{}, error) {
	scan, err := s.scanRepo.GetScanWithProjectByScanID(ctx, scanid)
	if err != nil {
		return nil, fmt.Errorf("failed to get scan with project: %w", err)
	}

	githubUrl := fmt.Sprintf("https://github.com/%s/%s", scan.Organisation, scan.Repository)

	artifactResponse, err := s.ssdClient.GetArtifact(ctx, commitsha, githubUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	if len(artifactResponse) == 0 {
		return nil, fmt.Errorf("no artifact found")
	}

	artifatcSha := artifactResponse[0].ArtifactNodes.ArtifactSha
	deployments, err := s.cspmClient.GetNetworkMap(ctx, client.GetNetworkMapParams{
		Sha: artifatcSha,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get network map: %w", err)
	}

	return deployments, nil
}

// GetCSPMDashboard returns per-service CSPM posture summary for a cloud account scan (proxied from SSD).
func (s *CSPMService) GetCSPMDashboard(ctx context.Context, accountName, scanID, accountType string) ([]client.CSPMDashboardServiceRow, error) {
	result, err := s.ssdClient.GetCSPMDashboard(ctx, accountName, scanID, accountType)
	if err != nil {
		s.logger.LogError(err, "failed to get CSPM dashboard", map[string]interface{}{
			"accountName": accountName, "scanId": scanID, "accountType": accountType,
		})
		return nil, fmt.Errorf("failed to get CSPM dashboard: %w", err)
	}
	return result, nil
}

// GetCSPMRulesStatusSummary returns rule-level findings for one cloud service in a scan (proxied from SSD).
func (s *CSPMService) GetCSPMRulesStatusSummary(ctx context.Context, accountName, scanID, accountType, service string) (*client.CSPMRulesStatusSummaryResponse, error) {
	result, err := s.ssdClient.GetCSPMRulesStatusSummary(ctx, accountName, scanID, accountType, service)
	if err != nil {
		s.logger.LogError(err, "failed to get CSPM rules status summary", map[string]interface{}{
			"accountName": accountName, "scanId": scanID, "accountType": accountType, "service": service,
		})
		return nil, fmt.Errorf("failed to get CSPM rules status summary: %w", err)
	}
	return result, nil
}

func (s *CSPMService) GetCSPMPolicy(ctx context.Context, policyID, accountType, accountName, scanID, service string) ([]client.CSPMPolicyAffectedResource, error) {
	result, err := s.ssdClient.GetCSPMPolicy(ctx, policyID, "", accountType, accountName, scanID, service)
	if err != nil {
		s.logger.LogError(err, "failed to get CSPM policy", map[string]interface{}{
			"policyId": policyID, "accountName": accountName, "scanId": scanID, "accountType": accountType, "service": service,
		})
		return nil, fmt.Errorf("failed to get CSPM policy: %w", err)
	}
	return result, nil
}

func (s *CSPMService) GetCSPMRegions(ctx context.Context, policyName, accountType, accountName, scanID, service string) (*client.CSPMRegionsResponse, error) {
	result, err := s.ssdClient.GetCSPMRegions(ctx, policyName, accountType, accountName, scanID, service)
	if err != nil {
		s.logger.LogError(err, "failed to get CSPM regions", map[string]interface{}{
			"policyName": policyName, "accountName": accountName, "scanId": scanID, "accountType": accountType, "service": service,
		})
		return nil, fmt.Errorf("failed to get CSPM regions: %w", err)
	}
	return result, nil
}

func (s *CSPMService) GetCSPMScanResult(ctx context.Context, fileName, cloudServiceProvider, cloudAccountName, scanOperation string) (map[string]interface{}, error) {
	result, err := s.ssdClient.GetCSPMScanResult(ctx, fileName, cloudServiceProvider, cloudAccountName, scanOperation)
	if err != nil {
		s.logger.LogError(err, "failed to get CSPM scanResult", map[string]interface{}{
			"fileName": fileName, "cloudServiceProvider": cloudServiceProvider, "cloudAccountName": cloudAccountName, "scanOperation": scanOperation,
		})
		return nil, fmt.Errorf("failed to get CSPM scanResult: %w", err)
	}
	return result, nil
}

// GetCloudSecurityIntegrationScan returns cloud integration rows from SSD filtered by account name and type (e.g. aws), including lastScanId.
func (s *CSPMService) GetCloudSecurityIntegrationScan(ctx context.Context, name, accountType string) ([]client.CSPMCloudSecurityIntegration, error) {
	all, err := s.ssdClient.GetCloudSecurityIntegrations(ctx)
	if err != nil {
		s.logger.LogError(err, "failed to get cloud security integrations", map[string]interface{}{
			"name": name, "type": accountType,
		})
		return nil, fmt.Errorf("failed to get cloud security integrations: %w", err)
	}
	wantName := strings.TrimSpace(name)
	wantType := strings.TrimSpace(accountType)
	var filtered []client.CSPMCloudSecurityIntegration
	for _, row := range all {
		if strings.TrimSpace(row.Name) == wantName && strings.TrimSpace(row.Type) == wantType {
			filtered = append(filtered, row)
		}
	}
	return filtered, nil
}

func (s *CSPMService) TriggerCSPMScan(ctx context.Context, req *models.CSPMScanRequest) (*client.Response, error) {
	// TO DO, handle organization name from the request in case of multiple organizations
	newReq := &client.CSPMScanRequestBody{
		OrganizationName:     "opsmx",
		TeamName:             req.HubName,
		CloudServiceProvider: req.CloudServiceProvider,
		CloudAccountName:     req.CloudAccountName,
	}

	resp, err := s.ssdClient.PostCSPMScan(ctx, newReq)
	if err != nil {
		s.logger.LogError(err, "failed to trigger CSPM scan", nil)
		return nil, fmt.Errorf("failed to trigger CSPM scan: %w", err)
	}
	return resp, nil
}
