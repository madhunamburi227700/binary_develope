package client

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/rs/zerolog/log"
)

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
}

type SMTPAttachment struct {
	Filename    string
	Content     []byte
	ContentType string
}

type SMTPMessage struct {
	From        string
	To          []string
	Subject     string
	Body        string
	Attachments []SMTPAttachment
}

type SMTPClient struct {
	Config SMTPConfig
}

func NewSMTPClient(config SMTPConfig) *SMTPClient {
	return &SMTPClient{
		Config: config,
	}
}

// TODO: Add concurrency to it later
// Send sends an email using the configured SMTP server
func (c *SMTPClient) Send(message *SMTPMessage) error {

	// For each recipient
	var failed []string
	for _, to := range message.To {
		// Build email content
		content := c.buildMessage(to, message)

		if err := c.sendToRecipient(to, content); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", to, err))
			continue
		}
		log.Info().Msgf("Sucessfully sent an email to: %s", to)
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed to send email to: %v", failed)
	}

	return nil
}

func (c *SMTPClient) buildMessage(to string, message *SMTPMessage) []byte {
	boundary := "----=_NextPart_000_0000_01D00000.00000000"

	var builder strings.Builder

	// Email headers
	builder.WriteString(fmt.Sprintf("From: %s\r\n", message.From))
	builder.WriteString(fmt.Sprintf("To: %s\r\n", to))
	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", message.Subject))
	builder.WriteString("MIME-Version: 1.0\r\n")
	builder.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
	builder.WriteString("\r\n")

	// Email body
	builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	builder.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	builder.WriteString("Content-Transfer-Encoding: 7bit\r\n")
	builder.WriteString("\r\n")
	builder.WriteString(message.Body)
	builder.WriteString("\r\n\r\n")

	// Add attachments
	for _, attachment := range message.Attachments {
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

func (c *SMTPClient) sendToRecipient(to string, message []byte) error {
	addr := fmt.Sprintf("%s:%s", c.Config.Host, c.Config.Port)

	// Setup authentication
	auth := smtp.PlainAuth("", c.Config.Username, c.Config.Password, c.Config.Host)

	// For TLS connections (port 465)
	if c.Config.Port == "465" {
		return c.sendTLS(addr, auth, to, message)
	}

	// For STARTTLS connections (port 587)
	return smtp.SendMail(addr, auth, c.Config.Username, []string{to}, message)
}

func (c *SMTPClient) sendTLS(addr string, auth smtp.Auth, to string, message []byte) error {
	// Connect to SMTP server with TLS
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		ServerName: strings.Split(addr, ":")[0],
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("tls dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, tlsConfig.ServerName)
	if err != nil {
		return fmt.Errorf("smtp client creation failed: %w", err)
	}
	defer client.Close()

	// Authenticate
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("smtp authentication failed: %w", err)
		}
	}

	// Set sender
	if err = client.Mail(c.Config.Username); err != nil {
		return fmt.Errorf("smtp set sender failed: %w", err)
	}

	// Set recipient
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp set recipient failed: %w", err)
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data command failed: %w", err)
	}

	_, err = w.Write(message)
	if err != nil {
		return fmt.Errorf("smtp write message failed: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("smtp data close failed: %w", err)
	}

	return client.Quit()
}
