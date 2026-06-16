package grpc

import (
	"context"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	pb "github.com/zapmarket/zapmarket/pkg/proto/payment"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PaymentGRPCHandler struct {
	pb.UnimplementedPaymentServiceServer
	svc service.PaymentService
}

func NewPaymentGRPCHandler(svc service.PaymentService) *PaymentGRPCHandler {
	return &PaymentGRPCHandler{svc: svc}
}

func (h *PaymentGRPCHandler) ChargeCard(ctx context.Context, req *pb.ChargeCardRequest) (*pb.ChargeCardResponse, error) {
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %s", req.OrderId)
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %s", req.UserId)
	}
	idempotencyKey, err := uuid.Parse(req.IdempotencyKey)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid idempotency_key: %s", req.IdempotencyKey)
	}

	payment, err := h.svc.ChargeCard(ctx, orderID, userID, req.Amount, req.Currency, idempotencyKey)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.ChargeCardResponse{
		PaymentId: payment.ID.String(),
		Status:    string(payment.Status),
	}, nil
}

func (h *PaymentGRPCHandler) RefundPayment(ctx context.Context, req *pb.RefundPaymentRequest) (*pb.RefundPaymentResponse, error) {
	paymentID, err := uuid.Parse(req.PaymentId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid payment_id: %s", req.PaymentId)
	}

	refund, err := h.svc.RefundPayment(ctx, paymentID, req.Amount, req.Reason)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.RefundPaymentResponse{
		RefundId: refund.ID.String(),
		Status:   string(refund.Status),
	}, nil
}

func (h *PaymentGRPCHandler) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.GetTransactionResponse, error) {
	paymentID, err := uuid.Parse(req.PaymentId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid payment_id: %s", req.PaymentId)
	}

	payment, err := h.svc.GetTransaction(ctx, paymentID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	resp := &pb.GetTransactionResponse{
		PaymentId: payment.ID.String(),
		OrderId:   payment.OrderID.String(),
		Status:    string(payment.Status),
		Amount:    payment.Amount,
		Currency:  payment.Currency,
	}
	if payment.GatewayTxnID != nil {
		resp.GatewayTxnId = *payment.GatewayTxnID
	}
	return resp, nil
}

func toGRPCError(err error) error {
	return pkgerrors.ToGRPCStatus(err)
}
