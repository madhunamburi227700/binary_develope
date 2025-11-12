package service

import (
	"context"
	"fmt"
	"time"

	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
	"github.com/rs/zerolog/log"
)

type AuditService interface {
	GetAuditReport(fromDate string) ([]*UserReport, error)
}

type auditService struct {
	logger    *utils.ErrorLogger
	auditRepo *repository.AuditRepository
}

type UserReport struct {
	Date                      string `json:"date"`
	Email                     string `json:"email"`
	Duration                  uint32 `json:"duration"`
	TotalScans                uint16 `json:"total_scans"`
	TotalRemediationAttempted uint16 `json:"total_remediation_attempted"`
	TotalRemediationApproved  uint16 `json:"total_remediation_approved"`

	// used for stat calculation
	lastLogin time.Time
}

func NewAuditService() AuditService {
	return &auditService{
		logger:    utils.NewErrorLogger("audit_service"),
		auditRepo: repository.NewAuditRepository(),
	}
}

func (f *auditService) GetAuditReport(fromDate string) ([]*UserReport, error) {
	auditList, err := f.auditRepo.ListAuditLogByDateTime(context.TODO(), fromDate)
	if err != nil {
		return nil, err
	}
	return genUserAuditReport(auditList), nil
}

func genUserAuditReport(auditList []*models.AuditLog) []*UserReport {
	var userAudit []*UserReport
	userDayIdx := map[string]int{}
	for _, auditlog := range auditList {
		// skipping anonymous user logs
		if auditlog.UserID == "anonymous" || auditlog.UserID == "" {
			continue
		}

		date := auditlog.CreatedAt.Format("2006-01-02")
		if idx, ok := userDayIdx[auditlog.UserID+date]; ok {
			analyseUserActionStat(userAudit[idx], auditlog)
		} else {
			email := fmt.Sprintf("%s@%s", auditlog.Email.String, auditlog.Provider.String)
			if !auditlog.Email.Valid {
				email = fmt.Sprintf("%s@%s", auditlog.UserID, auditlog.Provider.String)
			}
			userDayStat := &UserReport{
				Date:  date,
				Email: email,
			}
			analyseUserActionStat(userDayStat, auditlog)
			userAudit = append(userAudit, userDayStat)
			userDayIdx[auditlog.UserID+date] = len(userAudit) - 1
		}

	}
	return userAudit
}

func analyseUserActionStat(user *UserReport, auditLog *models.AuditLog) {
	// no action to analyse stat
	if !auditLog.Action.Valid {
		return
	}
	action := auditLog.Action.String
	// whenever a project get created or rescan get triggered a scan initiates
	if action == models.ActionProjectCreate || action == models.ActionRescanInit {
		user.TotalScans++
	}
	if action == models.ActionRemediationAttempt {
		user.TotalRemediationAttempted++
	}
	if action == models.ActionRemediationApprove {
		user.TotalRemediationApproved++
	}
	if action == models.ActionLogin {
		user.lastLogin = auditLog.CreatedAt
	}
	if action == models.ActionLogout && !user.lastLogin.IsZero() {
		user.Duration += uint32(auditLog.CreatedAt.Sub(user.lastLogin).Seconds())
		user.lastLogin = time.Time{}
		log.Info().Msgf("%t", user.lastLogin.IsZero())
	}
}
