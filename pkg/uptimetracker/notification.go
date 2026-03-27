package uptimetracker

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	smtpclient "github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/config"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// slackPostTimeout bounds each outbound Slack webhook request.
const slackPostTimeout = 30 * time.Second

// -------- INTERFACE --------
type Notifier interface {
	SendEmail(to []string, subject, body string)
	SendSlack(webhook, msg string)
}

// -------- NOTIFICATION CLIENT --------
type NotificationClient struct {
	logger *utils.ErrorLogger
	smtp   *smtpclient.SMTPClient
}

// -------- CONSTRUCTOR --------
func NewNotificationClient() *NotificationClient {

	host, port, user, pass := config.GetSMTPConfig()

	smtpClient := smtpclient.NewSMTPClient(smtpclient.SMTPConfig{
		Host:     host,
		Port:     port,
		Username: user,
		Password: pass,
	})

	return &NotificationClient{
		logger: utils.NewErrorLogger("uptime-notification"),
		smtp:   smtpClient,
	}
}

// -------- EMAIL --------
func (n *NotificationClient) SendEmail(to []string, subject, body string) {

	msg := &smtpclient.SMTPMessage{
		From:    n.smtp.Config.Username,
		To:      to,
		Subject: subject,
		Body:    body,
	}

	err := n.smtp.Send(msg)
	if err != nil {
		n.logger.LogError(err, "Failed to send email", map[string]interface{}{
			"to": to,
		})
		return
	}

	n.logger.LogInfo("Email sent successfully", map[string]interface{}{
		"to": to,
	})
}

// -------- SLACK --------
func (n *NotificationClient) SendSlack(webhook, msg string) {

	payload := map[string]interface{}{
		"text":       "@here " + msg,
		"link_names": true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		n.logger.LogError(err, "Failed to marshal Slack payload", nil)
		return
	}

	resp, err := postSlackWebhook(webhook, jsonData)
	if err != nil {
		n.logger.LogError(err, "Failed to send Slack notification", map[string]interface{}{
			"webhook": webhook,
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		n.logger.LogError(nil, "Slack notification failed", map[string]interface{}{
			"status":  resp.Status,
			"webhook": webhook,
		})
		return
	}

	n.logger.LogInfo("Slack notification sent", map[string]interface{}{
		"webhook": webhook,
	})
}

func postSlackWebhook(webhook string, jsonBody []byte) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), slackPostTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return http.DefaultClient.Do(req)
}
