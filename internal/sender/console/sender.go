package console

import (
	"context"

	"go.uber.org/zap"
)

// Sender logs notifications instead of sending real emails.
// Used in development so no SMTP server is required.
type Sender struct {
	log *zap.Logger
}

func NewSender(log *zap.Logger) *Sender {
	return &Sender{log: log}
}

func (s *Sender) Send(_ context.Context, to, subject, body string) error {
	s.log.Info("notification",
		zap.String("to", to),
		zap.String("subject", subject),
		zap.String("body", body),
	)
	return nil
}
