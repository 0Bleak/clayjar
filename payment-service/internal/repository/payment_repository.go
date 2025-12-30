package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/0Bleak/payment-service/internal/models"
	"github.com/jmoiron/sqlx"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *models.Payment) error
	FindByID(ctx context.Context, id int64) (*models.Payment, error)
	FindByOrderID(ctx context.Context, orderID int64) (*models.Payment, error)
	UpdateStatus(ctx context.Context, id int64, status, transactionID string) error
}

type paymentRepository struct {
	db *sqlx.DB
}

func NewPaymentRepository(db *sqlx.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	query := `
		INSERT INTO payments (order_id, amount, status, payment_method, transaction_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	now := time.Now()
	err := r.db.QueryRowContext(
		ctx,
		query,
		payment.OrderID,
		payment.Amount,
		payment.Status,
		payment.PaymentMethod,
		payment.TransactionID,
		now,
		now,
	).Scan(&payment.ID)

	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}

	payment.CreatedAt = now
	payment.UpdatedAt = now
	return nil
}

func (r *paymentRepository) FindByID(ctx context.Context, id int64) (*models.Payment, error) {
	var payment models.Payment
	query := `SELECT id, order_id, amount, status, payment_method, transaction_id, created_at, updated_at 
	          FROM payments WHERE id = $1`

	err := r.db.GetContext(ctx, &payment, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payment not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find payment: %w", err)
	}

	return &payment, nil
}

func (r *paymentRepository) FindByOrderID(ctx context.Context, orderID int64) (*models.Payment, error) {
	var payment models.Payment
	query := `SELECT id, order_id, amount, status, payment_method, transaction_id, created_at, updated_at 
	          FROM payments WHERE order_id = $1`

	err := r.db.GetContext(ctx, &payment, query, orderID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payment not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find payment: %w", err)
	}

	return &payment, nil
}

func (r *paymentRepository) UpdateStatus(ctx context.Context, id int64, status, transactionID string) error {
	query := `UPDATE payments SET status = $1, transaction_id = $2, updated_at = $3 WHERE id = $4`

	_, err := r.db.ExecContext(ctx, query, status, transactionID, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	return nil
}
