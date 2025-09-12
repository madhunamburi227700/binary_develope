package models

import (
	"time"

	"github.com/google/uuid"
)

type Integration struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Type      *string    `json:"type" db:"type"`
	Name      *string    `json:"name" db:"name"`
	Config    *string    `json:"config" db:"config"`
	IsActive  *bool      `json:"is_active" db:"is_active"`
	CreatedAt *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt *time.Time `json:"updated_at" db:"updated_at"`
}

type Setting struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	HubID     *uuid.UUID `json:"hub_id" db:"hub_id"`
	Key       *string    `json:"key" db:"key"`
	Value     *string    `json:"value" db:"value"`
	CreatedAt *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt *time.Time `json:"updated_at" db:"updated_at"`
}
