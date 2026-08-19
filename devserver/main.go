package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
)

type Service struct {
	Name  string
	Dir   string
	Color string
}

const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Bold    = "\033[1m"
)

func main() {
	fmt.Printf("%s%s======================================================%s\n", Bold, Cyan, Reset)
	fmt.Printf("%s%s  🚀 Starting Urban Prime Microservices (Go Ecosystem) %s\n", Bold, Cyan, Reset)
	fmt.Printf("%s%s======================================================%s\n\n", Bold, Cyan, Reset)

	services := []Service{
		{Name: "AUTH    ", Dir: filepath.Join("Services", "auth-service"), Color: Cyan},
		{Name: "TRIP    ", Dir: filepath.Join("Services", "trip-service"), Color: Green},
		{Name: "DRIVER  ", Dir: filepath.Join("Services", "driver-service"), Color: Yellow},
		{Name: "LOCATION", Dir: filepath.Join("Services", "location-service"), Color: Blue},
		{Name: "PAYMENT ", Dir: filepath.Join("Services", "payment-service"), Color: Magenta},
		{Name: "NOTIF   ", Dir: filepath.Join("Services", "notification-service"), Color: Red},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	for _, svc := range services {
		wg.Add(1)
		go runService(ctx, &wg, svc)
	}

	<-sigChan
	fmt.Printf("\n%s%s🛑 Received Shutdown Signal. Gracefully stopping all services...%s\n", Bold, Red, Reset)
	cancel()
	wg.Wait()
	fmt.Printf("%s%s✨ All services stopped cleanly.%s\n", Bold, Green, Reset)
}

func runService(ctx context.Context, wg *sync.WaitGroup, svc Service) {
	defer wg.Done()

	cmd := exec.CommandContext(ctx, "go", "run", "cmd/main.go")
	cmd.Dir = svc.Dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("%s[%s]%s Failed to capture stdout: %v\n", svc.Color, svc.Name, Reset, err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("%s[%s]%s Failed to capture stderr: %v\n", svc.Color, svc.Name, Reset, err)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("%s[%s]%s Failed to start service: %v\n", svc.Color, svc.Name, Reset, err)
		return
	}

	go pipeOutput(stdout, svc.Color, svc.Name)
	go pipeOutput(stderr, svc.Color, svc.Name)

	_ = cmd.Wait()
}

func pipeOutput(r io.Reader, color, name string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Printf("%s%s[%s]%s %s\n", Bold, color, name, Reset, scanner.Text())
	}
}
