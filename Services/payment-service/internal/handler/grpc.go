package handler

import (
	"context"
	"time"

	"github.com/cab-booking/payment-service/internal/domain"
	"github.com/cab-booking/pkg/logger"
	paymentv1 "github.com/cab-booking/proto/gen/payment/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TransactionStore interface {
	Create(ctx context.Context, tx *domain.Transaction) error
	UpdateStatus(ctx context.Context, txID string, status domain.TransactionStatus) error
	GetByTripID(ctx context.Context, tripID string) (*domain.Transaction, error)
}

type StripeAPI interface {
	AuthorizeHold(ctx context.Context, amountCents int64, currency, paymentMethodID string) (string, error)
	ReleaseHold(ctx context.Context, paymentIntentID string) error
	CapturePayment(ctx context.Context, paymentIntentID string, finalAmountCents int64) (string, error)
}

type PaymentHandler struct {
	paymentv1.UnimplementedPaymentServiceServer
	repo         TransactionStore
	stripeClient StripeAPI
}

func NewPaymentHandler(repo TransactionStore, stripeClient StripeAPI) *PaymentHandler {
	return &PaymentHandler{
		repo:         repo,
		stripeClient: stripeClient,
	}
}

func (h *PaymentHandler) AuthorizeHold(ctx context.Context, req *paymentv1.AuthorizeHoldRequest) (*paymentv1.AuthorizeHoldResponse, error) {
	if req.TripId == "" || req.RiderId == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id and rider_id are required")
	}
	if req.AmountCents <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount_cents must be greater than zero")
	}
	if req.Currency == "" {
		req.Currency = "INR"
	}

	// Attempt Stripe authorization
	paymentIntentID, err := h.stripeClient.AuthorizeHold(ctx, req.AmountCents, req.Currency, req.PaymentMethodId)

	now := time.Now()
	tx := &domain.Transaction{
		ID:                    uuid.New().String(),
		TripID:                req.TripId,
		RiderID:               req.RiderId,
		AmountCents:           req.AmountCents,
		Currency:              req.Currency,
		StripePaymentIntentID: paymentIntentID,
		Status:                domain.StatusHoldPending,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if err != nil {
		logger.Error(ctx, "AuthorizeHold failed at Stripe", "error", err)
		tx.Status = domain.StatusHoldFailed
		_ = h.repo.Create(ctx, tx)
		return &paymentv1.AuthorizeHoldResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	tx.Status = domain.StatusHoldSuccess
	if err := h.repo.Create(ctx, tx); err != nil {
		// Log error, but Stripe succeeded. In production, we'd queue a retry to save to DB or release the hold.
		logger.Error(ctx, "Failed to save transaction to DB", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to persist transaction")
	}

	return &paymentv1.AuthorizeHoldResponse{
		Success:       true,
		TransactionId: tx.ID,
	}, nil
}

func (h *PaymentHandler) ReleaseHold(ctx context.Context, req *paymentv1.ReleaseHoldRequest) (*paymentv1.ReleaseHoldResponse, error) {
	// Get transaction from DB
	tx, err := h.repo.GetByTripID(ctx, req.TripId)
	if err != nil {
		logger.Error(ctx, "Failed to find transaction for release", "trip_id", req.TripId)
		return nil, status.Errorf(codes.NotFound, "transaction not found")
	}

	if tx.Status != domain.StatusHoldSuccess {
		return &paymentv1.ReleaseHoldResponse{Success: true}, nil // Nothing to release
	}

	err = h.stripeClient.ReleaseHold(ctx, tx.StripePaymentIntentID)
	if err != nil {
		logger.Error(ctx, "ReleaseHold failed at Stripe", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to release hold")
	}

	err = h.repo.UpdateStatus(ctx, tx.ID, domain.StatusReleased)
	if err != nil {
		logger.Error(ctx, "Failed to update transaction status to RELEASED", "error", err)
	}

	return &paymentv1.ReleaseHoldResponse{Success: true}, nil
}

func (h *PaymentHandler) CapturePayment(ctx context.Context, req *paymentv1.CapturePaymentRequest) (*paymentv1.CapturePaymentResponse, error) {
	if req.TripId == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}
	if req.FinalAmountCents < 0 {
		return nil, status.Error(codes.InvalidArgument, "final_amount_cents cannot be negative")
	}
	tx, err := h.repo.GetByTripID(ctx, req.TripId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "transaction not found")
	}

	if tx.Status != domain.StatusHoldSuccess {
		return nil, status.Errorf(codes.FailedPrecondition, "transaction not in hold status")
	}

	receiptURL, err := h.stripeClient.CapturePayment(ctx, tx.StripePaymentIntentID, req.FinalAmountCents)
	if err != nil {
		logger.Error(ctx, "CapturePayment failed at Stripe", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to capture payment")
	}

	err = h.repo.UpdateStatus(ctx, tx.ID, domain.StatusCaptured)
	if err != nil {
		logger.Error(ctx, "Failed to update transaction status to CAPTURED", "error", err)
	}

	return &paymentv1.CapturePaymentResponse{
		Success:    true,
		ReceiptUrl: receiptURL,
	}, nil
}
