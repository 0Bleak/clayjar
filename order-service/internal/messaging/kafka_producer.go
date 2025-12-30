package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/0Bleak/order-service/internal/models"
	"github.com/segmentio/kafka-go"
)

type KafkaProducer interface {
	PublishOrderEvent(ctx context.Context, event *models.OrderEvent) error
	Close() error
}

type kafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokers []string, topic string) KafkaProducer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
		Compression:  kafka.Snappy,
	}

	return &kafkaProducer{
		writer: writer,
	}
}

func (p *kafkaProducer) PublishOrderEvent(ctx context.Context, event *models.OrderEvent) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal order event: %w", err)
	}

	message := kafka.Message{
		Key:   []byte(fmt.Sprintf("%d", event.OrderID)),
		Value: eventJSON,
		Time:  event.Timestamp,
		Headers: []kafka.Header{
			{Key: "event-type", Value: []byte(event.Type)},
		},
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := p.writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("failed to write order event to kafka: %w", err)
	}

	return nil
}

func (p *kafkaProducer) Close() error {
	return p.writer.Close()
}
