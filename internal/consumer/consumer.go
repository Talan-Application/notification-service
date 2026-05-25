package consumer

import (
	"context"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"github.com/Talan-Application/notification-service/pkg/rabbitmq"
)

const (
	mainExchange  = "talan.events"
	dlxExchange   = "talan.dlx"
	mainQueue     = "notification.queue"
	dlqQueue      = "notification.dlq"
	prefetchCount = 10
)

type MessageHandler func(ctx context.Context, msg amqp.Delivery) error

type Consumer struct {
	conn    *rabbitmq.Connection
	channel *amqp.Channel
	log     *zap.Logger
}

func New(conn *rabbitmq.Connection, log *zap.Logger) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}

	c := &Consumer{conn: conn, channel: ch, log: log}

	if err := c.declareTopology(); err != nil {
		ch.Close()
		return nil, fmt.Errorf("declare topology: %w", err)
	}

	return c, nil
}

func (c *Consumer) declareTopology() error {
	if err := c.channel.ExchangeDeclare(
		dlxExchange, amqp.ExchangeFanout, true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare dlx exchange: %w", err)
	}

	if _, err := c.channel.QueueDeclare(
		dlqQueue, true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare dlq: %w", err)
	}
	if err := c.channel.QueueBind(dlqQueue, "", dlxExchange, false, nil); err != nil {
		return fmt.Errorf("bind dlq: %w", err)
	}

	if err := c.channel.ExchangeDeclare(
		mainExchange, amqp.ExchangeTopic, true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare main exchange: %w", err)
	}

	if _, err := c.channel.QueueDeclare(
		mainQueue, true, false, false, false,
		amqp.Table{"x-dead-letter-exchange": dlxExchange},
	); err != nil {
		return fmt.Errorf("declare main queue: %w", err)
	}

	if err := c.channel.QueueBind(mainQueue, "user.*", mainExchange, false, nil); err != nil {
		return fmt.Errorf("bind main queue: %w", err)
	}

	return nil
}

func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	for {
		if err := c.channel.Qos(prefetchCount, 0, false); err != nil {
			return fmt.Errorf("set qos: %w", err)
		}

		deliveries, err := c.channel.Consume(mainQueue, "", false, false, false, false, nil)
		if err != nil {
			return fmt.Errorf("start consuming: %w", err)
		}

		c.log.Info("consumer started", zap.String("queue", mainQueue))

		if !c.drainUntilClosed(ctx, handler, deliveries) {
			return nil
		}

		c.log.Warn("consumer channel closed unexpectedly, reconnecting")
		c.channel.Close()

		if err := c.reconnectChannel(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("consumer reconnect failed: %w", err)
		}
	}
}

// drainUntilClosed processes messages until ctx is done (returns false) or the delivery channel closes (returns true).
func (c *Consumer) drainUntilClosed(ctx context.Context, handler MessageHandler, deliveries <-chan amqp.Delivery) bool {
	var wg sync.WaitGroup
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			c.log.Info("consumer drained, shutting down")
			return false
		case msg, ok := <-deliveries:
			if !ok {
				wg.Wait()
				return true
			}
			wg.Add(1)
			go func(m amqp.Delivery) {
				defer wg.Done()
				if err := handler(ctx, m); err != nil {
					c.log.Error("handler error, routing to DLQ",
						zap.String("routing_key", m.RoutingKey),
						zap.Error(err),
					)
					m.Nack(false, false)
					return
				}
				m.Ack(false)
			}(msg)
		}
	}
}

func (c *Consumer) reconnectChannel(ctx context.Context) error {
	const maxAttempts = 10
	for attempt := 0; attempt < maxAttempts; attempt++ {
		ch, err := c.conn.Channel()
		if err == nil {
			c.channel = ch
			if err := c.declareTopology(); err != nil {
				ch.Close()
				return fmt.Errorf("declare topology after reconnect: %w", err)
			}
			c.log.Info("consumer channel reconnected")
			return nil
		}
		wait := time.Duration(attempt+1) * 2 * time.Second
		c.log.Warn("channel reopen failed, retrying",
			zap.Int("attempt", attempt+1),
			zap.Duration("wait", wait),
			zap.Error(err),
		)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return fmt.Errorf("exhausted %d channel reconnect attempts", maxAttempts)
}

func (c *Consumer) Close() {
	c.channel.Close()
}
