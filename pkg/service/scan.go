package service

import (
	"context"
	"fmt"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type ScanService struct {
	ssdService *SSDService
	logger     *utils.ErrorLogger
}

func NewScanService() *ScanService {
	return &ScanService{
		logger: utils.NewErrorLogger("scan_service"),
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

	return s.ssdService.Rescan(ctx, req, scanResult)
}
