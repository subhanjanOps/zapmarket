package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/database"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/domain"
)

type InventoryRepository struct {
	db *sql.DB
}

func NewInventoryRepository(db *sql.DB) *InventoryRepository {
	return &InventoryRepository{db}
}

func (r *InventoryRepository) AddStock(ctx context.Context, skuID uuid.UUID, qty int) (int, error) {
	var newQty int

	err := database.WithTransaction(ctx, r.db, func(tx *sql.Tx) error {
		var inventoryID uuid.UUID
		var before int

		err := tx.QueryRowContext(ctx, `
			INSERT INTO inventory (sku_id, warehouse_id, qty_on_hand)
			VALUES ($1, $2, $3)
			ON CONFLICT (sku_id, warehouse_id) DO UPDATE
				SET qty_on_hand = inventory.qty_on_hand + EXCLUDED.qty_on_hand,
					updated_at = NOW()
			RETURNING id, qty_on_hand - $3
		`, skuID, domain.DefaultWarehouseID, qty).Scan(&inventoryID, &before)
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to add stock", err)
		}

		after := before + qty
		newQty = after

		return insertLedgerEntry(ctx, tx, inventoryID, nil, domain.MovementPurchaseOrder, qty, before, after, "")
	})

	return newQty, err
}

func (r *InventoryRepository) ReserveStock(ctx context.Context, skuID, orderID uuid.UUID, qty int) (*domain.Reservation, error) {
	var reservation *domain.Reservation

	err := database.WithTransaction(ctx, r.db, func(tx *sql.Tx) error {
		var inventoryID uuid.UUID
		var reservedBefore, reservedAfter int

		err := tx.QueryRowContext(ctx, `
			UPDATE inventory
			SET qty_reserved = qty_reserved + $2,
				updated_at = NOW()
			WHERE sku_id = $1
				AND warehouse_id = $3
				AND deleted_at IS NULL
				AND qty_on_hand - qty_reserved >= $2
			RETURNING id, qty_reserved - $2, qty_reserved
		`, skuID, qty, domain.DefaultWarehouseID).Scan(&inventoryID, &reservedBefore, &reservedAfter)

		if errors.Is(err, sql.ErrNoRows) {
			// Either no inventory row exists for this SKU, or there isn't
			// enough qty_available — both mean "cannot reserve right now",
			// which is a normal outcome the caller checks for, not an error.
			reservation = nil
			return nil
		}
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to reserve stock", err)
		}

		id := uuid.New()
		expiresAt := time.Now().Add(domain.ReservationTTL)

		_, err = tx.ExecContext(ctx, `
			INSERT INTO inventory_reservations (id, inventory_id, order_id, sku_id, qty, status, expires_at)
			VALUES ($1, $2, $3, $4, $5, 'RESERVED', $6)
		`, id, inventoryID, orderID, skuID, qty, expiresAt)
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to create reservation", err)
		}

		if err := insertLedgerEntry(ctx, tx, inventoryID, &orderID, domain.MovementReservation, qty, reservedBefore, reservedAfter, ""); err != nil {
			return err
		}

		reservation = &domain.Reservation{
			ID:          id,
			InventoryID: inventoryID,
			OrderID:     orderID,
			SKUID:       skuID,
			Qty:         qty,
			Status:      domain.ReservationReserved,
			ExpiresAt:   expiresAt,
		}
		return nil
	})

	return reservation, err
}

func (r *InventoryRepository) ReleaseStock(ctx context.Context, reservationID uuid.UUID) error {
	return database.WithTransaction(ctx, r.db, func(tx *sql.Tx) error {
		var inventoryID uuid.UUID
		var qty int

		err := tx.QueryRowContext(ctx, `
			UPDATE inventory_reservations
			SET status = 'RELEASED',
				updated_at = NOW()
			WHERE id = $1
				AND status = 'RESERVED'
				AND deleted_at IS NULL
			RETURNING inventory_id, qty
		`, reservationID).Scan(&inventoryID, &qty)

		if errors.Is(err, sql.ErrNoRows) {
			return reservationNotFoundOrConflict(ctx, tx, reservationID)
		}
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to release reservation", err)
		}

		var before, after int
		err = tx.QueryRowContext(ctx, `
			UPDATE inventory
			SET qty_reserved = qty_reserved - $2,
				updated_at = NOW()
			WHERE id = $1
			RETURNING qty_reserved + $2, qty_reserved
		`, inventoryID, qty).Scan(&before, &after)
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to release stock", err)
		}

		return insertLedgerEntry(ctx, tx, inventoryID, nil, domain.MovementReservationRelease, -qty, before, after, "")
	})
}

func (r *InventoryRepository) DeductStock(ctx context.Context, reservationID uuid.UUID) error {
	return database.WithTransaction(ctx, r.db, func(tx *sql.Tx) error {
		var inventoryID uuid.UUID
		var qty int

		err := tx.QueryRowContext(ctx, `
			UPDATE inventory_reservations
			SET status = 'CONFIRMED',
				updated_at = NOW()
			WHERE id = $1
				AND status = 'RESERVED'
				AND deleted_at IS NULL
			RETURNING inventory_id, qty
		`, reservationID).Scan(&inventoryID, &qty)

		if errors.Is(err, sql.ErrNoRows) {
			return reservationNotFoundOrConflict(ctx, tx, reservationID)
		}
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to confirm reservation", err)
		}

		var before, after int
		err = tx.QueryRowContext(ctx, `
			UPDATE inventory
			SET qty_on_hand = qty_on_hand - $2,
				qty_reserved = qty_reserved - $2,
				updated_at = NOW()
			WHERE id = $1
			RETURNING qty_on_hand + $2, qty_on_hand
		`, inventoryID, qty).Scan(&before, &after)
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to deduct stock", err)
		}

		return insertLedgerEntry(ctx, tx, inventoryID, nil, domain.MovementSale, -qty, before, after, "")
	})
}

func (r *InventoryRepository) GetStock(ctx context.Context, skuID uuid.UUID) (*domain.Inventory, error) {
	inv := &domain.Inventory{}

	err := r.db.QueryRowContext(ctx, `
		SELECT id, sku_id, warehouse_id, qty_on_hand, qty_reserved, qty_available, low_stock_threshold, created_at, updated_at
		FROM inventory
		WHERE sku_id = $1
			AND warehouse_id = $2
			AND deleted_at IS NULL
	`, skuID, domain.DefaultWarehouseID).Scan(
		&inv.ID, &inv.SKUID, &inv.WarehouseID, &inv.QtyOnHand, &inv.QtyReserved, &inv.QtyAvailable, &inv.LowStockThreshold,
		&inv.CreatedAt, &inv.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, pkgerrors.NewNotFound("STOCK_NOT_FOUND", "no stock recorded for this sku")
	}
	if err != nil {
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to get stock", err)
	}

	return inv, nil
}

// reservationNotFoundOrConflict distinguishes "this reservation id never
// existed" from "it exists but isn't in `reserved` state anymore" so
// callers attempting a double release/deduct get a meaningful error
// instead of a generic not-found.
func reservationNotFoundOrConflict(ctx context.Context, tx *sql.Tx, reservationID uuid.UUID) error {
	var status string
	err := tx.QueryRowContext(ctx, `SELECT status FROM inventory_reservations WHERE id = $1 AND deleted_at IS NULL`, reservationID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return pkgerrors.NewNotFound("RESERVATION_NOT_FOUND", "reservation not found")
	}
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to look up reservation", err)
	}
	return pkgerrors.NewConflict("RESERVATION_NOT_RESERVED", "reservation is already "+status)
}

func insertLedgerEntry(ctx context.Context, tx *sql.Tx, inventoryID uuid.UUID, orderID *uuid.UUID, movementType domain.MovementType, qtyDelta, qtyBefore, qtyAfter int, note string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO inventory_ledger (inventory_id, order_id, movement_type, qty_delta, qty_before, qty_after, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, inventoryID, orderID, movementType, qtyDelta, qtyBefore, qtyAfter, sql.NullString{String: note, Valid: note != ""})
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to write ledger entry", err)
	}
	return nil
}
