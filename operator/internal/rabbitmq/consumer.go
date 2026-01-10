package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// MessageType defines the type of message being sent
type MessageType string

const (
	MessageTypeDeploymentRequest MessageType = "deployment.request"
	MessageTypeDeploymentUpdate  MessageType = "deployment.update"
	MessageTypeDeploymentDelete  MessageType = "deployment.delete"
)

// Message is the envelope for all RabbitMQ messages
type Message struct {
	Type      MessageType     `json:"type"`
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string          `json:"source"`
	Payload   json.RawMessage `json:"payload"`
}

// DeploymentRequestPayload contains the data for a deployment request
type DeploymentRequestPayload struct {
	RequestID   string                 `json:"requestId"`
	TeamID      string                 `json:"teamId"`
	UserID      string                 `json:"userId"`
	AppName     string                 `json:"appName"`
	Namespace   string                 `json:"namespace"`
	ReleaseName string                 `json:"releaseName,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Values      map[string]interface{} `json:"values,omitempty"`
}

// DeploymentUpdatePayload contains the data for updating an existing deployment
type DeploymentUpdatePayload struct {
	RequestID string                 `json:"requestId"`
	TeamID    string                 `json:"teamId"`
	UserID    string                 `json:"userId"`
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace"`
	Version   string                 `json:"version,omitempty"`
	Values    map[string]interface{} `json:"values,omitempty"`
}

// DeploymentDeletePayload contains the data for deleting a deployment
type DeploymentDeletePayload struct {
	RequestID string `json:"requestId"`
	TeamID    string `json:"teamId"`
	UserID    string `json:"userId"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// MessageHandler is the interface for handling incoming messages
type MessageHandler interface {
	HandleDeploymentRequest(ctx context.Context, payload DeploymentRequestPayload) error
	HandleDeploymentUpdate(ctx context.Context, payload DeploymentUpdatePayload) error
	HandleDeploymentDelete(ctx context.Context, payload DeploymentDeletePayload) error
}

// ConsumerConfig holds the configuration for the RabbitMQ consumer
type ConsumerConfig struct {
	URL           string
	Exchange      string
	Queue         string
	RoutingKeys   []string
	ConsumerTag   string
	PrefetchCount int
}

// Consumer handles consuming messages from RabbitMQ
type Consumer struct {
	config    ConsumerConfig
	conn      *amqp.Connection
	channel   *amqp.Channel
	handler   MessageHandler
	done      chan struct{}
	reconnect chan struct{}
}

// NewConsumer creates a new RabbitMQ consumer
func NewConsumer(config ConsumerConfig, handler MessageHandler) *Consumer {
	return &Consumer{
		config:    config,
		handler:   handler,
		done:      make(chan struct{}),
		reconnect: make(chan struct{}, 1),
	}
}

// Start begins consuming messages from RabbitMQ
func (c *Consumer) Start(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("rabbitmq")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := c.connect(ctx); err != nil {
			logger.Error(err, "Failed to connect to RabbitMQ, retrying in 5 seconds")
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				continue
			}
		}

		logger.Info("Connected to RabbitMQ", "url", c.config.URL)

		if err := c.consume(ctx); err != nil {
			logger.Error(err, "Consumer error, reconnecting")
			c.cleanup()
			continue
		}

		return nil
	}
}

// Stop gracefully stops the consumer
func (c *Consumer) Stop() error {
	close(c.done)
	return c.cleanup()
}

func (c *Consumer) connect(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("rabbitmq")

	var err error
	c.conn, err = amqp.Dial(c.config.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	c.channel, err = c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	if err := c.channel.Qos(c.config.PrefetchCount, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Declare exchange
	if err := c.channel.ExchangeDeclare(
		c.config.Exchange,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	queue, err := c.channel.QueueDeclare(
		c.config.Queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange with routing keys
	for _, key := range c.config.RoutingKeys {
		if err := c.channel.QueueBind(
			queue.Name,
			key,
			c.config.Exchange,
			false,
			nil,
		); err != nil {
			return fmt.Errorf("failed to bind queue: %w", err)
		}
		logger.Info("Bound queue to exchange", "queue", queue.Name, "routingKey", key)
	}

	return nil
}

func (c *Consumer) consume(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("rabbitmq")

	msgs, err := c.channel.Consume(
		c.config.Queue,
		c.config.ConsumerTag,
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	// Monitor connection close
	connClose := c.conn.NotifyClose(make(chan *amqp.Error, 1))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.done:
			return nil
		case amqpErr := <-connClose:
			return fmt.Errorf("connection closed: %w", amqpErr)
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("message channel closed")
			}

			if err := c.handleMessage(ctx, msg); err != nil {
				logger.Error(err, "Failed to handle message", "messageId", msg.MessageId)
				// Nack and requeue on failure
				if nackErr := msg.Nack(false, true); nackErr != nil {
					logger.Error(nackErr, "Failed to nack message")
				}
			} else {
				if ackErr := msg.Ack(false); ackErr != nil {
					logger.Error(ackErr, "Failed to ack message")
				}
			}
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	logger := log.FromContext(ctx).WithName("rabbitmq")

	var envelope Message
	if err := json.Unmarshal(msg.Body, &envelope); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	logger.Info("Received message", "type", envelope.Type, "id", envelope.ID)

	switch envelope.Type {
	case MessageTypeDeploymentRequest:
		var payload DeploymentRequestPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal deployment request payload: %w", err)
		}
		return c.handler.HandleDeploymentRequest(ctx, payload)

	case MessageTypeDeploymentUpdate:
		var payload DeploymentUpdatePayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal deployment update payload: %w", err)
		}
		return c.handler.HandleDeploymentUpdate(ctx, payload)

	case MessageTypeDeploymentDelete:
		var payload DeploymentDeletePayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal deployment delete payload: %w", err)
		}
		return c.handler.HandleDeploymentDelete(ctx, payload)

	default:
		return fmt.Errorf("unknown message type: %s", envelope.Type)
	}
}

func (c *Consumer) cleanup() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			return err
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return err
		}
	}
	return nil
}
