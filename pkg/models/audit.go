package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type AuditLog struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	UserID    *uuid.UUID      `json:"user_id" db:"user_id"`
	HubID     *uuid.UUID      `json:"hub_id" db:"hub_id"`
	Action    *string         `json:"action" db:"action"`
	Metadata  *pq.StringArray `json:"metadata" db:"metadata"`
	CreatedAt *time.Time      `json:"created_at" db:"created_at"`
}
