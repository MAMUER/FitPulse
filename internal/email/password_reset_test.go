package email

import (
	"context"
	"errors"
	"net/smtp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendPasswordResetEmail_ContextCancelled(t *testing.T) {
	client := NewSMTPClient(Config{Host: "localhost", Port: 1025, From: "test@test.com"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.SendPasswordResetEmail(ctx, "user@example.com", "123456", "http://localhost")
	require.Error(t, err)
	assert.Equal(t, 0, client.dailySent)
}

func TestSendPasswordResetEmail_InvalidAddress(t *testing.T) {
	client := NewSMTPClient(Config{Host: "localhost", Port: 1025, From: "test@test.com"})

	err := client.SendPasswordResetEmail(context.Background(), "not-an-email", "123456", "http://localhost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid recipient address")
}

func TestSendPasswordResetEmail_SkipDomain(t *testing.T) {
	client := NewSMTPClient(Config{
		Host: "localhost", Port: 1025, From: "test@test.com",
		SkipSendDomains: []string{"test.local"},
	})

	err := client.SendPasswordResetEmail(context.Background(), "user@test.local", "123456", "http://localhost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "skipped")
}

func TestSendPasswordResetEmail_DailyLimit(t *testing.T) {
	client := NewSMTPClient(Config{
		Host: "localhost", Port: 1025, From: "test@test.com", DailyLimit: 1,
	})
	client.dailySent = 1

	err := client.SendPasswordResetEmail(context.Background(), "user@example.com", "123456", "http://localhost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "daily email limit exceeded")
}

func TestSendPasswordResetEmail_PlainSuccess(t *testing.T) {
	client := NewSMTPClient(Config{Host: "localhost", Port: 1025, From: "test@test.com"})

	origSend := smtpSendMailFn
	var gotAddr, gotFrom, gotMsg string
	var gotTo []string
	smtpSendMailFn = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		gotAddr, gotFrom, gotTo, gotMsg = addr, from, to, string(msg)
		return nil
	}
	t.Cleanup(func() { smtpSendMailFn = origSend })

	require.NoError(t, client.SendPasswordResetEmail(context.Background(), "user@example.com", "123456", "http://localhost"))

	assert.Equal(t, "localhost:1025", gotAddr)
	assert.Equal(t, "test@test.com", gotFrom)
	require.Len(t, gotTo, 1)
	assert.Equal(t, "user@example.com", gotTo[0])
	assert.Contains(t, gotMsg, "Subject: Восстановление пароля — FitPulse")
	assert.Contains(t, gotMsg, "123456")
	assert.Contains(t, gotMsg, "user@example.com")
	assert.Equal(t, 1, client.dailySent)
}

func TestSendPasswordResetEmail_WithAuth(t *testing.T) {
	client := NewSMTPClient(Config{
		Host: "localhost", Port: 1025, From: "test@test.com",
		User: "smtp-user", Password: "smtp-pass",
	})

	origSend := smtpSendMailFn
	var gotAuth smtp.Auth
	smtpSendMailFn = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		gotAuth = auth
		return nil
	}
	t.Cleanup(func() { smtpSendMailFn = origSend })

	require.NoError(t, client.SendPasswordResetEmail(context.Background(), "user@example.com", "123456", "http://localhost"))
	assert.NotNil(t, gotAuth)
}

func TestSendPasswordResetEmail_SendError(t *testing.T) {
	client := NewSMTPClient(Config{Host: "localhost", Port: 1025, From: "test@test.com"})

	origSend := smtpSendMailFn
	smtpSendMailFn = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		return errors.New("dial failed")
	}
	t.Cleanup(func() { smtpSendMailFn = origSend })

	err := client.SendPasswordResetEmail(context.Background(), "user@example.com", "123456", "http://localhost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send mail")
	assert.Equal(t, 0, client.dailySent) // not incremented on failure
}

func TestSendPasswordResetEmail_TLS(t *testing.T) {
	client := NewSMTPClient(Config{
		Host: "localhost", Port: 1025, From: "test@test.com", UseTLS: true,
	})

	origTLS := sendWithTLSImpl
	var called bool
	sendWithTLSImpl = func(ctx context.Context, c Config, addr, toEmail, msg string) error {
		called = true
		return nil
	}
	t.Cleanup(func() { sendWithTLSImpl = origTLS })

	require.NoError(t, client.SendPasswordResetEmail(context.Background(), "user@example.com", "123456", "http://localhost"))
	assert.True(t, called)
}

func TestBuildPasswordResetHTML(t *testing.T) {
	html := buildPasswordResetHTML("user@example.com", "123456")
	assert.True(t, strings.Contains(html, "<style>"))
	assert.Contains(t, html, "user@example.com")
	assert.Contains(t, html, "123456")
}

func TestBuildPasswordResetHTML_EscapesInput(t *testing.T) {
	html := buildPasswordResetHTML("a<b>@example.com", "<script>")
	assert.NotContains(t, html, "<script>")
	assert.Contains(t, html, "&lt;script&gt;")
}
