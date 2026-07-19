package email

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/config"
)

func TestLoadConfigDefaults(t *testing.T) {
	require.NoError(t, os.Unsetenv("SMTP_HOST"))
	require.NoError(t, os.Unsetenv("SMTP_PORT"))
	require.NoError(t, os.Unsetenv("SMTP_USER"))
	require.NoError(t, os.Unsetenv("SMTP_PASSWORD"))
	require.NoError(t, os.Unsetenv("SMTP_FROM"))
	require.NoError(t, os.Unsetenv("SMTP_TLS"))
	require.NoError(t, os.Unsetenv("EMAIL_DAILY_LIMIT"))
	require.NoError(t, os.Unsetenv("EMAIL_SKIP_DOMAINS"))

	cfg := LoadConfig()
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 1025, cfg.Port)
	assert.Equal(t, "noreply@fitpulse.app", cfg.From)
	assert.False(t, cfg.UseTLS)
	assert.Equal(t, 0, cfg.DailyLimit)
	assert.Empty(t, cfg.SkipSendDomains)
	assert.Empty(t, cfg.User)
	assert.Empty(t, cfg.Password)
}

func TestLoadConfigFromEnv(t *testing.T) {
	require.NoError(t, os.Setenv("SMTP_HOST", "smtp.yandex.ru"))
	require.NoError(t, os.Setenv("SMTP_PORT", "465"))
	require.NoError(t, os.Setenv("SMTP_USER", "test@yandex.ru"))
	require.NoError(t, os.Setenv("SMTP_PASSWORD", "password"))
	require.NoError(t, os.Setenv("SMTP_FROM", "test@yandex.ru"))
	require.NoError(t, os.Setenv("SMTP_TLS", "true"))
	require.NoError(t, os.Setenv("EMAIL_DAILY_LIMIT", "100"))
	require.NoError(t, os.Setenv("EMAIL_SKIP_DOMAINS", "test.local,example.test"))

	cfg := LoadConfig()
	assert.Equal(t, "smtp.yandex.ru", cfg.Host)
	assert.Equal(t, 465, cfg.Port)
	assert.Equal(t, "test@yandex.ru", cfg.User)
	assert.Equal(t, "password", cfg.Password)
	assert.Equal(t, "test@yandex.ru", cfg.From)
	assert.True(t, cfg.UseTLS)
	assert.Equal(t, 100, cfg.DailyLimit)
	assert.Len(t, cfg.SkipSendDomains, 2)
	assert.Equal(t, "test.local", cfg.SkipSendDomains[0])
	assert.Equal(t, "example.test", cfg.SkipSendDomains[1])

	require.NoError(t, os.Unsetenv("SMTP_HOST"))
	require.NoError(t, os.Unsetenv("SMTP_PORT"))
	require.NoError(t, os.Unsetenv("SMTP_USER"))
	require.NoError(t, os.Unsetenv("SMTP_PASSWORD"))
	require.NoError(t, os.Unsetenv("SMTP_FROM"))
	require.NoError(t, os.Unsetenv("SMTP_TLS"))
	require.NoError(t, os.Unsetenv("EMAIL_DAILY_LIMIT"))
	require.NoError(t, os.Unsetenv("EMAIL_SKIP_DOMAINS"))
}

func TestLoadConfigInvalidPort(t *testing.T) {
	require.NoError(t, os.Setenv("SMTP_PORT", "not-a-number"))
	cfg := LoadConfig()
	assert.Equal(t, 1025, cfg.Port)
	require.NoError(t, os.Unsetenv("SMTP_PORT"))
}

func TestLoadConfigInvalidDailyLimit(t *testing.T) {
	require.NoError(t, os.Setenv("EMAIL_DAILY_LIMIT", "invalid"))
	cfg := LoadConfig()
	assert.Equal(t, 0, cfg.DailyLimit)
	require.NoError(t, os.Unsetenv("EMAIL_DAILY_LIMIT"))
}

func TestSplitCSV(t *testing.T) {
	result := splitCSV("a,b,c")
	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func TestSplitCSVWithSpaces(t *testing.T) {
	result := splitCSV(" a , b , c ")
	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func TestSplitCSVEmpty(t *testing.T) {
	result := splitCSV("")
	assert.Len(t, result, 1)
	assert.Equal(t, "", result[0])
}

func TestGetEnv(t *testing.T) {
	require.NoError(t, os.Setenv("TEST_VAR", "hello"))
	assert.Equal(t, "hello", config.GetEnv("TEST_VAR", "fallback"))
	require.NoError(t, os.Unsetenv("TEST_VAR"))
}

func TestGetEnvFallback(t *testing.T) {
	require.NoError(t, os.Unsetenv("NONEXISTENT_VAR_XYZ"))
	assert.Equal(t, "fallback", config.GetEnv("NONEXISTENT_VAR_XYZ", "fallback"))
}

func TestNewSMTPClient(t *testing.T) {
	cfg := Config{Host: "localhost", Port: 1025}
	client := NewSMTPClient(cfg)
	require.NotNil(t, client)
	assert.Equal(t, cfg, client.cfg)
	assert.Equal(t, 0, client.dailySent)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantError bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Host: "smtp.example.com",
				Port: 587,
				From: "noreply@example.com",
			},
			wantError: false,
		},
		{
			name: "empty host",
			cfg: Config{
				Host: "",
				Port: 587,
				From: "noreply@example.com",
			},
			wantError: true,
		},
		{
			name: "invalid port zero",
			cfg: Config{
				Host: "localhost",
				Port: 0,
				From: "noreply@example.com",
			},
			wantError: true,
		},
		{
			name: "invalid port too high",
			cfg: Config{
				Host: "localhost",
				Port: 99999,
				From: "noreply@example.com",
			},
			wantError: true,
		},
		{
			name: "empty from",
			cfg: Config{
				Host: "localhost",
				Port: 1025,
				From: "",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSendVerificationEmailSkipDomain(t *testing.T) {
	cfg := Config{
		Host:            "localhost",
		Port:            1025,
		From:            "test@test.com",
		SkipSendDomains: []string{"test.local"},
	}
	client := NewSMTPClient(cfg)

	err := client.SendVerificationEmail(context.Background(), "user@test.local", "token123", "http://localhost:8080")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "skipped")
	assert.Equal(t, 0, client.dailySent)
}

func TestSendVerificationEmailSkipMultipleDomains(t *testing.T) {
	cfg := Config{
		Host:            "localhost",
		Port:            1025,
		From:            "test@test.com",
		SkipSendDomains: []string{"test.local", "example.test", "dev.local"},
	}
	client := NewSMTPClient(cfg)

	for _, domain := range cfg.SkipSendDomains {
		err := client.SendVerificationEmail(context.Background(), "user@"+domain, "token123", "http://localhost:8080")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "skipped")
	}
}

func TestSendVerificationEmailExceedsDailyLimit(t *testing.T) {
	cfg := Config{
		Host:       "localhost",
		Port:       1025,
		From:       "test@test.com",
		DailyLimit: 2,
	}
	client := NewSMTPClient(cfg)
	client.dailySent = 2

	err := client.SendVerificationEmail(context.Background(), "user@example.com", "token", "http://localhost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "daily email limit exceeded")
	assert.Contains(t, err.Error(), "2/2")
}

func TestSendVerificationEmailDailyLimitIncrement(t *testing.T) {
	cfg := Config{
		Host:       "localhost",
		Port:       1025,
		From:       "test@test.com",
		DailyLimit: 5,
	}
	client := NewSMTPClient(cfg)

	err := client.SendVerificationEmail(context.Background(), "user@example.com", "token123", "http://localhost:8080")
	if err != nil {
		assert.Equal(t, 0, client.dailySent, "counter should not increase on failure")
	} else {
		assert.Equal(t, 1, client.dailySent, "counter should increment on success")
	}
}

func TestSendVerificationEmailContextCancelled(t *testing.T) {
	cfg := Config{Host: "localhost", Port: 1025, From: "test@test.com"}
	client := NewSMTPClient(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.SendVerificationEmail(ctx, "user@example.com", "token123", "http://localhost:8080")
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestBuildMessage(t *testing.T) {
	msg := buildMessage("from@test.com", "to@test.com", "Test Subject", "Test Body")

	assert.Contains(t, msg, "From: from@test.com")
	assert.Contains(t, msg, "To: to@test.com")
	assert.Contains(t, msg, "Subject: Test Subject")
	assert.Contains(t, msg, "MIME-Version: 1.0")
	assert.Contains(t, msg, "Content-Type: text/html; charset=UTF-8")
	assert.Contains(t, msg, "Test Body")
}

func TestBuildMessageWithSpecialChars(t *testing.T) {
	msg := buildMessage("a@b.com", "c@d.com", "Тема", "Тело")

	assert.Contains(t, msg, "From: a@b.com")
	assert.Contains(t, msg, "To: c@d.com")
	assert.Contains(t, msg, "Subject: Тема")
	assert.Contains(t, msg, "Тело")
}

func TestBuildVerificationHTML(t *testing.T) {
	html := buildVerificationHTML("user@example.com", "abc123", "http://localhost/confirm?token=abc123")

	assert.Contains(t, html, "user@example.com")
	assert.Contains(t, html, "abc123")
	assert.Contains(t, html, "http://localhost/confirm?token=abc123")
	assert.Contains(t, html, "FitPulse")
	assert.Contains(t, html, "Подтвердите ваш email")
	assert.Contains(t, html, "<!DOCTYPE html>")
}

func TestBuildVerificationHTMLEmptyToken(t *testing.T) {
	html := buildVerificationHTML("", "", "")
	assert.Contains(t, html, "<!DOCTYPE html>")
}

func TestSendWithTLSConnectionError(t *testing.T) {
	cfg := Config{
		Host:     "nonexistent.smtp.invalid",
		Port:     465,
		User:     "test@test.com",
		Password: "password",
		From:     "test@test.com",
		UseTLS:   true,
	}
	client := NewSMTPClient(cfg)

	err := client.SendVerificationEmail(context.Background(), "user@example.com", "token123", "http://localhost:8080")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TLS connect to SMTP server")
	assert.Contains(t, err.Error(), "nonexistent.smtp.invalid:465")
}

func TestSendWithTLSAuthErrorFormat(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     1025,
		User:     "invalid-user",
		Password: "invalid-pass",
		From:     "test@test.com",
		UseTLS:   true,
	}
	client := NewSMTPClient(cfg)

	err := client.SendVerificationEmail(context.Background(), "user@example.com", "token123", "http://localhost:8080")
	require.Error(t, err)
}

func TestSendWithTLSDailyLimitCheck(t *testing.T) {
	cfg := Config{
		Host:       "localhost",
		Port:       1025,
		From:       "test@test.com",
		UseTLS:     true,
		DailyLimit: 0,
	}
	client := NewSMTPClient(cfg)
	client.dailySent = 1

	err := client.SendVerificationEmail(context.Background(), "user@example.com", "token", "http://localhost")
	require.Error(t, err)
}

func TestSendWithTLSSkipDomainBeforeLimit(t *testing.T) {
	cfg := Config{
		Host:            "localhost",
		Port:            1025,
		From:            "test@test.com",
		UseTLS:          true,
		DailyLimit:      1,
		SkipSendDomains: []string{"test.local"},
	}
	client := NewSMTPClient(cfg)
	client.dailySent = 1

	err := client.SendVerificationEmail(context.Background(), "user@test.local", "token", "http://localhost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "skipped")
	assert.Equal(t, 1, client.dailySent)
}

func TestBuildMessageEmptyFields(t *testing.T) {
	msg := buildMessage("", "", "", "")
	assert.Contains(t, msg, "From:")
	assert.Contains(t, msg, "To:")
	assert.Contains(t, msg, "Subject:")
	assert.Contains(t, msg, "MIME-Version: 1.0")
	assert.Contains(t, msg, "Content-Type: text/html; charset=UTF-8")
}

func TestBuildMessageLongSubjectAndBody(t *testing.T) {
	longSubject := strings.Repeat("A", 1000)
	longBody := strings.Repeat("B", 5000)
	msg := buildMessage("from@test.com", "to@test.com", longSubject, longBody)
	assert.Contains(t, msg, longSubject)
	assert.Contains(t, msg, longBody)
}

func TestBuildMessageWithNewlines(t *testing.T) {
	msg := buildMessage("from@test.com", "to@test.com", "Subject\nwith\nnewlines", "Body\nwith\nnewlines")
	assert.Contains(t, msg, "Subject\nwith\nnewlines")
	assert.Contains(t, msg, "Body\nwith\nnewlines")
}

func TestBuildMessageCRLFHeaders(t *testing.T) {
	msg := buildMessage("a@b.com", "c@d.com", "Test", "Body")
	assert.Contains(t, msg, "\r\n")
	assert.Contains(t, msg, "\r\n\r\n")
}

func TestBuildVerificationHTMLContainsRequiredElements(t *testing.T) {
	html := buildVerificationHTML("test@example.com", "tok123", "https://example.com/confirm")

	requiredElements := []string{
		"<!DOCTYPE html>",
		"<html",
		"<head>",
		"<body>",
		"<style>",
		"test@example.com",
		"tok123",
		"https://example.com/confirm",
		"FitPulse",
		"Подтвердите ваш email",
		"token-code",
		".btn",
		".container",
	}

	for _, el := range requiredElements {
		assert.Contains(t, html, el, "HTML should contain: %s", el)
	}
}

func TestLoadConfigSkipDomainsEmptyEntries(t *testing.T) {
	require.NoError(t, os.Setenv("EMAIL_SKIP_DOMAINS", ",,test.local,,example.test,,"))
	cfg := LoadConfig()
	for _, d := range cfg.SkipSendDomains {
		assert.NotEmpty(t, d, "skip domains should not contain empty strings")
	}
	assert.Len(t, cfg.SkipSendDomains, 2)
	require.NoError(t, os.Unsetenv("EMAIL_SKIP_DOMAINS"))
}

func TestLoadConfigDailyLimitZeroOrNegative(t *testing.T) {
	require.NoError(t, os.Setenv("EMAIL_DAILY_LIMIT", "-5"))
	cfg := LoadConfig()
	assert.Equal(t, 0, cfg.DailyLimit, "negative limit should be treated as 0")
	require.NoError(t, os.Unsetenv("EMAIL_DAILY_LIMIT"))
}

func TestLoadConfigPortOutOfRange(t *testing.T) {
	require.NoError(t, os.Setenv("SMTP_PORT", "99999"))
	cfg := LoadConfig()
	assert.Equal(t, 99999, cfg.Port)
	require.NoError(t, os.Unsetenv("SMTP_PORT"))
}

func TestLoadConfigTLSEnvVariations(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"TRUE", false},
		{"1", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			require.NoError(t, os.Setenv("SMTP_TLS", tt.value))
			cfg := LoadConfig()
			assert.Equal(t, tt.expected, cfg.UseTLS)
			require.NoError(t, os.Unsetenv("SMTP_TLS"))
		})
	}
}

func TestSendVerificationEmailDailyLimitReached(t *testing.T) {
	cfg := Config{
		Host:       "localhost",
		Port:       1025,
		From:       "test@test.com",
		DailyLimit: 2,
	}
	client := NewSMTPClient(cfg)
	client.dailySent = 2

	err := client.SendVerificationEmail(context.Background(), "user@example.com", "token", "http://localhost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "daily email limit exceeded")
	assert.Contains(t, err.Error(), "2/2")
}

func TestNewSMTPClientWithFullConfig(t *testing.T) {
	cfg := Config{
		Host:            "smtp.example.com",
		Port:            587,
		User:            "user@example.com",
		Password:        "pass",
		From:            "noreply@example.com",
		UseTLS:          true,
		DailyLimit:      500,
		SkipSendDomains: []string{"test.local"},
	}
	client := NewSMTPClient(cfg)
	require.NotNil(t, client)
	assert.Equal(t, cfg, client.cfg)
	assert.Equal(t, 0, client.dailySent)
}

func TestEmailSenderInterface(t *testing.T) {
	var _ EmailSender = (*SMTPClient)(nil)
}
