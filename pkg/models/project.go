package models

import (
	"time"
)

type Project struct {
	ID                    string     `json:"id" db:"id"`
	Name                  string     `json:"name" db:"name"`
	HubID                 string     `json:"hub_id" db:"hub_id"`
	IntegrationID         string     `json:"integration_id" db:"integration_id"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
	Organisation          string     `json:"organisation" db:"organisation"`
	LastScannedTime       *time.Time `json:"last_scanned_time,omitempty" db:"last_scanned_time"`
	ScheduledTime         *int       `json:"scheduled_time,omitempty" db:"scheduled_time"`
}

type ProjectExt struct {
	ProjectId    string `db:"project_id"`
	ProjectName  string `db:"project_name"`
	Organisation string `db:"organisation"`
	Scans        []*ScanExt
}

type WebhookRequest struct {
	PRNumber   string `json:"PR_NUMBER" validate:"required"`
	HeadBranch string `json:"HEAD_BRANCH" validate:"required"`
	BaseBranch string `json:"BASE_BRANCH" validate:"required"`
	RepoURL    string `json:"REPO_URL" validate:"required"`
}
