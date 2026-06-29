package notifier

import "context"

// Notification holds the data needed to dispatch a user-facing message.
type Notification struct {
	UserID    string
	EventType string
	Subject   string
	Body      string
}

// Notifier sends notifications to users via some channel (email, SMS, push).
type Notifier interface {
	Send(ctx context.Context, n Notification) error
}

// SMSNotifier sends an SMS to a phone number. Implementations are
// fire-and-forget; callers log failures and do not retry synchronously.
type SMSNotifier interface {
	SendSMS(ctx context.Context, phone, message string) error
}
