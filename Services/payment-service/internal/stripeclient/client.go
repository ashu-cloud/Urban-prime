package stripeclient

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cab-booking/pkg/logger"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
)

// Client is a real Stripe API client with automatic test token fallback.
type Client struct {
	secretKey string
}

func NewClient(secretKey string) *Client {
	if secretKey != "" {
		stripe.Key = secretKey
	}
	return &Client{
		secretKey: secretKey,
	}
}

func isLiveOrTestKey(key string) bool {
	return (strings.HasPrefix(key, "sk_test_") || strings.HasPrefix(key, "sk_live_")) && len(key) > 20
}

// AuthorizeHold places a hold (manual capture) on customer's payment method using Stripe SDK.
func (c *Client) AuthorizeHold(ctx context.Context, amountCents int64, currency, paymentMethodID string) (string, error) {
	logger.Info(ctx, "Stripe AuthorizeHold", "amount", amountCents, "currency", currency, "payment_method", paymentMethodID)

	if !isLiveOrTestKey(c.secretKey) {
		if paymentMethodID == "pm_fail" {
			return "", fmt.Errorf("mock stripe error: card declined")
		}
		mockPaymentIntentID := fmt.Sprintf("pi_mock_%d", time.Now().UnixNano())
		return mockPaymentIntentID, nil
	}

	curr := strings.ToLower(currency)
	if curr == "" {
		curr = "usd"
	}

	pm := paymentMethodID
	if pm == "" {
		pm = "pm_card_visa" // Default test payment method token
	}

	params := &stripe.PaymentIntentParams{
		Amount:        stripe.Int64(amountCents),
		Currency:      stripe.String(curr),
		PaymentMethod: stripe.String(pm),
		Confirm:       stripe.Bool(true),
		CaptureMethod: stripe.String(string(stripe.PaymentIntentCaptureMethodManual)),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled:        stripe.Bool(true),
			AllowRedirects: stripe.String("never"),
		},
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		logger.Error(ctx, "Stripe PaymentIntent authorization failed", "error", err)
		return "", fmt.Errorf("stripe authorization error: %w", err)
	}

	return pi.ID, nil
}

// ReleaseHold cancels an uncaptured PaymentIntent authorization.
func (c *Client) ReleaseHold(ctx context.Context, paymentIntentID string) error {
	logger.Info(ctx, "Stripe ReleaseHold", "payment_intent", paymentIntentID)

	if paymentIntentID == "" {
		return fmt.Errorf("invalid payment intent ID")
	}

	if !isLiveOrTestKey(c.secretKey) || strings.HasPrefix(paymentIntentID, "pi_mock_") {
		return nil
	}

	params := &stripe.PaymentIntentCancelParams{}
	_, err := paymentintent.Cancel(paymentIntentID, params)
	if err != nil {
		logger.Error(ctx, "Stripe PaymentIntent cancel failed", "payment_intent", paymentIntentID, "error", err)
		return fmt.Errorf("stripe release failed: %w", err)
	}

	return nil
}

// CapturePayment captures a previously held authorization.
func (c *Client) CapturePayment(ctx context.Context, paymentIntentID string, finalAmountCents int64) (string, error) {
	logger.Info(ctx, "Stripe CapturePayment", "payment_intent", paymentIntentID, "final_amount", finalAmountCents)

	if paymentIntentID == "" {
		return "", fmt.Errorf("invalid payment intent ID")
	}

	if !isLiveOrTestKey(c.secretKey) || strings.HasPrefix(paymentIntentID, "pi_mock_") {
		receiptURL := fmt.Sprintf("https://mock-stripe.com/receipts/%s", paymentIntentID)
		return receiptURL, nil
	}

	params := &stripe.PaymentIntentCaptureParams{
		AmountToCapture: stripe.Int64(finalAmountCents),
	}
	pi, err := paymentintent.Capture(paymentIntentID, params)
	if err != nil {
		logger.Error(ctx, "Stripe PaymentIntent capture failed", "payment_intent", paymentIntentID, "error", err)
		return "", fmt.Errorf("stripe capture failed: %w", err)
	}

	receiptURL := fmt.Sprintf("https://dashboard.stripe.com/test/payments/%s", pi.ID)
	return receiptURL, nil
}
