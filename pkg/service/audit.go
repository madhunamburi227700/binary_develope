package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type AuditService interface {
	GetAuditReport(fromDate string) ([]*UserReport, error)
}

type auditService struct {
	logger          *utils.ErrorLogger
	ssdService      *SSDService
	auditRepo       *repository.AuditRepository
	userRepo        *repository.UserRepository
	scanRepo        *repository.ScanRepository
	remediationRepo *repository.RemediationRepository
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
		logger:          utils.NewErrorLogger("audit_service"),
		ssdService:      NewSSDService(),
		auditRepo:       repository.NewAuditRepository(),
		userRepo:        repository.NewUserRepository(),
		remediationRepo: repository.NewRemediationRepository(),
		scanRepo:        repository.NewScanRepository(),
	}
}

func (f *auditService) GetAuditReport(fromDate string) ([]*UserReport, error) {
	if fromDate == "" {
		return f.getAuditReportViaEntities()
	}
	auditList, err := f.auditRepo.ListAuditLogByDateTime(context.TODO(), fromDate)
	if err != nil {
		return nil, err
	}
	return genUserAuditReport(auditList), nil
}

func genUserAuditReport(auditList []*models.AuditLog) []*UserReport {
	var userAudit []*UserReport
	userDayIdx := map[string]int{}
	processedRemediationAttempts := map[string]bool{}
	for _, auditlog := range auditList {
		// skipping anonymous user logs
		if auditlog.UserID == "anonymous" || auditlog.UserID == "" {
			continue
		}

		// skipping remediation attempt count if repeated
		// one chat Id is equal to one attempt.
		if auditlog.Action.String == models.ActionRemediationAttempt {
			type remRequest struct {
				ID string `json:"id"`
			}
			var remReq remRequest
			err := json.Unmarshal([]byte(auditlog.RequestBody.String), &remReq)
			if err == nil {
				if _, raOk := processedRemediationAttempts[remReq.ID]; raOk {
					continue
				}
			}
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
	}
}

// getAuditReportViaEntities directly report data
// via entity tables from users, scans, vulns, remediations
// this will include past data also
func (a *auditService) getAuditReportViaEntities() ([]*UserReport, error) {
	auditReport := []*UserReport{}
	ctx := context.TODO()

	orgResponse, err := a.ssdService.GetOrganizationsAndTeams(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get org team details")
		return nil, err
	}

	teamMap := map[string]client.Hub{}
	for _, org := range orgResponse.QueryOrganization {
		for _, team := range org.Teams {
			teamMap[team.ID] = team
		}
	}

	// getting users
	users, err := a.userRepo.GetAllUsers(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get users")
		return nil, err
	}
	usersMap := map[string]*models.User{}
	for _, user := range users {
		out := *user
		usersMap[user.ProviderUserID] = &out
	}

	// getting remediations
	remediations, err := a.remediationRepo.List(ctx, &repository.QueryOptions{
		OrderBy: "created_at"})
	if err != nil {
		log.Error().Err(err).Msg("failed to get remediations")
		return nil, err
	}

	remediationsMap := map[string][]models.Remediation{}
	for _, r := range remediations.Data {
		remediationsMap[r.VulnerabilityID.String()] = append(remediationsMap[r.VulnerabilityID.String()], r)
	}

	hubsScanData, err := a.scanRepo.GetScansVulns(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get scansvulns")
		return nil, err
	}

	userDayIdx := map[string]int{}
	userDayRemediations := map[string]struct {
		Attempted uint16
		Approved  uint16
	}{}
	for _, hub := range hubsScanData {
		team, tok := teamMap[hub.ID.String()]
		// if hub is not matching then skip that hub
		if !tok {
			continue
		}

		userEmail := team.Email
		if user, uOk := usersMap[userEmail]; uOk && user.Email.Valid {
			userEmail = user.Email.String + "@" + user.Provider
		}
		for _, project := range hub.Projects {
			for _, scan := range project.Scans {
				for _, vuln := range scan.Vulnerabilites {
					for _, rem := range remediationsMap[vuln.ID.String()] {
						if rem.Status != nil {
							approved := *rem.Status == "PR_RAISED"
							rdate := rem.CreatedAt.Format("2006-01-02")
							udr, udrOk := userDayRemediations[userEmail+"@@"+rdate]
							if udrOk {
								udr.Attempted++
								if approved {
									udr.Approved++
								}
								userDayRemediations[userEmail+"@@"+rdate] = udr
							} else {
								stat := struct {
									Attempted uint16
									Approved  uint16
								}{Attempted: 1}
								if approved {
									stat.Approved++
								}
								userDayRemediations[userEmail+"@@"+rdate] = stat
							}
						}
					}
				}
				date := scan.CreatedAt.Format("2006-01-02")
				userAuditIdx, uaOk := userDayIdx[userEmail+"@@"+date]
				if uaOk {
					auditReport[userAuditIdx].Date = date
					auditReport[userAuditIdx].TotalScans++
				} else {
					auditReport = append(auditReport, &UserReport{
						Date:       date,
						Email:      userEmail,
						TotalScans: 1,
					})
					userDayIdx[userEmail+"@@"+date] = len(auditReport) - 1
				}
			}
		}
	}

	for key, stats := range userDayRemediations {
		if idx, udOk := userDayIdx[key]; udOk {
			auditReport[idx].TotalRemediationAttempted = stats.Attempted
			auditReport[idx].TotalRemediationApproved = stats.Approved
		} else {
			keyParts := strings.Split(key, "@@")
			if len(keyParts) > 1 {
				auditReport = append(auditReport, &UserReport{
					Date:                      keyParts[1],
					Email:                     keyParts[0],
					TotalRemediationAttempted: stats.Attempted,
					TotalRemediationApproved:  stats.Approved,
				})
			}
		}
	}
	sort.SliceStable(auditReport, func(i, j int) bool {
		return auditReport[i].Date < auditReport[j].Date
	})
	return auditReport, nil
}
