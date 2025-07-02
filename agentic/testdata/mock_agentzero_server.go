package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type AgentZeroStatus struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type AgentZeroTask struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func mockAgentZeroHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/v1/status":
		status := AgentZeroStatus{Status: "ready", Version: "1.2.3"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	case "/api/v1/tasks":
		tasks := []AgentZeroTask{
			{ID: "a1", Name: "Sync", Status: "completed"},
			{ID: "a2", Name: "Provision", Status: "pending"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)
	default:
		http.NotFound(w, r)
	}
}

// StartMockAgentZeroServer starts the mock Agent-Zero API server on the given address.
func StartMockAgentZeroServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/status", mockAgentZeroHandler)
	mux.HandleFunc("/api/v1/tasks", mockAgentZeroHandler)

	server := &http.Server{Addr: addr, Handler: mux}
	go func() {
		log.Printf("Mock Agent-Zero server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Mock Agent-Zero server error: %v", err)
		}
	}()
	return server
}

func main() {
	addr := os.Getenv("AGENTZERO_MOCK_ADDR")
	if addr == "" {
		addr = ":8081"
	}
	server := StartMockAgentZeroServer(addr)

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("Shutting down Mock Agent-Zero server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Mock Agent-Zero server Shutdown: %v", err)
	}
	log.Println("Mock Agent-Zero server stopped gracefully")
}
