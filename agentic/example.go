package agentic

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"golang.org/x/xerrors"
)

// ExampleServer demonstrates how to use the agentic auth and secrets system.
type ExampleServer struct {
	secretManager *SecretManager
	authProvider  *GitHubAuthProvider
	config        *Config
	registry      *Registry
	scheduler     *Scheduler
}

// NewExampleServer creates a new example server with authentication and secrets.
func NewExampleServer() (*ExampleServer, error) {
	// Initialize configuration
	config := DefaultConfig()

	// Initialize secret manager with multiple stores
	secretManager := NewSecretManager()
	secretManager.SetLogger(log.Printf)

	// Add environment variable store (highest priority)
	envStore := NewEnvSecretStore()
	secretManager.AddStore("env", envStore)

	// Add file-based encrypted store
	secretsPath := os.Getenv("AGENTIC_SECRETS_FILE")
	if secretsPath == "" {
		secretsPath = config.Secrets.File.Path
	}

	secretsPassword := os.Getenv("AGENTIC_SECRETS_PASSWORD")
	if secretsPassword == "" {
		secretsPassword = "default-dev-password" // In production, this should be required
	}

	fileStore := NewFileSecretStore(secretsPath, secretsPassword)
	secretManager.AddStore("file", fileStore)

	// Load configuration from secrets
	if err := config.LoadFromSecrets(secretManager); err != nil {
		return nil, xerrors.Errorf("failed to load config from secrets: %w", err)
	}

	// Initialize GitHub OAuth provider
	authProvider := NewGitHubAuthProvider(config.Auth.GitHub, fileStore)
	authProvider.SetLogger(log.Printf)

	// Initialize agent registry and scheduler
	registry := NewRegistry()
	scheduler := NewScheduler(registry, 100)

	// Register HuggingFace connector if API key is available
	if config.HuggingFace.APIKey != "" {
		hfClient := NewHFClient(config.HuggingFace)
		registry.Register(hfClient)
		log.Printf("Registered HuggingFace connector")
	}

	return &ExampleServer{
		secretManager: secretManager,
		authProvider:  authProvider,
		config:        config,
		registry:      registry,
		scheduler:     scheduler,
	}, nil
}

// SetupRoutes sets up HTTP routes for the example server.
func (es *ExampleServer) SetupRoutes() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Configure appropriately for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Session-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Authentication handlers
	authHandler := NewAuthHandler(es.authProvider)
	authHandler.SetLogger(log.Printf)

	r.Route("/auth", func(r chi.Router) {
		r.Get("/login", authHandler.LoginHandler)
		r.Get("/callback", authHandler.CallbackHandler)
		r.Post("/logout", authHandler.LogoutHandler)
	})

	// Protected routes
	r.Route("/api", func(r chi.Router) {
		r.Use(es.authProvider.AuthMiddleware())

		r.Get("/user", es.GetUserHandler)
		r.Get("/secrets", es.ListSecretsHandler)
		r.Post("/secrets", es.SetSecretHandler)
		r.Delete("/secrets/{key}", es.DeleteSecretHandler)
		r.Post("/tasks", es.CreateTaskHandler)
		r.Get("/tasks/{id}", es.GetTaskHandler)
	})

	// Health check (no auth required)
	r.Get("/health", es.HealthHandler)

	return r
}

// GetUserHandler returns current user information.
func (es *ExampleServer) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]interface{}{
		"user": map[string]interface{}{
			"id":         session.UserID,
			"login":      session.Login,
			"email":      session.Email,
			"name":       session.Name,
			"avatar_url": session.AvatarURL,
		},
	})
}

// ListSecretsHandler returns a list of secret keys (not values).
func (es *ExampleServer) ListSecretsHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Only allow listing secrets for authenticated users
	log.Printf("User %s requested secrets list", session.Login)

	// Get keys from file store (don't expose env vars)
	if fileStore, exists := es.secretManager.stores["file"]; exists {
		keys, err := fileStore.List()
		if err != nil {
			http.Error(w, "Failed to list secrets", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, map[string]interface{}{
			"keys": keys,
		})
		return
	}

	http.Error(w, "Secret store not available", http.StatusInternalServerError)
}

// SetSecretHandler stores a secret.
func (es *ExampleServer) SetSecretHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	if err := parseJSON(r, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Key == "" || req.Value == "" {
		http.Error(w, "Key and value are required", http.StatusBadRequest)
		return
	}

	// Store in file store by default
	if err := es.secretManager.Set("file", req.Key, req.Value); err != nil {
		log.Printf("Failed to store secret for user %s: %v", session.Login, err)
		http.Error(w, "Failed to store secret", http.StatusInternalServerError)
		return
	}

	log.Printf("User %s stored secret key: %s", session.Login, req.Key)

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Secret stored successfully",
	})
}

// DeleteSecretHandler deletes a secret.
func (es *ExampleServer) DeleteSecretHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	key := chi.URLParam(r, "key")
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	if err := es.secretManager.Delete("file", key); err != nil {
		log.Printf("Failed to delete secret for user %s: %v", session.Login, err)
		http.Error(w, "Failed to delete secret", http.StatusInternalServerError)
		return
	}

	log.Printf("User %s deleted secret key: %s", session.Login, key)

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Secret deleted successfully",
	})
}

// CreateTaskHandler creates a new agentic task.
func (es *ExampleServer) CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Type    string                 `json:"type"`
		Payload map[string]interface{} `json:"payload"`
	}

	if err := parseJSON(r, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	task := &Task{
		Type:    req.Type,
		Payload: req.Payload,
		Status:  "queued",
	}

	es.scheduler.Schedule(task)

	log.Printf("User %s created task: %s", session.Login, req.Type)

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]interface{}{
		"success": true,
		"task":    task,
	})
}

// GetTaskHandler returns task status (simplified implementation).
func (es *ExampleServer) GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	taskID := chi.URLParam(r, "id")

	log.Printf("User %s requested task: %s", session.Login, taskID)

	// This is a simplified implementation
	// In a real system, you'd store tasks in a database
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]interface{}{
		"id":     taskID,
		"status": "completed",
		"result": "Task completed successfully",
	})
}

// HealthHandler returns server health status.
func (es *ExampleServer) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]interface{}{
		"status": "healthy",
		"auth":   "enabled",
		"secrets": map[string]interface{}{
			"env":  true,
			"file": true,
		},
	})
}

// Start starts the example server.
func (es *ExampleServer) Start(ctx context.Context, addr string) error {
	// Start the scheduler
	es.scheduler.Run(3) // 3 worker goroutines

	// Setup HTTP server
	handler := es.SetupRoutes()
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	log.Printf("Starting agentic server on %s", addr)
	log.Printf("GitHub OAuth configured: %t", es.config.Auth.GitHub.ClientID != "")

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	log.Printf("Shutting down server...")
	es.scheduler.Stop()
	return server.Shutdown(context.Background())
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON: %v", err)
	}
}

func parseJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

