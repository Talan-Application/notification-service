package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IdempotencyRepository struct {
	db *pgxpool.Pool
}

func NewIdempotencyRepository(db *pgxpool.Pool) *IdempotencyRepository {
	return &IdempotencyRepository{db: db}
}

// Claim uses INSERT with unique constraint as an atomic check-and-set.
// Two concurrent workers racing on the same event_id: only one INSERT wins,
// the other gets a unique violation and returns (false, nil) — safe to skip.
func (r *IdempotencyRepository) Claim(ctx context.Context, eventID, eventType string) (bool, error) {
	_, err := r.db.Exec(ctx,
		`INSERT INTO processed_events (event_id, event_type, processed_at)
		 VALUES ($1, $2, NOW())`,
		eventID, eventType,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return false, nil
		}
		return false, fmt.Errorf("claim event: %w", err)
	}

	return true, nil
}
