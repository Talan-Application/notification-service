package rabbitmq

import (
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type Connection struct {
	url  string
	conn *amqp.Connection
	mu   sync.RWMutex
	log  *zap.Logger
	done chan struct{}
}

func NewConnection(url string, log *zap.Logger) (*Connection, error) {
	c := &Connection{url: url, log: log, done: make(chan struct{})}

	if err := c.connect(); err != nil {
		return nil, err
	}

	go c.watchAndReconnect()
	return c, nil
}

func (c *Connection) connect() error {
	const maxAttempts = 10
	for attempt := range maxAttempts {
		conn, err := amqp.Dial(c.url)
		if err == nil {
			c.mu.Lock()
			c.conn = conn
			c.mu.Unlock()
			c.log.Info("rabbitmq connected")
			return nil
		}

		wait := time.Duration(attempt+1) * 2 * time.Second
		c.log.Warn("rabbitmq connection failed, retrying",
			zap.Int("attempt", attempt+1),
			zap.Duration("wait", wait),
			zap.Error(err),
		)

		select {
		case <-c.done:
			return fmt.Errorf("connection closed during retry")
		case <-time.After(wait):
		}
	}

	return fmt.Errorf("exhausted %d connection attempts to rabbitmq", maxAttempts)
}

func (c *Connection) watchAndReconnect() {
	for {
		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()

		closed := conn.NotifyClose(make(chan *amqp.Error, 1))

		select {
		case <-c.done:
			return
		case err, ok := <-closed:
			if !ok || err == nil {
				return
			}
			c.log.Warn("rabbitmq connection lost, reconnecting", zap.Error(err))
			if reconnErr := c.connect(); reconnErr != nil {
				c.log.Error("rabbitmq reconnect failed", zap.Error(reconnErr))
				return
			}
		}
	}
}

func (c *Connection) Channel() (*amqp.Channel, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn.Channel()
}

func (c *Connection) Close() {
	select {
	case <-c.done:
	default:
		close(c.done)
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.conn.Close()
}
