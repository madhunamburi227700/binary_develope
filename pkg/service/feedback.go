package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strings"
	"text/template"

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
	logger *utils.ErrorLogger
}

func NewFeedbackService() *FeedbackService {
	return &FeedbackService{
		logger: utils.NewErrorLogger("feedback_service"),
	}
}

type EmailTemplateData struct {
	UserEmail string
	Message   string
}

// SendFeedback sends feedback email to admin users
func (s *FeedbackService) SendFeedback(ctx context.Context, req *SendFeedbackRequest) error {
	// Get feedback configuration
	smtpHost, smtpPort, smtpUser, smtpPassword, emailSubject, emailBodyTemplate, adminEmails := config.GetFeedbackConfig()

	if smtpHost == "" || smtpPort == "" || len(adminEmails) == 0 {
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

	// Parse email body template
	tmpl, err := template.New("email").Parse(emailBodyTemplate)
	if err != nil {
		s.logger.LogError(err, "Failed to parse email template", nil)
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var bodyBuf bytes.Buffer
	templateData := EmailTemplateData{
		UserEmail: req.UserEmail,
		Message:   req.Message,
	}
	if err := tmpl.Execute(&bodyBuf, templateData); err != nil {
		s.logger.LogError(err, "Failed to execute email template", nil)
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	// Send email to each admin
	for _, adminEmail := range adminEmails {
		message := s.buildEmailMessage(smtpUser, adminEmail, emailSubject, bodyBuf.String(), req.Attachments)
		
		if err := s.sendEmail(smtpHost, smtpPort, smtpUser, smtpPassword, adminEmail, message); err != nil {
			s.logger.LogError(err, "Failed to send email", map[string]interface{}{
				"admin_email": adminEmail,
				"user_email":  req.UserEmail,
			})
			continue
		}
		
		s.logger.LogInfo("Feedback email sent", map[string]interface{}{
			"admin_email":       adminEmail,
			"user_email":        req.UserEmail,
			"attachments_count": len(req.Attachments),
		})
	}

	return nil
}

// buildEmailMessage constructs the email message with attachments
func (s *FeedbackService) buildEmailMessage(from, to, subject, body string, attachments []FileAttachment) []byte {
	boundary := "----=_NextPart_000_0000_01D00000.00000000"
	
	var builder strings.Builder
	
	// Email headers
	builder.WriteString(fmt.Sprintf("From: %s\r\n", from))
	builder.WriteString(fmt.Sprintf("To: %s\r\n", to))
	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	builder.WriteString("MIME-Version: 1.0\r\n")
	builder.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
	builder.WriteString("\r\n")

	// Email body
	builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	builder.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	builder.WriteString("Content-Transfer-Encoding: 7bit\r\n")
	builder.WriteString("\r\n")
	builder.WriteString(body)
	builder.WriteString("\r\n\r\n")

	// Add attachments
	for _, attachment := range attachments {
		builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		builder.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", 
			attachment.ContentType, attachment.Filename))
		builder.WriteString("Content-Transfer-Encoding: base64\r\n")
		builder.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", 
			attachment.Filename))
		builder.WriteString("\r\n")
		
		// Encode content to base64
		encoded := base64.StdEncoding.EncodeToString(attachment.Content)
		builder.WriteString(encoded)
		builder.WriteString("\r\n")
	}

	// End boundary
	builder.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	return []byte(builder.String())
}

// sendEmail sends the email using SMTP
func (s *FeedbackService) sendEmail(host, port, user, password, to string, message []byte) error {
	addr := fmt.Sprintf("%s:%s", host, port)
	
	// Setup authentication
	auth := smtp.PlainAuth("", user, password, host)
	
	// For TLS connections (port 465)
	if port == "465" {
		return s.sendEmailTLS(addr, auth, user, to, message)
	}
	
	// For STARTTLS connections (port 587)
	return smtp.SendMail(addr, auth, user, []string{to}, message)
}

// sendEmailTLS sends email using TLS (for port 465)
func (s *FeedbackService) sendEmailTLS(addr string, auth smtp.Auth, from, to string, message []byte) error {
	// Connect to SMTP server with TLS
	tlsConfig := &tls.Config{
		ServerName: strings.Split(addr, ":")[0],
	}
	
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()
	
	client, err := smtp.NewClient(conn, tlsConfig.ServerName)
	if err != nil {
		return err
	}
	defer client.Close()
	
	// Authenticate
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}
	
	// Set sender
	if err = client.Mail(from); err != nil {
		return err
	}
	
	// Set recipient
	if err = client.Rcpt(to); err != nil {
		return err
	}
	
	// Send message
	w, err := client.Data()
	if err != nil {
		return err
	}
	
	_, err = w.Write(message)
	if err != nil {
		return err
	}
	
	err = w.Close()
	if err != nil {
		return err
	}
	
	return client.Quit()
}
