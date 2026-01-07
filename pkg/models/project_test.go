package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectJSONMarshaling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	project := Project{
		ID:            "project-123",
		Name:          "Test Project",
		HubID:         "hub-456",
		IntegrationID: "integration-789",
		Organisation:  "Example Org",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Marshal to JSON
	data, err := json.Marshal(project)
	require.NoError(t, err, "Failed to marshal Project to JSON")
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var unmarshaled Project
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err, "Failed to unmarshal JSON to Project")

	// Verify fields
	assert.Equal(t, project.ID, unmarshaled.ID)
	assert.Equal(t, project.Name, unmarshaled.Name)
	assert.Equal(t, project.HubID, unmarshaled.HubID)
	assert.Equal(t, project.IntegrationID, unmarshaled.IntegrationID)
	assert.Equal(t, project.Organisation, unmarshaled.Organisation)
}

func TestProjectExtStruct(t *testing.T) {
	projectExt := ProjectExt{
		ProjectId:   "project-123",
		ProjectName: "Test Project",
		Organisation: "Example Org",
		Scans:       []*ScanExt{},
	}

	assert.Equal(t, "project-123", projectExt.ProjectId)
	assert.Equal(t, "Test Project", projectExt.ProjectName)
	assert.Equal(t, "Example Org", projectExt.Organisation)
	assert.NotNil(t, projectExt.Scans)
	assert.Empty(t, projectExt.Scans)
}

func TestProjectStructTags(t *testing.T) {
	project := Project{}

	data, err := json.Marshal(project)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Verify expected JSON keys exist
	expectedKeys := []string{"id", "name", "hub_id", "integration_id", "organisation", "created_at", "updated_at"}

	for _, key := range expectedKeys {
		assert.Contains(t, result, key, "JSON should contain key: %s", key)
	}
}
