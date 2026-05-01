package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/Talan-Application/notification-service/internal/domain"
	"github.com/Talan-Application/notification-service/internal/sender"
)

type PasswordResetHandler struct {
	email sender.EmailSender
	log   *zap.Logger
}

func NewPasswordResetHandler(email sender.EmailSender, log *zap.Logger) *PasswordResetHandler {
	return &PasswordResetHandler{email: email, log: log}
}

func (h *PasswordResetHandler) Handle(ctx context.Context, event domain.Event) error {
	var p domain.PasswordResetPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	subject := "Password reset request"
	body := fmt.Sprintf(
		"Your password reset code is: %s\n\nThis code expires in 10 minutes.\n\nIf you did not request this, you can safely ignore this email.",
		p.Code,
	)

	if err := h.email.Send(ctx, p.Email, subject, body); err != nil {
		return fmt.Errorf("send password reset email: %w", err)
	}

	h.log.Info("password reset email sent",
		zap.String("event_id", event.ID),
		zap.Int64("user_id", p.UserID),
	)
	return nil
}
