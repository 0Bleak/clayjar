package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/0Bleak/clayjar-jar-service/internal/models"
	"github.com/segmentio/kafka-go"
)

type KafkaProducer interface {
	PublishJarEvent(ctx context.Context, event *models.JarEvent) error
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

func (p *kafkaProducer) PublishJarEvent(ctx context.Context, event *models.JarEvent) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal jar event: %w", err)
	}

	message := kafka.Message{
		Key:   []byte(event.JarID),
		Value: eventJSON,
		Time:  event.Timestamp,
		Headers: []kafka.Header{
			{Key: "event-type", Value: []byte(event.Type)},
		},
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := p.writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("failed to write jar event to kafka: %w", err)
	}

	return nil
}

func (p *kafkaProducer) Close() error {
	return p.writer.Close()
}
