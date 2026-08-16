package main

import (
	"context"
	"fmt"

	"github.com/cab-booking/pkg/logger"
)

func main() {
	ctx := context.Background()
	logger.Info(ctx, "Starting Trip Service (Saga Orchestrator & State Machine)...")
	fmt.Println("Trip Service listening on :50051")
}
