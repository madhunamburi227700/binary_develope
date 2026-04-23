package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/opsmx/ai-guardian-api/pkg/models"
)

// NLIRepository handles nli chat persistence.
type NLIRepository struct {
	*BaseRepository
}

func NewNLIRepository() *NLIRepository {
	return &NLIRepository{
		BaseRepository: NewBaseRepository("nli"),
	}
}

func (r *NLIRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.NLIChat, error) {
	query := `SELECT conversation FROM nli WHERE id = $1`

	chat := &models.NLIChat{ID: id}
	err := r.db.QueryRow(ctx, query, id).Scan(&chat.Conversation)
	if err != nil {
		// Keep error text compatible with existing controller check.
		// BaseRepository uses "record not found" for pgx.ErrNoRows.
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("record not found")
		}
		return nil, fmt.Errorf("failed to get nli conversation: %w", err)
	}

	return chat, nil
}

func (r *NLIRepository) ListSummariesByHubID(ctx context.Context, hubID uuid.UUID) ([]*models.NLIChatSummary, error) {
	var out []*models.NLIChatSummary

	// Postgres arrays are 1-indexed: conversation[1] is the first element.
	// We only fetch the first element to keep payload tiny.
	query := `
		SELECT
			id,
			coalesce(conversation[1], '') AS first_message
		FROM nli
		WHERE hub_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, hubID)
	if err != nil {
		r.logger.LogError(err, "Failed to list nli chat summaries by hub_id", map[string]interface{}{
			"hub_id": hubID.String(),
		})
		return nil, fmt.Errorf("failed to list nli chat summaries: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		summary := &models.NLIChatSummary{}
		if err := rows.Scan(&summary.ID, &summary.Title); err != nil {
			return nil, fmt.Errorf("failed to scan nli chat summary row: %w", err)
		}
		out = append(out, summary)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("failed to iterate nli chat summary rows: %w", rows.Err())
	}

	return out, nil
}
