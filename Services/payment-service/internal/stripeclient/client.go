package stripeclient

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/cab-booking/pkg/logger"
)

// Client is a mock implementation of the Stripe API.
type Client struct {
	secretKey string
}

func NewClient(secretKey string) *Client {
	return &Client{
		secretKey: secretKey,
	}
}

// AuthorizeHold simulates placing a hold on a customer's payment method.
func (c *Client) AuthorizeHold(ctx context.Context, amountCents int64, currency, paymentMethodID string) (string, error) {
	logger.Info(ctx, "MOCK STRIPE: AuthorizeHold", "amount", amountCents, "currency", currency, "payment_method", paymentMethodID)
	
	// Simulate network latency
	time.Sleep(500 * time.Millisecond)

	// Simulate failures for specific mock payment methods if needed, or 5% random failure rate
	if paymentMethodID == "pm_fail" || rand.Float32() < 0.05 {
		return "", fmt.Errorf("mock stripe error: card declined")
	}

	mockPaymentIntentID := fmt.Sprintf("pi_mock_%d", time.Now().UnixNano())
	return mockPaymentIntentID, nil
}

// ReleaseHold simulates releasing an uncaptured hold.
func (c *Client) ReleaseHold(ctx context.Context, paymentIntentID string) error {
	logger.Info(ctx, "MOCK STRIPE: ReleaseHold", "payment_intent", paymentIntentID)
	
	time.Sleep(300 * time.Millisecond)

	if paymentIntentID == "" {
		return fmt.Errorf("mock stripe error: invalid payment intent ID")
	}

	return nil
}

// CapturePayment simulates capturing a previously held authorization.
func (c *Client) CapturePayment(ctx context.Context, paymentIntentID string, finalAmountCents int64) (string, error) {
	logger.Info(ctx, "MOCK STRIPE: CapturePayment", "payment_intent", paymentIntentID, "final_amount", finalAmountCents)
	
	time.Sleep(500 * time.Millisecond)

	if paymentIntentID == "" {
		return "", fmt.Errorf("mock stripe error: invalid payment intent ID")
	}

	receiptURL := fmt.Sprintf("https://mock-stripe.com/receipts/%s", paymentIntentID)
	return receiptURL, nil
}
