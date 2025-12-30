package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/0Bleak/inventory-service/internal/models"
	"github.com/jmoiron/sqlx"
)

type InventoryRepository interface {
	Create(ctx context.Context, inventory *models.Inventory) error
	FindByJarID(ctx context.Context, jarID string) (*models.Inventory, error)
	Update(ctx context.Context, inventory *models.Inventory) error
	ReserveStock(ctx context.Context, jarID string, quantity int) error
	ReleaseStock(ctx context.Context, jarID string, quantity int) error
}

type inventoryRepository struct {
	db *sqlx.DB
}

func NewInventoryRepository(db *sqlx.DB) InventoryRepository {
	return &inventoryRepository{db: db}
}

func (r *inventoryRepository) Create(ctx context.Context, inventory *models.Inventory) error {
	query := `
		INSERT INTO inventory (jar_id, quantity, reserved, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	now := time.Now()
	err := r.db.QueryRowContext(
		ctx,
		query,
		inventory.JarID,
		inventory.Quantity,
		inventory.Reserved,
		now,
		now,
	).Scan(&inventory.ID)

	if err != nil {
		return fmt.Errorf("failed to create inventory: %w", err)
	}

	inventory.CreatedAt = now
	inventory.UpdatedAt = now
	return nil
}

func (r *inventoryRepository) FindByJarID(ctx context.Context, jarID string) (*models.Inventory, error) {
	var inventory models.Inventory
	query := `SELECT id, jar_id, quantity, reserved, created_at, updated_at FROM inventory WHERE jar_id = $1`

	err := r.db.GetContext(ctx, &inventory, query, jarID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("inventory not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find inventory: %w", err)
	}

	return &inventory, nil
}

func (r *inventoryRepository) Update(ctx context.Context, inventory *models.Inventory) error {
	query := `
		UPDATE inventory 
		SET quantity = $1, reserved = $2, updated_at = $3 
		WHERE jar_id = $4
	`

	_, err := r.db.ExecContext(ctx, query, inventory.Quantity, inventory.Reserved, time.Now(), inventory.JarID)
	if err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	return nil
}

func (r *inventoryRepository) ReserveStock(ctx context.Context, jarID string, quantity int) error {
	query := `
		UPDATE inventory 
		SET quantity = quantity - $1, reserved = reserved + $1, updated_at = $2
		WHERE jar_id = $3 AND quantity >= $1
	`

	result, err := r.db.ExecContext(ctx, query, quantity, time.Now(), jarID)
	if err != nil {
		return fmt.Errorf("failed to reserve stock: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("insufficient stock")
	}

	return nil
}

func (r *inventoryRepository) ReleaseStock(ctx context.Context, jarID string, quantity int) error {
	query := `
		UPDATE inventory 
		SET quantity = quantity + $1, reserved = reserved - $1, updated_at = $2
		WHERE jar_id = $3
	`

	_, err := r.db.ExecContext(ctx, query, quantity, time.Now(), jarID)
	if err != nil {
		return fmt.Errorf("failed to release stock: %w", err)
	}

	return nil
}
