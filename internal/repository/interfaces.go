package repository

import "context"

// IdempotencyRepository tracks processed events to prevent duplicate processing.
type IdempotencyRepository interface {
	// Claim inserts the event as processed. Returns true if this call claimed it
	// (first time seen), false if a prior worker already processed it.
	Claim(ctx context.Context, eventID, eventType string) (bool, error)
}
