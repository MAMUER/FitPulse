package email

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// SendPasswordResetEmail sends a password reset code to the given address.
// Mirrors SendVerificationEmail (rate limit, skip domains, TLS) but renders the
// password-reset template with a 6-digit code instead of a confirm link.
func (s *SMTPClient) SendPasswordResetEmail(ctx context.Context, toEmail, code, baseURL string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

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

	subject := "Восстановление пароля — FitPulse"
	body := buildPasswordResetHTML(toEmail, code)
	msg := buildMessage(s.cfg.From, toEmail, subject, body)
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

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

const passwordResetHTMLTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Восстановление пароля — FitPulse</title>
<style>%s</style>
</head>
<body>
  <div class="container">
    <div class="header">
      <div class="icon">&#128274;</div>
      <h1>Восстановление пароля</h1>
      <p>Сброс пароля для FitPulse</p>
    </div>
    <div class="body">
      <p>Здравствуйте!</p>
      <p>Получен запрос на восстановление пароля для <strong>%s</strong>. Используйте код ниже, чтобы задать новый пароль:</p>
      <div class="token-section">
        <p>Код подтверждения:</p>
        <div class="token-code" style="font-size: 28px; letter-spacing: 6px;">%s</div>
      </div>
      <p style="margin-top: 20px; font-size: 13px; color: #8b949e;">
        Код действителен 15 минут. Если вы не запрашивали восстановление пароля, просто проигнорируйте это письмо — ваш пароль не изменится.
      </p>
    </div>
    <div class="footer">
      <p>С уважением, команда <span class="app-name">FitPulse</span></p>
      <p style="margin-top: 4px;">Ваш персональный фитнес-ассистент</p>
    </div>
  </div>
</body>
</html>`

func buildPasswordResetHTML(emailAddr, code string) string {
	return fmt.Sprintf(passwordResetHTMLTemplate, emailCSS, emailAddr, code)
}
