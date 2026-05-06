package domain

import (
	"encoding/json"
	"time"
)

type EventType string

const (
	EventUserRegistered EventType = "user.registered"
	EventPasswordReset  EventType = "user.password_reset"
	EventLoginOTP       EventType = "user.login_otp"
)

type Event struct {
	ID         string          `json:"id"`
	Type       EventType       `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	OccurredAt time.Time       `json:"occurred_at"`
}

type UserRegisteredPayload struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Code   string `json:"code"`
}

type PasswordResetPayload struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Code   string `json:"code"`
}

type LoginOTPPayload struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Code   string `json:"code"`
}
