package grpc

import (
	"context"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	pb "github.com/zapmarket/zapmarket/pkg/proto/inventory"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type InventoryGRPCHandler struct {
	pb.UnimplementedInventoryServiceServer
	svc service.InventoryService
}

func NewInventoryGRPCHandler(svc service.InventoryService) *InventoryGRPCHandler {
	return &InventoryGRPCHandler{svc: svc}
}

func (h *InventoryGRPCHandler) AddStock(ctx context.Context, req *pb.AddStockRequest) (*pb.AddStockResponse, error) {
	skuID, err := uuid.Parse(req.SkuId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid sku_id: %s", req.SkuId)
	}

	qty, err := h.svc.AddStock(ctx, skuID, int(req.Quantity))
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.AddStockResponse{QtyOnHand: int32(qty)}, nil
}

func (h *InventoryGRPCHandler) ReserveStock(ctx context.Context, req *pb.ReserveStockRequest) (*pb.ReserveStockResponse, error) {
	skuID, err := uuid.Parse(req.SkuId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid sku_id: %s", req.SkuId)
	}
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %s", req.OrderId)
	}

	reservation, err := h.svc.ReserveStock(ctx, skuID, orderID, int(req.Quantity))
	if err != nil {
		return nil, toGRPCError(err)
	}
	if reservation == nil {
		// Not enough stock available — an expected outcome the caller
		// (Order Management's saga) branches on, not a transport error.
		return &pb.ReserveStockResponse{Reserved: false}, nil
	}

	return &pb.ReserveStockResponse{
		Reserved:      true,
		ReservationId: reservation.ID.String(),
	}, nil
}

func (h *InventoryGRPCHandler) ReleaseStock(ctx context.Context, req *pb.ReleaseStockRequest) (*pb.ReleaseStockResponse, error) {
	reservationID, err := uuid.Parse(req.ReservationId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid reservation_id: %s", req.ReservationId)
	}

	if err := h.svc.ReleaseStock(ctx, reservationID); err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.ReleaseStockResponse{Released: true}, nil
}

func (h *InventoryGRPCHandler) DeductStock(ctx context.Context, req *pb.DeductStockRequest) (*pb.DeductStockResponse, error) {
	reservationID, err := uuid.Parse(req.ReservationId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid reservation_id: %s", req.ReservationId)
	}

	if err := h.svc.DeductStock(ctx, reservationID); err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.DeductStockResponse{Deducted: true}, nil
}

func (h *InventoryGRPCHandler) GetStock(ctx context.Context, req *pb.GetStockRequest) (*pb.GetStockResponse, error) {
	skuID, err := uuid.Parse(req.SkuId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid sku_id: %s", req.SkuId)
	}

	inv, err := h.svc.GetStock(ctx, skuID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetStockResponse{
		QtyOnHand:    int32(inv.QtyOnHand),
		QtyReserved:  int32(inv.QtyReserved),
		QtyAvailable: int32(inv.QtyAvailable),
	}, nil
}

func toGRPCError(err error) error {
	return pkgerrors.ToGRPCStatus(err)
}
