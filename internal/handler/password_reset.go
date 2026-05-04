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

	msg := sender.Message{
		To:      p.Email,
		Subject: "Talan — Password reset request",
		Body:    buildResetPlain(p.Code),
		HTML:    buildResetHTML(p.Code),
	}

	if err := h.email.Send(ctx, msg); err != nil {
		return fmt.Errorf("send password reset email: %w", err)
	}

	h.log.Info("password reset email sent",
		zap.String("event_id", event.ID),
		zap.Int64("user_id", p.UserID),
	)
	return nil
}

func buildResetPlain(code string) string {
	return fmt.Sprintf(
		"We received a request to reset your Talan account password.\n\nYour password reset code is: %s\n\nThis code expires in 10 minutes.\n\nIf you did not request this, you can safely ignore this email.",
		code,
	)
}

func buildResetHTML(code string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width,initial-scale=1.0">
</head>
<body style="margin:0;padding:0;background:#f4f6f9;font-family:Arial,Helvetica,sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background:#f4f6f9;padding:40px 20px;">
    <tr><td align="center">
      <table width="600" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;overflow:hidden;max-width:600px;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
        <tr>
          <td style="background:#4F46E5;padding:32px 40px;">
            <h1 style="margin:0;color:#ffffff;font-size:28px;font-weight:700;letter-spacing:-0.5px;">Talan</h1>
          </td>
        </tr>
        <tr>
          <td style="padding:40px;">
            <h2 style="margin:0 0 16px;color:#1a1a2e;font-size:22px;font-weight:700;">Password Reset</h2>
            <p style="margin:0 0 16px;color:#4a5568;font-size:15px;line-height:1.7;">
              We received a request to reset your <strong>Talan</strong> account password.
            </p>
            <p style="margin:0 0 24px;color:#4a5568;font-size:15px;line-height:1.7;">
              Use the code below to reset your password:
            </p>
            <div style="background:#f4f6f9;border-radius:8px;padding:28px 24px;text-align:center;margin:0 0 24px;">
              <p style="margin:0 0 8px;color:#718096;font-size:13px;text-transform:uppercase;letter-spacing:1px;">Reset code</p>
              <p style="margin:0;font-size:38px;font-weight:700;letter-spacing:10px;color:#4F46E5;font-family:monospace;">%s</p>
            </div>
            <p style="margin:0;color:#a0aec0;font-size:13px;">This code expires in <strong>10 minutes</strong>.</p>
          </td>
        </tr>
        <tr>
          <td style="padding:20px 40px 32px;border-top:1px solid #e2e8f0;">
            <p style="margin:0;color:#a0aec0;font-size:12px;text-align:center;line-height:1.6;">
              If you did not request a password reset, you can safely ignore this email. Your account is secure.
            </p>
          </td>
        </tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`, code)
}
