package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemediationJSONMarshaling(t *testing.T) {
	remediationID := uuid.New()
	vulnID := uuid.New()
	scanResultID := uuid.New()
	promptID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	status := "completed"
	fixCommit := "abc123def456"
	prLink := "https://github.com/example/repo/pull/42"

	remediation := Remediation{
		ID:              remediationID,
		VulnerabilityID: vulnID,
		ScanResultID:    scanResultID,
		Status:          &status,
		FixCommitSHA:    &fixCommit,
		PRLink:          &prLink,
		PromptID:        &promptID,
		StartedAt:       &now,
		CompletedAt:     &now,
		CreatedAt:       now,
	}

	// Marshal to JSON
	data, err := json.Marshal(remediation)
	require.NoError(t, err, "Failed to marshal Remediation to JSON")
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var unmarshaled Remediation
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err, "Failed to unmarshal JSON to Remediation")

	// Verify fields
	assert.Equal(t, remediation.ID, unmarshaled.ID)
	assert.Equal(t, remediation.VulnerabilityID, unmarshaled.VulnerabilityID)
	assert.Equal(t, remediation.ScanResultID, unmarshaled.ScanResultID)
	assert.Equal(t, *remediation.Status, *unmarshaled.Status)
	assert.Equal(t, *remediation.FixCommitSHA, *unmarshaled.FixCommitSHA)
	assert.Equal(t, *remediation.PRLink, *unmarshaled.PRLink)
}

func TestRemediationWithNullFields(t *testing.T) {
	remediationID := uuid.New()
	vulnID := uuid.New()
	scanResultID := uuid.New()
	now := time.Now().UTC()

	remediation := Remediation{
		ID:              remediationID,
		VulnerabilityID: vulnID,
		ScanResultID:    scanResultID,
		CreatedAt:       now,
		// All pointer fields left as nil
	}

	// Marshal to JSON
	data, err := json.Marshal(remediation)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshaled Remediation
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify pointer fields are nil
	assert.Nil(t, unmarshaled.Status)
	assert.Nil(t, unmarshaled.FixCommitSHA)
	assert.Nil(t, unmarshaled.PRLink)
	assert.Nil(t, unmarshaled.PromptID)
	assert.Nil(t, unmarshaled.StartedAt)
	assert.Nil(t, unmarshaled.CompletedAt)
}

func TestRemediationVerificationJSONMarshaling(t *testing.T) {
	verificationID := uuid.New()
	vulnID := uuid.New()
	remediationID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	tool := "semgrep"
	status := "verified"
	description := "Vulnerability successfully remediated"

	verification := RemediationVerification{
		ID:               verificationID,
		VulnerabilityID:  &vulnID,
		RemediationID:    &remediationID,
		VerificationTool: &tool,
		Status:           &status,
		Description:      &description,
		CreatedAt:        &now,
	}

	// Marshal to JSON
	data, err := json.Marshal(verification)
	require.NoError(t, err, "Failed to marshal RemediationVerification to JSON")
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var unmarshaled RemediationVerification
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err, "Failed to unmarshal JSON to RemediationVerification")

	// Verify fields
	assert.Equal(t, verification.ID, unmarshaled.ID)
	assert.Equal(t, *verification.VulnerabilityID, *unmarshaled.VulnerabilityID)
	assert.Equal(t, *verification.RemediationID, *unmarshaled.RemediationID)
	assert.Equal(t, *verification.VerificationTool, *unmarshaled.VerificationTool)
	assert.Equal(t, *verification.Status, *unmarshaled.Status)
}

func TestRemediationFeedbackJSONMarshaling(t *testing.T) {
	feedbackID := uuid.New()
	remediationID := uuid.New()
	vulnID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	comments := "Great fix!"
	rating := 4.5

	feedback := RemediationFeedback{
		ID:              feedbackID,
		RemediationID:   remediationID,
		VulnerabilityID: vulnID,
		Comments:        &comments,
		Rating:          &rating,
		CreatedAt:       &now,
	}

	// Marshal to JSON
	data, err := json.Marshal(feedback)
	require.NoError(t, err, "Failed to marshal RemediationFeedback to JSON")
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var unmarshaled RemediationFeedback
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err, "Failed to unmarshal JSON to RemediationFeedback")

	// Verify fields
	assert.Equal(t, feedback.ID, unmarshaled.ID)
	assert.Equal(t, feedback.RemediationID, unmarshaled.RemediationID)
	assert.Equal(t, feedback.VulnerabilityID, unmarshaled.VulnerabilityID)
	assert.Equal(t, *feedback.Comments, *unmarshaled.Comments)
	assert.Equal(t, *feedback.Rating, *unmarshaled.Rating)
}

func TestRemediationStructTags(t *testing.T) {
	remediationID := uuid.New()
	vulnID := uuid.New()
	scanResultID := uuid.New()

	remediation := Remediation{
		ID:              remediationID,
		VulnerabilityID: vulnID,
		ScanResultID:    scanResultID,
		CreatedAt:       time.Now(),
	}

	data, err := json.Marshal(remediation)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Verify expected JSON keys exist
	expectedKeys := []string{
		"id", "vulnerability_id", "scan_result_id", "status",
		"fix_commit_sha", "pr_link", "prompt_id", "started_at",
		"completed_at", "created_at",
	}

	for _, key := range expectedKeys {
		assert.Contains(t, result, key, "JSON should contain key: %s", key)
	}
}
