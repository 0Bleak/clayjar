package service

import (
	"context"
	"fmt"
	"log"

	"github.com/0Bleak/inventory-service/internal/messaging"
	"github.com/0Bleak/inventory-service/internal/models"
	"github.com/0Bleak/inventory-service/internal/repository"
)

type InventoryService interface {
	CreateInventory(ctx context.Context, req *models.CreateInventoryRequest) (*models.Inventory, error)
	GetInventory(ctx context.Context, jarID string) (*models.Inventory, error)
	UpdateInventory(ctx context.Context, jarID string, req *models.UpdateInventoryRequest) (*models.Inventory, error)
	HandleOrderEvent(ctx context.Context, event *models.OrderEvent) error
}

type inventoryService struct {
	repo     repository.InventoryRepository
	producer messaging.KafkaProducer
}

func NewInventoryService(repo repository.InventoryRepository, producer messaging.KafkaProducer) InventoryService {
	return &inventoryService{
		repo:     repo,
		producer: producer,
	}
}

func (s *inventoryService) CreateInventory(ctx context.Context, req *models.CreateInventoryRequest) (*models.Inventory, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	inventory := &models.Inventory{
		JarID:    req.JarID,
		Quantity: req.Quantity,
		Reserved: 0,
	}

	if err := s.repo.Create(ctx, inventory); err != nil {
		return nil, fmt.Errorf("failed to create inventory: %w", err)
	}

	// Publish inventory created event
	event := &models.InventoryEvent{
		Type:      "inventory.created",
		JarID:     inventory.JarID,
		Quantity:  inventory.Quantity,
		Reserved:  inventory.Reserved,
		Timestamp: inventory.CreatedAt,
	}

	if err := s.producer.PublishInventoryEvent(ctx, event); err != nil {
		log.Printf("Failed to publish inventory created event: %v", err)
	}

	return inventory, nil
}

func (s *inventoryService) GetInventory(ctx context.Context, jarID string) (*models.Inventory, error) {
	return s.repo.FindByJarID(ctx, jarID)
}

func (s *inventoryService) UpdateInventory(ctx context.Context, jarID string, req *models.UpdateInventoryRequest) (*models.Inventory, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	inventory, err := s.repo.FindByJarID(ctx, jarID)
	if err != nil {
		return nil, fmt.Errorf("inventory not found: %w", err)
	}

	inventory.Quantity = req.Quantity

	if err := s.repo.Update(ctx, inventory); err != nil {
		return nil, fmt.Errorf("failed to update inventory: %w", err)
	}

	// Publish inventory updated event
	event := &models.InventoryEvent{
		Type:      "inventory.updated",
		JarID:     inventory.JarID,
		Quantity:  inventory.Quantity,
		Reserved:  inventory.Reserved,
		Timestamp: inventory.UpdatedAt,
	}

	if err := s.producer.PublishInventoryEvent(ctx, event); err != nil {
		log.Printf("Failed to publish inventory updated event: %v", err)
	}

	return inventory, nil
}

func (s *inventoryService) HandleOrderEvent(ctx context.Context, event *models.OrderEvent) error {
	log.Printf("Handling order event: %s for order %d", event.Type, event.OrderID)

	switch event.Type {
	case "order.created":
		// Reserve stock
		if err := s.repo.ReserveStock(ctx, event.JarID, event.Quantity); err != nil {
			log.Printf("Failed to reserve stock: %v", err)
			return err
		}

		log.Printf("Reserved %d units of jar %s for order %d", event.Quantity, event.JarID, event.OrderID)

		// Publish inventory reserved event
		inventoryEvent := &models.InventoryEvent{
			Type:      "inventory.reserved",
			JarID:     event.JarID,
			Quantity:  event.Quantity,
			Timestamp: event.Timestamp,
		}

		if err := s.producer.PublishInventoryEvent(ctx, inventoryEvent); err != nil {
			log.Printf("Failed to publish inventory reserved event: %v", err)
		}

	case "order.status_updated":
		if event.Status == "cancelled" {
			// Release stock
			if err := s.repo.ReleaseStock(ctx, event.JarID, event.Quantity); err != nil {
				log.Printf("Failed to release stock: %v", err)
				return err
			}

			log.Printf("Released %d units of jar %s for cancelled order %d", event.Quantity, event.JarID, event.OrderID)

			// Publish inventory released event
			inventoryEvent := &models.InventoryEvent{
				Type:      "inventory.released",
				JarID:     event.JarID,
				Quantity:  event.Quantity,
				Timestamp: event.Timestamp,
			}

			if err := s.producer.PublishInventoryEvent(ctx, inventoryEvent); err != nil {
				log.Printf("Failed to publish inventory released event: %v", err)
			}
		}
	}

	return nil
}
