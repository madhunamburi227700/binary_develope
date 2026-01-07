package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   ScanStatus
		expected ScanStatus
	}{
		{"Pending status", ScanStatusPending, "pending"},
		{"Fail status", ScanStatusFail, "fail"},
		{"Completed status", ScanStatusCompleted, "completed"},
		{"Scanning status", ScanStatusScanning, "scanning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status)
		})
	}
}

func TestScanJSONMarshaling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	settings := json.RawMessage(`{"key":"value"}`)

	scan := Scan{
		ID:            "scan-123",
		ProjectID:     "project-456",
		Status:        string(ScanStatusCompleted),
		TriggeredBy:   "user-789",
		HubID:         "hub-101",
		Remediated:    5,
		Repository:    "github.com/example/repo",
		Branch:        "main",
		CommitSHA:     "abc123",
		PullRequestID: "PR-42",
		Tag:           "v1.0.0",
		Settings:      settings,
		StartTime:     now,
		EndTime:       now.Add(5 * time.Minute),
		CreatedAt:     now,
	}

	// Marshal to JSON
	data, err := json.Marshal(scan)
	require.NoError(t, err, "Failed to marshal Scan to JSON")
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var unmarshaled Scan
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err, "Failed to unmarshal JSON to Scan")

	// Verify fields
	assert.Equal(t, scan.ID, unmarshaled.ID)
	assert.Equal(t, scan.ProjectID, unmarshaled.ProjectID)
	assert.Equal(t, scan.Status, unmarshaled.Status)
	assert.Equal(t, scan.TriggeredBy, unmarshaled.TriggeredBy)
	assert.Equal(t, scan.Remediated, unmarshaled.Remediated)
	assert.Equal(t, scan.Repository, unmarshaled.Repository)
	assert.Equal(t, scan.Branch, unmarshaled.Branch)
	assert.Equal(t, scan.CommitSHA, unmarshaled.CommitSHA)
}

func TestScanTypeJSONMarshaling(t *testing.T) {
	rawJSON := json.RawMessage(`{"findings": []}`)

	scanType := ScanType{
		ID:            "scantype-123",
		ScanID:        "scan-456",
		ScanType:      "SAST",
		Tool:          "semgrep",
		FileName:      "report.json",
		FileURL:       "https://example.com/report.json",
		RawJSON:       rawJSON,
		FindingsCount: 10,
		CriticalCount: 2,
		HighCount:     3,
		MediumCount:   4,
		LowCount:      1,
	}

	// Marshal to JSON
	data, err := json.Marshal(scanType)
	require.NoError(t, err, "Failed to marshal ScanType to JSON")
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var unmarshaled ScanType
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err, "Failed to unmarshal JSON to ScanType")

	// Verify fields
	assert.Equal(t, scanType.ID, unmarshaled.ID)
	assert.Equal(t, scanType.ScanID, unmarshaled.ScanID)
	assert.Equal(t, scanType.ScanType, unmarshaled.ScanType)
	assert.Equal(t, scanType.FindingsCount, unmarshaled.FindingsCount)
	assert.Equal(t, scanType.CriticalCount, unmarshaled.CriticalCount)
	assert.Equal(t, scanType.HighCount, unmarshaled.HighCount)
	assert.Equal(t, scanType.MediumCount, unmarshaled.MediumCount)
	assert.Equal(t, scanType.LowCount, unmarshaled.LowCount)
}

func TestScanExtStruct(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	scanExt := ScanExt{
		ScanId:     "scan-123",
		ProjectId:  "project-456",
		Status:     string(ScanStatusCompleted),
		Repository: "github.com/example/repo",
		Branch:     "main",
		CommitSHA:  "abc123",
		EndTime:    &now,
		CreatedAt:  now,
		Vulnerabilites: []*Vulnerability{},
	}

	assert.Equal(t, "scan-123", scanExt.ScanId)
	assert.Equal(t, "project-456", scanExt.ProjectId)
	assert.Equal(t, string(ScanStatusCompleted), scanExt.Status)
	assert.NotNil(t, scanExt.EndTime)
	assert.NotNil(t, scanExt.Vulnerabilites)
}

func TestScanStructTags(t *testing.T) {
	scan := Scan{}

	data, err := json.Marshal(scan)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Verify expected JSON keys exist
	expectedKeys := []string{
		"id", "project_id", "status", "triggered_by", "hub_id",
		"remediated", "repository", "branch", "commit_sha",
		"pull_request_id", "tag", "settings", "start_time",
		"end_time", "created_at",
	}

	for _, key := range expectedKeys {
		assert.Contains(t, result, key, "JSON should contain key: %s", key)
	}
}
