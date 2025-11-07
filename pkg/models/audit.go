package models

import (
	"database/sql"
	"time"
)

// Action constants for audit logging
const (
	ActionLogin              = "LOGIN"
	ActionLogout             = "LOGOUT"
	ActionRescanInit         = "RESCAN_INITIATED"
	ActionScanInit           = "SCAN_INITIATED"
	ActionProjectCreate      = "PROJECT_CREATE"
	ActionProjectUpdate      = "PROJECT_UPDATE"
	ActionProjectDelete      = "PROJECT_DELETE"
	ActionRemediationAttempt = "REMEDIATION_ATTEMPTED"
	ActionRemediationApprove = "REMEDIATION_APPROVED"
)

type AuditLog struct {
	ID             int64          `json:"id" db:"id"`
	UserID         string         `json:"user_id" db:"user_id"`
	HTTPMethod     sql.NullString `json:"http_method" db:"http_method"`
	Action         sql.NullString `json:"action" db:"action"`
	Endpoint       sql.NullString `json:"endpoint" db:"endpoint"`
	EntityName     sql.NullString `json:"entity_name" db:"entity_name"`
	EntityID       sql.NullString `json:"entity_id" db:"entity_id"`
	RequestBody    sql.NullString `json:"request_body" db:"request_body"`
	ResponseStatus sql.NullInt16  `json:"response_status" db:"response_status"`
	ResponseBody   sql.NullString `json:"response_body" db:"response_body"`
	DurationMs     sql.NullInt32  `json:"duration_ms" db:"duration_ms"`
	ServiceName    sql.NullString `json:"service_name" db:"service_name"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
}
