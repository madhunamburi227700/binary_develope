package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/opsmx/ai-guardian-api/pkg/database"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
	"github.com/redis/go-redis/v9"
)

const (
	webhookRedisKeyPrefix     = "webhook:pr:"
	webhookScanKeyPrefix      = "webhook:scan:"
	webhookRedisTTL           = 24 * time.Hour
	webhookStatusCompleted    = "true"
	webhookStatusNotCompleted = "false"
	webhookStatusProcessed    = "true"
)

// WebhookScanPairRepository handles Redis operations for webhook scan pairs
type WebhookScanPairRepository struct {
	redis  *redis.Client
	logger *utils.ErrorLogger
}

// NewWebhookScanPairRepository creates a new webhook scan pair repository
func NewWebhookScanPairRepository() *WebhookScanPairRepository {
	return &WebhookScanPairRepository{
		redis:  database.GetRedis(),
		logger: utils.NewErrorLogger("webhook_scan_pair_repository"),
	}
}

// ProjectPairData represents the data stored in Redis for a project pair
type ProjectPairData struct {
	BaseProjectID string
	HeadProjectID string
	BaseBranch    string
	HeadBranch    string
	RepoURL       string
	PRNumber      string
	BaseScanID    string
	HeadScanID    string
	BaseCompleted bool
	HeadCompleted bool
	DiffProcessed bool
	CreatedAt     string
}

// StoreProjectPair stores project pair information when webhook is received
func (r *WebhookScanPairRepository) StoreProjectPair(ctx context.Context, prNumber, repoURL, baseProjectID, headProjectID, baseBranch, headBranch string) error {
	if prNumber == "" || baseProjectID == "" || headProjectID == "" {
		return fmt.Errorf("pr_number, base_project_id, and head_project_id are required")
	}

	key := r.getRedisKey(prNumber)

	data := map[string]interface{}{
		"base_project_id": baseProjectID,
		"head_project_id": headProjectID,
		"base_branch":     baseBranch,
		"head_branch":     headBranch,
		"repo_url":        repoURL,
		"pr_number":       prNumber,
		"base_scan_id":    "",
		"head_scan_id":    "",
		"base_completed":  webhookStatusNotCompleted,
		"head_completed":  webhookStatusNotCompleted,
		"diff_processed":  webhookStatusNotCompleted,
		"created_at":      time.Now().UTC().Format(time.RFC3339),
	}

	// Use pipeline for atomic operations
	pipe := r.redis.Pipeline()
	pipe.HSet(ctx, key, data)
	pipe.Expire(ctx, key, webhookRedisTTL)
	_, err := pipe.Exec(ctx)

	if err != nil {
		r.logger.LogError(err, "Failed to store project pair in Redis", map[string]interface{}{
			"pr_number":       prNumber,
			"base_project_id": baseProjectID,
			"head_project_id": headProjectID,
		})
		return fmt.Errorf("failed to store project pair: %w", err)
	}
	return nil
}

// StoreScanIDMapping stores a mapping from scan ID to PR number and updates the pair
func (r *WebhookScanPairRepository) StoreScanIDMapping(ctx context.Context, scanID, prNumber string, isBase bool) error {
	if scanID == "" || prNumber == "" {
		return fmt.Errorf("scan_id and pr_number are required")
	}

	// Store reverse mapping: scan_id -> pr_number
	scanKey := r.getScanRedisKey(scanID)
	err := r.redis.Set(ctx, scanKey, prNumber, webhookRedisTTL).Err()
	if err != nil {
		r.logger.LogError(err, "Failed to store scan ID mapping", map[string]interface{}{
			"scan_id":   scanID,
			"pr_number": prNumber,
		})
		return fmt.Errorf("failed to store scan ID mapping: %w", err)
	}

	// Update the PR pair with the scan ID
	prKey := r.getRedisKey(prNumber)
	fieldName := "head_scan_id"
	if isBase {
		fieldName = "base_scan_id"
	}

	err = r.redis.HSet(ctx, prKey, fieldName, scanID).Err()
	if err != nil {
		r.logger.LogError(err, "Failed to update scan ID in pair", map[string]interface{}{
			"scan_id":   scanID,
			"pr_number": prNumber,
			"field":     fieldName,
		})
		return fmt.Errorf("failed to update scan ID in pair: %w", err)
	}

	return nil
}

// GetPRNumberFromScanID retrieves the PR number for a given scan ID
func (r *WebhookScanPairRepository) GetPRNumberFromScanID(ctx context.Context, scanID string) (string, error) {
	if scanID == "" {
		return "", fmt.Errorf("scan_id is required")
	}

	scanKey := r.getScanRedisKey(scanID)
	prNumber, err := r.redis.Get(ctx, scanKey).Result()
	if err == redis.Nil {
		return "", nil // Not a webhook-triggered scan
	}
	if err != nil {
		return "", fmt.Errorf("failed to get PR number from scan ID: %w", err)
	}

	return prNumber, nil
}

// CheckAndUpdateProjectCompletion checks if a scan ID is part of a pair and updates completion status
// Returns: (isPairComplete, pairData, error)
// isPairComplete: true if both projects are completed and diff not yet processed
// pairData: contains all pair information if found
func (r *WebhookScanPairRepository) CheckAndUpdateProjectCompletion(ctx context.Context, scanID string) (bool, *ProjectPairData, error) {
	if scanID == "" {
		return false, nil, fmt.Errorf("scan_id is required")
	}

	// Get PR number from scan ID
	prNumber, err := r.GetPRNumberFromScanID(ctx, scanID)
	if err != nil {
		return false, nil, err
	}
	if prNumber == "" {
		// Not a webhook-triggered scan
		return false, nil, nil
	}

	// Get the PR pair data
	key := r.getRedisKey(prNumber)
	pairData, err := r.getPairData(ctx, key)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get pair data: %w", err)
	}

	// Determine which field to update based on scan ID
	var fieldToUpdate string
	if scanID == pairData.BaseScanID {
		fieldToUpdate = "base_completed"
	} else if scanID == pairData.HeadScanID {
		fieldToUpdate = "head_completed"
	} else {
		return false, nil, fmt.Errorf("scan ID %s doesn't match pair for PR %s", scanID, prNumber)
	}

	// Update Redis atomically
	err = r.redis.HSet(ctx, key, fieldToUpdate, webhookStatusCompleted).Err()
	if err != nil {
		return false, nil, fmt.Errorf("failed to update project completion: %w", err)
	}

	// Use a pipeline to atomically get both completion statuses
	pipe := r.redis.Pipeline()
	baseCompletedCmd := pipe.HGet(ctx, key, "base_completed")
	headCompletedCmd := pipe.HGet(ctx, key, "head_completed")
	diffProcessedCmd := pipe.HGet(ctx, key, "diff_processed")
	_, err = pipe.Exec(ctx)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get completion status: %w", err)
	}

	// Parse the values (HGet returns empty string if field doesn't exist)
	baseCompleted := baseCompletedCmd.Val() == webhookStatusCompleted
	headCompleted := headCompletedCmd.Val() == webhookStatusCompleted
	diffProcessed := diffProcessedCmd.Val() == webhookStatusProcessed

	// Re-read full pair data for return value
	updatedPairData, err := r.getPairData(ctx, key)
	if err != nil {
		return false, nil, fmt.Errorf("failed to re-read pair data after update: %w", err)
	}

	// Update the struct with the atomic values we just read
	updatedPairData.BaseCompleted = baseCompleted
	updatedPairData.HeadCompleted = headCompleted
	updatedPairData.DiffProcessed = diffProcessed

	// Check if both projects are completed and diff not yet processed
	if updatedPairData.BaseCompleted && updatedPairData.HeadCompleted && !updatedPairData.DiffProcessed {
		return true, updatedPairData, nil
	}

	// One completed, waiting for the other
	r.logger.LogInfo("One project completed", nil)

	return false, updatedPairData, nil
}

// MarkDiffProcessed marks the diff as processed to prevent duplicate processing
func (r *WebhookScanPairRepository) MarkDiffProcessed(ctx context.Context, prNumber string) error {
	if prNumber == "" {
		return fmt.Errorf("pr_number is required")
	}

	key := r.getRedisKey(prNumber)
	err := r.redis.HSet(ctx, key, "diff_processed", webhookStatusProcessed).Err()
	if err != nil {
		r.logger.LogError(err, "Failed to mark diff as processed", map[string]interface{}{
			"pr_number": prNumber,
		})
		return fmt.Errorf("failed to mark diff as processed: %w", err)
	}
	return nil
}

// getPairData retrieves and parses pair data from Redis
func (r *WebhookScanPairRepository) getPairData(ctx context.Context, key string) (*ProjectPairData, error) {
	data, err := r.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get pair data: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("pair data not found for key: %s", key)
	}

	pairData := &ProjectPairData{
		BaseProjectID: data["base_project_id"],
		HeadProjectID: data["head_project_id"],
		BaseBranch:    data["base_branch"],
		HeadBranch:    data["head_branch"],
		RepoURL:       data["repo_url"],
		PRNumber:      data["pr_number"],
		BaseScanID:    data["base_scan_id"],
		HeadScanID:    data["head_scan_id"],
		CreatedAt:     data["created_at"],
	}

	// Parse boolean fields
	pairData.BaseCompleted = data["base_completed"] == webhookStatusCompleted
	pairData.HeadCompleted = data["head_completed"] == webhookStatusCompleted
	pairData.DiffProcessed = data["diff_processed"] == webhookStatusProcessed

	return pairData, nil
}

// getRedisKey generates the Redis key for a PR number
func (r *WebhookScanPairRepository) getRedisKey(prNumber string) string {
	return webhookRedisKeyPrefix + prNumber
}

// getScanRedisKey generates the Redis key for a scan ID
func (r *WebhookScanPairRepository) getScanRedisKey(scanID string) string {
	return webhookScanKeyPrefix + scanID
}

// GetPRNumberFromKey extracts PR number from Redis key
func (r *WebhookScanPairRepository) GetPRNumberFromKey(key string) string {
	// Extract PR number from key "webhook:pr:123" -> "123"
	if strings.HasPrefix(key, webhookRedisKeyPrefix) {
		return strings.TrimPrefix(key, webhookRedisKeyPrefix)
	}
	return ""
}

// DeleteProjectPair deletes a project pair from Redis (used when scan fails)
func (r *WebhookScanPairRepository) DeleteProjectPair(ctx context.Context, prNumber string) error {
	if prNumber == "" {
		return fmt.Errorf("pr_number is required")
	}

	key := r.getRedisKey(prNumber)

	// Get pair data to delete scan ID mappings
	pairData, err := r.getPairData(ctx, key)
	if err == nil {
		// Delete scan ID mappings
		if pairData.BaseScanID != "" {
			r.redis.Del(ctx, r.getScanRedisKey(pairData.BaseScanID))
		}
		if pairData.HeadScanID != "" {
			r.redis.Del(ctx, r.getScanRedisKey(pairData.HeadScanID))
		}
	}

	err = r.redis.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete project pair: %w", err)
	}

	return nil
}

// DeleteProjectPairByScanID finds and deletes a project pair by scan ID
func (r *WebhookScanPairRepository) DeleteProjectPairByScanID(ctx context.Context, scanID string) error {
	if scanID == "" {
		return fmt.Errorf("scan_id is required")
	}

	prNumber, err := r.GetPRNumberFromScanID(ctx, scanID)
	if err != nil {
		return fmt.Errorf("failed to get PR number from scan ID: %w", err)
	}
	if prNumber == "" {
		// Not a webhook-triggered scan
		return nil
	}

	return r.DeleteProjectPair(ctx, prNumber)
}
