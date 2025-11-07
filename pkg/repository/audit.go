package repository

import (
	"context"

	"github.com/opsmx/ai-guardian-api/pkg/models"
)

// AuditRepository handles audit log database operations
type AuditRepository struct {
	*BaseRepository
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository() *AuditRepository {
	return &AuditRepository{
		BaseRepository: NewBaseRepository("audit_logs"),
	}
}

// CreateAuditLog creates a new audit log entry
func (r *AuditRepository) CreateAuditLog(ctx context.Context, auditLog *models.AuditLog) error {
	query := `INSERT INTO audit_logs (
		user_id, http_method, action, endpoint, entity_name, entity_id,
		request_body, response_status, response_body, duration_ms, service_name, created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id`

	var id int64
	err := r.db.QueryRow(ctx, query,
		auditLog.UserID,
		auditLog.HTTPMethod,
		auditLog.Action,
		auditLog.Endpoint,
		auditLog.EntityName,
		auditLog.EntityID,
		auditLog.RequestBody,
		auditLog.ResponseStatus,
		auditLog.ResponseBody,
		auditLog.DurationMs,
		auditLog.ServiceName,
		auditLog.CreatedAt,
	).Scan(&id)

	if err != nil {
		r.logger.LogError(err, "Failed to create audit log", map[string]interface{}{
			"user_id": auditLog.UserID,
		})
		return err
	}

	auditLog.ID = id
	return nil
}
