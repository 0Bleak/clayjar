package models

import (
	"errors"
	"time"
)

type Payment struct {
	ID            int64     `db:"id" json:"id"`
	OrderID       int64     `db:"order_id" json:"order_id"`
	Amount        float64   `db:"amount" json:"amount"`
	Status        string    `db:"status" json:"status"`
	PaymentMethod string    `db:"payment_method" json:"payment_method"`
	TransactionID string    `db:"transaction_id" json:"transaction_id"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

type CreatePaymentRequest struct {
	OrderID       int64   `json:"order_id"`
	Amount        float64 `json:"amount"`
	PaymentMethod string  `json:"payment_method"`
}

type PaymentEvent struct {
	Type      string    `json:"type"`
	PaymentID int64     `json:"payment_id"`
	OrderID   int64     `json:"order_id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type OrderEvent struct {
	Type      string    `json:"type"`
	OrderID   int64     `json:"order_id"`
	UserID    int64     `json:"user_id"`
	JarID     string    `json:"jar_id"`
	Quantity  int       `json:"quantity"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func (r *CreatePaymentRequest) Validate() error {
	if r.OrderID <= 0 {
		return errors.New("order_id is required")
	}
	if r.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	if r.PaymentMethod == "" {
		return errors.New("payment_method is required")
	}
	return nil
}
