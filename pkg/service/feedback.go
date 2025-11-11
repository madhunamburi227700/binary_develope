package service

import (
	"context"
	"fmt"

	"github.com/opsmx/ai-guardian-api/pkg/config"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type FileAttachment struct {
	Filename    string
	Content     []byte
	ContentType string
	Size        int64
}

type SendFeedbackRequest struct {
	UserEmail   string
	Message     string
	Attachments []FileAttachment
}

type FeedbackService struct {
	logger          *utils.ErrorLogger
	notificationSvc Notifier
}

func NewFeedbackService() *FeedbackService {
	return &FeedbackService{
		logger:          utils.NewErrorLogger("feedback_service"),
		notificationSvc: NewEmailNotifier(),
	}
}

type EmailTemplateData struct {
	UserEmail string
	Message   string
}

// SendFeedback sends feedback email to admin users
func (s *FeedbackService) SendFeedback(ctx context.Context, req *SendFeedbackRequest) error {
	// Get feedback configuration
	emailSubject, emailBodyTemplate, adminEmails := config.GetFeedbackConfig()

	if len(adminEmails) == 0 {
		s.logger.LogWarning("Feedback email not configured", map[string]interface{}{
			"user_email":        req.UserEmail,
			"attachments_count": len(req.Attachments),
		})
		return fmt.Errorf("feedback email not configured")
	}

	// Default subject and body if not configured
	if emailSubject == "" {
		emailSubject = "New Feedback Received"
	}
	if emailBodyTemplate == "" {
		emailBodyTemplate = "New feedback received from {{.UserEmail}}\n\nMessage:\n{{.Message}}"
	}

	// Convert attachments
	var attachments []Attachment
	for _, att := range req.Attachments {
		attachments = append(attachments, Attachment{
			Filename:    att.Filename,
			Content:     att.Content,
			ContentType: att.ContentType,
		})
	}

	// Send email using notification service
	err := s.notificationSvc.Notify(ctx, &EmailNotification{
		To:       adminEmails,
		Subject:  emailSubject,
		Template: emailBodyTemplate,
		TemplateData: EmailTemplateData{
			UserEmail: req.UserEmail,
			Message:   req.Message,
		},
		Attachments: attachments,
	})
	if err != nil {
		s.logger.LogError(err, "Failed to send feedback email", nil)
	}

	return nil
}
