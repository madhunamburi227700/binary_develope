package service

import (
	"context"

	"github.com/opsmx/ai-guardian-api/pkg/repository"
)

// WebhookScanPairService provides a service layer wrapper around the repository
type WebhookScanPairService struct {
	repo *repository.WebhookScanPairRepository
}

// NewWebhookScanPairService creates a new webhook scan pair service
func NewWebhookScanPairService() *WebhookScanPairService {
	return &WebhookScanPairService{
		repo: repository.NewWebhookScanPairRepository(),
	}
}

// StoreProjectPair stores project pair information when webhook is received
func (s *WebhookScanPairService) StoreProjectPair(ctx context.Context, prNumber, repoURL, baseProjectID, headProjectID, baseBranch, headBranch string) error {
	return s.repo.StoreProjectPair(ctx, prNumber, repoURL, baseProjectID, headProjectID, baseBranch, headBranch)
}

// CheckAndUpdateProjectCompletion checks if a project ID is part of a pair and updates completion status
func (s *WebhookScanPairService) CheckAndUpdateProjectCompletion(ctx context.Context, projectID string) (bool, *repository.ProjectPairData, error) {
	return s.repo.CheckAndUpdateProjectCompletion(ctx, projectID)
}

// MarkDiffProcessed marks the diff as processed
func (s *WebhookScanPairService) MarkDiffProcessed(ctx context.Context, prNumber string) error {
	return s.repo.MarkDiffProcessed(ctx, prNumber)
}
