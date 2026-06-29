package domain

import "time"

type ShipmentStatus string

const (
	ShipmentStatusCreated         ShipmentStatus = "CREATED"
	ShipmentStatusPickedUp        ShipmentStatus = "PICKED_UP"
	ShipmentStatusInTransit       ShipmentStatus = "IN_TRANSIT"
	ShipmentStatusOutForDelivery  ShipmentStatus = "OUT_FOR_DELIVERY"
	ShipmentStatusDelivered       ShipmentStatus = "DELIVERED"
	ShipmentStatusFailed          ShipmentStatus = "DELIVERY_FAILED"
)

type Shipment struct {
	ID                string
	OrderID           string
	CarrierShipmentID string
	Carrier           string
	TrackingURL       string
	Status            ShipmentStatus
	LabelURL          string
	EstimatedDelivery *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type TrackingEvent struct {
	Status      string
	Description string
	Location    string
	OccurredAt  time.Time
}
