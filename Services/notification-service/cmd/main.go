package main

import (
	"context"
	"fmt"

	"github.com/cab-booking/pkg/logger"
)

func main() {
	ctx := context.Background()
	logger.Info(ctx, "Starting Notification Service (Centrifugo WS Gateway)...")
	fmt.Println("Notification Service listening on :50055")
}
