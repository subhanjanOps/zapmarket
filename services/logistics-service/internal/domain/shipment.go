package domain

import "time"

type ShipmentStatus string

const (
	ShipmentStatusCreated             ShipmentStatus = "CREATED"
	ShipmentStatusPickedUp            ShipmentStatus = "PICKED_UP"
	ShipmentStatusInTransit           ShipmentStatus = "IN_TRANSIT"
	ShipmentStatusOutForDelivery      ShipmentStatus = "OUT_FOR_DELIVERY"
	ShipmentStatusDelivered           ShipmentStatus = "DELIVERED"
	ShipmentStatusFailed              ShipmentStatus = "DELIVERY_FAILED"
	ShipmentStatusReattemptScheduled  ShipmentStatus = "REATTEMPT_SCHEDULED"
	ShipmentStatusUndelivered         ShipmentStatus = "UNDELIVERED"
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
	AssignedAgentID   *string
	AttemptCount      int
	NextAttemptAt     *time.Time
	LastAttemptAt     *time.Time
	ShipmentType      string
	ParentShipmentID  *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type TrackingEvent struct {
	Status      string
	Description string
	Location    string
	OccurredAt  time.Time
}

type DeliveryAgent struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Phone       string    `json:"phone"`
	VehicleType string    `json:"vehicle_type"`
	Zone        string    `json:"zone"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type ProofOfDelivery struct {
	ID          string    `json:"id"`
	ShipmentID  string    `json:"shipment_id"`
	Method      string    `json:"method"`
	OTPVerified bool      `json:"otp_verified"`
	PhotoURL    *string   `json:"photo_url,omitempty"`
	DeliveredAt time.Time `json:"delivered_at"`
	DeliveredBy string    `json:"delivered_by"`
}
