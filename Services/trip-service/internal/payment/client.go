package payment

import (
	"context"
	"fmt"
	"time"

	paymentv1 "github.com/cab-booking/proto/gen/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn *grpc.ClientConn
	svc  paymentv1.PaymentServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to payment service: %w", err)
	}

	return &Client{
		conn: conn,
		svc:  paymentv1.NewPaymentServiceClient(conn),
	}, nil
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) AuthorizeHold(ctx context.Context, tripID, riderID string, amountCents int64, currency, paymentMethodID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req := &paymentv1.AuthorizeHoldRequest{
		TripId:          tripID,
		RiderId:         riderID,
		AmountCents:     amountCents,
		Currency:        currency,
		PaymentMethodId: paymentMethodID,
	}

	resp, err := c.svc.AuthorizeHold(ctx, req)
	if err != nil {
		return "", fmt.Errorf("payment hold rpc failed: %w", err)
	}
	if !resp.Success {
		return "", fmt.Errorf("payment hold declined: %s", resp.ErrorMessage)
	}

	return resp.TransactionId, nil
}

func (c *Client) ReleaseHold(ctx context.Context, transactionID, tripID, reason string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req := &paymentv1.ReleaseHoldRequest{
		TransactionId: transactionID,
		TripId:        tripID,
		Reason:        reason,
	}

	resp, err := c.svc.ReleaseHold(ctx, req)
	if err != nil {
		return fmt.Errorf("payment release rpc failed: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("payment release failed")
	}

	return nil
}

func (c *Client) CapturePayment(ctx context.Context, transactionID, tripID string, finalAmountCents int64) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req := &paymentv1.CapturePaymentRequest{
		TransactionId:    transactionID,
		TripId:           tripID,
		FinalAmountCents: finalAmountCents,
	}

	resp, err := c.svc.CapturePayment(ctx, req)
	if err != nil {
		return "", fmt.Errorf("payment capture rpc failed: %w", err)
	}
	if !resp.Success {
		return "", fmt.Errorf("payment capture failed")
	}

	return resp.ReceiptUrl, nil
}
