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

func (r *AuditRepository) ListAuditLogByDateTime(ctx context.Context, datetime string) ([]*models.AuditLog, error) {
	query := `SELECT al.user_id, u.email, u.provider, al.http_method, al.action, al.endpoint, 
		al.entity_name, al.entity_id, al.request_body, al.response_status, al.response_body, 
		al.duration_ms, al.service_name, al.created_at
		FROM audit_logs al
		LEFT JOIN users u ON u.provider_user_id = al.user_id
		WHERE al.created_at >= $1 ORDER BY al.created_at`

	rows, err := r.db.Query(ctx, query, datetime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		var log models.AuditLog
		err := rows.Scan(
			&log.UserID,
			&log.Email,
			&log.Provider,
			&log.HTTPMethod,
			&log.Action,
			&log.Endpoint,
			&log.EntityName,
			&log.EntityID,
			&log.RequestBody,
			&log.ResponseStatus,
			&log.ResponseBody,
			&log.DurationMs,
			&log.ServiceName,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, &log)
	}
	return logs, nil
}
