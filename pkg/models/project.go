package models

import (
	"time"
)

type Project struct {
	ID            string    `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	HubID         string    `json:"hub_id" db:"hub_id"`
	IntegrationID string    `json:"integration_id" db:"integration_id"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type ProjectExt struct {
	ProjectId string `db:"project_id"`
	Scans     []*ScanExt
}
