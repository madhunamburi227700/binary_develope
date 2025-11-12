package models

import (
	"time"

	"github.com/google/uuid"
)

type Hub struct {
	ID             uuid.UUID   `json:"id" db:"id"`
	Name           string      `json:"name" db:"name"`
	Description    *string     `json:"description" db:"description"`
	OwnerID        *uuid.UUID  `json:"owner_id" db:"owner_id"`
	CollaboratorID []uuid.UUID `json:"collaborator_id" db:"collaborator_id"`
	CreatedAt      *time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      *time.Time  `json:"updated_at" db:"updated_at"`

	Projects []*ProjectExt
}
