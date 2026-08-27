package stripeclient

import (
	"context"
	"testing"
)

func TestAuthorizeHoldDeterministicDecline(t *testing.T) {
	c := NewClient("sk_test")
	if _, err := c.AuthorizeHold(context.Background(), 1000, "INR", "pm_fail"); err == nil {
		t.Fatal("pm_fail must decline")
	}
	id, err := c.AuthorizeHold(context.Background(), 1000, "INR", "pm_ok")
	if err != nil || id == "" {
		t.Fatalf("pm_ok should succeed: %v %s", err, id)
	}
}

func TestReleaseAndCaptureRequireIntent(t *testing.T) {
	c := NewClient("sk_test")
	if err := c.ReleaseHold(context.Background(), ""); err == nil {
		t.Fatal("empty intent must fail")
	}
	if _, err := c.CapturePayment(context.Background(), "", 100); err == nil {
		t.Fatal("empty intent capture must fail")
	}
	url, err := c.CapturePayment(context.Background(), "pi_mock_1", 100)
	if err != nil || url == "" {
		t.Fatalf("capture: %v %s", err, url)
	}
}
