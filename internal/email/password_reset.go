package email

import (
	"context"
	_ "embed"
	"fmt"
	"html"
	"net/mail"
	"net/smtp"
	"strings"
)

//go:embed password_reset.html
var passwordResetHTMLTemplate string

// SendPasswordResetEmail sends a password reset code to the given address.
// Mirrors SendVerificationEmail (rate limit, skip domains, TLS) but renders the
// password-reset template with a 6-digit code instead of a confirm link.
func (s *SMTPClient) SendPasswordResetEmail(ctx context.Context, toEmail, code, baseURL string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Защита от SMTP-инъекции: убираем CR/LF из адреса получателя.
	toEmail = stripCRLF(toEmail)
	// Валидируем адрес через net/mail — барьер для CodeQL и гарантия корректного
	// конверта/заголовка без инъекций.
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

func buildPasswordResetHTML(emailAddr, code string) string {
	// CSS-блок собираем отдельно и подставляем в <head> (шаблон не содержит
	// <style>, поэтому HTML/CSS-линтер не ругается на плейсхолдер).
	style := "<style>" + emailCSS + "</style>"
	return fmt.Sprintf(
		passwordResetHTMLTemplate,
		style,
		html.EscapeString(emailAddr),
		html.EscapeString(code),
	)
}
