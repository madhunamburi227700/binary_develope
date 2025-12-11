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

	r.logger.LogInfo("Stored project pair in Redis", map[string]interface{}{
		"pr_number":       prNumber,
		"base_project_id": baseProjectID,
		"head_project_id": headProjectID,
		"key":             key,
	})

	return nil
}

// CheckAndUpdateProjectCompletion checks if a project ID is part of a pair and updates completion status
// Returns: (isPairComplete, pairData, error)
// isPairComplete: true if both projects are completed and diff not yet processed
// pairData: contains all pair information if found
func (r *WebhookScanPairRepository) CheckAndUpdateProjectCompletion(ctx context.Context, projectID string) (bool, *ProjectPairData, error) {
	if projectID == "" {
		return false, nil, fmt.Errorf("project_id is required")
	}

	// Find all webhook pairs
	keys, err := r.redis.Keys(ctx, webhookRedisKeyPrefix+"*").Result()
	if err != nil {
		return false, nil, fmt.Errorf("failed to find project pairs: %w", err)
	}

	if len(keys) == 0 {
		// No webhook pairs found, this is not a webhook-triggered scan
		return false, nil, nil
	}

	// Iterate through keys to find matching project ID
	for _, key := range keys {
		pairData, err := r.getPairData(ctx, key)
		if err != nil {
			r.logger.LogError(err, "Failed to get pair data", map[string]interface{}{
				"key": key,
			})
			continue
		}

		// Check if this project ID matches base or head
		if projectID != pairData.BaseProjectID && projectID != pairData.HeadProjectID {
			continue
		}

		// Update the appropriate completion status atomically
		fieldToUpdate := ""
		if projectID == pairData.BaseProjectID {
			fieldToUpdate = "base_completed"
			pairData.BaseCompleted = true
		} else if projectID == pairData.HeadProjectID {
			fieldToUpdate = "head_completed"
			pairData.HeadCompleted = true
		}

		// Update Redis atomically
		err = r.redis.HSet(ctx, key, fieldToUpdate, webhookStatusCompleted).Err()
		if err != nil {
			return false, nil, fmt.Errorf("failed to update project completion: %w", err)
		}

		// Check if both projects are completed and diff not yet processed
		if pairData.BaseCompleted && pairData.HeadCompleted && !pairData.DiffProcessed {
			r.logger.LogInfo("Both projects completed, ready for diff", map[string]interface{}{
				"pr_number":       pairData.PRNumber,
				"base_project_id": pairData.BaseProjectID,
				"head_project_id": pairData.HeadProjectID,
			})
			return true, pairData, nil
		}

		// One completed, waiting for the other
		r.logger.LogInfo("One project completed, waiting for the other", map[string]interface{}{
			"pr_number":      pairData.PRNumber,
			"project_id":     projectID,
			"base_completed": pairData.BaseCompleted,
			"head_completed": pairData.HeadCompleted,
		})

		return false, pairData, nil
	}

	// Project not found in any pair (not a webhook-triggered scan)
	return false, nil, nil
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

	r.logger.LogInfo("Marked diff as processed", map[string]interface{}{
		"pr_number": prNumber,
	})

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
	err := r.redis.Del(ctx, key).Err()
	if err != nil {
		r.logger.LogError(err, "Failed to delete project pair from Redis", map[string]interface{}{
			"pr_number": prNumber,
			"key":       key,
		})
		return fmt.Errorf("failed to delete project pair: %w", err)
	}

	r.logger.LogInfo("Deleted project pair from Redis", map[string]interface{}{
		"pr_number": prNumber,
		"key":       key,
	})

	return nil
}

// DeleteProjectPairByProjectID finds and deletes a project pair by project ID
func (r *WebhookScanPairRepository) DeleteProjectPairByProjectID(ctx context.Context, projectID string) error {
	if projectID == "" {
		return fmt.Errorf("project_id is required")
	}

	// Find all webhook pairs
	keys, err := r.redis.Keys(ctx, webhookRedisKeyPrefix+"*").Result()
	if err != nil {
		return fmt.Errorf("failed to find project pairs: %w", err)
	}

	// Find the key containing this project ID
	for _, key := range keys {
		pairData, err := r.getPairData(ctx, key)
		if err != nil {
			continue
		}

		// Check if this project ID matches base or head
		if projectID == pairData.BaseProjectID || projectID == pairData.HeadProjectID {
			// Delete the pair
			err = r.redis.Del(ctx, key).Err()
			if err != nil {
				r.logger.LogError(err, "Failed to delete project pair", map[string]interface{}{
					"project_id": projectID,
					"pr_number":  pairData.PRNumber,
				})
				return fmt.Errorf("failed to delete project pair: %w", err)
			}

			r.logger.LogInfo("Deleted project pair due to scan failure", map[string]interface{}{
				"project_id": projectID,
				"pr_number":  pairData.PRNumber,
				"key":        key,
			})

			return nil
		}
	}

	// Project not found in any pair
	return nil
}
