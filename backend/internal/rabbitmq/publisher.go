package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"appstore/backend/pkg/models"
)

// PublisherConfig holds the configuration for the RabbitMQ publisher
type PublisherConfig struct {
	URL      string
	Exchange string
}

// Publisher handles publishing messages to RabbitMQ
type Publisher struct {
	config  PublisherConfig
	conn    *amqp.Connection
	channel *amqp.Channel
	mu      sync.Mutex
}

// NewPublisher creates a new RabbitMQ publisher
func NewPublisher(config PublisherConfig) *Publisher {
	return &Publisher{
		config: config,
	}
}

// Connect establishes a connection to RabbitMQ
func (p *Publisher) Connect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var err error
	p.conn, err = amqp.Dial(p.config.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	p.channel, err = p.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange
	if err := p.channel.ExchangeDeclare(
		p.config.Exchange,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	return nil
}

// Close closes the connection to RabbitMQ
func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			return err
		}
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			return err
		}
	}
	return nil
}

// publish sends a message to RabbitMQ
func (p *Publisher) publish(ctx context.Context, routingKey string, msg models.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return p.channel.PublishWithContext(ctx,
		p.config.Exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    msg.ID,
			Timestamp:    msg.Timestamp,
			Body:         body,
		},
	)
}

// PublishDeploymentRequest publishes a deployment request message
func (p *Publisher) PublishDeploymentRequest(ctx context.Context, payload models.DeploymentRequestPayload) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := models.Message{
		Type:      models.MessageTypeDeploymentRequest,
		ID:        payload.RequestID,
		Timestamp: time.Now().UTC(),
		Source:    "backend-api",
		Payload:   payloadBytes,
	}

	return p.publish(ctx, models.RoutingKeyDeploymentRequest, msg)
}

// PublishDeploymentUpdate publishes a deployment update message
func (p *Publisher) PublishDeploymentUpdate(ctx context.Context, payload models.DeploymentUpdatePayload) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := models.Message{
		Type:      models.MessageTypeDeploymentUpdate,
		ID:        payload.RequestID,
		Timestamp: time.Now().UTC(),
		Source:    "backend-api",
		Payload:   payloadBytes,
	}

	return p.publish(ctx, models.RoutingKeyDeploymentUpdate, msg)
}

// PublishDeploymentDelete publishes a deployment delete message
func (p *Publisher) PublishDeploymentDelete(ctx context.Context, payload models.DeploymentDeletePayload) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := models.Message{
		Type:      models.MessageTypeDeploymentDelete,
		ID:        payload.RequestID,
		Timestamp: time.Now().UTC(),
		Source:    "backend-api",
		Payload:   payloadBytes,
	}

	return p.publish(ctx, models.RoutingKeyDeploymentDelete, msg)
}
