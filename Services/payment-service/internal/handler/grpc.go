package handler

import (
	"context"

	"github.com/cab-booking/payment-service/internal/domain"
	"github.com/cab-booking/payment-service/internal/repository"
	"github.com/cab-booking/payment-service/internal/stripeclient"
	"github.com/cab-booking/pkg/logger"
	paymentv1 "github.com/cab-booking/proto/gen/payment/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PaymentHandler struct {
	paymentv1.UnimplementedPaymentServiceServer
	repo         *repository.TransactionRepository
	stripeClient *stripeclient.Client
}

func NewPaymentHandler(repo *repository.TransactionRepository, stripeClient *stripeclient.Client) *PaymentHandler {
	return &PaymentHandler{
		repo:         repo,
		stripeClient: stripeClient,
	}
}

func (h *PaymentHandler) AuthorizeHold(ctx context.Context, req *paymentv1.AuthorizeHoldRequest) (*paymentv1.AuthorizeHoldResponse, error) {
	// Attempt Stripe authorization
	paymentIntentID, err := h.stripeClient.AuthorizeHold(ctx, req.AmountCents, req.Currency, req.PaymentMethodId)
	
	tx := &domain.Transaction{
		ID:                    uuid.New().String(),
		TripID:                req.TripId,
		RiderID:               req.RiderId,
		AmountCents:           req.AmountCents,
		Currency:              req.Currency,
		StripePaymentIntentID: paymentIntentID,
		Status:                domain.StatusHoldPending,
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
