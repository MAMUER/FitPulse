// Package email provides SMTP email sending for verification emails.
// Supports TLS connections for production SMTP servers (Yandex, Mail.ru, Gmail).
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/MAMUER/project/internal/config"
)

// Config holds SMTP server configuration.
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	From            string
	UseTLS          bool
	DailyLimit      int      // максимальное количество писем в день (0 = без лимита)
	SkipSendDomains []string // домены, для которых отправка писем пропускается (тестовые)
}

// LoadConfig reads SMTP configuration from environment variables.
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
		Host:            config.GetEnv("SMTP_HOST", "localhost"),
		Port:            port,
		User:            config.GetEnv("SMTP_USER"),
		Password:        config.GetEnv("SMTP_PASSWORD"),
		From:            config.GetEnv("SMTP_FROM", "noreply@fitpulse.app"),
		UseTLS:          config.GetEnv("SMTP_TLS", "false") == "true",
		DailyLimit:      dailyLimit,
		SkipSendDomains: skipDomains,
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

// Sender is the SMTP client used to send emails.
type Sender struct {
	cfg       Config
	dailySent int // счётчик отправленных писем за день
}

// NewSender creates a new email sender with the given configuration.
func NewSender(cfg Config) *Sender {
	return &Sender{cfg: cfg}
}

// SendVerificationEmail sends an email verification message to the given address.
// The email contains a clickable link and a manual token for confirmation.
func (s *Sender) SendVerificationEmail(toEmail, verifyToken, baseURL string) error {
	// Пропуск отправки для тестовых доменов (api-test, load-test)
	for _, domain := range s.cfg.SkipSendDomains {
		if strings.HasSuffix(toEmail, "@"+domain) {
			fmt.Printf("SKIP email send (test domain %s): %s\n", domain, toEmail)
			return nil
		}
	}

	// Проверка дневного лимита (защита от случайной рассылки)
	if s.cfg.DailyLimit > 0 && s.dailySent >= s.cfg.DailyLimit {
		return fmt.Errorf("превышен дневной лимит отправки писем: %d/%d", s.dailySent, s.cfg.DailyLimit)
	}

	confirmURL := fmt.Sprintf("%s/confirm?token=%s", baseURL, verifyToken)

	subject := "Подтвердите ваш email — FitPulse"
	body := buildVerificationHTML(toEmail, verifyToken, confirmURL)

	msg := buildMessage(s.cfg.From, toEmail, subject, body)

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	var err error
	if s.cfg.UseTLS {
		err = s.sendWithTLS(addr, toEmail, msg)
	} else {
		var auth smtp.Auth
		if s.cfg.User != "" && s.cfg.Password != "" {
			auth = smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)
		}
		err = smtp.SendMail(addr, auth, s.cfg.From, []string{toEmail}, []byte(msg))
	}

	// Увеличиваем счётчик только при успешной отправке
	if err == nil {
		s.dailySent++
	}

	return fmt.Errorf("send mail: %w", err)
}

// sendWithTLS sends email using TLS connection (для Yandex, Mail.ru, Gmail).
func (s *Sender) sendWithTLS(addr string, toEmail string, msg string) error {
	// TLS конфигурация
	tlsConfig := &tls.Config{
		ServerName: s.cfg.Host,
		MinVersion: tls.VersionTLS12,
	}

	// Подключаемся к серверу
	conn, err := (&tls.Dialer{
		NetDialer: &net.Dialer{Timeout: 10 * time.Second},
		Config:    tlsConfig,
	}).DialContext(context.Background(), "tcp", addr)
	if err != nil {
		return fmt.Errorf("TLS подключение к SMTP серверу (%s) не удалось: %w", addr, err)
	}

	// Создаём SMTP клиент
	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("создание SMTP клиента не удалось: %w", err)
	}

	// Аутентификация
	if s.cfg.User != "" && s.cfg.Password != "" {
		auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)
		if authErr := client.Auth(auth); authErr != nil {
			return fmt.Errorf("SMTP аутентификация не удалась: %w", authErr)
		}
	}

	// Отправка письма
	if mailErr := client.Mail(s.cfg.From); mailErr != nil {
		return fmt.Errorf("ошибка при установке отправителя: %w", mailErr)
	}

	if rcptErr := client.Rcpt(toEmail); rcptErr != nil {
		return fmt.Errorf("ошибка при установке получателя (%s): %w", toEmail, rcptErr)
	}

	// Отправка данных письма
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("ошибка при начале передачи данных: %w", err)
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("ошибка при записи данных письма: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("ошибка при завершении передачи данных: %w", err)
	}

	// Завершение сессии
	_ = client.Quit() // Игнорируем ошибку — сессия может быть уже закрыта

	return nil
}

func buildMessage(from, to, subject, body string) string {
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

const emailCSS = `
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    background-color: #0d1117;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, sans-serif;
    color: #c9d1d9;
    line-height: 1.6;
    padding: 20px;
  }
  .container {
    max-width: 520px;
    margin: 0 auto;
    background-color: #161b22;
    border-radius: 12px;
    border: 1px solid #30363d;
    overflow: hidden;
  }
  .header {
    background: linear-gradient(135deg, #1a7f37 0%, #2ea043 100%);
    padding: 32px 24px;
    text-align: center;
  }
  .header .icon { font-size: 48px; margin-bottom: 12px; }
  .header h1 { color: #ffffff; font-size: 22px; font-weight: 700; margin-bottom: 4px; }
  .header p { color: rgba(255,255,255,0.85); font-size: 14px; }
  .body { padding: 28px 24px; }
  .body p { margin-bottom: 16px; font-size: 15px; color: #c9d1d9; }
  .btn {
    display: block; width: 100%; text-align: center;
    background-color: #238636; color: #ffffff; text-decoration: none;
    padding: 14px 24px; border-radius: 8px; font-size: 16px; font-weight: 600;
    margin: 20px 0; transition: background-color 0.2s;
  }
  .btn:hover { background-color: #2ea043; }
  .token-section { margin-top: 24px; padding-top: 20px; border-top: 1px solid #21262d; }
  .token-section p { font-size: 13px; color: #8b949e; margin-bottom: 10px; }
  .token-code {
    background-color: #0d1117; border: 1px solid #30363d; border-radius: 6px;
    padding: 12px 16px; font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
    font-size: 14px; letter-spacing: 1px; color: #58a6ff; text-align: center;
    word-break: break-all; user-select: all;
  }
  .footer { padding: 20px 24px; text-align: center; border-top: 1px solid #21262d; font-size: 12px; color: #484f58; }
  .footer .app-name { color: #2ea043; font-weight: 600; }
  .email-display {
    background-color: #0d1117; border: 1px solid #30363d; border-radius: 6px;
    padding: 8px 12px; font-size: 14px; color: #58a6ff; margin: 12px 0;
  }
`

const emailHTMLTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Подтвердите email — FitPulse</title>
<style>%s</style>
</head>
<body>
  <div class="container">
    <div class="header">
      <div class="icon">&#128233;</div>
      <h1>Подтвердите ваш email</h1>
      <p>Остался один шаг до старта!</p>
    </div>
    <div class="body">
      <p>Здравствуйте!</p>
      <p>Спасибо за регистрацию в <strong>FitPulse</strong>. Для активации аккаунта подтвердите ваш email, нажав на кнопку ниже:</p>
      <div class="email-display">%s</div>
      <a href="%s" class="btn">Подтвердить email</a>
      <div class="token-section">
        <p>Если кнопка не работает, скопируйте этот токен и введите его вручную:</p>
        <div class="token-code">%s</div>
      </div>
      <p style="margin-top: 20px; font-size: 13px; color: #8b949e;">
        Ссылка действительна 24 часа. Если вы не регистрировались в FitPulse, просто проигнорируйте это письмо.
      </p>
    </div>
    <div class="footer">
      <p>С уважением, команда <span class="app-name">FitPulse</span></p>
      <p style="margin-top: 4px;">Ваш персональный фитнес-ассистент</p>
    </div>
  </div>
</body>
</html>`

func buildVerificationHTML(emailAddr, token, confirmURL string) string {
	return fmt.Sprintf(emailHTMLTemplate, emailCSS, emailAddr, confirmURL, token)
}
