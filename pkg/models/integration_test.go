package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationJSONMarshaling(t *testing.T) {
	integrationID := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	integrationType := "github"
	name := "GitHub Integration"
	config := `{"api_key":"secret","webhook_url":"https://example.com/webhook"}`
	isActive := true

	integration := Integration{
		ID:        integrationID,
		UserID:    userID,
		Type:      &integrationType,
		Name:      &name,
		Config:    &config,
		IsActive:  &isActive,
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	// Marshal to JSON
	data, err := json.Marshal(integration)
	require.NoError(t, err, "Failed to marshal Integration to JSON")
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var unmarshaled Integration
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err, "Failed to unmarshal JSON to Integration")

	// Verify fields
	assert.Equal(t, integration.ID, unmarshaled.ID)
	assert.Equal(t, integration.UserID, unmarshaled.UserID)
	assert.Equal(t, *integration.Type, *unmarshaled.Type)
	assert.Equal(t, *integration.Name, *unmarshaled.Name)
	assert.Equal(t, *integration.Config, *unmarshaled.Config)
	assert.Equal(t, *integration.IsActive, *unmarshaled.IsActive)
}

func TestIntegrationWithNullFields(t *testing.T) {
	integrationID := uuid.New()
	userID := uuid.New()

	integration := Integration{
		ID:     integrationID,
		UserID: userID,
		// All pointer fields left as nil
	}

	// Marshal to JSON
	data, err := json.Marshal(integration)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshaled Integration
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify pointer fields are nil
	assert.Nil(t, unmarshaled.Type)
	assert.Nil(t, unmarshaled.Name)
	assert.Nil(t, unmarshaled.Config)
	assert.Nil(t, unmarshaled.IsActive)
	assert.Nil(t, unmarshaled.CreatedAt)
	assert.Nil(t, unmarshaled.UpdatedAt)
}

func TestSettingJSONMarshaling(t *testing.T) {
	settingID := uuid.New()
	hubID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	key := "theme"
	value := "dark"

	setting := Setting{
		ID:        settingID,
		HubID:     &hubID,
		Key:       &key,
		Value:     &value,
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	// Marshal to JSON
	data, err := json.Marshal(setting)
	require.NoError(t, err, "Failed to marshal Setting to JSON")
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var unmarshaled Setting
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err, "Failed to unmarshal JSON to Setting")

	// Verify fields
	assert.Equal(t, setting.ID, unmarshaled.ID)
	assert.Equal(t, *setting.HubID, *unmarshaled.HubID)
	assert.Equal(t, *setting.Key, *unmarshaled.Key)
	assert.Equal(t, *setting.Value, *unmarshaled.Value)
}

func TestSettingWithNullFields(t *testing.T) {
	settingID := uuid.New()

	setting := Setting{
		ID: settingID,
		// All pointer fields left as nil
	}

	// Marshal to JSON
	data, err := json.Marshal(setting)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshaled Setting
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify pointer fields are nil
	assert.Nil(t, unmarshaled.HubID)
	assert.Nil(t, unmarshaled.Key)
	assert.Nil(t, unmarshaled.Value)
	assert.Nil(t, unmarshaled.CreatedAt)
	assert.Nil(t, unmarshaled.UpdatedAt)
}

func TestIntegrationStructTags(t *testing.T) {
	integrationID := uuid.New()
	userID := uuid.New()

	integration := Integration{
		ID:     integrationID,
		UserID: userID,
	}

	data, err := json.Marshal(integration)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Verify expected JSON keys exist
	expectedKeys := []string{"id", "user_id", "type", "name", "config", "is_active", "created_at", "updated_at"}

	for _, key := range expectedKeys {
		assert.Contains(t, result, key, "JSON should contain key: %s", key)
	}
}

func TestSettingStructTags(t *testing.T) {
	settingID := uuid.New()

	setting := Setting{
		ID: settingID,
	}

	data, err := json.Marshal(setting)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Verify expected JSON keys exist
	expectedKeys := []string{"id", "hub_id", "key", "value", "created_at", "updated_at"}

	for _, key := range expectedKeys {
		assert.Contains(t, result, key, "JSON should contain key: %s", key)
	}
}
