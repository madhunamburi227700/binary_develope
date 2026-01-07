package models

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditLogJSONMarshaling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	auditLog := AuditLog{
		ID:             123,
		UserID:         "user-123",
		HTTPMethod:     sql.NullString{String: "POST", Valid: true},
		Action:         sql.NullString{String: ActionLogin, Valid: true},
		Endpoint:       sql.NullString{String: "/api/auth/login", Valid: true},
		EntityName:     sql.NullString{String: "user", Valid: true},
		EntityID:       sql.NullString{String: "user-123", Valid: true},
		RequestBody:    sql.NullString{String: `{"email":"test@example.com"}`, Valid: true},
		ResponseStatus: sql.NullInt16{Int16: 200, Valid: true},
		ResponseBody:   sql.NullString{String: `{"success":true}`, Valid: true},
		DurationMs:     sql.NullInt32{Int32: 150, Valid: true},
		ServiceName:    sql.NullString{String: "auth-service", Valid: true},
		CreatedAt:      now,
		Email:          sql.NullString{String: "test@example.com", Valid: true},
		Provider:       sql.NullString{String: "google", Valid: true},
	}

	// Marshal to JSON
	data, err := json.Marshal(auditLog)
	require.NoError(t, err, "Failed to marshal AuditLog to JSON")
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var unmarshaled AuditLog
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err, "Failed to unmarshal JSON to AuditLog")

	// Verify fields
	assert.Equal(t, auditLog.ID, unmarshaled.ID)
	assert.Equal(t, auditLog.UserID, unmarshaled.UserID)
	assert.Equal(t, auditLog.HTTPMethod.String, unmarshaled.HTTPMethod.String)
	assert.Equal(t, auditLog.Action.String, unmarshaled.Action.String)
	assert.Equal(t, auditLog.Endpoint.String, unmarshaled.Endpoint.String)
	assert.Equal(t, auditLog.ResponseStatus.Int16, unmarshaled.ResponseStatus.Int16)
	assert.Equal(t, auditLog.DurationMs.Int32, unmarshaled.DurationMs.Int32)
}

func TestAuditLogWithNullFields(t *testing.T) {
	now := time.Now().UTC()

	auditLog := AuditLog{
		ID:        456,
		UserID:    "user-456",
		CreatedAt: now,
		// All sql.Null* fields left as invalid/null
	}

	// Marshal to JSON
	data, err := json.Marshal(auditLog)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshaled AuditLog
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify null fields are handled correctly
	assert.False(t, unmarshaled.HTTPMethod.Valid)
	assert.False(t, unmarshaled.Action.Valid)
	assert.False(t, unmarshaled.ResponseStatus.Valid)
	assert.False(t, unmarshaled.DurationMs.Valid)
}

func TestAuditActionConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"Login action", ActionLogin, "LOGIN"},
		{"Logout action", ActionLogout, "LOGOUT"},
		{"Rescan init action", ActionRescanInit, "RESCAN_INITIATED"},
		{"Scan init action", ActionScanInit, "SCAN_INITIATED"},
		{"Project create action", ActionProjectCreate, "PROJECT_CREATE"},
		{"Project update action", ActionProjectUpdate, "PROJECT_UPDATE"},
		{"Project delete action", ActionProjectDelete, "PROJECT_DELETE"},
		{"Remediation attempt action", ActionRemediationAttempt, "REMEDIATION_ATTEMPTED"},
		{"Remediation approve action", ActionRemediationApprove, "REMEDIATION_APPROVED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestAuditLogStructTags(t *testing.T) {
	// Verify struct has correct JSON and DB tags
	auditLog := AuditLog{}

	data, err := json.Marshal(auditLog)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Verify expected JSON keys exist
	expectedKeys := []string{
		"id", "user_id", "http_method", "action", "endpoint",
		"entity_name", "entity_id", "request_body", "response_status",
		"response_body", "duration_ms", "service_name", "created_at",
		"email", "provider",
	}

	for _, key := range expectedKeys {
		assert.Contains(t, result, key, "JSON should contain key: %s", key)
	}
}
