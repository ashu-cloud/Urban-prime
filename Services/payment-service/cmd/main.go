package main

import (
	"context"
	"fmt"

	"github.com/cab-booking/pkg/logger"
)

func main() {
	ctx := context.Background()
	logger.Info(ctx, "Starting Payment Service (Stripe Integration & Saga Handler)...")
	fmt.Println("Payment Service listening on :50054")
}
