package uptimetracker

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	smtpclient "github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

func TestNewNotificationClient_NotNil(t *testing.T) {
	c := NewNotificationClient()
	if c == nil || c.logger == nil || c.smtp == nil {
		t.Fatal("NewNotificationClient returned incomplete client")
	}
}

func TestPostSlackWebhook_InvalidURL(t *testing.T) {
	_, err := postSlackWebhook(":", []byte(`{"text":"x"}`))
	if err == nil {
		t.Fatal("expected error for invalid webhook URL")
	}
}

// -------- TEST EMAIL (BASIC EXECUTION TEST) --------
func TestSendEmail_NoPanic(t *testing.T) {

	client := &NotificationClient{
		logger: utils.NewErrorLogger("test"),
		smtp: &smtpclient.SMTPClient{
			Config: smtpclient.SMTPConfig{
				Host:     "invalid-host",
				Port:     "123",
				Username: "user",
				Password: "pass",
			},
		},
	}

	// We can't mock SMTP, so just ensure it doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SendEmail panicked: %v", r)
		}
	}()

	client.SendEmail([]string{"test@mail.com"}, "subject", "body")
}

// -------- TEST SLACK SUCCESS --------
func TestSendSlack_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type %q, want application/json", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["text"] != "test message" {
			t.Errorf("payload text %q, want %q", payload["text"], "test message")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &NotificationClient{logger: utils.NewErrorLogger("test")}
	client.SendSlack(srv.URL, "test message")
}

// -------- TEST SLACK REQUEST ERROR (e.g. connection refused) --------
func TestSendSlack_RequestError(t *testing.T) {
	client := &NotificationClient{logger: utils.NewErrorLogger("test")}
	// Port 1 is typically no listener; expect Do() to fail.
	client.SendSlack("http://127.0.0.1:1/webhook", "test message")
}

// -------- TEST SLACK NON-2XX RESPONSE --------
func TestSendSlack_BadHTTPStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := &NotificationClient{logger: utils.NewErrorLogger("test")}
	client.SendSlack(srv.URL, "test message")
}

// -------- TEST SLACK NO PANIC --------
func TestSendSlack_NoPanic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &NotificationClient{logger: utils.NewErrorLogger("test")}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SendSlack panicked: %v", r)
		}
	}()

	client.SendSlack(srv.URL, "test message")
}
