package models

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	HubID         *uuid.UUID `json:"hub_id" db:"hub_id"`
	IntegrationID *uuid.UUID `json:"integration_id" db:"integration_id"`
	Name          *string    `json:"name" db:"name"`
	RepoURL       *string    `json:"repo_url" db:"repo_url"`
	Description   *string    `json:"description" db:"description"`
	CreatedAt     *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at" db:"updated_at"`
}
