package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/zapmarket/zapmarket/pkg/database"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/domain"
)

// defaultWarehouseID is the seed warehouse from migrations/0001_init.up.sql.
// Replace with a zone-lookup call once multi-warehouse routing is live.
var defaultWarehouseID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

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
		`, skuID, defaultWarehouseID, qty).Scan(&inventoryID, &before)
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
		`, skuID, qty, defaultWarehouseID).Scan(&inventoryID, &reservedBefore, &reservedAfter)

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
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
				// Already reserved for this (order_id, sku_id) — return existing reservation idempotently.
				var res domain.Reservation
				err2 := tx.QueryRowContext(ctx, `
					SELECT id, inventory_id, order_id, sku_id, qty, status, expires_at, created_at
					FROM inventory_reservations
					WHERE order_id = $1 AND sku_id = $2
				`, orderID, skuID).Scan(&res.ID, &res.InventoryID, &res.OrderID, &res.SKUID, &res.Qty, &res.Status, &res.ExpiresAt, &res.CreatedAt)
				if err2 != nil {
					return pkgerrors.NewInternal("DATABASE_ERROR", "failed to fetch existing reservation", err2)
				}
				reservation = &res
				return nil
			}
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

		payload, err := json.Marshal(map[string]any{
			"reservation_id": id.String(), "sku_id": skuID.String(),
			"order_id": orderID.String(), "qty": qty, "status": "RESERVED",
		})
		if err != nil {
			return fmt.Errorf("marshal outbox payload: %w", err)
		}
		return insertOutboxRow(ctx, tx, id, "inventory", "inventory.reserved", payload)
	})

	return reservation, err
}

func (r *InventoryRepository) ReleaseStock(ctx context.Context, reservationID uuid.UUID) (skuID uuid.UUID, qty int64, err error) {
	txErr := database.WithTransaction(ctx, r.db, func(tx *sql.Tx) error {
		var inventoryID uuid.UUID
		var releaseQty int

		err := tx.QueryRowContext(ctx, `
			UPDATE inventory_reservations
			SET status = 'RELEASED',
				updated_at = NOW()
			WHERE id = $1
				AND status = 'RESERVED'
				AND deleted_at IS NULL
			RETURNING inventory_id, sku_id, qty
		`, reservationID).Scan(&inventoryID, &skuID, &releaseQty)

		if errors.Is(err, sql.ErrNoRows) {
			return reservationNotFoundOrConflict(ctx, tx, reservationID)
		}
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to release reservation", err)
		}

		qty = int64(releaseQty)

		var before, after int
		err = tx.QueryRowContext(ctx, `
			UPDATE inventory
			SET qty_reserved = qty_reserved - $2,
				updated_at = NOW()
			WHERE id = $1
			RETURNING qty_reserved + $2, qty_reserved
		`, inventoryID, releaseQty).Scan(&before, &after)
		if err != nil {
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to release stock", err)
		}

		if err := insertLedgerEntry(ctx, tx, inventoryID, nil, domain.MovementReservationRelease, -releaseQty, before, after, ""); err != nil {
			return err
		}

		payload, err := json.Marshal(map[string]any{
			"reservation_id": reservationID.String(), "qty": releaseQty, "status": "RELEASED",
		})
		if err != nil {
			return fmt.Errorf("marshal outbox payload: %w", err)
		}
		return insertOutboxRow(ctx, tx, reservationID, "inventory", "inventory.released", payload)
	})
	if txErr != nil {
		return uuid.Nil, 0, txErr
	}
	return skuID, qty, nil
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
	`, skuID, defaultWarehouseID).Scan(
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

func insertOutboxRow(ctx context.Context, tx *sql.Tx, aggregateID uuid.UUID, aggregateType, eventType string, payload []byte) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO outbox (aggregate_id, aggregate_type, event_type, payload)
		VALUES ($1, $2, $3, $4)
	`, aggregateID, aggregateType, eventType, payload)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to write outbox row", err)
	}
	return nil
}

func (r *InventoryRepository) GetReservationDetails(ctx context.Context, reservationID uuid.UUID) (uuid.UUID, int, error) {
	var skuID uuid.UUID
	var qty int
	err := r.db.QueryRowContext(ctx, `
		SELECT sku_id, qty FROM inventory_reservations WHERE id = $1 AND deleted_at IS NULL
	`, reservationID).Scan(&skuID, &qty)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, 0, pkgerrors.NewNotFound("RESERVATION_NOT_FOUND", "reservation not found")
	}
	if err != nil {
		return uuid.Nil, 0, pkgerrors.NewInternal("DATABASE_ERROR", "failed to get reservation details", err)
	}
	return skuID, qty, nil
}

func (r *InventoryRepository) FindExpiredReservations(ctx context.Context, before time.Time) ([]*domain.Reservation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, inventory_id, order_id, sku_id, qty, status, expires_at, created_at
		FROM inventory_reservations
		WHERE status = 'RESERVED' AND expires_at < $1 AND deleted_at IS NULL
	`, before)
	if err != nil {
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to find expired reservations", err)
	}
	defer rows.Close()

	var out []*domain.Reservation
	for rows.Next() {
		var res domain.Reservation
		if err := rows.Scan(&res.ID, &res.InventoryID, &res.OrderID, &res.SKUID, &res.Qty, &res.Status, &res.ExpiresAt, &res.CreatedAt); err != nil {
			return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to scan expired reservation", err)
		}
		out = append(out, &res)
	}
	return out, rows.Err()
}

func (r *InventoryRepository) WriteReservationExpiredEvent(ctx context.Context, reservationID, orderID uuid.UUID) error {
	payload, err := json.Marshal(map[string]string{
		"reservation_id": reservationID.String(),
		"order_id":       orderID.String(),
		"status":         "EXPIRED",
	})
	if err != nil {
		return fmt.Errorf("marshal reservation.expired payload: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO outbox (aggregate_id, aggregate_type, event_type, payload)
		VALUES ($1, 'inventory', 'reservation.expired', $2)
	`, reservationID, payload)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to write reservation.expired outbox row", err)
	}
	return nil
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
