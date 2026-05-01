package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"github.com/Talan-Application/notification-service/internal/domain"
	"github.com/Talan-Application/notification-service/internal/handler"
	"github.com/Talan-Application/notification-service/internal/repository"
)

// NotificationConsumer routes incoming AMQP deliveries to the correct EventHandler.
// It implements the MessageHandler signature expected by Consumer.Consume.
type NotificationConsumer struct {
	handlers   map[domain.EventType]handler.EventHandler
	idempotent repository.IdempotencyRepository
	log        *zap.Logger
}

func NewNotificationConsumer(
	handlers map[domain.EventType]handler.EventHandler,
	idempotent repository.IdempotencyRepository,
	log *zap.Logger,
) *NotificationConsumer {
	return &NotificationConsumer{
		handlers:   handlers,
		idempotent: idempotent,
		log:        log,
	}
}

// Handle is the MessageHandler implementation.
// Returning nil → ACK. Returning error → NACK to DLQ (done by Consumer).
func (c *NotificationConsumer) Handle(ctx context.Context, msg amqp.Delivery) error {
	var event domain.Event
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		// Malformed JSON can never be fixed by retrying — ACK to discard it.
		c.log.Error("malformed message body, discarding",
			zap.String("routing_key", msg.RoutingKey),
			zap.Error(err),
		)
		msg.Ack(false) //nolint:errcheck
		return nil
	}

	// INSERT-first idempotency: only one concurrent worker can claim a given event_id.
	// The unique constraint on processed_events makes this race-free.
	claimed, err := c.idempotent.Claim(ctx, event.ID, string(event.Type))
	if err != nil {
		return fmt.Errorf("idempotency claim: %w", err)
	}
	if !claimed {
		c.log.Info("duplicate event skipped",
			zap.String("event_id", event.ID),
			zap.String("type", string(event.Type)),
		)
		msg.Ack(false) //nolint:errcheck
		return nil
	}

	h, ok := c.handlers[event.Type]
	if !ok {
		c.log.Warn("no handler registered for event type, discarding",
			zap.String("type", string(event.Type)),
		)
		msg.Ack(false) //nolint:errcheck
		return nil
	}

	if err := h.Handle(ctx, event); err != nil {
		return fmt.Errorf("handle %s: %w", event.Type, err)
	}

	return nil
}
