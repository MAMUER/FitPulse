package email

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigDefaults(t *testing.T) {
	// Clear env to test defaults
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

	// Cleanup
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
	assert.Equal(t, 1025, cfg.Port) // should use default
	require.NoError(t, os.Unsetenv("SMTP_PORT"))
}

func TestLoadConfigInvalidDailyLimit(t *testing.T) {
	require.NoError(t, os.Setenv("EMAIL_DAILY_LIMIT", "invalid"))
	cfg := LoadConfig()
	assert.Equal(t, 0, cfg.DailyLimit) // should use default
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
	assert.Equal(t, "hello", getEnv("TEST_VAR", "fallback"))
	require.NoError(t, os.Unsetenv("TEST_VAR"))
}

func TestGetEnvFallback(t *testing.T) {
	require.NoError(t, os.Unsetenv("NONEXISTENT_VAR_XYZ"))
	assert.Equal(t, "fallback", getEnv("NONEXISTENT_VAR_XYZ", "fallback"))
}

func TestNewSender(t *testing.T) {
	cfg := Config{Host: "localhost", Port: 1025}
	sender := NewSender(cfg)
	require.NotNil(t, sender)
	assert.Equal(t, cfg, sender.cfg)
	assert.Equal(t, 0, sender.dailySent)
}

func TestSendVerificationEmailSkipDomain(t *testing.T) {
	cfg := Config{
		Host:            "localhost",
		Port:            1025,
		From:            "test@test.com",
		SkipSendDomains: []string{"test.local"},
	}
	sender := NewSender(cfg)

	err := sender.SendVerificationEmail("user@test.local", "token123", "http://localhost:8080")
	require.NoError(t, err)
	assert.Equal(t, 0, sender.dailySent) // counter should not increase for skipped
}

func TestSendVerificationEmailSkipMultipleDomains(t *testing.T) {
	cfg := Config{
		Host:            "localhost",
		Port:            1025,
		From:            "test@test.com",
		SkipSendDomains: []string{"test.local", "example.test", "dev.local"},
	}
	sender := NewSender(cfg)

	// Test each skip domain
	for _, domain := range cfg.SkipSendDomains {
		err := sender.SendVerificationEmail("user@"+domain, "token123", "http://localhost:8080")
		require.NoError(t, err)
	}
}

func TestSendVerificationEmailExceedsDailyLimit(t *testing.T) {
	cfg := Config{
		Host:       "localhost",
		Port:       1025,
		From:       "test@test.com",
		DailyLimit: 1,
	}
	sender := NewSender(cfg)
	sender.dailySent = 1 // already at limit

	err := sender.SendVerificationEmail("user@example.com", "token123", "http://localhost:8080")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "превышен дневной лимит")
}

func TestSendVerificationEmailNoSkipWhenUnderLimit(t *testing.T) {
	cfg := Config{
		Host:            "localhost",
		Port:            1025,
		From:            "test@test.com",
		DailyLimit:      5,
		SkipSendDomains: []string{"test.local"},
	}
	sender := NewSender(cfg)

	err := sender.SendVerificationEmail("user@example.com", "token123", "http://localhost:8080")

	if err != nil {
		assert.Equal(t, 0, sender.dailySent, "counter should not increase on failure")
	} else {
		assert.Equal(t, 1, sender.dailySent, "counter should increment on success")
	}
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
	// Should still produce valid HTML
	assert.Contains(t, html, "<!DOCTYPE html>")
}

func TestSendWithTLSConnectionError(t *testing.T) {
	// sendWithTLS will fail because there's no real TLS SMTP server on the given address
	// but we can verify the error message format
	cfg := Config{
		Host:     "nonexistent.smtp.invalid",
		Port:     465,
		User:     "test@test.com",
		Password: "password",
		From:     "test@test.com",
		UseTLS:   true,
	}
	sender := NewSender(cfg)

	err := sender.SendVerificationEmail("user@example.com", "token123", "http://localhost:8080")
	require.Error(t, err)
	// Error message should mention TLS connection failure
	assert.Contains(t, err.Error(), "TLS подключение к SMTP серверу")
	assert.Contains(t, err.Error(), "nonexistent.smtp.invalid:465")
}

func TestSendWithTLSAuthErrorFormat(t *testing.T) {
	// Test with invalid credentials to verify error path through auth
	cfg := Config{
		Host:     "localhost",
		Port:     1025,
		User:     "invalid-user",
		Password: "invalid-pass",
		From:     "test@test.com",
		UseTLS:   true,
	}
	sender := NewSender(cfg)

	err := sender.SendVerificationEmail("user@example.com", "token123", "http://localhost:8080")
	// Connection will fail before auth, but we verify the TLS path is taken
	require.Error(t, err)
}

func TestSendWithTLSDailyLimitCheck(t *testing.T) {
	// Verify daily limit is checked before TLS connection attempt
	cfg := Config{
		Host:       "localhost",
		Port:       1025,
		From:       "test@test.com",
		UseTLS:     true,
		DailyLimit: 0,
	}
	sender := NewSender(cfg)
	sender.dailySent = 1 // any value > 0 with DailyLimit=0 means no limit

	err := sender.SendVerificationEmail("user@example.com", "token", "http://localhost")
	// Should attempt TLS send (will fail due to no server), but limit check should pass
	require.Error(t, err) // error from TLS, not from limit
}

func TestSendWithTLSSkipDomainBeforeLimit(t *testing.T) {
	// Verify skip domain check happens before daily limit check
	cfg := Config{
		Host:            "localhost",
		Port:            1025,
		From:            "test@test.com",
		UseTLS:          true,
		DailyLimit:      1,
		SkipSendDomains: []string{"test.local"},
	}
	sender := NewSender(cfg)
	sender.dailySent = 1 // at limit, but skip domain should bypass

	err := sender.SendVerificationEmail("user@test.local", "token", "http://localhost")
	require.NoError(t, err)
	assert.Equal(t, 1, sender.dailySent)
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
	// Headers should be separated by \r\n
	assert.Contains(t, msg, "\r\n")
	// Empty line between headers and body
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
	// Empty entries should be filtered out
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
	assert.Equal(t, 99999, cfg.Port) // large port is valid, just won't connect
	require.NoError(t, os.Unsetenv("SMTP_PORT"))
}

func TestLoadConfigTLSEnvVariations(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"TRUE", false}, // case-sensitive
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
	sender := NewSender(cfg)
	sender.dailySent = 2

	err := sender.SendVerificationEmail("user@example.com", "token", "http://localhost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "превышен дневной лимит")
	assert.Contains(t, err.Error(), "2/2")
}

func TestNewSenderWithFullConfig(t *testing.T) {
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
	sender := NewSender(cfg)
	require.NotNil(t, sender)
	assert.Equal(t, cfg, sender.cfg)
	assert.Equal(t, 0, sender.dailySent)
}
