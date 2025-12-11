package models

import (
	"time"
)

type Project struct {
	ID            string    `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	HubID         string    `json:"hub_id" db:"hub_id"`
	IntegrationID string    `json:"integration_id" db:"integration_id"`
	Organisation  string    `json:"organisation" db:"organisation"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
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
