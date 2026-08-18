package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/cab-booking/payment-service/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepository struct {
	db *pgxpool.Pool
}

func NewTransactionRepository(db *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, tx *domain.Transaction) error {
	query := `
		INSERT INTO transactions (
			id, trip_id, rider_id, amount_cents, currency, stripe_payment_intent_id, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`
	_, err := r.db.Exec(ctx, query,
		tx.ID,
		tx.TripID,
		tx.RiderID,
		tx.AmountCents,
		tx.Currency,
		tx.StripePaymentIntentID,
		tx.Status,
		tx.CreatedAt,
		tx.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}
	return nil
}

func (r *TransactionRepository) UpdateStatus(ctx context.Context, txID string, status domain.TransactionStatus) error {
	query := `
		UPDATE transactions
		SET status = $1, updated_at = $2
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, status, time.Now(), txID)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}
	return nil
}

func (r *TransactionRepository) GetByTripID(ctx context.Context, tripID string) (*domain.Transaction, error) {
	query := `
		SELECT id, trip_id, rider_id, amount_cents, currency, stripe_payment_intent_id, status, created_at, updated_at
		FROM transactions
		WHERE trip_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	tx := &domain.Transaction{}
	err := r.db.QueryRow(ctx, query, tripID).Scan(
		&tx.ID,
		&tx.TripID,
		&tx.RiderID,
		&tx.AmountCents,
		&tx.Currency,
		&tx.StripePaymentIntentID,
		&tx.Status,
		&tx.CreatedAt,
		&tx.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction by trip id: %w", err)
	}
	return tx, nil
}
