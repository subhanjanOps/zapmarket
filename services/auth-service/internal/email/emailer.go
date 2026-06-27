// Package email defines the Emailer interface and provides implementations
// for sending transactional emails from auth-service.
package email

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
)

// Emailer sends transactional emails.
type Emailer interface {
	SendPasswordResetEmail(ctx context.Context, to, resetLink string) error
	SendOTPEmail(ctx context.Context, to, otp string) error
}

// LogEmailer logs emails instead of sending them. Used in development.
type LogEmailer struct{}

func NewLogEmailer() *LogEmailer { return &LogEmailer{} }

func (e *LogEmailer) SendPasswordResetEmail(_ context.Context, to, resetLink string) error {
	slog.Info("password reset email (dev mode — not sent)",
		"to", to,
		"reset_link", resetLink,
	)
	return nil
}

func (e *LogEmailer) SendOTPEmail(_ context.Context, to, otp string) error {
	slog.Info("OTP email (dev mode — not sent)", "to", to, "otp", otp)
	return nil
}

// SMTPEmailer sends emails via an SMTP server.
type SMTPEmailer struct {
	host     string
	port     int
	user     string
	password string
	from     string
}

func NewSMTPEmailer(host string, port int, user, password, from string) *SMTPEmailer {
	return &SMTPEmailer{host: host, port: port, user: user, password: password, from: from}
}

func (e *SMTPEmailer) SendOTPEmail(_ context.Context, to, otp string) error {
	auth := smtp.PlainAuth("", e.user, e.password, e.host)
	subject := "Your ZapMarket verification code"
	body := fmt.Sprintf(
		"Your verification code is: %s\r\n\r\nThis code expires in 10 minutes.\r\n\r\n— The ZapMarket Team",
		otp,
	)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", e.from, to, subject, body)
	addr := fmt.Sprintf("%s:%d", e.host, e.port)
	if err := smtp.SendMail(addr, auth, e.from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("send OTP email: %w", err)
	}
	return nil
}

func (e *SMTPEmailer) SendPasswordResetEmail(_ context.Context, to, resetLink string) error {
	auth := smtp.PlainAuth("", e.user, e.password, e.host)

	subject := "Reset your ZapMarket password"
	body := fmt.Sprintf(
		"Hi,\r\n\r\n"+
			"You requested a password reset. Click the link below to choose a new password.\r\n"+
			"This link expires in 30 minutes.\r\n\r\n"+
			"%s\r\n\r\n"+
			"If you did not request this, you can safely ignore this email.\r\n\r\n"+
			"— The ZapMarket Team",
		resetLink,
	)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", e.from, to, subject, body)
	addr := fmt.Sprintf("%s:%d", e.host, e.port)

	if err := smtp.SendMail(addr, auth, e.from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("send password reset email: %w", err)
	}
	return nil
}
