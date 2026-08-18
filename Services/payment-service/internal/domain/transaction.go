package domain

import (
	"time"
)

type TransactionStatus string

const (
	StatusHoldPending TransactionStatus = "HOLD_PENDING"
	StatusHoldSuccess TransactionStatus = "HOLD_SUCCESS"
	StatusHoldFailed  TransactionStatus = "HOLD_FAILED"
	StatusCaptured    TransactionStatus = "CAPTURED"
	StatusReleased    TransactionStatus = "RELEASED"
)

type Transaction struct {
	ID                     string
	TripID                 string
	RiderID                string
	AmountCents            int64
	Currency               string
	StripePaymentIntentID  string
	Status                 TransactionStatus
	CreatedAt              time.Time
	UpdatedAt              time.Time
}
