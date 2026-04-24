package models

import (
	"bytes"
	"encoding/json"
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

func (c NLIChat) MarshalJSON() ([]byte, error) {
	type Alias NLIChat
	normalized := make([]any, 0, len(c.Conversation))
	for _, s := range c.Conversation {
		b := bytes.TrimSpace([]byte(s))
		if len(b) == 0 {
			normalized = append(normalized, nil) // or "" if you prefer
			continue
		}
		var v any
		if err := json.Unmarshal(b, &v); err != nil {
			// If not valid JSON, you must pick a fallback:
			// - keep as string (mixed types), or
			// - return error (enforce JSON-only)
			normalized = append(normalized, s)
			continue
		}
		normalized = append(normalized, v) // <-- this is the key change
	}
	out := struct {
		Alias
		Conversation []any `json:"conversation,omitempty"`
	}{
		Alias:        Alias(c),
		Conversation: normalized,
	}
	return json.Marshal(out)
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
