package service

import (
	"context"
	"fmt"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// CSPMService handles CSPM MCP operations using the client layer.
type CSPMService struct {
	cspmClient client.CspmMcpClient
	scanRepo   *repository.ScanRepository
	ssdClient  *client.SSDClient
	logger     *utils.ErrorLogger
}

// NewCSPMService creates a new CSPM service.
func NewCSPMService() *CSPMService {
	return &CSPMService{
		cspmClient: client.NewCspmMcpClient(),
		scanRepo:   repository.NewScanRepository(),
		ssdClient:  client.NewSSDClient(),
		logger:     utils.NewErrorLogger("cspm_service"),
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
