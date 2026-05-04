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

func (c *NotificationConsumer) Handle(ctx context.Context, msg amqp.Delivery) error {
	var event domain.Event
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		c.log.Error("malformed message body, discarding",
			zap.String("routing_key", msg.RoutingKey),
			zap.Error(err),
		)
		msg.Ack(false)
		return nil
	}

	fmt.Printf("[notification-service] event received  type=%-25s  id=%s  occurred_at=%s\n",
		event.Type, event.ID, event.OccurredAt.Format("2006-01-02T15:04:05Z"))

	claimed, err := c.idempotent.Claim(ctx, event.ID, string(event.Type))
	if err != nil {
		return fmt.Errorf("idempotency claim: %w", err)
	}
	if !claimed {
		c.log.Info("duplicate event skipped",
			zap.String("event_id", event.ID),
			zap.String("type", string(event.Type)),
		)
		msg.Ack(false)
		return nil
	}

	h, ok := c.handlers[event.Type]
	if !ok {
		c.log.Warn("no handler registered for event type, discarding",
			zap.String("type", string(event.Type)),
		)
		msg.Ack(false)
		return nil
	}

	if err := h.Handle(ctx, event); err != nil {
		return fmt.Errorf("handle %s: %w", event.Type, err)
	}

	return nil
}
