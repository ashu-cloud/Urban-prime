package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
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

	// Start the embedded API Gateway
	go startGateway()

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

	var cmd *exec.Cmd
	// Check if a precompiled binary 'main' exists in the service directory
	binPath := filepath.Join(svc.Dir, "main")
	if _, err := os.Stat(binPath); err == nil {
		cmd = exec.CommandContext(ctx, "./main")
	} else {
		cmd = exec.CommandContext(ctx, "go", "run", "cmd/main.go")
	}
	// Strip PORT from child processes to prevent them from binding to the gateway's port
	cmd.Env = filterEnv(os.Environ(), "PORT")
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

func filterEnv(env []string, keyToStrip string) []string {
	var filtered []string
	prefix := keyToStrip + "="
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func startGateway() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9080"
	}

	targets := map[string]string{
		"/auth":              "http://localhost:8080",
		"/api/v1/auth":       "http://localhost:8080",
		"/trip":              "http://localhost:8051",
		"/api/v1/trip":       "http://localhost:8051",
		"/driver":            "http://localhost:8052",
		"/api/v1/driver":     "http://localhost:8052",
		"/api/v1/drivers":    "http://localhost:8052",
		"/location":          "http://localhost:8053",
		"/api/v1/location":   "http://localhost:8053",
	}

	mux := http.NewServeMux()

	for prefix, target := range targets {
		prefixCopy := prefix
		targetURL, _ := url.Parse(target)
		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			if strings.HasPrefix(req.URL.Path, "/api/v1") {
				// Strip /api/v1/drivers -> /drivers, etc.
				req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api/v1")
			}
		}

		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(fmt.Sprintf("Gateway Error: %v", err)))
		}

		mux.Handle(prefixCopy+"/", proxy)
		
		// Exact match fallback for missing trailing slash on exact endpoints
		if !strings.HasSuffix(prefixCopy, "/") {
			mux.Handle(prefixCopy, proxy)
		}
	}

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Global CORS Middleware
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, Origin")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	fmt.Printf("%s%s🌐 Built-in API Gateway & CORS Proxy listening on port %s%s\n", Bold, Magenta, port, Reset)
	if err := http.ListenAndServe(":"+port, corsMiddleware(mux)); err != nil {
		fmt.Printf("Gateway error: %v\n", err)
	}
}
