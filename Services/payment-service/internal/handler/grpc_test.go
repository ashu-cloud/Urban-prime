package handler

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/cab-booking/payment-service/internal/domain"
	"github.com/cab-booking/payment-service/internal/stripeclient"
	paymentv1 "github.com/cab-booking/proto/gen/payment/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type memTxRepo struct {
	mu  sync.Mutex
	txs map[string]*domain.Transaction
}

func newMemTxRepo() *memTxRepo {
	return &memTxRepo{txs: make(map[string]*domain.Transaction)}
}

func (m *memTxRepo) Create(ctx context.Context, tx *domain.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *tx
	m.txs[tx.TripID] = &cp
	return nil
}

func (m *memTxRepo) UpdateStatus(ctx context.Context, txID string, status domain.TransactionStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, tx := range m.txs {
		if tx.ID == txID {
			tx.Status = status
			return nil
		}
	}
	return fmt.Errorf("not found")
}

func (m *memTxRepo) GetByTripID(ctx context.Context, tripID string) (*domain.Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tx, ok := m.txs[tripID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *tx
	return &cp, nil
}

func TestAuthorizeHoldSuccessAndCapture(t *testing.T) {
	h := NewPaymentHandler(newMemTxRepo(), stripeclient.NewClient("sk_test"))
	hold, err := h.AuthorizeHold(context.Background(), &paymentv1.AuthorizeHoldRequest{
		TripId:          "trip-1",
		RiderId:         "rider-1",
		AmountCents:     19000,
		Currency:        "INR",
		PaymentMethodId: "pm_ok",
	})
	if err != nil || !hold.Success {
		t.Fatalf("hold err=%v resp=%+v", err, hold)
	}

	cap, err := h.CapturePayment(context.Background(), &paymentv1.CapturePaymentRequest{
		TripId:           "trip-1",
		FinalAmountCents: 19000,
	})
	if err != nil || !cap.Success || cap.ReceiptUrl == "" {
		t.Fatalf("capture err=%v resp=%+v", err, cap)
	}

	_, err = h.CapturePayment(context.Background(), &paymentv1.CapturePaymentRequest{
		TripId:           "trip-1",
		FinalAmountCents: 19000,
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("double capture code %v", status.Code(err))
	}
}

func TestAuthorizeHoldValidation(t *testing.T) {
	h := NewPaymentHandler(newMemTxRepo(), stripeclient.NewClient("sk_test"))
	_, err := h.AuthorizeHold(context.Background(), &paymentv1.AuthorizeHoldRequest{
		AmountCents: 100,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("missing ids code %v", status.Code(err))
	}
	_, err = h.AuthorizeHold(context.Background(), &paymentv1.AuthorizeHoldRequest{
		TripId: "t", RiderId: "r", AmountCents: -5,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("negative amount code %v", status.Code(err))
	}
}

func TestAuthorizeHoldDeclinedCard(t *testing.T) {
	h := NewPaymentHandler(newMemTxRepo(), stripeclient.NewClient("sk_test"))
	resp, err := h.AuthorizeHold(context.Background(), &paymentv1.AuthorizeHoldRequest{
		TripId: "trip-fail", RiderId: "r", AmountCents: 1000, PaymentMethodId: "pm_fail",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Success {
		t.Fatal("declined card should not succeed")
	}
}

func TestReleaseHold(t *testing.T) {
	h := NewPaymentHandler(newMemTxRepo(), stripeclient.NewClient("sk_test"))
	_, err := h.AuthorizeHold(context.Background(), &paymentv1.AuthorizeHoldRequest{
		TripId: "trip-rel", RiderId: "r", AmountCents: 5000, PaymentMethodId: "pm_ok",
	})
	if err != nil {
		t.Fatal(err)
	}
	rel, err := h.ReleaseHold(context.Background(), &paymentv1.ReleaseHoldRequest{TripId: "trip-rel"})
	if err != nil || !rel.Success {
		t.Fatalf("release err=%v", err)
	}
}

func TestConcurrentAuthorizeHolds(t *testing.T) {
	h := NewPaymentHandler(newMemTxRepo(), stripeclient.NewClient("sk_test"))
	const n = 30
	var wg sync.WaitGroup
	wg.Add(n)
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			resp, err := h.AuthorizeHold(context.Background(), &paymentv1.AuthorizeHoldRequest{
				TripId:          fmt.Sprintf("trip-%d", i),
				RiderId:         "rider",
				AmountCents:     2500,
				PaymentMethodId: "pm_ok",
			})
			if err != nil {
				errCh <- err
				return
			}
			if !resp.Success {
				errCh <- fmt.Errorf("hold failed for trip-%d: %s", i, resp.ErrorMessage)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
}
