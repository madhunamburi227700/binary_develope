package models

import (
	"time"

	"github.com/google/uuid"
)

type AuthUser struct {
	Username      string
	Authenticated bool
}

type User struct {
    ID        uuid.UUID  `json:"id" db:"id"`
    Email     string     `json:"email" db:"email"`
    Name      *string    `json:"name" db:"name"`
    GoogleID  *string    `json:"google_id" db:"google_id"`
    Picture   *string    `json:"picture" db:"picture"`
    Status    *string    `json:"status" db:"status"`
    CreatedAt *time.Time `json:"created_at" db:"created_at"`
    UpdatedAt *time.Time `json:"updated_at" db:"updated_at"`
}
