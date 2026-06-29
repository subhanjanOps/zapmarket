package usecases

import "context"

type ReserveStockUseCase interface {
	Execute(ctx context.Context, req ReserveStockRequest) (*ReserveStockResult, error)
}

type ReleaseStockUseCase interface {
	Execute(ctx context.Context, reservationID string) error
}

type DeductStockUseCase interface {
	Execute(ctx context.Context, reservationID string) error
}

type AddStockUseCase interface {
	Execute(ctx context.Context, skuID, warehouseID string, qty int) error
}

type GetStockUseCase interface {
	Execute(ctx context.Context, skuID, warehouseID string) (*StockResult, error)
}
