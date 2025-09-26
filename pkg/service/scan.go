package service

import (
	"context"
	"fmt"

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
	ProjectID   string `json:"projectId"`
	ProjectName string `json:"projectName"`
	Platform    string `json:"platform"`
	ScanID      string `json:"scanId"`
	ScanType    string `json:"scanType"`
}

type RescanResponse struct {
	Message string `json:"message"`
}

func (s *ScanService) ValidateRescanRequest(req *RescanRequest) error {
	if req.ProjectID == "" {
		return fmt.Errorf("projectId is required")
	}
	if req.ProjectName == "" {
		return fmt.Errorf("projectName is required")
	}
	if req.Platform == "" {
		return fmt.Errorf("platform is required")
	}
	if req.ScanID == "" {
		return fmt.Errorf("scanId is required")
	}
	if req.ScanType == "" {
		return fmt.Errorf("scanType is required")
	}
	return nil
}

func (s *ScanService) Rescan(ctx context.Context, req *RescanRequest) (*RescanResponse, error) {
	return s.ssdService.Rescan(ctx, req)
}
