package models

import (
	"encoding/json"
	"time"
)

// ScanStatus represents the possible states of a process.
type ScanStatus string

const (
	ScanStatusPending   ScanStatus = "pending"
	ScanStatusFail      ScanStatus = "fail"
	ScanStatusCompleted ScanStatus = "completed"
	ScanStatusScanning  ScanStatus = "scanning"
)

type Scan struct {
	ID            string          `json:"id" db:"id"`
	ProjectID     string          `json:"project_id" db:"project_id"`
	Status        string          `json:"status" db:"status"`
	TriggeredBy   string          `json:"triggered_by" db:"triggered_by"`
	HubID         string          `json:"hub_id" db:"hub_id"`
	Remediated    int             `json:"remediated" db:"remediated"`
	Repository    string          `json:"repository" db:"repository"`
	Branch        string          `json:"branch" db:"branch"`
	CommitSHA     string          `json:"commit_sha" db:"commit_sha"`
	PullRequestID string          `json:"pull_request_id" db:"pull_request_id"`
	Tag           string          `json:"tag" db:"tag"`
	Settings      json.RawMessage `json:"settings" db:"settings"`
	StartTime     time.Time       `json:"start_time" db:"start_time"`
	EndTime       time.Time       `json:"end_time" db:"end_time"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
}

// scan type entry
type ScanType struct {
	ID            string          `json:"id" db:"id"`
	ScanID        string          `json:"scan_id" db:"scan_id"`
	ScanType      string          `json:"scan_type" db:"scan_type"`
	Tool          string          `json:"tool" db:"tool"`
	FileName      string          `json:"file_name" db:"file_name"`
	FileURL       string          `json:"file_url" db:"file_url"`
	RawJSON       json.RawMessage `json:"raw_json" db:"raw_json"`
	FindingsCount int             `json:"findings_count" db:"findings_count"`
	CriticalCount int             `json:"critical_count" db:"critical_count"`
	HighCount     int             `json:"high_count" db:"high_count"`
	MediumCount   int             `json:"medium_count" db:"medium_count"`
	LowCount      int             `json:"low_count" db:"low_count"`
}

type ScanExt struct {
	ScanId         string    `db:"scan_id"`
	ProjectId      string    `db:"project_id"`
	Status         string    `db:"status"`
	Branch         string    `db:"branch"`
	CommitSHA      string    `db:"commit_sha"`
	EndTime        time.Time `db:"end_time"`
	Vulnerabilites []*Vulnerability
}
