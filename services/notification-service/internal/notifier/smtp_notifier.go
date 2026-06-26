package notifier

import (
	"context"
	"fmt"
	"net/smtp"
)

// SMTPNotifier sends notifications as plain-text emails via SMTP.
// It implements the Notifier interface.
type SMTPNotifier struct {
	host string
	port int
	user string
	pass string
	from string
}

// NewSMTPNotifier creates a notifier that sends email via the given SMTP server.
func NewSMTPNotifier(host string, port int, user, pass, from string) *SMTPNotifier {
	return &SMTPNotifier{host: host, port: port, user: user, pass: pass, from: from}
}

func (n *SMTPNotifier) Send(_ context.Context, notif Notification) error {
	auth := smtp.PlainAuth("", n.user, n.pass, n.host)

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		n.from, notif.UserID, notif.Subject, notif.Body,
	)
	addr := fmt.Sprintf("%s:%d", n.host, n.port)
	if err := smtp.SendMail(addr, auth, n.from, []string{notif.UserID}, []byte(msg)); err != nil {
		return fmt.Errorf("smtp send (event=%s user=%s): %w", notif.EventType, notif.UserID, err)
	}
	return nil
}
