package email

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

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

	assert.Equal(t, "mihnikolaenko12@yandex.ru", cfg.From)

	assert.True(t, cfg.UseTLS)

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
		name string

		cfg Config

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

		Host: "localhost",

		Port: 1025,

		From: "test@test.com",

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

		Host: "localhost",

		Port: 1025,

		From: "test@test.com",

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

		Host: "localhost",

		Port: 1025,

		From: "test@test.com",

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

		Host: "localhost",

		Port: 1025,

		From: "test@test.com",

		UseTLS: true,

		DailyLimit: 5,
	}

	client := NewSMTPClient(cfg)

	origTLS := sendWithTLSImpl

	sendWithTLSImpl = func(ctx context.Context, c Config, addr, toEmail, msg string) error {

		return nil

	}

	t.Cleanup(func() { sendWithTLSImpl = origTLS })

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

		Host: "nonexistent.smtp.invalid",

		Port: 465,

		User: "test@test.com",

		Password: "password",

		From: "test@test.com",

		UseTLS: true,
	}

	client := NewSMTPClient(cfg)

	origTLS := sendWithTLSImpl

	sendWithTLSImpl = func(ctx context.Context, c Config, addr, toEmail, msg string) error {

		return fmt.Errorf("TLS connect to SMTP server (%s) failed: connection refused", addr)

	}

	t.Cleanup(func() { sendWithTLSImpl = origTLS })

	err := client.SendVerificationEmail(context.Background(), "user@example.com", "token123", "http://localhost:8080")

	require.Error(t, err)

	assert.Contains(t, err.Error(), "TLS connect to SMTP server")

	assert.Contains(t, err.Error(), "nonexistent.smtp.invalid:465")

}

func TestSendWithTLSAuthErrorFormat(t *testing.T) {

	cfg := Config{

		Host: "localhost",

		Port: 1025,

		User: "invalid-user",

		Password: "invalid-pass",

		From: "test@test.com",

		UseTLS: true,
	}

	client := NewSMTPClient(cfg)

	origTLS := sendWithTLSImpl

	sendWithTLSImpl = func(ctx context.Context, c Config, addr, toEmail, msg string) error {

		return errors.New("SMTP authentication failed: invalid credentials")

	}

	t.Cleanup(func() { sendWithTLSImpl = origTLS })

	err := client.SendVerificationEmail(context.Background(), "user@example.com", "token123", "http://localhost:8080")

	require.Error(t, err)

	assert.Contains(t, err.Error(), "SMTP authentication failed")

}

func TestSendWithTLSDailyLimitCheck(t *testing.T) {

	cfg := Config{

		Host: "localhost",

		Port: 1025,

		From: "test@test.com",

		UseTLS: true,

		DailyLimit: 1,
	}

	client := NewSMTPClient(cfg)

	client.dailySent = 1

	err := client.SendVerificationEmail(context.Background(), "user@example.com", "token", "http://localhost")

	require.Error(t, err)

	assert.Contains(t, err.Error(), "daily email limit exceeded")

	assert.Contains(t, err.Error(), "1/1")

}

func TestSendWithTLSSkipDomainBeforeLimit(t *testing.T) {

	cfg := Config{

		Host: "localhost",

		Port: 1025,

		From: "test@test.com",

		UseTLS: true,

		DailyLimit: 1,

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
		value string

		expected bool
	}{

		{"true", true},

		{"false", false},

		{"TRUE", false},

		{"1", false},

		{"", true},
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

func TestNewSMTPClientWithFullConfig(t *testing.T) {

	cfg := Config{

		Host: "smtp.example.com",

		Port: 587,

		User: "user@example.com",

		Password: "pass",

		From: "noreply@example.com",

		UseTLS: true,

		DailyLimit: 500,

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

func TestSendVerificationEmailPlainTextSuccess(t *testing.T) {

	cfg := Config{Host: "localhost", Port: 1025, From: "test@test.com"}

	client := NewSMTPClient(cfg)

	origSend := smtpSendMailFn

	var gotAddr, gotFrom string

	var gotTo []string

	var gotMsg string

	smtpSendMailFn = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {

		gotAddr = addr

		gotFrom = from

		gotTo = to

		gotMsg = string(msg)

		return nil

	}

	t.Cleanup(func() { smtpSendMailFn = origSend })

	require.NoError(t, client.SendVerificationEmail(context.Background(), "user@example.com", "tok123", "http://localhost:8080"))

	assert.Equal(t, "localhost:1025", gotAddr)

	assert.Equal(t, "test@test.com", gotFrom)

	require.Len(t, gotTo, 1)

	assert.Equal(t, "user@example.com", gotTo[0])

	assert.Contains(t, gotMsg, "Subject: Подтвердите ваш email — FitPulse")

	assert.Contains(t, gotMsg, "http://localhost:8080/confirm?token=tok123")

	assert.Contains(t, gotMsg, "user@example.com")

	assert.Equal(t, 1, client.dailySent)

}

func TestSendVerificationEmailPlainTextWithAuth(t *testing.T) {

	cfg := Config{
		Host: "localhost",

		Port: 1025,

		User: "user@test.com",

		Password: "password",

		From: "test@test.com",
	}

	client := NewSMTPClient(cfg)

	origSend := smtpSendMailFn

	var gotAuth smtp.Auth

	smtpSendMailFn = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {

		gotAuth = auth

		return nil

	}

	t.Cleanup(func() { smtpSendMailFn = origSend })

	require.NoError(t, client.SendVerificationEmail(context.Background(), "user@example.com", "tok123", "http://localhost"))

	assert.NotNil(t, gotAuth)

	assert.Equal(t, 1, client.dailySent)

}

func TestSendVerificationEmailPlainTextError(t *testing.T) {

	cfg := Config{Host: "localhost", Port: 1025, From: "test@test.com"}

	client := NewSMTPClient(cfg)

	origSend := smtpSendMailFn

	smtpSendMailFn = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {

		return errors.New("smtp server down")

	}

	t.Cleanup(func() { smtpSendMailFn = origSend })

	err := client.SendVerificationEmail(context.Background(), "user@example.com", "tok123", "http://localhost")

	require.Error(t, err)

	assert.Contains(t, err.Error(), "send mail")

	assert.Equal(t, 0, client.dailySent)

}

func TestDefaultSMTPSendMailFn(t *testing.T) {

	err := smtpSendMailFn("127.0.0.1:1", nil, "test@test.com", []string{"user@example.com"}, []byte("Subject: hi\r\n\r\nbody"))

	require.Error(t, err)

}

func TestSendWithTLSImplDialFailure(t *testing.T) {

	cfg := Config{Host: "127.0.0.1", Port: 1}

	t.Run("without deadline", func(t *testing.T) {

		err := sendWithTLSImpl(context.Background(), cfg, "127.0.0.1:1", "user@example.com", "message body")

		require.Error(t, err)

		assert.Contains(t, err.Error(), "TLS connect to SMTP server (127.0.0.1:1) failed")

	})

	t.Run("with deadline", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		defer cancel()

		err := sendWithTLSImpl(ctx, cfg, "127.0.0.1:1", "user@example.com", "message body")

		require.Error(t, err)

		assert.Contains(t, err.Error(), "TLS connect to SMTP server")

	})

}

func TestSendWithTLSImplSession(t *testing.T) {

	tests := []struct {
		name    string
		failOn  string
		wantErr string
	}{
		{name: "session completes", failOn: "", wantErr: "close SMTP connection failed"},
		{name: "auth failure", failOn: "auth", wantErr: "SMTP authentication failed"},
		{name: "mail rejected", failOn: "mail", wantErr: "set sender failed"},
		{name: "rcpt rejected", failOn: "rcpt", wantErr: "set recipient (user@example.com) failed"},
		{name: "data rejected", failOn: "data", wantErr: "begin data transfer failed"},
		{name: "body rejected", failOn: "close", wantErr: "close data writer failed"},
		{name: "quit rejected", failOn: "quit", wantErr: "SMTP QUIT failed"},
		{name: "greeting missing", failOn: "greeting", wantErr: "create SMTP client failed"},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			addr := startFakeTLSSMTPServerForEmailTest(t, tt.failOn)

			cfg := Config{

				Host: "127.0.0.1",

				Port: 1025,

				User: "user@test.com",

				Password: "password",

				From: "test@test.com",

				UseTLS: true,
			}

			err := sendWithTLSImpl(context.Background(), cfg, addr, "user@example.com", "Subject: hi\r\n\r\nbody")

			require.Error(t, err)

			assert.Contains(t, err.Error(), tt.wantErr)

		})

	}

}

func TestSendVerificationEmailWithRealTLSServer(t *testing.T) {

	addr := startFakeTLSSMTPServerForEmailTest(t, "")

	_, portStr, err := net.SplitHostPort(addr)

	require.NoError(t, err)

	port, err := strconv.Atoi(portStr)

	require.NoError(t, err)

	cfg := Config{

		Host: "127.0.0.1",

		Port: port,

		User: "user@test.com",

		Password: "password",

		From: "test@test.com",

		UseTLS: true,
	}

	client := NewSMTPClient(cfg)

	sendErr := client.SendVerificationEmail(context.Background(), "user@example.com", "tok123", "http://localhost:8080")

	require.Error(t, sendErr)

	assert.Contains(t, sendErr.Error(), "close SMTP connection failed")

	assert.Equal(t, 0, client.dailySent)
}

func startFakeTLSSMTPServerForEmailTest(t *testing.T, failOn string) string {

	t.Helper()

	certPEM, keyPEM := selfSignedCertForEmailTest(t)

	certFile := fakeTLSCAFileForEmailTest()

	t.Setenv("SSL_CERT_FILE", certFile)

	cert, err := tls.X509KeyPair(certPEM, keyPEM)

	require.NoError(t, err)

	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	})

	require.NoError(t, err)

	done := make(chan struct{})

	go func() {

		defer close(done)

		serveFakeTLSSMTPForEmailTest(listener, failOn)

	}()

	t.Cleanup(func() {

		_ = listener.Close()

		<-done

	})

	return listener.Addr().String()

}

func selfSignedCertForEmailTest(t *testing.T) (certPEM, keyPEM []byte) {

	t.Helper()

	certPEM, keyPEM = sharedSelfSignedCertForEmailTest()

	require.NotEmpty(t, certPEM)

	require.NotEmpty(t, keyPEM)

	return certPEM, keyPEM

}

func fakeTLSCAFileForEmailTest() string {

	certPEM, _ := sharedSelfSignedCertForEmailTest()

	file := filepath.Join(os.TempDir(), "fitpulse-email-smtp-ca.pem")

	if err := os.WriteFile(file, certPEM, 0o600); err != nil {

		return ""

	}

	return file

}

var sharedSelfSignedCertForEmailTest = sync.OnceValues(func() ([]byte, []byte) {

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	if err != nil {

		return nil, nil

	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	if err != nil {

		return nil, nil

	}

	template := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              []string{"127.0.0.1", "localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)

	if err != nil {

		return nil, nil

	}

	keyDER, err := x509.MarshalECPrivateKey(key)

	if err != nil {

		return nil, nil

	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM

})

func fakeSMTPCommandResponse(command, failOn string, advertiseAuth bool) string {
	switch {
	case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
		if advertiseAuth {
			return "250-smtp.test.local\r\n250-AUTH PLAIN\r\n250 HELP\r\n"
		}

		return "250-smtp.test.local\r\n250 HELP\r\n"

	case strings.HasPrefix(command, "AUTH"):
		if failOn == "auth" {
			return "535 authentication failed\r\n"
		}

		return "235 OK authenticated\r\n"

	case strings.HasPrefix(command, "MAIL"):
		if failOn == "mail" {
			return "550 sender rejected\r\n"
		}

		return "250 OK\r\n"

	case strings.HasPrefix(command, "RCPT"):
		if failOn == "rcpt" {
			return "550 recipient rejected\r\n"
		}

		return "250 OK\r\n"

	case strings.HasPrefix(command, "DATA"):
		if failOn == "data" {
			return "550 data rejected\r\n"
		}

		return "354 End data with <CR><LF>.<CR><LF>\r\n"

	case strings.HasPrefix(command, "QUIT"):
		if failOn == "quit" {
			return "500 quit rejected\r\n"
		}

		return "221 bye\r\n"

	default:
		return "250 OK\r\n"
	}
}

func fakeSMTPDataResponse(failOn string) string {
	if failOn == "close" {
		return "550 message rejected\r\n"
	}

	return "250 OK queued\r\n"
}

func serveFakeTLSSMTPForEmailTest(listener net.Listener, failOn string) {

	conn, err := listener.Accept()

	if err != nil {

		return
	}

	defer func() { _ = conn.Close() }()

	tlsConn, ok := conn.(*tls.Conn)

	if !ok {

		return
	}

	if err := tlsConn.Handshake(); err != nil {

		return
	}

	if failOn == "greeting" {

		return
	}

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	if _, err := conn.Write([]byte("220 smtp.test.local ESMTP ready\r\n")); err != nil {

		return
	}

	reader := bufio.NewReader(conn)

	inData := false

	for {

		line, err := reader.ReadString('\n')

		if err != nil {

			return
		}

		command := strings.ToUpper(strings.TrimSpace(line))

		if inData {

			if command != "." {

				continue
			}

			inData = false

			_, _ = conn.Write([]byte(fakeSMTPDataResponse(failOn)))

			continue

		}

		_, _ = conn.Write([]byte(fakeSMTPCommandResponse(command, failOn, true)))

		inData = strings.HasPrefix(command, "DATA") && failOn != "data"

	}

}

func TestDeliverSMTPSession(t *testing.T) {

	tests := []struct {
		name    string
		failOn  string
		wantErr string
	}{
		{name: "success", failOn: "", wantErr: ""},
		{name: "mail rejected", failOn: "mail", wantErr: "set sender failed"},
		{name: "rcpt rejected", failOn: "rcpt", wantErr: "set recipient (user@example.com) failed"},
		{name: "data rejected", failOn: "data", wantErr: "begin data transfer failed"},
		{name: "body rejected", failOn: "close", wantErr: "close data writer failed"},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			server := startFakeSMTPServerForEmailTest(t, tt.failOn)

			conn, err := net.Dial("tcp", server.addr())

			require.NoError(t, err)

			client, err := smtp.NewClient(conn, "127.0.0.1")

			require.NoError(t, err)

			defer func() { _ = client.Close() }()

			cfg := Config{Host: "127.0.0.1", Port: 1025, From: "test@test.com"}

			err = deliverSMTPSession(client, cfg, "user@example.com", "Subject: hi\r\n\r\nbody")

			if tt.wantErr == "" {

				require.NoError(t, err)

				return

			}

			require.Error(t, err)

			assert.Contains(t, err.Error(), tt.wantErr)

		})

	}

}

type fakeSMTPServerForEmailTest struct {
	listener net.Listener

	failOn string

	done chan struct{}
}

func startFakeSMTPServerForEmailTest(t *testing.T, failOn string) *fakeSMTPServerForEmailTest {

	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")

	require.NoError(t, err)

	server := &fakeSMTPServerForEmailTest{

		listener: listener,

		failOn: failOn,

		done: make(chan struct{}),
	}

	go server.serve()

	t.Cleanup(func() {

		_ = listener.Close()

		<-server.done

	})

	return server

}

func (s *fakeSMTPServerForEmailTest) addr() string {

	return s.listener.Addr().String()

}

func (s *fakeSMTPServerForEmailTest) serve() {

	defer close(s.done)

	conn, err := s.listener.Accept()

	if err != nil {

		return

	}

	defer func() { _ = conn.Close() }()

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	if _, err := conn.Write([]byte("220 fake.smtp.local ESMTP ready\r\n")); err != nil {

		return
	}

	reader := bufio.NewReader(conn)

	inData := false

	for {

		line, err := reader.ReadString('\n')

		if err != nil {

			return

		}

		command := strings.ToUpper(strings.TrimSpace(line))

		if inData {

			if command != "." {

				continue

			}

			inData = false

			_, _ = conn.Write([]byte(fakeSMTPDataResponse(s.failOn)))

			continue

		}

		_, _ = conn.Write([]byte(fakeSMTPCommandResponse(command, s.failOn, false)))

		inData = strings.HasPrefix(command, "DATA") && s.failOn != "data"

	}

}
