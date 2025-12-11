// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	version = "0.1.0"
)

// AINode is the main AI node server
type AINode struct {
	config  Config
	mu      sync.RWMutex
	miners  map[string]*MinerInfo
	tasks   map[string]*Task
	models  map[string]*ModelInfo
	server  *http.Server
	running bool
}

// Config holds node configuration
type Config struct {
	Port           int      `json:"port"`
	DataDir        string   `json:"data_dir"`
	NodeURL        string   `json:"node_url"`
	EnableCORS     bool     `json:"enable_cors"`
	AllowedOrigins []string `json:"allowed_origins"`
}

// MinerInfo tracks connected miners
type MinerInfo struct {
	ID           string    `json:"id"`
	WalletAddr   string    `json:"wallet_address"`
	Endpoint     string    `json:"endpoint"`
	GPUEnabled   bool      `json:"gpu_enabled"`
	LastSeen     time.Time `json:"last_seen"`
	TasksHandled uint64    `json:"tasks_handled"`
}

// Task represents an AI task
type Task struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Model      string          `json:"model"`
	Input      json.RawMessage `json:"input"`
	Output     json.RawMessage `json:"output,omitempty"`
	Status     string          `json:"status"`
	AssignedTo string          `json:"assigned_to,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// ModelInfo describes available models
type ModelInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Capabilities []string `json:"capabilities"`
	ContextSize  int      `json:"context_size"`
}

// ChatRequest represents a chat API request
type ChatRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	Stream      bool    `json:"stream,omitempty"`
}

// ChatResponse represents a chat API response
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func main() {
	var (
		port        = flag.Int("port", 9090, "API port")
		dataDir     = flag.String("data", "./data", "Data directory")
		nodeURL     = flag.String("node", "http://localhost:9650", "Lux node URL")
		enableCORS  = flag.Bool("cors", true, "Enable CORS")
		showVersion = flag.Bool("version", false, "Show version")
	)

	flag.Parse()

	if *showVersion {
		fmt.Printf("lux-ai %s\n", version)
		os.Exit(0)
	}

	config := Config{
		Port:           *port,
		DataDir:        *dataDir,
		NodeURL:        *nodeURL,
		EnableCORS:     *enableCORS,
		AllowedOrigins: []string{"*"},
	}

	node := NewAINode(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
		_ = node.Stop()
	}()

	fmt.Printf("Starting Lux AI Node %s\n", version)
	fmt.Printf("Port: %d\n", *port)
	fmt.Printf("Data Dir: %s\n", *dataDir)
	fmt.Printf("Node URL: %s\n", *nodeURL)

	if err := node.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting node: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("AI Node started. Press Ctrl+C to stop.")

	<-ctx.Done()
	fmt.Println("AI Node stopped.")
}

// NewAINode creates a new AI node
func NewAINode(config Config) *AINode {
	return &AINode{
		config: config,
		miners: make(map[string]*MinerInfo),
		tasks:  make(map[string]*Task),
		models: defaultModels(),
	}
}

// defaultModels returns the default available models
func defaultModels() map[string]*ModelInfo {
	return map[string]*ModelInfo{
		"zen-coder-1.5b": {
			ID:           "zen-coder-1.5b",
			Name:         "Zen Coder 1.5B",
			Type:         "chat",
			Capabilities: []string{"code", "chat", "completion"},
			ContextSize:  32768,
		},
		"zen-mini-0.5b": {
			ID:           "zen-mini-0.5b",
			Name:         "Zen Mini 0.5B",
			Type:         "chat",
			Capabilities: []string{"chat", "completion"},
			ContextSize:  8192,
		},
		"qwen3-8b": {
			ID:           "qwen3-8b",
			Name:         "Qwen3 8B",
			Type:         "chat",
			Capabilities: []string{"chat", "code", "reasoning"},
			ContextSize:  131072,
		},
	}
}

// Start begins the AI node server
func (n *AINode) Start(ctx context.Context) error {
	n.mu.Lock()
	if n.running {
		n.mu.Unlock()
		return fmt.Errorf("already running")
	}
	n.running = true
	n.mu.Unlock()

	// Create data directory
	if err := os.MkdirAll(n.config.DataDir, 0755); err != nil {
		return err
	}

	mux := http.NewServeMux()

	// OpenAI-compatible API
	mux.HandleFunc("/v1/chat/completions", n.corsMiddleware(n.handleChatCompletions))
	mux.HandleFunc("/v1/models", n.corsMiddleware(n.handleModels))
	mux.HandleFunc("/v1/embeddings", n.corsMiddleware(n.handleEmbeddings))

	// Lux AI API
	mux.HandleFunc("/api/miners", n.corsMiddleware(n.handleMiners))
	mux.HandleFunc("/api/miners/register", n.corsMiddleware(n.handleMinerRegister))
	mux.HandleFunc("/api/tasks", n.corsMiddleware(n.handleTasks))
	mux.HandleFunc("/api/tasks/pending", n.corsMiddleware(n.handlePendingTasks))
	mux.HandleFunc("/api/tasks/submit", n.corsMiddleware(n.handleSubmitResult))
	mux.HandleFunc("/api/stats", n.corsMiddleware(n.handleStats))

	// Health check
	mux.HandleFunc("/health", n.handleHealth)

	n.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", n.config.Port),
		Handler: mux,
	}

	go n.server.ListenAndServe()

	return nil
}

// Stop halts the AI node server
func (n *AINode) Stop() error {
	n.mu.Lock()
	if !n.running {
		n.mu.Unlock()
		return nil
	}
	n.running = false
	n.mu.Unlock()

	if n.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return n.server.Shutdown(ctx)
	}
	return nil
}

// corsMiddleware adds CORS headers
func (n *AINode) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if n.config.EnableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		next(w, r)
	}
}

// handleChatCompletions handles OpenAI-compatible chat API
func (n *AINode) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if model exists
	n.mu.RLock()
	model, ok := n.models[req.Model]
	n.mu.RUnlock()

	if !ok {
		// Use default model
		req.Model = "zen-mini-0.5b"
		model = n.models[req.Model]
	}

	// Generate response (placeholder - would route to miner)
	response := ChatResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
	}
	response.Choices = append(response.Choices, struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	}{
		Index: 0,
		Message: struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			Role:    "assistant",
			Content: fmt.Sprintf("Hello! I'm %s running on the Lux AI network. How can I help you today?", model.Name),
		},
		FinishReason: "stop",
	})
	response.Usage.PromptTokens = 10
	response.Usage.CompletionTokens = 20
	response.Usage.TotalTokens = 30

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleModels returns available models
func (n *AINode) handleModels(w http.ResponseWriter, r *http.Request) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	models := make([]map[string]interface{}, 0, len(n.models))
	for _, m := range n.models {
		models = append(models, map[string]interface{}{
			"id":       m.ID,
			"object":   "model",
			"created":  time.Now().Unix(),
			"owned_by": "lux-ai",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data":   models,
	})
}

// handleEmbeddings handles embedding requests
func (n *AINode) handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Input string `json:"input"`
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Placeholder embedding
	embedding := make([]float64, 1536)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{
				"object":    "embedding",
				"embedding": embedding,
				"index":     0,
			},
		},
		"model": req.Model,
		"usage": map[string]int{
			"prompt_tokens": 8,
			"total_tokens":  8,
		},
	})
}

// handleMiners returns connected miners
func (n *AINode) handleMiners(w http.ResponseWriter, r *http.Request) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	miners := make([]*MinerInfo, 0, len(n.miners))
	for _, m := range n.miners {
		miners = append(miners, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(miners)
}

// handleMinerRegister registers a new miner
func (n *AINode) handleMinerRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var miner MinerInfo
	if err := json.NewDecoder(r.Body).Decode(&miner); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	miner.LastSeen = time.Now()

	n.mu.Lock()
	n.miners[miner.ID] = &miner
	n.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "registered",
		"id":     miner.ID,
	})
}

// handleTasks returns all tasks
func (n *AINode) handleTasks(w http.ResponseWriter, r *http.Request) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	tasks := make([]*Task, 0, len(n.tasks))
	for _, t := range n.tasks {
		tasks = append(tasks, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// handlePendingTasks returns pending tasks for miners
func (n *AINode) handlePendingTasks(w http.ResponseWriter, r *http.Request) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	pending := make([]*Task, 0)
	for _, t := range n.tasks {
		if t.Status == "pending" {
			pending = append(pending, t)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pending)
}

// handleSubmitResult handles task result submission
func (n *AINode) handleSubmitResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	n.mu.Lock()
	if existing, ok := n.tasks[task.ID]; ok {
		existing.Output = task.Output
		existing.Status = task.Status
	}
	n.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleStats returns node statistics
func (n *AINode) handleStats(w http.ResponseWriter, r *http.Request) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var pending, completed, failed int
	for _, t := range n.tasks {
		switch t.Status {
		case "pending":
			pending++
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"miners_connected": len(n.miners),
		"models_available": len(n.models),
		"tasks_pending":    pending,
		"tasks_completed":  completed,
		"tasks_failed":     failed,
	})
}

// handleHealth returns health status
func (n *AINode) handleHealth(w http.ResponseWriter, r *http.Request) {
	n.mu.RLock()
	running := n.running
	n.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"running": running,
		"version": version,
	})
}
