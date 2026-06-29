package entities

import "time"

type Warehouse struct {
	ID        string
	Name      string
	City      string
	State     string
	Pincode   string
	IsActive  bool
	CreatedAt time.Time
}

type Inventory struct {
	ID           string
	SKUID        string
	WarehouseID  string
	QtyOnHand    int
	QtyReserved  int
	QtyAvailable int
	UpdatedAt    time.Time
}

type Reservation struct {
	ID          string
	InventoryID string
	OrderID     string
	SKUID       string
	Qty         int
	Status      string
	ExpiresAt   time.Time
	CreatedAt   time.Time
}

type LedgerEntry struct {
	ID           string
	InventoryID  string
	OrderID      string
	MovementType string
	QtyDelta     int
	QtyBefore    int
	QtyAfter     int
	Note         string
	CreatedAt    time.Time
}
