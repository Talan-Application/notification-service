package repository

import "context"

type IdempotencyRepository interface {
	Claim(ctx context.Context, eventID, eventType string) (bool, error)
}
