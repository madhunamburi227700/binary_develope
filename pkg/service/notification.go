package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/config"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type NotificationService struct {
	notifier Notifier
	logger   *utils.ErrorLogger
}

func NewNotificationService(notifier Notifier) *NotificationService {
	return &NotificationService{
		notifier: notifier,
		logger:   utils.NewErrorLogger("notification_service"),
	}
}

// NotifyScanCompletion sends a plain-text email notification detailing the completed scan.
func (s *NotificationService) NotifyScanCompletion(ctx context.Context, notifyTo, projectID, repository, branch, commitSHA string) error {
	_, recipients := config.GetNotificationConfig()

	if notifyTo == "" {
		return errors.New("no email recipient found to notify to")
	}

	recipients = append(recipients, notifyTo)

	data := struct {
		ProjectID  string
		Repository string
		Branch     string
		CommitSHA  string
	}{
		ProjectID:  projectID,
		Repository: repository,
		Branch:     branch,
		CommitSHA:  commitSHA,
	}

	emailTemplate := `
The security scan has been completed for the following:

Details:
ProjectID:  {{.ProjectID}}
Repository: {{.Repository}}
Branch:     {{.Branch}}
CommitSHA:  {{.CommitSHA}}

Please check the latest results in the AI Guardian dashboard.

- AI Guardian Team
`

	notification := &EmailNotification{
		To:           recipients,
		Subject:      fmt.Sprintf("AI Guardian - Scan Completed - %s/%s", repository, branch),
		Template:     emailTemplate,
		TemplateData: data,
		Attachments:  nil,
	}

	if err := s.notifier.Notify(ctx, notification); err != nil {
		return err
	}
	return nil
}

type Notifier interface {
	Notify(ctx context.Context, message any) error
}

// EmailNotification represents an email notification request
type EmailNotification struct {
	To           []string
	Subject      string
	Body         string
	Template     string
	TemplateData interface{}
	Attachments  []Attachment
}

// Attachment represents a file attachment
type Attachment struct {
	Filename    string
	Content     []byte
	ContentType string
}

// EmailNotifier handles email notifications
type EmailNotifier struct {
	client *client.SMTPClient
}

func NewEmailNotifier() *EmailNotifier {
	smtpHost, smtpPort, smtpUser, smtpPassword := config.GetSMTPConfig()
	return &EmailNotifier{
		client: client.NewSMTPClient(
			client.SMTPConfig{
				Host:     smtpHost,
				Port:     smtpPort,
				Username: smtpUser,
				Password: smtpPassword,
			},
		),
	}
}

// Send sends an email notification
func (s *EmailNotifier) Notify(ctx context.Context, message any) error {
	if s.client.Config.Host == "" || s.client.Config.Port == "" {
		return fmt.Errorf("smtp not configured")
	}

	notification, ok := message.(*EmailNotification)
	if !ok {
		return fmt.Errorf("invalid notification type expected EmailNotification got %T", notification)
	}

	var body string
	if notification.Template != "" {
		// Parse email template
		tmpl, err := template.New("email").Parse(notification.Template)
		if err != nil {
			return fmt.Errorf("failed to parse email template: %w", err)
		}

		var bodyBuf bytes.Buffer
		if err := tmpl.Execute(&bodyBuf, notification.TemplateData); err != nil {
			return fmt.Errorf("failed to execute email template: %w", err)
		}
		body = bodyBuf.String()
	} else {
		body = notification.Body
	}

	// Convert attachments
	var smtpAttachments []client.SMTPAttachment
	for _, att := range notification.Attachments {
		smtpAttachments = append(smtpAttachments, client.SMTPAttachment{
			Filename:    att.Filename,
			Content:     att.Content,
			ContentType: att.ContentType,
		})
	}

	smtpMessage := client.SMTPMessage{
		From:        s.client.Config.Username,
		To:          notification.To,
		Subject:     notification.Subject,
		Body:        body,
		Attachments: smtpAttachments,
	}

	if err := s.client.Send(&smtpMessage); err != nil {
		return err
	}

	return nil
}
