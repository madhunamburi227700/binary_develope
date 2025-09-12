package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Scan struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	ProjectID     *uuid.UUID      `json:"project_id" db:"project_id"`
	ScanType      *string         `json:"scan_type" db:"scan_type"`
	Tool          *string         `json:"tool" db:"tool"`
	Status        *string         `json:"status" db:"status"`
	TriggeredBy   *uuid.UUID      `json:"triggered_by" db:"triggered_by"`
	FileName      *string         `json:"file_name" db:"file_name"`
	FileURL       *string         `json:"file_url" db:"file_url"`
	RawJSON       *pq.StringArray `json:"raw_json" db:"raw_json"`
	FindingsCount *int            `json:"findings_count" db:"findings_count"`
	CriticalCount *int            `json:"critical_count" db:"critical_count"`
	HighCount     *int            `json:"high_count" db:"high_count"`
	MediumCount   *int            `json:"medium_count" db:"medium_count"`
	LowCount      *int            `json:"low_count" db:"low_count"`
	Remediated    *int            `json:"remediated" db:"remediated"`
	Branch        *string         `json:"branch" db:"branch"`
	CommitSHA     *string         `json:"commit_sha" db:"commit_sha"`
	PullRequestID *string         `json:"pull_request_id" db:"pull_request_id"`
	Tag           *string         `json:"tag" db:"tag"`
	Settings      *pq.StringArray `json:"settings" db:"settings"`
	StartTime     *time.Time      `json:"start_time" db:"start_time"`
	EndTime       *time.Time      `json:"end_time" db:"end_time"`
	CreatedAt     *time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt     *time.Time      `json:"updated_at" db:"updated_at"`
}
