package models

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthUserStruct(t *testing.T) {
	authUser := AuthUser{
		Username:      "testuser",
		Authenticated: true,
	}

	assert.Equal(t, "testuser", authUser.Username)
	assert.True(t, authUser.Authenticated)
}

func TestUserJSONMarshaling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	user := User{
		ID:             123,
		Email:          sql.NullString{String: "test@example.com", Valid: true},
		Name:           sql.NullString{String: "Test User", Valid: true},
		Provider:       "google",
		ProviderUserID: "google-123",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Marshal to JSON
	data, err := json.Marshal(user)
	require.NoError(t, err, "Failed to marshal User to JSON")
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var unmarshaled User
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err, "Failed to unmarshal JSON to User")

	// Verify fields
	assert.Equal(t, user.ID, unmarshaled.ID)
	assert.Equal(t, user.Email.String, unmarshaled.Email.String)
	assert.Equal(t, user.Name.String, unmarshaled.Name.String)
	assert.Equal(t, user.Provider, unmarshaled.Provider)
	assert.Equal(t, user.ProviderUserID, unmarshaled.ProviderUserID)
}

func TestUserWithNullFields(t *testing.T) {
	now := time.Now().UTC()

	user := User{
		ID:             456,
		Provider:       "github",
		ProviderUserID: "github-456",
		CreatedAt:      now,
		UpdatedAt:      now,
		// Email and Name left as null
	}

	// Marshal to JSON
	data, err := json.Marshal(user)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshaled User
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify null fields are handled correctly
	assert.False(t, unmarshaled.Email.Valid)
	assert.False(t, unmarshaled.Name.Valid)
}

func TestUserStructTags(t *testing.T) {
	user := User{}

	data, err := json.Marshal(user)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Verify expected JSON keys exist
	expectedKeys := []string{"id", "email", "name", "provider", "provider_user_id", "created_at", "updated_at"}

	for _, key := range expectedKeys {
		assert.Contains(t, result, key, "JSON should contain key: %s", key)
	}
}
