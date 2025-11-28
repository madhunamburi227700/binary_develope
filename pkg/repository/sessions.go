package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opsmx/ai-guardian-api/pkg/database"
	"github.com/opsmx/ai-guardian-api/pkg/models"
)

// SessionRepository handles Sessions-related database operations
type SessionRepository struct {
	db *pgxpool.Pool
}

// NewSessionRepository creates a new Sessions repository
func NewSessionRepository() *SessionRepository {
	return &SessionRepository{
		db: database.GetPostgres(),
	}
}

// Get the session's modified time
func (r *SessionRepository) Get(ctx context.Context, sessionId string) (*models.AuthUser, error) {
	query := `
        SELECT modified_on
        FROM http_sessions 
        WHERE key = $1
    `
	var authUser models.AuthUser
	err := r.db.QueryRow(
		ctx,
		query,
		sessionId,
	).Scan(&authUser.ModifiedOn)
	if err != nil {
		return nil, err
	}
	return &authUser, nil
}
