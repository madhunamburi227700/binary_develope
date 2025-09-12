package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Remediation struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	ScanResultID uuid.UUID       `json:"scan_result_id" db:"scan_result_id"`
	Status       *string         `json:"status" db:"status"`
	FixCommitSHA *string         `json:"fix_commit_sha" db:"fix_commit_sha"`
	PRLink       *string         `json:"pr_link" db:"pr_link"`
	PromptID     *uuid.UUID      `json:"prompt_id" db:"prompt_id"`
	Conversation *pq.StringArray `json:"conversation" db:"conversation"`
	StartedAt    *time.Time      `json:"started_at" db:"started_at"`
	CompletedAt  *time.Time      `json:"completed_at" db:"completed_at"`
}

type RemediationVerification struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	VulnerabilityID  *uuid.UUID `json:"vulnerability_id" db:"vulnerability_id"`
	RemediationID    *uuid.UUID `json:"remediation_id" db:"remediation_id"`
	VerificationTool *string    `json:"verification_tool" db:"verification_tool"`
	Status           *string    `json:"status" db:"status"`
	Description      *string    `json:"description" db:"description"`
	CreatedAt        *time.Time `json:"created_at" db:"created_at"`
}

type RemediationFeedback struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	RemediationID   *uuid.UUID `json:"remediation_id" db:"remediation_id"`
	VulnerabilityID *uuid.UUID `json:"vulnerability_id" db:"vulnerability_id"`
	Comments        *string    `json:"comments" db:"comments"`
	Rating          *float64   `json:"rating" db:"rating"`
}
