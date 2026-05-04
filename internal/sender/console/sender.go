package console

import (
	"context"

	"go.uber.org/zap"

	"github.com/Talan-Application/notification-service/internal/sender"
)

type Sender struct {
	log *zap.Logger
}

func NewSender(log *zap.Logger) *Sender {
	return &Sender{log: log}
}

func (s *Sender) Send(_ context.Context, msg sender.Message) error {
	s.log.Info("notification",
		zap.String("to", msg.To),
		zap.String("subject", msg.Subject),
		zap.String("body", msg.Body),
	)
	return nil
}
