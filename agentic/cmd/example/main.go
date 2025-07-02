package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/coder/coder/v2/agentic"
)

func main() {
	addr := os.Getenv("AGENTIC_EXAMPLE_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	server, err := agentic.NewExampleServer()
	if err != nil {
		log.Fatalf("Failed to create example server: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		log.Println("Shutting down agentic example server...")
		cancel()
	}()

	if err := server.Start(ctx, addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server exited with error: %v", err)
	}
	log.Println("Agentic example server stopped gracefully")
}
