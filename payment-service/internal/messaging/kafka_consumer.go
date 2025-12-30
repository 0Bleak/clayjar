package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/0Bleak/payment-service/internal/models"
	"github.com/0Bleak/payment-service/internal/service"
	"github.com/segmentio/kafka-go"
)

type KafkaConsumer interface {
	ConsumeOrderEvents(ctx context.Context, paymentService service.PaymentService) error
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

func (c *kafkaConsumer) ConsumeOrderEvents(ctx context.Context, paymentService service.PaymentService) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return fmt.Errorf("failed to read message: %w", err)
		}

		var orderEvent models.OrderEvent
		if err := json.Unmarshal(msg.Value, &orderEvent); err != nil {
			log.Printf("Failed to unmarshal order event: %v", err)
			continue
		}

		log.Printf("Received order event: %s for order %d", orderEvent.Type, orderEvent.OrderID)

		if err := paymentService.HandleOrderEvent(ctx, &orderEvent); err != nil {
			log.Printf("Failed to handle order event: %v", err)
		}
	}
}

func (c *kafkaConsumer) Close() error {
	return c.reader.Close()
}
