package models

import (
	"errors"
	"time"
)

type Order struct {
	ID         int64     `db:"id" json:"id"`
	UserID     int64     `db:"user_id" json:"user_id"`
	JarID      string    `db:"jar_id" json:"jar_id"`
	Quantity   int       `db:"quantity" json:"quantity"`
	TotalPrice float64   `db:"total_price" json:"total_price"`
	Status     string    `db:"status" json:"status"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

type CreateOrderRequest struct {
	UserID     int64   `json:"user_id"`
	JarID      string  `json:"jar_id"`
	Quantity   int     `json:"quantity"`
	TotalPrice float64 `json:"total_price"`
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

type PaymentEvent struct {
	Type      string    `json:"type"`
	PaymentID int64     `json:"payment_id"`
	OrderID   int64     `json:"order_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func (r *CreateOrderRequest) Validate() error {
	if r.UserID <= 0 {
		return errors.New("user_id is required")
	}
	if r.JarID == "" {
		return errors.New("jar_id is required")
	}
	if r.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	if r.TotalPrice <= 0 {
		return errors.New("total_price must be positive")
	}
	return nil
}
