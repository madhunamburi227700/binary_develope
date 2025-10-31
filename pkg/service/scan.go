package service

import (
	"context"
	"fmt"

	"github.com/opsmx/ai-guardian-api/pkg/client"
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

func (s *ScanService) Rescan(ctx context.Context, req *RescanRequest) (string, error) {
	scanResult, err := s.ssdService.GetScanResultData(ctx, &client.ScanResultDataRequest{
		ProjectID:  req.ProjectID,
		TeamID:     req.HubID,
		Repository: req.Repository,
		Type:       "sourceScan",
		Branch:     req.Branch,
	})
	if err != nil {
		return "", err
	}

	if scanResult.ScanID == "" {
		return "", fmt.Errorf("scan result not found")
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
		return "", err
	}

	return s.ssdService.Rescan(ctx, req, scanResult)
}
