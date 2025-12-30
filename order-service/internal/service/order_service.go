package service

import (
	"context"
	"fmt"
	"log"

	"github.com/0Bleak/order-service/internal/messaging"
	"github.com/0Bleak/order-service/internal/models"
	"github.com/0Bleak/order-service/internal/repository"
)

type OrderService interface {
	CreateOrder(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error)
	GetOrderByID(ctx context.Context, id int64) (*models.Order, error)
	GetAllOrders(ctx context.Context, limit, offset int64) ([]*models.Order, error)
	GetOrdersByUserID(ctx context.Context, userID int64) ([]*models.Order, error)
	HandlePaymentEvent(ctx context.Context, event *models.PaymentEvent) error
}

type orderService struct {
	repo     repository.OrderRepository
	producer messaging.KafkaProducer
}

func NewOrderService(repo repository.OrderRepository, producer messaging.KafkaProducer) OrderService {
	return &orderService{
		repo:     repo,
		producer: producer,
	}
}

func (s *orderService) CreateOrder(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	order := &models.Order{
		UserID:     req.UserID,
		JarID:      req.JarID,
		Quantity:   req.Quantity,
		TotalPrice: req.TotalPrice,
		Status:     "pending",
	}

	if err := s.repo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Publish order created event
	event := &models.OrderEvent{
		Type:      "order.created",
		OrderID:   order.ID,
		UserID:    order.UserID,
		JarID:     order.JarID,
		Quantity:  order.Quantity,
		Status:    order.Status,
		Timestamp: order.CreatedAt,
	}

	if err := s.producer.PublishOrderEvent(ctx, event); err != nil {
		log.Printf("Failed to publish order created event: %v", err)
	}

	return order, nil
}

func (s *orderService) GetOrderByID(ctx context.Context, id int64) (*models.Order, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *orderService) GetAllOrders(ctx context.Context, limit, offset int64) ([]*models.Order, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.FindAll(ctx, limit, offset)
}

func (s *orderService) GetOrdersByUserID(ctx context.Context, userID int64) ([]*models.Order, error) {
	return s.repo.FindByUserID(ctx, userID)
}

func (s *orderService) HandlePaymentEvent(ctx context.Context, event *models.PaymentEvent) error {
	log.Printf("Handling payment event: %s for order %d", event.Type, event.OrderID)

	var newStatus string
	switch event.Type {
	case "payment.completed":
		newStatus = "confirmed"
	case "payment.failed":
		newStatus = "cancelled"
	default:
		return fmt.Errorf("unknown payment event type: %s", event.Type)
	}

	if err := s.repo.UpdateStatus(ctx, event.OrderID, newStatus); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Publish order status updated event
	order, err := s.repo.FindByID(ctx, event.OrderID)
	if err != nil {
		return fmt.Errorf("failed to find order: %w", err)
	}

	orderEvent := &models.OrderEvent{
		Type:      "order.status_updated",
		OrderID:   order.ID,
		UserID:    order.UserID,
		JarID:     order.JarID,
		Quantity:  order.Quantity,
		Status:    order.Status,
		Timestamp: order.UpdatedAt,
	}

	if err := s.producer.PublishOrderEvent(ctx, orderEvent); err != nil {
		log.Printf("Failed to publish order status updated event: %v", err)
	}

	return nil
}
