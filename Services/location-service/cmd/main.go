package main

import (
	"context"
	"fmt"

	"github.com/cab-booking/pkg/logger"
)

func main() {
	ctx := context.Background()
	logger.Info(ctx, "Starting Location Service (GPS Ingestion)...")
	fmt.Println("Location Service listening on :50053")
}
