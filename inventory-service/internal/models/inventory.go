package models

import (
	"errors"
	"time"
)

type Inventory struct {
	ID        int64     `db:"id" json:"id"`
	JarID     string    `db:"jar_id" json:"jar_id"`
	Quantity  int       `db:"quantity" json:"quantity"`
	Reserved  int       `db:"reserved" json:"reserved"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type CreateInventoryRequest struct {
	JarID    string `json:"jar_id"`
	Quantity int    `json:"quantity"`
}

type UpdateInventoryRequest struct {
	Quantity int `json:"quantity"`
}

type InventoryEvent struct {
	Type      string    `json:"type"`
	JarID     string    `json:"jar_id"`
	Quantity  int       `json:"quantity"`
	Reserved  int       `json:"reserved"`
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

func (r *CreateInventoryRequest) Validate() error {
	if r.JarID == "" {
		return errors.New("jar_id is required")
	}
	if r.Quantity < 0 {
		return errors.New("quantity cannot be negative")
	}
	return nil
}

func (r *UpdateInventoryRequest) Validate() error {
	if r.Quantity < 0 {
		return errors.New("quantity cannot be negative")
	}
	return nil
}
