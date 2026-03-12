package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHubJSONMarshaling(t *testing.T) {
	ownerID := uuid.New()
	collabID1 := uuid.New()
	collabID2 := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	description := "Test Hub Description"

	hub := Hub{
		ID:             "7922c93d-848a-4bf8-ba9a-cd141b0e1149",
		Name:           "Test Hub",
		Description:    &description,
		OwnerID:        &ownerID,
		CollaboratorID: []uuid.UUID{collabID1, collabID2},
		CreatedAt:      &now,
		UpdatedAt:      &now,
		Projects:       []*ProjectExt{},
	}

	// Marshal to JSON
	data, err := json.Marshal(hub)
	require.NoError(t, err, "Failed to marshal Hub to JSON")
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var unmarshaled Hub
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err, "Failed to unmarshal JSON to Hub")

	// Verify fields
	assert.Equal(t, hub.ID, unmarshaled.ID)
	assert.Equal(t, hub.Name, unmarshaled.Name)
	assert.Equal(t, *hub.Description, *unmarshaled.Description)
	assert.Equal(t, *hub.OwnerID, *unmarshaled.OwnerID)
	assert.Equal(t, len(hub.CollaboratorID), len(unmarshaled.CollaboratorID))
}

func TestHubWithNullFields(t *testing.T) {
	hub := Hub{
		ID:             "7922c93d-848a-4bf8-ba9a-cd141b0e1149",
		Name:           "Minimal Hub",
		CollaboratorID: []uuid.UUID{},
		Projects:       []*ProjectExt{},
		// Description, OwnerID, CreatedAt, UpdatedAt left as nil
	}

	// Marshal to JSON
	data, err := json.Marshal(hub)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshaled Hub
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify pointer fields are nil
	assert.Nil(t, unmarshaled.Description)
	assert.Nil(t, unmarshaled.OwnerID)
	assert.Nil(t, unmarshaled.CreatedAt)
	assert.Nil(t, unmarshaled.UpdatedAt)
}

func TestHubStructTags(t *testing.T) {
	hub := Hub{
		ID:             "7922c93d-848a-4bf8-ba9a-cd141b0e1149",
		Name:           "Test Hub",
		CollaboratorID: []uuid.UUID{},
	}

	data, err := json.Marshal(hub)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Verify expected JSON keys exist
	expectedKeys := []string{"id", "name", "description", "owner_id", "collaborator_id", "created_at", "updated_at"}

	for _, key := range expectedKeys {
		assert.Contains(t, result, key, "JSON should contain key: %s", key)
	}
}
