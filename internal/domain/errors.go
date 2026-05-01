package domain

import "errors"

var (
	ErrEventAlreadyProcessed = errors.New("event already processed")
	ErrUnknownEventType      = errors.New("unknown event type")
)
