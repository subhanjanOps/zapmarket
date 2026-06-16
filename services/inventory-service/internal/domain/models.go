package domain

import (
	"time"

	"github.com/google/uuid"
)

// DefaultWarehouseID is the single warehouse every SKU's stock lives in for
// this stage — see migrations/0001_init.up.sql. The schema supports
// multiple warehouses; the gRPC surface just doesn't expose warehouse_id
// yet (see pkg/proto/inventory/inventory.proto).
var DefaultWarehouseID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type Warehouse struct {
	ID        uuid.UUID
	Name      string
	City      string
	State     string
	Pincode   string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// Inventory is the per-SKU-per-warehouse stock row. QtyAvailable is a
// generated column in Postgres (qty_on_hand - qty_reserved); it's read-only
// here, never written by the application.
type Inventory struct {
	ID                uuid.UUID
	SKUID             uuid.UUID
	WarehouseID       uuid.UUID
	QtyOnHand         int
	QtyReserved       int
	QtyAvailable      int
	LowStockThreshold int
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}

type MovementType string

const (
	MovementPurchaseOrder      MovementType = "purchase_order"
	MovementReservation        MovementType = "reservation"
	MovementReservationRelease MovementType = "reservation_release"
	MovementSale               MovementType = "sale"
	MovementReturn             MovementType = "return"
	MovementAdjustment         MovementType = "adjustment"
)

// LedgerEntry is an immutable record of a stock movement — no UpdatedAt or
// DeletedAt, append-only by design.
type LedgerEntry struct {
	ID           uuid.UUID
	InventoryID  uuid.UUID
	OrderID      *uuid.UUID
	MovementType MovementType
	QtyDelta     int
	QtyBefore    int
	QtyAfter     int
	Note         string
	CreatedAt    time.Time
}

type ReservationStatus string

const (
	ReservationReserved  ReservationStatus = "reserved"
	ReservationConfirmed ReservationStatus = "confirmed"
	ReservationReleased  ReservationStatus = "released"
)

// Reservation TTL: how long a `reserved` row is honored before it's
// considered abandoned. No sweep job exists yet to actively release expired
// rows (out of scope for this stage) — ExpiresAt is recorded so one can be
// added later without a schema change.
const ReservationTTL = 15 * time.Minute

type Reservation struct {
	ID          uuid.UUID
	InventoryID uuid.UUID
	OrderID     uuid.UUID
	SKUID       uuid.UUID
	Qty         int
	Status      ReservationStatus
	ExpiresAt   time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}
