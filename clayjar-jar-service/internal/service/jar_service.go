package service

import (
	"context"
	"fmt"

	"github.com/0Bleak/clayjar-jar-service/internal/messaging"
	"github.com/0Bleak/clayjar-jar-service/internal/models"
	"github.com/0Bleak/clayjar-jar-service/internal/repository"
)

type JarService interface {
	CreateJar(ctx context.Context, req *models.CreateJarRequest) (*models.Jar, error)
	GetJarByID(ctx context.Context, id string) (*models.Jar, error)
	GetAllJars(ctx context.Context, limit, offset int64) ([]*models.Jar, error)
	UpdateJar(ctx context.Context, id string, req *models.CreateJarRequest) (*models.Jar, error)
	DeleteJar(ctx context.Context, id string) error
}

type jarService struct {
	repo     repository.JarRepository
	producer messaging.KafkaProducer
}

func NewJarService(repo repository.JarRepository, producer messaging.KafkaProducer) JarService {
	return &jarService{
		repo:     repo,
		producer: producer,
	}
}

func (s *jarService) CreateJar(ctx context.Context, req *models.CreateJarRequest) (*models.Jar, error) {
	jar := &models.Jar{
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Price:       req.Price,
		StockQty:    req.StockQty,
		ImageUrl:    req.ImageURL,
		Attributes:  req.Attributes,
	}

	if err := jar.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := s.repo.Create(ctx, jar); err != nil {
		return nil, fmt.Errorf("failed to create jar: %w", err)
	}

	event := models.JarEvent{
		Type:      "jar.created",
		JarID:     jar.ID.Hex(),
		Payload:   jar,
		Timestamp: jar.CreatedAt,
	}

	if err := s.producer.PublishJarEvent(ctx, &event); err != nil {
		return nil, fmt.Errorf("failed to publish jar created event: %w", err)
	}

	return jar, nil
}

func (s *jarService) GetJarByID(ctx context.Context, id string) (*models.Jar, error) {
	jar, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch jar: %w", err)
	}

	return jar, nil
}

func (s *jarService) GetAllJars(ctx context.Context, limit, offset int64) ([]*models.Jar, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	jars, err := s.repo.FindAll(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch jars: %w", err)
	}

	return jars, nil
}

func (s *jarService) UpdateJar(ctx context.Context, id string, req *models.CreateJarRequest) (*models.Jar, error) {
	existingJar, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("jar not found: %w", err)
	}

	existingJar.Name = req.Name
	existingJar.Description = req.Description
	existingJar.Category = req.Category
	existingJar.Price = req.Price
	existingJar.StockQty = req.StockQty
	existingJar.ImageUrl = req.ImageURL
	existingJar.Attributes = req.Attributes

	if err := existingJar.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := s.repo.Update(ctx, id, existingJar); err != nil {
		return nil, fmt.Errorf("failed to update jar: %w", err)
	}

	event := models.JarEvent{
		Type:      "jar.updated",
		JarID:     existingJar.ID.Hex(),
		Payload:   existingJar,
		Timestamp: existingJar.UpdatedAt,
	}

	if err := s.producer.PublishJarEvent(ctx, &event); err != nil {
		return nil, fmt.Errorf("failed to publish jar updated event: %w", err)
	}

	return existingJar, nil
}

func (s *jarService) DeleteJar(ctx context.Context, id string) error {
	jar, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("jar not found: %w", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete jar: %w", err)
	}

	event := models.JarEvent{
		Type:      "jar.deleted",
		JarID:     jar.ID.Hex(),
		Payload:   nil,
		Timestamp: jar.UpdatedAt,
	}

	if err := s.producer.PublishJarEvent(ctx, &event); err != nil {
		return fmt.Errorf("failed to publish jar deleted event: %w", err)
	}

	return nil
}
