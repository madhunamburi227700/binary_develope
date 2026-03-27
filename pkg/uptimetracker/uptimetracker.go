package uptimetracker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/opsmx/ai-guardian-api/pkg/config"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

const (
	defaultProbeInterval = 300 * time.Second
	defaultProbeTimeout  = 10 * time.Second
)

// -------- INTERFACE --------
type UptimeTrackerManager interface {
	Start(ctx context.Context)
	Stop()
}

// ServiceDetails holds per-goroutine monitor runtime: health transitions and probe timeout.
type ServiceDetails struct {
	IsDown    bool
	DownStart time.Time
	// CheckTimeout is the HTTP client deadline for each probe (from config.Service.TimeoutSeconds).
	CheckTimeout time.Duration
}

// -------- STRUCT --------
type UptimeTracker struct {
	services []config.Service

	notifier Notifier

	cancel context.CancelFunc
	logger *utils.ErrorLogger
}

// -------- CONSTRUCTOR --------
func NewUptimeTracker() *UptimeTracker {
	return &UptimeTracker{
		services: config.GetUptimeServices(),
		notifier: NewNotificationClient(),
		logger:   utils.NewErrorLogger("uptime-tracker"),
	}
}

// -------- START --------
func (u *UptimeTracker) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	u.cancel = cancel

	for _, svc := range u.services {
		service := svc
		go u.monitorService(ctx, service)
	}
}

// -------- STOP --------
func (u *UptimeTracker) Stop() {
	if u.cancel != nil {
		u.logger.LogInfo("Stopping uptime tracker", nil)
		u.cancel()
	}
}

func effectiveProbeInterval(svc config.Service) time.Duration {
	if svc.IntervalSeconds <= 0 {
		return defaultProbeInterval
	}
	return time.Duration(svc.IntervalSeconds) * time.Second
}

func effectiveProbeTimeout(svc config.Service) time.Duration {
	if svc.TimeoutSeconds <= 0 {
		return defaultProbeTimeout
	}
	return time.Duration(svc.TimeoutSeconds) * time.Second
}

// -------- MONITOR --------
func (u *UptimeTracker) monitorService(ctx context.Context, svc config.Service) {

	interval := effectiveProbeInterval(svc)
	timeout := effectiveProbeTimeout(svc)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	details := ServiceDetails{CheckTimeout: timeout}

	for {
		select {
		case <-ctx.Done():
			u.logger.LogInfo("Stopping service monitor", map[string]interface{}{
				"service": svc.Name,
			})
			return

		case <-ticker.C:
			u.checkAndNotify(svc, &details)
		}
	}
}

// -------- URL CHECK --------
func (u *UptimeTracker) checkURL(url string, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// -------- CORE LOGIC --------
func (u *UptimeTracker) checkAndNotify(
	svc config.Service,
	details *ServiceDetails,
) {
	isHealthy := u.checkURL(svc.URL, details.CheckTimeout)
	now := time.Now()

	var (
		shouldNotify bool
		downStart    time.Time
		subject      string
		message      string
	)

	if !isHealthy && !details.IsDown {
		details.IsDown = true
		details.DownStart = now
		shouldNotify = true
	} else if isHealthy && details.IsDown {
		downStart = details.DownStart
		details.IsDown = false
		shouldNotify = true
	}

	if shouldNotify {
		if !isHealthy {
			subject = "Service Down Alert"
			message = fmt.Sprintf(
				"SERVICE DOWN \nService: %s\nURL: %s\nTime: %s",
				svc.Name,
				svc.URL,
				now.Format(time.RFC3339),
			)
		} else {
			downtime := now.Sub(downStart)

			subject = "Service Recovery Alert"
			message = fmt.Sprintf(
				"SERVICE RECOVERED \nService: %s\nURL: %s\nDowntime: %s\nRecovered At: %s",
				svc.Name,
				svc.URL,
				downtime.String(),
				now.Format(time.RFC3339),
			)
		}

		go func(s config.Service, subj, msg string) {
			u.sendNotifications(s, subj, msg)
		}(svc, subject, message)
	}
}

// -------- NOTIFICATIONS --------
func (u *UptimeTracker) sendNotifications(svc config.Service, subject, message string) {

	// -------- SERVICE LEVEL EMAIL --------
	if svc.Notifications.Email.Enabled && len(svc.Notifications.Email.Addresses) > 0 {
		u.notifier.SendEmail(svc.Notifications.Email.Addresses, subject, message)
	}

	// -------- SERVICE LEVEL SLACK --------
	if svc.Notifications.Slack.Enabled && len(svc.Notifications.Slack.Addresses) > 0 {
		for _, webhook := range svc.Notifications.Slack.Addresses {
			u.notifier.SendSlack(webhook, message)
		}
	}
}
