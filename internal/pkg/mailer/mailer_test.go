package mailer_test

import (
	"os"
	"strconv"
	"testing"

	"golang-base/internal/pkg/mailer"
)

func TestGoMailer_Creation(t *testing.T) {
	cfg := mailer.Config{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "password",
		From:     "noreply@example.com",
	}

	m := mailer.NewGoMailer(cfg)
	if m == nil {
		t.Fatal("expected mailer instance, got nil")
	}

	var _ mailer.Mailer = m
}

func TestMessage_Structure(t *testing.T) {
	msg := mailer.Message{
		To:          []string{"recipient@example.com"},
		CC:          []string{"cc@example.com"},
		BCC:         []string{"bcc@example.com"},
		Subject:     "Test Subject",
		Body:        "<h1>Hello</h1>",
		IsHTML:      true,
		Attachments: []string{"test.pdf"},
	}

	if msg.Subject != "Test Subject" {
		t.Errorf("expected subject 'Test Subject', got %s", msg.Subject)
	}
	if !msg.IsHTML {
		t.Error("expected IsHTML to be true")
	}
	if len(msg.To) != 1 || msg.To[0] != "recipient@example.com" {
		t.Errorf("unexpected To: %v", msg.To)
	}
	if len(msg.CC) != 1 || msg.CC[0] != "cc@example.com" {
		t.Errorf("unexpected CC: %v", msg.CC)
	}
	if len(msg.BCC) != 1 || msg.BCC[0] != "bcc@example.com" {
		t.Errorf("unexpected BCC: %v", msg.BCC)
	}
	if len(msg.Attachments) != 1 || msg.Attachments[0] != "test.pdf" {
		t.Errorf("unexpected Attachments: %v", msg.Attachments)
	}
}

func TestMessage_PlainText(t *testing.T) {
	msg := mailer.Message{
		To:      []string{"recipient@example.com"},
		Subject: "Plain Text Email",
		Body:    "Hello, this is plain text.",
		IsHTML:  false,
	}

	if msg.IsHTML {
		t.Error("expected IsHTML to be false")
	}
	if msg.Body != "Hello, this is plain text." {
		t.Errorf("unexpected Body: %s", msg.Body)
	}
}

func TestGoMailer_Ping_InvalidHost(t *testing.T) {
	cfg := mailer.Config{
		Host:     "127.0.0.1",
		Port:     1, // Unused port that will fail connection
		Username: "user",
		Password: "password",
		From:     "noreply@example.com",
	}

	m := mailer.NewGoMailer(cfg)
	err := m.Ping()
	if err == nil {
		t.Error("expected ping to fail for invalid host/port, got nil")
	}
}

func TestGoMailer_WithEnvConfig(t *testing.T) {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		t.Skip("SMTP_HOST not set, skipping env config test")
	}

	portStr := os.Getenv("SMTP_PORT")
	port, _ := strconv.Atoi(portStr)
	if port == 0 {
		port = 587
	}

	user := os.Getenv("SMTP_USERNAME")
	if user == "" {
		user = os.Getenv("SMTP_USER")
	}

	pass := os.Getenv("SMTP_PASSWORD")
	if pass == "" {
		pass = os.Getenv("SMTP_PASS")
	}

	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = "test@example.com"
	}

	cfg := mailer.Config{
		Host:     host,
		Port:     port,
		Username: user,
		Password: pass,
		From:     from,
	}

	m := mailer.NewGoMailer(cfg)
	if m == nil {
		t.Fatal("expected mailer instance from env config, got nil")
	}
}
