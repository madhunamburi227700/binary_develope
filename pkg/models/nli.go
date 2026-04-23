package models

import (
	"time"

	"github.com/google/uuid"
)

// NLIChat represents a row in the nli table.
type NLIChat struct {
	ID           uuid.UUID  `json:"id,omitempty" db:"id"`
	HubID        *uuid.UUID `json:"hub_id,omitempty" db:"hub_id"`
	Status       *string    `json:"status,omitempty" db:"status"`
	Conversation []string   `json:"conversation,omitempty" db:"conversation"`
	Agents       []string   `json:"agents,omitempty" db:"agents"`
	CreatedAt    *time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

type NLIChatSummary struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
}

// NLIConversationItem is the JSON shape stored inside conversation[] entries.
// Example: {"type":"user","data":"..."}
type NLIConversationItem struct {
	Type string `json:"type,omitempty"`
	Data string `json:"data,omitempty"`
}
