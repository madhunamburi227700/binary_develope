package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opsmx/ai-guardian-api/pkg/database"
	"github.com/opsmx/ai-guardian-api/pkg/models"
)

type UserSessionsRepository interface {
	GetByID(ctx context.Context, extSessionID string) (*models.AuthUser, error)
}

// userSessionsRepository handles User-sessions-related database operations
type userSessionsRepository struct {
	db *pgxpool.Pool
}

// NewUserSessionsRepository creates a new User sessions repository
func NewUserSessionsRepository() UserSessionsRepository {
	return &userSessionsRepository{
		db: database.GetPostgres(),
	}
}

// GetByID retrieves a user session by id
func (r *userSessionsRepository) GetByID(ctx context.Context, extSessionID string) (*models.AuthUser, error) {
	query := `
		SELECT id, created_at, last_accessed 
		FROM user_sessions WHERE id = $1
		`

	var user models.AuthUser
	err := r.db.QueryRow(ctx, query, extSessionID).Scan(
		&user.ExtSessionId,
		&user.CreatedAt,
		&user.LastAccessed,
	)
	return &user, err
}
