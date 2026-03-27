package uptimetracker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/opsmx/ai-guardian-api/pkg/config"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// -------- MOCK NOTIFIER --------
type MockNotifier struct {
	EmailsSent [][]string
	SlackSent  []string
}

func (m *MockNotifier) SendEmail(to []string, subject, body string) {
	m.EmailsSent = append(m.EmailsSent, to)
}

func (m *MockNotifier) SendSlack(webhook, msg string) {
	m.SlackSent = append(m.SlackSent, webhook)
}

// waitUntil polls until cond() is true or timeout (for async sendNotifications).
func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

// -------- HELPER --------
func newTestService() config.Service {
	return config.Service{
		Name:            "test-service",
		URL:             "http://placeholder.invalid",
		IntervalSeconds: 10,
		TimeoutSeconds:  2,

		Notifications: struct {
			Email struct {
				Enabled   bool     `yaml:"enabled"`
				Addresses []string `yaml:"addresses"`
			} `yaml:"email"`

			Slack struct {
				Enabled   bool     `yaml:"enabled"`
				Addresses []string `yaml:"addresses"`
			} `yaml:"slack"`
		}{
			Email: struct {
				Enabled   bool     `yaml:"enabled"`
				Addresses []string `yaml:"addresses"`
			}{
				Enabled:   false,
				Addresses: []string{},
			},
			Slack: struct {
				Enabled   bool     `yaml:"enabled"`
				Addresses []string `yaml:"addresses"`
			}{
				Enabled:   true,
				Addresses: []string{"http://slack-webhook"},
			},
		},
	}
}

func TestCheckURL_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	u := &UptimeTracker{}
	if !u.checkURL(srv.URL, 2*time.Second) {
		t.Fatal("expected URL check to succeed for 200 OK")
	}
}

func TestCheckURL_Non2xxStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	u := &UptimeTracker{}
	if u.checkURL(srv.URL, 2*time.Second) {
		t.Fatal("expected URL check to fail for non-2xx")
	}
}

func TestCheckURL_2xxNoContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	u := &UptimeTracker{}
	if !u.checkURL(srv.URL, 2*time.Second) {
		t.Fatal("expected 204 No Content to count as healthy")
	}
}

func TestCheckURL_InvalidURL(t *testing.T) {
	u := &UptimeTracker{}
	if u.checkURL(":", time.Second) {
		t.Fatal("expected false when request URL cannot be parsed")
	}
}

func TestCheckURL_ProbeTimesOut(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
		w.WriteHeader(http.StatusOK)
	}))
	// Unblock handler before srv.Close(); otherwise Close waits forever on the stuck handler.
	defer srv.Close()
	defer close(block)

	u := &UptimeTracker{}
	if u.checkURL(srv.URL, 30*time.Millisecond) {
		t.Fatal("expected false when probe exceeds CheckTimeout")
	}
}

func TestNewUptimeTracker_NotNil(t *testing.T) {
	u := NewUptimeTracker()
	if u == nil || u.notifier == nil || u.logger == nil {
		t.Fatal("NewUptimeTracker returned incomplete tracker")
	}
}

func TestStop_WithoutStart_NoPanic(t *testing.T) {
	u := &UptimeTracker{}
	u.Stop()
}

func TestEffectiveProbeTimingDefaults(t *testing.T) {
	s := newTestService()
	s.IntervalSeconds = 0
	s.TimeoutSeconds = 0
	if effectiveProbeInterval(s) != defaultProbeInterval {
		t.Fatalf("interval: got %v want %v", effectiveProbeInterval(s), defaultProbeInterval)
	}
	if effectiveProbeTimeout(s) != defaultProbeTimeout {
		t.Fatalf("timeout: got %v want %v", effectiveProbeTimeout(s), defaultProbeTimeout)
	}
}

func TestStartStop_MonitorsReceiveCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := newTestService()
	svc.URL = srv.URL
	svc.IntervalSeconds = 3600

	u := &UptimeTracker{
		services: []config.Service{svc},
		notifier: &MockNotifier{},
		logger:   utils.NewErrorLogger("test"),
	}

	u.Start(context.Background())
	u.Stop()
	time.Sleep(50 * time.Millisecond)
}

func TestStartStop_TickerRunsProbe(t *testing.T) {
	var probes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		probes.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := newTestService()
	svc.URL = srv.URL
	svc.IntervalSeconds = 1
	svc.TimeoutSeconds = 2
	svc.Notifications.Slack.Enabled = false

	u := &UptimeTracker{
		services: []config.Service{svc},
		notifier: &MockNotifier{},
		logger:   utils.NewErrorLogger("test"),
	}

	u.Start(context.Background())
	time.Sleep(2500 * time.Millisecond)
	u.Stop()
	time.Sleep(100 * time.Millisecond)

	if probes.Load() < 1 {
		t.Fatalf("expected at least one probe from ticker, got %d", probes.Load())
	}
}

func TestSendNotifications_EmailOnly(t *testing.T) {
	mock := &MockNotifier{}
	u := &UptimeTracker{notifier: mock}

	svc := newTestService()
	svc.Notifications.Slack.Enabled = false
	svc.Notifications.Slack.Addresses = nil
	svc.Notifications.Email.Enabled = true
	svc.Notifications.Email.Addresses = []string{"ops@example.com"}

	u.sendNotifications(svc, "subj", "body")

	if len(mock.EmailsSent) != 1 || len(mock.EmailsSent[0]) != 1 || mock.EmailsSent[0][0] != "ops@example.com" {
		t.Fatalf("email notifier: got %#v", mock.EmailsSent)
	}
	if len(mock.SlackSent) != 0 {
		t.Fatalf("expected no slack, got %v", mock.SlackSent)
	}
}

func TestSendNotifications_MultipleSlackWebhooks(t *testing.T) {
	mock := &MockNotifier{}
	u := &UptimeTracker{notifier: mock}

	svc := newTestService()
	svc.Notifications.Email.Enabled = false
	svc.Notifications.Slack.Addresses = []string{"https://hooks.slack/a", "https://hooks.slack/b"}

	u.sendNotifications(svc, "s", "m")

	if len(mock.SlackSent) != 2 {
		t.Fatalf("want 2 slack posts, got %d (%v)", len(mock.SlackSent), mock.SlackSent)
	}
}

func TestCheckAndNotify_HealthySteadyNoNotification(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	mock := &MockNotifier{}
	u := &UptimeTracker{notifier: mock}

	svc := newTestService()
	svc.URL = srv.URL
	details := ServiceDetails{CheckTimeout: time.Second}

	u.checkAndNotify(svc, &details)
	time.Sleep(100 * time.Millisecond)
	if len(mock.SlackSent) != 0 {
		t.Errorf("expected no notification when already healthy")
	}
}

// -------- TEST: SERVICE DOWN --------
func TestServiceDownSendsNotification(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	mockNotifier := &MockNotifier{}
	u := &UptimeTracker{notifier: mockNotifier}

	svc := newTestService()
	svc.URL = srv.URL
	details := ServiceDetails{CheckTimeout: time.Second}

	u.checkAndNotify(svc, &details)

	waitUntil(t, func() bool { return len(mockNotifier.SlackSent) > 0 })

	if mockNotifier.SlackSent[0] != "http://slack-webhook" {
		t.Errorf("unexpected webhook: %v", mockNotifier.SlackSent[0])
	}
}

// -------- TEST: SERVICE RECOVERY --------
func TestServiceRecoverySendsNotification(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	mockNotifier := &MockNotifier{}
	u := &UptimeTracker{notifier: mockNotifier}

	svc := newTestService()
	svc.URL = srv.URL
	details := ServiceDetails{
		IsDown:       true,
		DownStart:    time.Now().Add(-5 * time.Minute),
		CheckTimeout: time.Second,
	}

	u.checkAndNotify(svc, &details)

	waitUntil(t, func() bool { return len(mockNotifier.SlackSent) > 0 })
}

// -------- TEST: NO DUPLICATE DOWN ALERT --------
func TestNoDuplicateDownAlert(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	mockNotifier := &MockNotifier{}
	u := &UptimeTracker{notifier: mockNotifier}

	svc := newTestService()
	svc.URL = srv.URL
	details := ServiceDetails{IsDown: true, CheckTimeout: time.Second}

	u.checkAndNotify(svc, &details)
	time.Sleep(100 * time.Millisecond)
	if len(mockNotifier.SlackSent) != 0 {
		t.Errorf("expected no duplicate alert")
	}
}
