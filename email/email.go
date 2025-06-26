package email

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"flugo.com/logger"
)

type EmailConfig struct {
	SMTPHost   string `json:"smtp_host"`
	SMTPPort   int    `json:"smtp_port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	FromEmail  string `json:"from_email"`
	FromName   string `json:"from_name"`
	ReplyTo    string `json:"reply_to"`
	EnableSSL  bool   `json:"enable_ssl"`
	EnableAuth bool   `json:"enable_auth"`
}

type Email struct {
	To          []string
	CC          []string
	BCC         []string
	Subject     string
	Body        string
	HTMLBody    string
	Attachments []Attachment
	Headers     map[string]string
}

type Attachment struct {
	Filename string
	Content  []byte
	MimeType string
}

type EmailService struct {
	config *EmailConfig
	auth   smtp.Auth
}

var DefaultEmailService *EmailService

func Init(cfg *EmailConfig) {
	DefaultEmailService = NewEmailService(cfg)
}

func NewEmailService(cfg *EmailConfig) *EmailService {
	var auth smtp.Auth
	if cfg.EnableAuth {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	}

	return &EmailService{
		config: cfg,
		auth:   auth,
	}
}

func (es *EmailService) Send(email *Email) error {
	if len(email.To) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	message := es.buildMessage(email)

	addr := fmt.Sprintf("%s:%d", es.config.SMTPHost, es.config.SMTPPort)
	recipients := append(email.To, email.CC...)
	recipients = append(recipients, email.BCC...)

	err := smtp.SendMail(addr, es.auth, es.config.FromEmail, recipients, message)
	if err != nil {
		logger.Error("Failed to send email: %v", err)
		return err
	}

	logger.Info("Email sent successfully to %v", email.To)
	return nil
}

func (es *EmailService) buildMessage(email *Email) []byte {
	var buffer bytes.Buffer

	// Headers
	buffer.WriteString(fmt.Sprintf("From: %s <%s>\r\n", es.config.FromName, es.config.FromEmail))
	buffer.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ", ")))

	if len(email.CC) > 0 {
		buffer.WriteString(fmt.Sprintf("CC: %s\r\n", strings.Join(email.CC, ", ")))
	}

	if es.config.ReplyTo != "" {
		buffer.WriteString(fmt.Sprintf("Reply-To: %s\r\n", es.config.ReplyTo))
	}

	buffer.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))

	// Custom headers
	for key, value := range email.Headers {
		buffer.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	buffer.WriteString("MIME-Version: 1.0\r\n")

	if email.HTMLBody != "" {
		// HTML email
		buffer.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		buffer.WriteString(email.HTMLBody)
	} else {
		// Plain text email
		buffer.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		buffer.WriteString(email.Body)
	}

	return buffer.Bytes()
}

func (es *EmailService) SendTemplate(templateName string, data interface{}, email *Email) error {
	tmpl, err := template.New(templateName).Parse(getTemplate(templateName))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	email.HTMLBody = buf.String()
	return es.Send(email)
}

func getTemplate(name string) string {
	templates := map[string]string{
		"welcome": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #007bff; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f8f9fa; }
        .footer { padding: 20px; text-align: center; color: #666; }
        .btn { display: inline-block; padding: 10px 20px; background: #007bff; color: white; text-decoration: none; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to {{.AppName}}!</h1>
        </div>
        <div class="content">
            <h2>Hello {{.Name}},</h2>
            <p>Thank you for joining {{.AppName}}. We're excited to have you on board!</p>
            <p>{{.Message}}</p>
            <p><a href="{{.ActivationLink}}" class="btn">Activate Your Account</a></p>
        </div>
        <div class="footer">
            <p>&copy; 2024 {{.AppName}}. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,

		"reset_password": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Reset Password</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #dc3545; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f8f9fa; }
        .footer { padding: 20px; text-align: center; color: #666; }
        .btn { display: inline-block; padding: 10px 20px; background: #dc3545; color: white; text-decoration: none; border-radius: 5px; }
        .warning { background: #fff3cd; border: 1px solid #ffeaa7; padding: 10px; border-radius: 5px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Reset Your Password</h1>
        </div>
        <div class="content">
            <h2>Hello {{.Name}},</h2>
            <p>We received a request to reset your password for your {{.AppName}} account.</p>
            <div class="warning">
                <strong>Important:</strong> This link will expire in {{.ExpirationTime}} minutes.
            </div>
            <p><a href="{{.ResetLink}}" class="btn">Reset Password</a></p>
            <p>If you didn't request this password reset, please ignore this email.</p>
        </div>
        <div class="footer">
            <p>&copy; 2024 {{.AppName}}. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,

		"notification": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.Title}}</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #28a745; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f8f9fa; }
        .footer { padding: 20px; text-align: center; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Title}}</h1>
        </div>
        <div class="content">
            <h2>Hello {{.Name}},</h2>
            <p>{{.Message}}</p>
            {{if .ActionURL}}
            <p><a href="{{.ActionURL}}" style="display: inline-block; padding: 10px 20px; background: #28a745; color: white; text-decoration: none; border-radius: 5px;">{{.ActionText}}</a></p>
            {{end}}
        </div>
        <div class="footer">
            <p>&copy; 2024 {{.AppName}}. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,
	}

	if tmpl, exists := templates[name]; exists {
		return tmpl
	}
	return templates["notification"]
}

func Send(email *Email) error {
	if DefaultEmailService == nil {
		return fmt.Errorf("email service not initialized")
	}
	return DefaultEmailService.Send(email)
}

func SendTemplate(templateName string, data interface{}, email *Email) error {
	if DefaultEmailService == nil {
		return fmt.Errorf("email service not initialized")
	}
	return DefaultEmailService.SendTemplate(templateName, data, email)
}

func SendWelcome(to, name, appName, activationLink string) error {
	data := map[string]interface{}{
		"Name":           name,
		"AppName":        appName,
		"ActivationLink": activationLink,
		"Message":        "Your account has been created successfully. Please activate it by clicking the button below.",
	}

	email := &Email{
		To:      []string{to},
		Subject: fmt.Sprintf("Welcome to %s", appName),
	}

	return SendTemplate("welcome", data, email)
}

func SendPasswordReset(to, name, appName, resetLink string, expirationMinutes int) error {
	data := map[string]interface{}{
		"Name":           name,
		"AppName":        appName,
		"ResetLink":      resetLink,
		"ExpirationTime": expirationMinutes,
	}

	email := &Email{
		To:      []string{to},
		Subject: "Reset Your Password",
	}

	return SendTemplate("reset_password", data, email)
}

func SendNotification(to, name, title, message, appName string) error {
	data := map[string]interface{}{
		"Name":    name,
		"Title":   title,
		"Message": message,
		"AppName": appName,
	}

	email := &Email{
		To:      []string{to},
		Subject: title,
	}

	return SendTemplate("notification", data, email)
}

func SendBulk(emails []*Email) error {
	if DefaultEmailService == nil {
		return fmt.Errorf("email service not initialized")
	}

	for i, email := range emails {
		if err := DefaultEmailService.Send(email); err != nil {
			logger.Error("Failed to send bulk email %d: %v", i, err)
			return err
		}
	}

	logger.Info("Successfully sent %d bulk emails", len(emails))
	return nil
}

func ValidateEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

func TestConnection() error {
	if DefaultEmailService == nil {
		return fmt.Errorf("email service not initialized")
	}

	addr := fmt.Sprintf("%s:%d", DefaultEmailService.config.SMTPHost, DefaultEmailService.config.SMTPPort)
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	if DefaultEmailService.config.EnableAuth {
		if err := client.Auth(DefaultEmailService.auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	logger.Info("Email service connection test successful")
	return nil
}
