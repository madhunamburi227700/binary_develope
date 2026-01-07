package models

import (
	"database/sql"
	"time"
)

type AuthUser struct {
	Username      string
	Authenticated bool

	// used for session access and duration
	ExtSessionId string    `db:"id"`
	CreatedAt    time.Time `db:"created_at"`
	LastAccessed time.Time `db:"last_accessed"`
}

// User represents a row in the "users" table.
type User struct {
	ID             int64          `db:"id" json:"id"`
	Email          sql.NullString `db:"email" json:"email"`
	Name           sql.NullString `db:"name" json:"name"`
	Provider       string         `db:"provider" json:"provider"`
	ProviderUserID string         `db:"provider_user_id" json:"provider_user_id"`
	CreatedAt      time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time      `db:"updated_at" json:"updated_at"`
}
