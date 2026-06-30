package usecases_test

import (
	"context"
	"testing"

	"github.com/zapmarket/zapmarket/services/logistics-service/internal/application/usecases"
	"github.com/zapmarket/zapmarket/services/logistics-service/internal/domain"
)

type fakeCarrierClient struct{}

func (f *fakeCarrierClient) CreateShipment(_ context.Context, _ usecases.ShipmentRequest) (awb, carrier, trackingURL string, err error) {
	return "AWB123", "Delhivery", "https://track.example.com/AWB123", nil
}

func (f *fakeCarrierClient) CreateReversePickup(_ context.Context, _, _ string) (awb, carrier, trackingURL string, err error) {
	return "AWB123-R", "Delhivery", "https://track.example.com/AWB123-R", nil
}

type fakeShipmentRepo struct{ saved bool }

func (f *fakeShipmentRepo) Save(_ context.Context, _ *domain.Shipment) error {
	f.saved = true
	return nil
}

func TestCreateShipment_SavesWithAWB(t *testing.T) {
	client := &fakeCarrierClient{}
	repo := &fakeShipmentRepo{}
	uc := usecases.NewCreateShipmentUseCase(client, repo)

	shipment, err := uc.Execute(context.Background(), usecases.ShipmentRequest{
		OrderID:         "ord-1",
		DeliveryPincode: "400001",
		WeightGrams:     500,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.saved {
		t.Fatal("expected shipment to be saved")
	}
	if shipment.CarrierShipmentID != "AWB123" {
		t.Fatalf("expected AWB123, got %s", shipment.CarrierShipmentID)
	}
}
