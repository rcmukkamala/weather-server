package notification

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"time"

	"github.com/smukkama/weather-server/internal/protocol"
	"github.com/smukkama/weather-server/pkg/config"
)

// EmailNotifier sends email notifications
type EmailNotifier struct {
	config *config.SMTPConfig
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier(cfg *config.SMTPConfig) *EmailNotifier {
	return &EmailNotifier{config: cfg}
}

// SendAlarmNotification sends an email for an alarm notification
func (e *EmailNotifier) SendAlarmNotification(notification *protocol.AlarmNotification) error {
	var subject string
	var body string
	var err error

	switch notification.Type {
	case protocol.AlarmTypeTriggered:
		subject = fmt.Sprintf("🚨 Weather Alarm TRIGGERED - %s, %s", notification.City, notification.Zipcode)
		body, err = e.renderTriggeredTemplate(notification)
	case protocol.AlarmTypeCleared:
		subject = fmt.Sprintf("✅ Weather Alarm CLEARED - %s, %s", notification.City, notification.Zipcode)
		body, err = e.renderClearedTemplate(notification)
	default:
		return fmt.Errorf("unknown notification type: %s", notification.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return e.sendEmail(subject, body)
}

func (e *EmailNotifier) renderTriggeredTemplate(notification *protocol.AlarmNotification) (string, error) {
	tmpl := `
Weather Alarm Triggered
=======================

Location: {{.City}}, {{.Zipcode}}
Metric: {{.Metric}}
Current Value: {{.Value}}
Threshold: {{.Operator}} {{.Threshold}}
Duration: {{.Duration}} minutes
Start Time: {{.StartTime}}
Alarm ID: {{.AlarmID}}

Description:
The {{.Metric}} at {{.City}} ({{.Zipcode}}) has breached the threshold 
({{.Operator}} {{.Threshold}}) for {{.Duration}} minutes. The current value 
is {{.Value}}.

This alarm was triggered at {{.StartTime}}.

Please take appropriate action.

---
Weather Server Notification System
`

	t, err := template.New("triggered").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, notification); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (e *EmailNotifier) renderClearedTemplate(notification *protocol.AlarmNotification) (string, error) {
	tmpl := `
Weather Alarm Cleared
=====================

Location: {{.City}}, {{.Zipcode}}
Metric: {{.Metric}}
Alarm ID: {{.AlarmID}}

Description:
The alarm for {{.Metric}} at {{.City}} ({{.Zipcode}}) has been cleared.
The metric has returned to normal levels.

---
Weather Server Notification System
`

	t, err := template.New("cleared").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, notification); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (e *EmailNotifier) sendEmail(subject, body string) error {
	// Skip sending if SMTP is not configured
	if e.config.Username == "" || e.config.Password == "" {
		fmt.Printf("SMTP not configured, skipping email:\nSubject: %s\n%s\n", subject, body)
		return nil
	}

	// Construct message
	message := fmt.Sprintf("From: %s\r\n", e.config.From)
	message += fmt.Sprintf("To: %s\r\n", e.config.To)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	message += "\r\n"
	message += body

	// Setup authentication
	auth := smtp.PlainAuth("", e.config.Username, e.config.Password, e.config.Host)

	// Send email
	addr := fmt.Sprintf("%s:%d", e.config.Host, e.config.Port)
	err := smtp.SendMail(addr, auth, e.config.From, []string{e.config.To}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Printf("Email sent successfully: %s\n", subject)
	return nil
}

// TestConnection tests the SMTP connection
func (e *EmailNotifier) TestConnection() error {
	if e.config.Username == "" {
		return fmt.Errorf("SMTP not configured")
	}

	// Try to connect
	addr := fmt.Sprintf("%s:%d", e.config.Host, e.config.Port)
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	fmt.Println("SMTP connection test successful")
	return nil
}
