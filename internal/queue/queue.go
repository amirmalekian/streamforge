package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"streamforge/internal/config"
)

type Message struct {
	JobID     string                 `json:"job_id"`
	Action    string                 `json:"action"`
	Payload   map[string]interface{} `json:"payload"`
	CreatedAt string                 `json:"created_at"`
}

type Service struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	config  config.QueueConfig
	mu      sync.RWMutex
}

func Connect(cfg config.RabbitMQConfig) (*amqp.Connection, error) {
	conn, err := amqp.Dial(cfg.URI())
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func NewService(conn *amqp.Connection, cfg config.QueueConfig) *Service {
	ch, err := conn.Channel()
	if err != nil {
		panic(fmt.Sprintf("Failed to open channel: %v", err))
	}

	if err := ch.Qos(1, 0, false); err != nil {
		panic(fmt.Sprintf("Failed to set QoS: %v", err))
	}

	err = ch.ExchangeDeclare(
		cfg.Exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to declare exchange: %v", err))
	}

	_, err = ch.QueueDeclare(
		cfg.Queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to declare queue: %v", err))
	}

	err = ch.QueueBind(
		cfg.Queue,
		cfg.RoutingKey,
		cfg.Exchange,
		false,
		nil,
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to bind queue: %v", err))
	}

	return &Service{
		conn:    conn,
		channel: ch,
		config:  cfg,
	}
}

func (s *Service) Publish(ctx context.Context, msg Message) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.channel == nil {
		return fmt.Errorf("channel is closed")
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return s.channel.PublishWithContext(ctx,
		s.config.Exchange,
		s.config.RoutingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
}

func (s *Service) Consume(ctx context.Context, jobChan chan<- Message) error {
	s.mu.RLock()
	ch := s.channel
	s.mu.RUnlock()

	if ch == nil {
		return fmt.Errorf("channel is closed")
	}

	msgs, err := ch.Consume(
		s.config.Queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case delivery, ok := <-msgs:
				if !ok {
					return
				}

				var msg Message
				if err := json.Unmarshal(delivery.Body, &msg); err != nil {
					_ = delivery.Nack(false, false)
					continue
				}

				select {
				case jobChan <- msg:
					_ = delivery.Ack(false)
				case <-ctx.Done():
					_ = delivery.Nack(false, true)
					return
				default:
					_ = delivery.Nack(false, true)
				}
			}
		}
	}()

	return nil
}

func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.channel != nil {
		_ = s.channel.Close()
		s.channel = nil
	}
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
	return nil
}
