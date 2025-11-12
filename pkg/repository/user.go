package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opsmx/ai-guardian-api/pkg/database"
	"github.com/opsmx/ai-guardian-api/pkg/models"
)

// UserRepository handles Users-related database operations
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new Users repository
func NewUserRepository() *UserRepository {
	return &UserRepository{
		db: database.GetPostgres(),
	}
}

// Create inserts a new user into the database and returns the new ID.
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	now := time.Now()

	query := `
		INSERT INTO users (email, name, provider, provider_user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id int64
	err := r.db.QueryRow(
		ctx,
		query,
		user.Email,
		user.Name,
		user.Provider,
		user.ProviderUserID,
		now,
		now,
	).Scan(&id)

	if err != nil {
		return err
	}

	user.ID = id
	user.CreatedAt = now
	user.UpdatedAt = now
	return nil
}

// Updates the user's email and name
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET email = $1, name = $2, updated_at = $3
		WHERE provider_user_id = $4
		RETURNING id
	`

	var id int64
	return r.db.QueryRow(
		ctx,
		query,
		user.Email,
		user.Name,
		time.Now(),
		user.ProviderUserID,
	).Scan(&id)
}

// GetByProviderUserID retrieves a user by provider_user_id
func (r *UserRepository) GetByProviderUserID(ctx context.Context, providerUserID string) (*models.User, error) {
	query := `
		SELECT id, email, name, provider, provider_user_id, created_at, updated_at
		FROM users WHERE provider_user_id = $1 LIMIT 1`

	var user models.User
	err := r.db.QueryRow(ctx, query, providerUserID).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Provider,
		&user.ProviderUserID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	return &user, err
}

// GetAllUsers retrieves all user
func (r *UserRepository) GetAllUsers(ctx context.Context) ([]*models.User, error) {
	query := `
		SELECT id, email, name, provider, provider_user_id, created_at, updated_at
		FROM users`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	var users []*models.User
	for rows.Next() {
		var user models.User
		err = rows.Scan(
			&user.ID,
			&user.Email,
			&user.Name,
			&user.Provider,
			&user.ProviderUserID,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, nil
}
