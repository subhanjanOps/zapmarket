package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/logistics-service/internal/domain"
)

type ShipmentRequest struct {
	OrderID         string
	DeliveryPincode string
	WeightGrams     int
	SellerPincode   string
}

type carrierClient interface {
	CreateShipment(ctx context.Context, req ShipmentRequest) (awb, carrier, trackingURL string, err error)
	CreateReversePickup(ctx context.Context, parentAWB, orderID string) (awb, carrier, trackingURL string, err error)
}

type shipmentSaver interface {
	Save(ctx context.Context, s *domain.Shipment) error
}

type CreateShipmentUseCase struct {
	carrier carrierClient
	repo    shipmentSaver
}

func NewCreateShipmentUseCase(carrier carrierClient, repo shipmentSaver) *CreateShipmentUseCase {
	return &CreateShipmentUseCase{carrier: carrier, repo: repo}
}

func (uc *CreateShipmentUseCase) Execute(ctx context.Context, req ShipmentRequest) (*domain.Shipment, error) {
	awb, carrier, trackingURL, err := uc.carrier.CreateShipment(ctx, req)
	if err != nil {
		return nil, err
	}
	shipment := &domain.Shipment{
		ID:                uuid.NewString(),
		OrderID:           req.OrderID,
		CarrierShipmentID: awb,
		Carrier:           carrier,
		TrackingURL:       trackingURL,
		Status:            domain.ShipmentStatusCreated,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	if err := uc.repo.Save(ctx, shipment); err != nil {
		return nil, err
	}
	return shipment, nil
}
