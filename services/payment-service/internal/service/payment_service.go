package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/domain/contracts"
)

const paymentIdempotencyTTL = 24 * time.Hour

// PaymentService defines the interface for payment operations.
type PaymentService interface {
	ChargeCard(ctx context.Context, orderID, userID uuid.UUID, amount int64, currency string, idempotencyKey uuid.UUID, paymentMethodID string) (*domain.Payment, error)
	RefundPayment(ctx context.Context, paymentID uuid.UUID, amount int64, reason string) (*domain.Refund, error)
	GetTransaction(ctx context.Context, paymentID uuid.UUID) (*domain.Payment, error)

	// HandleCaptureWebhook and HandleFailureWebhook process a payment
	// gateway's async callback. Both are idempotent: re-processing an
	// already-resolved payment is a no-op since gateways commonly retry.
	HandleCaptureWebhook(ctx context.Context, paymentID uuid.UUID, gatewayTxnID string) error
	HandleFailureWebhook(ctx context.Context, paymentID uuid.UUID, reason string) error
	GetByGatewayTxnID(ctx context.Context, txnID string) (*domain.Payment, error)
}

type paymentService struct {
	repo    contracts.PaymentRepository
	gateway contracts.PaymentGateway
	cache   contracts.IdempotencyCache
	logger  *slog.Logger
}

func NewPaymentService(repo contracts.PaymentRepository, gateway contracts.PaymentGateway, cache contracts.IdempotencyCache, logger *slog.Logger) PaymentService {
	return &paymentService{repo: repo, gateway: gateway, cache: cache, logger: logger}
}

func paymentIdempKey(key uuid.UUID) string {
	return fmt.Sprintf("payment:idem:%s", key)
}

func (s *paymentService) ChargeCard(ctx context.Context, orderID, userID uuid.UUID, amount int64, currency string, idempotencyKey uuid.UUID, paymentMethodID string) (*domain.Payment, error) {
	if orderID == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "order_id is required")
	}
	if userID == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "user_id is required")
	}
	if amount <= 0 {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "amount must be greater than zero")
	}
	if currency == "" {
		currency = "INR"
	}
	if idempotencyKey == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "idempotency_key is required")
	}

	// Cache-first idempotency check (24h TTL), then DB with a NX lock to
	// prevent duplicate payment creation under concurrent requests.
	cacheKey := paymentIdempKey(idempotencyKey)
	lockKey := cacheKey + ":lock"
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var p domain.Payment
		if json.Unmarshal(cached, &p) == nil {
			s.logger.Info("idempotent replay from cache", "payment_id", p.ID)
			return &p, nil
		}
	}

	const lockTTL = 10 * time.Second
	acquired, lockErr := s.cache.SetNX(ctx, lockKey, "1", lockTTL)
	if lockErr != nil {
		s.logger.Warn("idempotency lock unavailable, proceeding without lock", "error", lockErr)
	}
	if !acquired && lockErr == nil {
		// Another request is already creating a payment for this idempotency key.
		// Return 409 immediately — the client should retry after a short delay
		// rather than us spinning and burning cache connections.
		return nil, pkgerrors.NewConflict("IDEMPOTENCY_CONFLICT", "payment creation already in progress for this idempotency key; retry after a moment")
	}
	defer func() {
		if acquired {
			_ = s.cache.Del(ctx, lockKey)
		}
	}()

	// DB fallback.
	existing, err := s.repo.GetByIdempotencyKey(ctx, idempotencyKey)
	if err == nil {
		s.logger.Info("idempotent replay from db", "payment_id", existing.ID, "status", existing.Status)
		if b, err := json.Marshal(existing); err == nil {
			_ = s.cache.Set(ctx, cacheKey, b, paymentIdempotencyTTL)
		}
		return existing, nil
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != pkgerrors.NotFound {
		return nil, err
	}

	// COD orders: create payment in PENDING_COD state and return immediately.
	// Payment transitions to CAPTURED when shipment.delivered event arrives.
	if paymentMethodID == "cod" {
		payment := &domain.Payment{
			OrderID:        orderID,
			UserID:         userID,
			IdempotencyKey: idempotencyKey,
			Status:         domain.PaymentPendingCOD,
			Amount:         amount,
			Currency:       currency,
			Gateway:        "cod",
		}
		if err := s.repo.CreatePayment(ctx, payment); err != nil {
			return nil, err
		}
		s.logger.Info("COD order payment created", "payment_id", payment.ID, "order_id", orderID)
		if b, err := json.Marshal(payment); err == nil {
			_ = s.cache.Set(ctx, cacheKey, b, paymentIdempotencyTTL)
		}
		return payment, nil
	}

	payment := &domain.Payment{
		OrderID:        orderID,
		UserID:         userID,
		IdempotencyKey: idempotencyKey,
		Status:         domain.PaymentPending,
		Amount:         amount,
		Currency:       currency,
		Gateway:        s.gateway.Name(),
	}
	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return nil, err
	}

	s.logger.Info("charging card", "payment_id", payment.ID, "order_id", orderID, "amount", amount, "currency", currency)

	result, chargeErr := s.gateway.Charge(ctx, amount, currency, idempotencyKey, paymentMethodID)
	if chargeErr != nil {
		s.logger.Info("charge declined", "payment_id", payment.ID, "error", chargeErr)
		if err := s.repo.MarkFailed(ctx, payment.ID, chargeErr.Error()); err != nil {
			return nil, err
		}
		payment.Status = domain.PaymentFailed
		reason := chargeErr.Error()
		payment.FailureReason = &reason
		// Don't cache failed payments — caller may retry with a real card.
		return payment, nil
	}

	entries := []*domain.LedgerEntry{
		{EntryType: domain.LedgerDebit, Account: "accounts_receivable", Amount: amount, Currency: currency, Description: "charge for order " + orderID.String()},
		{EntryType: domain.LedgerCredit, Account: "revenue", Amount: amount, Currency: currency, Description: "charge for order " + orderID.String()},
	}
	if err := s.repo.MarkCaptured(ctx, payment.ID, result.GatewayTxnID, entries); err != nil {
		// Card was charged but we can't persist CAPTURED. Attempt a gateway refund
		// so the customer is not debited for an order that won't be fulfilled.
		compensateCtx := context.WithoutCancel(ctx)
		if _, refundErr := s.gateway.Refund(compensateCtx, result.GatewayTxnID, amount, currency); refundErr != nil {
			s.logger.Error("CRITICAL: gateway refund after MarkCaptured failure failed — manual intervention required",
				"payment_id", payment.ID, "gateway_txn_id", result.GatewayTxnID, "refund_error", refundErr)
		} else {
			if mfErr := s.repo.MarkFailed(compensateCtx, payment.ID, "db_capture_failed_gateway_refunded"); mfErr != nil {
				s.logger.Error("CRITICAL: failed to mark payment failed after gateway refund — manual reconciliation required",
					"payment_id", payment.ID,
					"gateway_txn_id", result.GatewayTxnID,
					"mark_failed_error", mfErr,
				)
			}
		}
		return nil, err
	}

	payment.Status = domain.PaymentCaptured
	payment.GatewayTxnID = &result.GatewayTxnID

	// Cache captured payment for 24h.
	if b, err := json.Marshal(payment); err == nil {
		_ = s.cache.Set(ctx, cacheKey, b, paymentIdempotencyTTL)
	}
	return payment, nil
}

func (s *paymentService) RefundPayment(ctx context.Context, paymentID uuid.UUID, amount int64, reason string) (*domain.Refund, error) {
	if paymentID == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "payment_id is required")
	}

	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if payment.Status != domain.PaymentCaptured {
		return nil, pkgerrors.NewConflict("PAYMENT_NOT_CAPTURED", "only a captured payment can be refunded")
	}

	if amount <= 0 {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "refund amount must be greater than zero")
	}
	if amount > payment.Amount {
		return nil, pkgerrors.NewValidation("INVALID_REFUND_AMOUNT", fmt.Sprintf("refund amount %d exceeds payment amount %d", amount, payment.Amount))
	}

	if payment.GatewayTxnID == nil {
		return nil, pkgerrors.NewInternal("MISSING_GATEWAY_TXN", "captured payment has no gateway transaction id", nil)
	}

	gatewayRefundID, err := s.gateway.Refund(ctx, *payment.GatewayTxnID, amount, payment.Currency)
	if err != nil {
		return nil, pkgerrors.NewInternal("REFUND_FAILED", "gateway refund failed", err)
	}

	newStatus := domain.PaymentPartiallyRefunded
	if amount == payment.Amount {
		newStatus = domain.PaymentRefunded
	}

	refund := &domain.Refund{
		PaymentID:       paymentID,
		OrderID:         payment.OrderID,
		Amount:          amount,
		Currency:        payment.Currency,
		Reason:          reason,
		Status:          domain.RefundProcessed,
		GatewayRefundID: &gatewayRefundID,
	}

	entries := []*domain.LedgerEntry{
		{EntryType: domain.LedgerDebit, Account: "refund", Amount: amount, Currency: payment.Currency, Description: "refund for payment " + paymentID.String()},
		{EntryType: domain.LedgerCredit, Account: "accounts_receivable", Amount: amount, Currency: payment.Currency, Description: "refund for payment " + paymentID.String()},
	}

	s.logger.Info("refunding payment", "payment_id", paymentID, "amount", amount, "new_status", newStatus)

	if err := s.repo.CreateRefund(ctx, refund, payment.UserID, newStatus, entries); err != nil {
		return nil, err
	}

	return refund, nil
}

func (s *paymentService) GetTransaction(ctx context.Context, paymentID uuid.UUID) (*domain.Payment, error) {
	if paymentID == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "payment_id is required")
	}
	return s.repo.GetByID(ctx, paymentID)
}

func (s *paymentService) GetByGatewayTxnID(ctx context.Context, txnID string) (*domain.Payment, error) {
	return s.repo.GetByGatewayTxnID(ctx, txnID)
}

func (s *paymentService) HandleCaptureWebhook(ctx context.Context, paymentID uuid.UUID, gatewayTxnID string) error {
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return err
	}
	if payment.Status != domain.PaymentPending && payment.Status != domain.PaymentAuthorised && payment.Status != domain.PaymentPendingCOD {
		s.logger.Info("ignoring capture webhook for already-resolved payment", "payment_id", paymentID, "status", payment.Status)
		return nil
	}

	entries := []*domain.LedgerEntry{
		{EntryType: domain.LedgerDebit, Account: "accounts_receivable", Amount: payment.Amount, Currency: payment.Currency, Description: "webhook capture for order " + payment.OrderID.String()},
		{EntryType: domain.LedgerCredit, Account: "revenue", Amount: payment.Amount, Currency: payment.Currency, Description: "webhook capture for order " + payment.OrderID.String()},
	}
	return s.repo.MarkCaptured(ctx, paymentID, gatewayTxnID, entries)
}

func (s *paymentService) HandleFailureWebhook(ctx context.Context, paymentID uuid.UUID, reason string) error {
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return err
	}
	if payment.Status != domain.PaymentPending && payment.Status != domain.PaymentAuthorised {
		s.logger.Info("ignoring failure webhook for already-resolved payment", "payment_id", paymentID, "status", payment.Status)
		return nil
	}

	return s.repo.MarkFailed(ctx, paymentID, reason)
}
