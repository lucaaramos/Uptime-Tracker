package messaging

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

const (
	ExchangeChecks    = "uptime.checks"
	QueuePending      = "checks.pending"
	QueueResults      = "checks.results"
	QueueAlerts       = "checks.alerts"
	RoutingKeyPending = "check.pending"
	RoutingKeyResult  = "check.result"
	RoutingKeyAlert   = "check.alert"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQ(uri string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	rmq := &RabbitMQ{conn: conn, channel: ch}
	if err := rmq.setup(); err != nil {
		return nil, fmt.Errorf("failed to setup exchanges/queues: %w", err)
	}

	return rmq, nil
}

func (r *RabbitMQ) setup() error {
	if err := r.channel.ExchangeDeclare(
		ExchangeChecks,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("exchange declare: %w", err)
	}

	queues := []string{QueuePending, QueueResults, QueueAlerts}
	keys := []string{RoutingKeyPending, RoutingKeyResult, RoutingKeyAlert}

	for i, q := range queues {
		_, err := r.channel.QueueDeclare(q, true, false, false, false, nil)
		if err != nil {
			return fmt.Errorf("queue declare %s: %w", q, err)
		}
		if err := r.channel.QueueBind(q, keys[i], ExchangeChecks, false, nil); err != nil {
			return fmt.Errorf("queue bind %s: %w", q, err)
		}
	}

	// Set prefetch for workers
	if err := r.channel.Qos(1, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}

	return nil
}

func (r *RabbitMQ) Publish(ctx context.Context, routingKey string, body []byte) error {
	return r.channel.PublishWithContext(ctx,
		ExchangeChecks,
		routingKey,
		true,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

func (r *RabbitMQ) Consume(queue string) (<-chan amqp.Delivery, error) {
	return r.channel.Consume(queue, "", false, false, false, false, nil)
}

func (r *RabbitMQ) Close() error {
	if err := r.channel.Close(); err != nil {
		return err
	}
	return r.conn.Close()
}

func (r *RabbitMQ) Channel() *amqp.Channel {
	return r.channel
}

func (r *RabbitMQ) NotifyClose() chan *amqp.Error {
	return r.conn.NotifyClose(make(chan *amqp.Error))
}

func (r *RabbitMQ) IsClosed() bool {
	return r.conn.IsClosed()
}

func (r *RabbitMQ) Reconnect(uri string) error {
	log.Info().Msg("attempting rabbitmq reconnection...")

	if err := r.Close(); err != nil {
		log.Warn().Err(err).Msg("error closing rabbitmq connection")
	}

	conn, err := amqp.Dial(uri)
	if err != nil {
		return fmt.Errorf("failed to reconnect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel after reconnect: %w", err)
	}

	r.conn = conn
	r.channel = ch

	if err := r.setup(); err != nil {
		return fmt.Errorf("failed to setup after reconnect: %w", err)
	}

	log.Info().Msg("rabbitmq reconnected successfully")
	return nil
}
