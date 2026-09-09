package integration

import (
	"os"
	"testing"

	"golang-base/internal/pkg/mailer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestMailer(t *testing.T) mailer.Mailer {
	t.Helper()

	if TestConfig == nil || TestConfig.SMTPHost == "" || TestConfig.SMTPPassword == "" {
		t.Skip("SMTP credentials not configured, skipping SMTP integration tests")
	}

	from := TestConfig.SMTPFrom
	if from == "" {
		from = "no-reply@example.com"
	}

	cfg := mailer.Config{
		Host:     TestConfig.SMTPHost,
		Port:     TestConfig.SMTPPort,
		Username: TestConfig.SMTPUsername,
		Password: TestConfig.SMTPPassword,
		From:     from,
	}

	m := mailer.NewGoMailer(cfg)
	require.NotNil(t, m, "expected mailer instance")
	return m
}

func TestMailer_E2E_Ping(t *testing.T) {
	m := getTestMailer(t)

	err := m.Ping()
	assert.NoError(t, err, "SMTP ping / handshake should succeed")
}

func TestMailer_E2E_SendEmail(t *testing.T) {
	m := getTestMailer(t)

	recipient := os.Getenv("SMTP_TEST_TO")
	if recipient == "" {
		// Fall back to super admin email, default test email, or SMTP from
		recipient = os.Getenv("SUPER_ADMIN_EMAIL")
		if recipient == "" {
			recipient = "taufiqqurrohman333@gmail.com"
		}
	}

	msg := &mailer.Message{
		To:      []string{recipient},
		Subject: "E2E Test: Golang Base Mailer Verification",
		Body: `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
  <h2 style="color: #2563eb;">E2E Mailer Test Passed</h2>
  <p>Halo, ini adalah email verifikasi otomatis dari suite <strong>E2E Integration Test</strong> di Golang Base.</p>
  <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 16px 0;" />
  <p style="color: #6b7280; font-size: 13px;">Sent via Resend SMTP</p>
</body>
</html>`,
		IsHTML: true,
	}

	err := m.Send(msg)
	assert.NoError(t, err, "sending email via configured SMTP should succeed")
}
