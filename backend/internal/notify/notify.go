package notify

import "context"

// Channel abstractions: Email / SMS / WhatsApp providers plug in here.
type EmailProvider interface {
	Send(ctx context.Context, to, subject, body string) error
}
type SMSProvider interface {
	Send(ctx context.Context, to, message string) error
}
type WhatsAppProvider interface {
	Send(ctx context.Context, to, template string, params map[string]string) error
}

type LogEmail struct{}

func (LogEmail) Send(_ context.Context, to, subject, body string) error { return nil }
