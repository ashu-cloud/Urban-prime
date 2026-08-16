package main

import (
	"context"
	"fmt"

	"github.com/cab-booking/pkg/logger"
)

func main() {
	ctx := context.Background()
	logger.Info(ctx, "Starting Driver Service (Matchmaking Engine)...")
	fmt.Println("Driver Service listening on :50052")
}
