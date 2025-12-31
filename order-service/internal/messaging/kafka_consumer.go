package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/0Bleak/order-service/internal/models"
	"github.com/segmentio/kafka-go"
)

// PaymentEventHandler defines the interface for handling payment events
type PaymentEventHandler interface {
	HandlePaymentEvent(ctx context.Context, event *models.PaymentEvent) error
}

type KafkaConsumer interface {
	ConsumePaymentEvents(ctx context.Context, handler PaymentEventHandler) error
	Close() error
}

type kafkaConsumer struct {
	reader *kafka.Reader
}

func NewKafkaConsumer(brokers []string, topic, groupID string) KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	return &kafkaConsumer{
		reader: reader,
	}
}

func (c *kafkaConsumer) ConsumePaymentEvents(ctx context.Context, handler PaymentEventHandler) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return fmt.Errorf("failed to read message: %w", err)
		}

		var paymentEvent models.PaymentEvent
		if err := json.Unmarshal(msg.Value, &paymentEvent); err != nil {
			log.Printf("Failed to unmarshal payment event: %v", err)
			continue
		}

		log.Printf("Received payment event: %s for order %d", paymentEvent.Type, paymentEvent.OrderID)

		if err := handler.HandlePaymentEvent(ctx, &paymentEvent); err != nil {
			log.Printf("Failed to handle payment event: %v", err)
		}
	}
}

func (c *kafkaConsumer) Close() error {
	return c.reader.Close()
}
