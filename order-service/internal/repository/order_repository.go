package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/0Bleak/order-service/internal/models"
	"github.com/jmoiron/sqlx"
)

type OrderRepository interface {
	Create(ctx context.Context, order *models.Order) error
	FindByID(ctx context.Context, id int64) (*models.Order, error)
	FindAll(ctx context.Context, limit, offset int64) ([]*models.Order, error)
	FindByUserID(ctx context.Context, userID int64) ([]*models.Order, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
}

type orderRepository struct {
	db *sqlx.DB
}

func NewOrderRepository(db *sqlx.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(ctx context.Context, order *models.Order) error {
	query := `
		INSERT INTO orders (user_id, jar_id, quantity, total_price, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	now := time.Now()
	err := r.db.QueryRowContext(
		ctx,
		query,
		order.UserID,
		order.JarID,
		order.Quantity,
		order.TotalPrice,
		order.Status,
		now,
		now,
	).Scan(&order.ID)

	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	order.CreatedAt = now
	order.UpdatedAt = now
	return nil
}

func (r *orderRepository) FindByID(ctx context.Context, id int64) (*models.Order, error) {
	var order models.Order
	query := `SELECT id, user_id, jar_id, quantity, total_price, status, created_at, updated_at FROM orders WHERE id = $1`

	err := r.db.GetContext(ctx, &order, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find order: %w", err)
	}

	return &order, nil
}

func (r *orderRepository) FindAll(ctx context.Context, limit, offset int64) ([]*models.Order, error) {
	var orders []*models.Order
	query := `SELECT id, user_id, jar_id, quantity, total_price, status, created_at, updated_at 
	          FROM orders ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	err := r.db.SelectContext(ctx, &orders, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find orders: %w", err)
	}

	return orders, nil
}

func (r *orderRepository) FindByUserID(ctx context.Context, userID int64) ([]*models.Order, error) {
	var orders []*models.Order
	query := `SELECT id, user_id, jar_id, quantity, total_price, status, created_at, updated_at 
	          FROM orders WHERE user_id = $1 ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &orders, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find orders by user: %w", err)
	}

	return orders, nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, id int64, status string) error {
	query := `UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3`

	_, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}
