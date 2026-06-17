package notifier

import (
	"context"
	"log/slog"
)

// LogNotifier is a fake notifier that logs the notification instead of sending
// a real email or SMS. Used until a real provider (SendGrid, Twilio) is wired.
type LogNotifier struct {
	logger *slog.Logger
}

func NewLogNotifier(logger *slog.Logger) *LogNotifier {
	return &LogNotifier{logger: logger}
}

func (n *LogNotifier) Send(_ context.Context, notif Notification) error {
	n.logger.Info("notification dispatched",
		"user_id", notif.UserID,
		"event_type", notif.EventType,
		"subject", notif.Subject,
		"body", notif.Body,
	)
	return nil
}
