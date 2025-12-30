package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/0Bleak/payment-service/internal/messaging"
	"github.com/0Bleak/payment-service/internal/models"
	"github.com/0Bleak/payment-service/internal/repository"
	"github.com/google/uuid"
)

type PaymentService interface {
	CreatePayment(ctx context.Context, req *models.CreatePaymentRequest) (*models.Payment, error)
	GetPaymentByID(ctx context.Context, id int64) (*models.Payment, error)
	GetPaymentByOrderID(ctx context.Context, orderID int64) (*models.Payment, error)
	HandleOrderEvent(ctx context.Context, event *models.OrderEvent) error
}

type paymentService struct {
	repo     repository.PaymentRepository
	producer messaging.KafkaProducer
}

func NewPaymentService(repo repository.PaymentRepository, producer messaging.KafkaProducer) PaymentService {
	return &paymentService{
		repo:     repo,
		producer: producer,
	}
}

func (s *paymentService) CreatePayment(ctx context.Context, req *models.CreatePaymentRequest) (*models.Payment, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	payment := &models.Payment{
		OrderID:       req.OrderID,
		Amount:        req.Amount,
		Status:        "pending",
		PaymentMethod: req.PaymentMethod,
		TransactionID: "",
	}

	if err := s.repo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Simulate payment processing
	go s.processPayment(context.Background(), payment)

	return payment, nil
}

func (s *paymentService) GetPaymentByID(ctx context.Context, id int64) (*models.Payment, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *paymentService) GetPaymentByOrderID(ctx context.Context, orderID int64) (*models.Payment, error) {
	return s.repo.FindByOrderID(ctx, orderID)
}

func (s *paymentService) HandleOrderEvent(ctx context.Context, event *models.OrderEvent) error {
	log.Printf("Handling order event: %s for order %d", event.Type, event.OrderID)

	if event.Type == "order.created" {
		// Check if payment already exists for this order
		existingPayment, _ := s.repo.FindByOrderID(ctx, event.OrderID)
		if existingPayment != nil {
			log.Printf("Payment already exists for order %d", event.OrderID)
			return nil
		}

		// Create payment automatically for new order
		// In a real system, you'd get amount from the order service or event
		payment := &models.Payment{
			OrderID:       event.OrderID,
			Amount:        100.00, // Mocked amount
			Status:        "pending",
			PaymentMethod: "credit_card",
			TransactionID: "",
		}

		if err := s.repo.Create(ctx, payment); err != nil {
			return fmt.Errorf("failed to create payment: %w", err)
		}

		log.Printf("Created payment %d for order %d", payment.ID, event.OrderID)

		// Simulate payment processing
		go s.processPayment(context.Background(), payment)
	}

	return nil
}

// processPayment simulates payment processing (MOCKED)
func (s *paymentService) processPayment(ctx context.Context, payment *models.Payment) {
	// Simulate processing delay
	time.Sleep(3 * time.Second)

	// Mock: 90% success rate
	rand.Seed(time.Now().UnixNano())
	success := rand.Float32() < 0.9

	var status string
	var eventType string
	transactionID := uuid.New().String()

	if success {
		status = "completed"
		eventType = "payment.completed"
		log.Printf("Payment %d processed successfully", payment.ID)
	} else {
		status = "failed"
		eventType = "payment.failed"
		log.Printf("Payment %d failed", payment.ID)
	}

	// Update payment status
	if err := s.repo.UpdateStatus(ctx, payment.ID, status, transactionID); err != nil {
		log.Printf("Failed to update payment status: %v", err)
		return
	}

	// Publish payment event
	event := &models.PaymentEvent{
		Type:      eventType,
		PaymentID: payment.ID,
		OrderID:   payment.OrderID,
		Amount:    payment.Amount,
		Status:    status,
		Timestamp: time.Now(),
	}

	if err := s.producer.PublishPaymentEvent(ctx, event); err != nil {
		log.Printf("Failed to publish payment event: %v", err)
	}
}
