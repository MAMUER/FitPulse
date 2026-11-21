// Package email provides SMTP email sending for verification emails.

// Supports TLS connections for production SMTP servers (Yandex, Mail.ru, Gmail).

package email

import (
	"context"
	"crypto/tls"
	_ "embed"
	"errors"
	"fmt"
	"html"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MAMUER/project/internal/config"
)

// Config holds SMTP server configuration.

type Config struct {
	Host string

	Port int

	User string

	Password string

	From string

	UseTLS bool

	DailyLimit int // maximum emails per day (0 = unlimited)

	SkipSendDomains []string // domains for which sending is skipped (test)

	DryRun bool // if true, log the email instead of sending (SMTP_DRY_RUN=true)

}

// Validate checks that the configuration is valid.

func (c Config) Validate() error {

	if c.Host == "" {

		return errors.New("SMTP host is required")

	}

	if c.Port <= 0 || c.Port > 65535 {

		return fmt.Errorf("SMTP port must be between 1 and 65535, got %d", c.Port)

	}

	if c.From == "" {

		return errors.New("SMTP from address is required")

	}

	return nil

}

// LoadConfig reads SMTP configuration from environment variables.

// Supports _FILE suffix for Docker/Kubernetes secrets.

func LoadConfig() Config {

	port := 1025

	if p := config.GetEnv("SMTP_PORT"); p != "" {

		if parsed, err := strconv.Atoi(p); err == nil {

			port = parsed

		}

	}

	dailyLimit := 0

	if limit := config.GetEnv("EMAIL_DAILY_LIMIT"); limit != "" {

		if parsed, err := strconv.Atoi(limit); err == nil && parsed > 0 {

			dailyLimit = parsed

		}

	}

	skipDomains := []string{}

	if domains := config.GetEnv("EMAIL_SKIP_DOMAINS"); domains != "" {

		for _, d := range splitCSV(domains) {

			if d != "" {

				skipDomains = append(skipDomains, d)

			}

		}

	}

	return Config{

		Host: config.GetEnv("SMTP_HOST", "localhost"),

		Port: port,

		User: config.GetEnv("SMTP_USER"),

		Password: config.GetEnv("SMTP_PASSWORD"),

		From: config.GetEnv("SMTP_FROM", "mihnikolaenko12@yandex.ru"),

		UseTLS: config.GetEnv("SMTP_TLS", "true") == "true",

		DailyLimit: dailyLimit,

		SkipSendDomains: skipDomains,

		DryRun: config.GetEnv("SMTP_DRY_RUN", "false") == "true",
	}

}

func splitCSV(s string) []string {

	parts := strings.Split(s, ",")

	result := make([]string, 0, len(parts))

	for _, part := range parts {

		result = append(result, strings.TrimSpace(part))

	}

	return result

}

// EmailSender is the port for sending emails.

type EmailSender interface {
	SendVerificationEmail(ctx context.Context, toEmail, verifyToken, baseURL string) error
	SendPasswordResetEmail(ctx context.Context, toEmail, code, baseURL string) error
}

// SMTPClient is an SMTP implementation of EmailSender.

type SMTPClient struct {
	cfg Config

	dailySent int

	mu sync.Mutex
}

// NewSMTPClient creates a new SMTP email sender with the given configuration.

func NewSMTPClient(cfg Config) *SMTPClient {

	return &SMTPClient{cfg: cfg}

}

// SendVerificationEmail sends an email verification message to the given address.

// The email contains a clickable link and a manual token for confirmation.

func (s *SMTPClient) SendVerificationEmail(ctx context.Context, toEmail, verifyToken, baseURL string) error {

	if err := ctx.Err(); err != nil {

		return err

	}
	// Защита от SMTP-инъекции: убираем CR/LF из адреса получателя.
	toEmail = stripCRLF(toEmail)
	// Валидируем адрес через net/mail — это распознаётся CodeQL как барьер и
	// гарантирует корректный конверт/заголовок без инъекций.
	parsed, addrErr := mail.ParseAddress(toEmail)
	if addrErr != nil {
		return fmt.Errorf("invalid recipient address: %w", addrErr)
	}
	toEmail = parsed.Address

	for _, domain := range s.cfg.SkipSendDomains {

		if strings.HasSuffix(toEmail, "@"+domain) {

			return fmt.Errorf("skipped: test domain %s", domain)

		}

	}

	s.mu.Lock()

	if s.cfg.DailyLimit > 0 && s.dailySent >= s.cfg.DailyLimit {

		s.mu.Unlock()

		return fmt.Errorf("daily email limit exceeded: %d/%d", s.dailySent, s.cfg.DailyLimit)

	}

	s.mu.Unlock()

	confirmURL := fmt.Sprintf("%s/confirm?token=%s", baseURL, verifyToken)

	subject := "Подтвердите ваш email — FitPulse"

	body := buildVerificationHTML(toEmail, verifyToken, confirmURL)

	msg := buildMessage(s.cfg.From, toEmail, subject, body)

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	if s.cfg.DryRun {
		log.Printf("[email] DRY-RUN: не отправляю письмо на %s (subject=%q)", toEmail, subject)
		return nil
	}

	var err error

	if s.cfg.UseTLS {

		err = sendWithTLSImpl(ctx, s.cfg, addr, toEmail, msg)

	} else {

		var auth smtp.Auth

		if s.cfg.User != "" && s.cfg.Password != "" {

			auth = smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)

		}

		err = smtpSendMailFn(addr, auth, s.cfg.From, []string{toEmail}, []byte(msg))

	}

	s.mu.Lock()

	if err == nil {

		s.dailySent++

	}

	s.mu.Unlock()

	if err != nil {

		return fmt.Errorf("send mail: %w", err)

	}

	return nil

}

// smtpSendMailFn is the actual implementation of smtp.SendMail.

// Tests can override this to avoid real network connections.

var smtpSendMailFn = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {

	return smtp.SendMail(addr, auth, from, to, msg)

}

// sendWithTLSImpl is the actual implementation of sendWithTLS.

// Tests can override this to avoid real network connections.

var sendWithTLSImpl = func(ctx context.Context, cfg Config, addr, toEmail, msg string) (sendErr error) {

	tlsConfig := &tls.Config{

		ServerName: cfg.Host,

		MinVersion: tls.VersionTLS12,
	}

	dialer := &tls.Dialer{

		Config: tlsConfig,
	}

	if deadline, ok := ctx.Deadline(); ok {

		dialer.NetDialer = &net.Dialer{Timeout: time.Until(deadline)}

	} else {

		dialer.NetDialer = &net.Dialer{Timeout: 10 * time.Second}

	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("TLS connect to SMTP server (%s) failed: %w", addr, err)
	}
	deadline := time.Now().Add(10 * time.Second)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return fmt.Errorf("set SMTP connection deadline: %w", err)
	}

	defer func() {
		if closeErr := conn.Close(); closeErr != nil && sendErr == nil {
			sendErr = fmt.Errorf("close SMTP connection failed: %w", closeErr)
		}
	}()

	client, err := smtp.NewClient(conn, cfg.Host) // NOSONAR go:S5332 - conn is already a TLS connection created by tls.Dialer with MinVersion TLS 1.2; clear-text traffic is not used here

	if err != nil {

		return fmt.Errorf("create SMTP client failed: %w", err)

	}

	defer func() {
		if quitErr := client.Quit(); quitErr != nil && sendErr == nil {
			sendErr = fmt.Errorf("SMTP QUIT failed: %w", quitErr)
		}
	}()

	if cfg.User != "" && cfg.Password != "" {

		auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)

		if authErr := client.Auth(auth); authErr != nil {

			return fmt.Errorf("SMTP authentication failed: %w", authErr)

		}

	}

	if err := deliverSMTPSession(client, cfg, toEmail, msg); err != nil {

		return err

	}

	return nil

}

// deliverSMTPSession проводит SMTP-сессию: MAIL FROM, RCPT TO и тело письма.
func deliverSMTPSession(client *smtp.Client, cfg Config, toEmail, msg string) error {

	if mailErr := client.Mail(cfg.From); mailErr != nil {

		return fmt.Errorf("set sender failed: %w", mailErr)

	}

	if rcptErr := client.Rcpt(toEmail); rcptErr != nil {

		return fmt.Errorf("set recipient (%s) failed: %w", toEmail, rcptErr)

	}

	w, err := client.Data()

	if err != nil {

		return fmt.Errorf("begin data transfer failed: %w", err)

	}

	_, err = w.Write([]byte(msg))

	if err != nil {

		return fmt.Errorf("write message body failed: %w", err)

	}

	if closeErr := w.Close(); closeErr != nil {

		return fmt.Errorf("close data writer failed: %w", closeErr)

	}

	return nil
}

// stripCRLF removes CR and LF characters so user-controlled values cannot
// inject extra SMTP headers or break out of an HTML context.
func stripCRLF(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	return strings.ReplaceAll(s, "\n", "")
}

func buildMessage(from, to, subject, body string) string {
	// Валидируем From/To через net/mail.ParseAddress — это распознаваемый
	// CodeQL-барьер (CWE-93/CWE-640). Если адрес некорректен (в т.ч. содержит
	// CR/LF), сбрасываем его. Тему чистим от CR/LF.
	if addr, err := mail.ParseAddress(from); err == nil {
		from = addr.Address
	} else {
		from = ""
	}
	if addr, err := mail.ParseAddress(to); err == nil {
		to = addr.Address
	} else {
		to = ""
	}
	subject = stripCRLF(subject)

	return fmt.Sprintf(

		"From: %s\r\n"+

			"To: %s\r\n"+

			"Subject: %s\r\n"+

			"MIME-Version: 1.0\r\n"+

			"Content-Type: text/html; charset=UTF-8\r\n"+

			"\r\n"+

			"%s",

		from, to, subject, body,
	)

}

//go:embed email.css
var emailCSS string

//go:embed verification.html
var emailHTMLTemplate string

func buildVerificationHTML(emailAddr, token, confirmURL string) string {
	// Escape user-controlled values before embedding them in the HTML body
	// (prevents HTML/content injection in the email). The CSS block is injected
	// into <head> so the template file itself stays CSS-free.
	style := "<style>" + emailCSS + "</style>"
	return fmt.Sprintf(
		emailHTMLTemplate,
		style,
		html.EscapeString(emailAddr),
		html.EscapeString(confirmURL),
		html.EscapeString(token),
	)
}
