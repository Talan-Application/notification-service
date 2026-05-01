package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/Talan-Application/notification-service/internal/domain"
	"github.com/Talan-Application/notification-service/internal/sender"
)

type UserRegisteredHandler struct {
	email sender.EmailSender
	log   *zap.Logger
}

func NewUserRegisteredHandler(email sender.EmailSender, log *zap.Logger) *UserRegisteredHandler {
	return &UserRegisteredHandler{email: email, log: log}
}

func (h *UserRegisteredHandler) Handle(ctx context.Context, event domain.Event) error {
	var p domain.UserRegisteredPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	subject := "Confirm your email address"
	body := fmt.Sprintf(
		"Welcome!\n\nYour confirmation code is: %s\n\nThis code expires in 15 minutes.",
		p.Code,
	)

	if err := h.email.Send(ctx, p.Email, subject, body); err != nil {
		return fmt.Errorf("send confirmation email: %w", err)
	}

	h.log.Info("confirmation email sent",
		zap.String("event_id", event.ID),
		zap.Int64("user_id", p.UserID),
	)
	return nil
}
