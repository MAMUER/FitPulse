package email

import (
	"context"
	_ "embed"
	"fmt"
	"html"
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
	toEmail, err := s.prepareSend(toEmail)
	if err != nil {
		return err
	}
	subject := "Восстановление пароля — FitPulse"
	body := buildPasswordResetHTML(toEmail, code)
	return s.deliverMessage(ctx, toEmail, subject, body)
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
