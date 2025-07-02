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

type OpenCodeUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func mockOpenCodeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/v1/users":
		users := []OpenCodeUser{
			{ID: "1", Name: "Alice Example", Email: "alice@opencode.local"},
			{ID: "2", Name: "Bob Example", Email: "bob@opencode.local"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	case "/api/v1/ping":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	default:
		http.NotFound(w, r)
	}
}

// StartMockOpenCodeServer starts the mock OpenCode API server on the given address.
func StartMockOpenCodeServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", mockOpenCodeHandler)
	mux.HandleFunc("/api/v1/ping", mockOpenCodeHandler)

	server := &http.Server{Addr: addr, Handler: mux}
	go func() {
		log.Printf("Mock OpenCode server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Mock OpenCode server error: %v", err)
		}
	}()
	return server
}

func main() {
	addr := os.Getenv("OPENCODE_MOCK_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	server := StartMockOpenCodeServer(addr)

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("Shutting down Mock OpenCode server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Mock OpenCode server Shutdown: %v", err)
	}
	log.Println("Mock OpenCode server stopped gracefully")
}
