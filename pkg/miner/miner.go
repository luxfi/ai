// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package miner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/luxfi/ai/pkg/miner/backend"
	"github.com/luxfi/ai/pkg/miner/backend/noop"
	"github.com/luxfi/ai/pkg/miner/backend/openai"
)

var (
	ErrNotRunning     = errors.New("miner not running")
	ErrAlreadyRunning = errors.New("miner already running")
	ErrNoGPU          = errors.New("no GPU available")
	ErrInvalidTask    = errors.New("invalid task")
)

// TaskType represents the type of AI task
type TaskType string

const (
	TaskInference TaskType = "inference"
	TaskTraining  TaskType = "training"
	TaskEmbedding TaskType = "embedding"
	TaskChat      TaskType = "chat"
)

// Task represents an AI computation task
type Task struct {
	ID        string          `json:"id"`
	Type      TaskType        `json:"type"`
	Model     string          `json:"model"`
	Input     json.RawMessage `json:"input"`
	Output    json.RawMessage `json:"output,omitempty"`
	Status    string          `json:"status"`
	Reward    uint64          `json:"reward"`
	CreatedAt time.Time       `json:"created_at"`
	StartedAt *time.Time      `json:"started_at,omitempty"`
	EndedAt   *time.Time      `json:"ended_at,omitempty"`
}

// Stats tracks miner statistics
type Stats struct {
	TasksCompleted   uint64        `json:"tasks_completed"`
	TasksFailed      uint64        `json:"tasks_failed"`
	TotalRewards     uint64        `json:"total_rewards"`
	Uptime           time.Duration `json:"uptime"`
	GPUUtilization   float64       `json:"gpu_utilization"`
	MemoryUsed       uint64        `json:"memory_used"`
	InferenceLatency time.Duration `json:"inference_latency"`
}

// Config holds miner configuration
type Config struct {
	WalletAddress string `json:"wallet_address"`
	NodeURL       string `json:"node_url"`
	GPUEnabled    bool   `json:"gpu_enabled"`
	MaxTasks      int    `json:"max_tasks"`
	CacheSize     int64  `json:"cache_size"` // in bytes
	ModelDir      string `json:"model_dir"`
	APIPort       int    `json:"api_port"`

	// Backend selects the inference-engine adapter used by the miner.
	// Supported values: "noop" (default, deterministic mock), "openai"
	// (OpenAI-compatible HTTP — works for the public OpenAI API and for
	// local engines like llama.cpp server, vllm, ollama, and LocalAI that
	// expose the same dialect).
	//
	// When the value is empty the miner falls back to the noop backend, so
	// existing callers see no behaviour change.
	Backend string `json:"backend,omitempty"`

	// OpenAIBase overrides the OpenAI API base URL (e.g.
	// "http://localhost:8080/v1" for llama.cpp, "http://localhost:11434/v1"
	// for ollama). Only used when Backend == "openai".
	OpenAIBase string `json:"openai_base,omitempty"`

	// OpenAIAPIKey is the bearer token sent with OpenAI requests. Empty is
	// fine for local engines that don't authenticate.
	OpenAIAPIKey string `json:"openai_api_key,omitempty"`

	// OpenAIModel is the default model name passed to the OpenAI backend
	// when the caller doesn't set Task.Model.
	OpenAIModel string `json:"openai_model,omitempty"`

	// OpenAIEmbeddingModel overrides OpenAIModel for embedding tasks.
	OpenAIEmbeddingModel string `json:"openai_embedding_model,omitempty"`
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		NodeURL:    "http://localhost:9650",
		GPUEnabled: true,
		MaxTasks:   10,
		CacheSize:  10 * 1024 * 1024 * 1024, // 10GB
		ModelDir:   "./models",
		APIPort:    8888,
	}
}

// Miner represents an AI mining node
type Miner struct {
	config    Config
	running   bool
	startTime time.Time
	stats     Stats
	tasks     map[string]*Task
	mu        sync.RWMutex

	// Pluggable inference backend. The miner's run* methods dispatch
	// through this interface; Config.Backend selects the default at
	// construction time, and callers can override via WithBackend.
	backend backend.InferenceBackend

	// Optional GPU-telemetry hook; see SetGPUStatsProvider. Leaving it nil
	// keeps GetStats zero-cost on systems without GPU telemetry wired.
	gpuStatsProvider GPUStatsProvider

	// Channels
	taskCh   chan *Task
	resultCh chan *Task
	stopCh   chan struct{}

	// HTTP server
	server *http.Server
}

// New creates a new miner instance. The inference backend is selected from
// config.Backend; when unset, a deterministic noop backend is used so legacy
// callers see no behaviour change.
func New(config Config) *Miner {
	return &Miner{
		config:   config,
		tasks:    make(map[string]*Task),
		backend:  newBackend(config),
		taskCh:   make(chan *Task, config.MaxTasks),
		resultCh: make(chan *Task, config.MaxTasks),
		stopCh:   make(chan struct{}),
	}
}

// newBackend picks a backend.InferenceBackend from config. Unknown or empty
// Backend values fall back to noop (safe default).
func newBackend(cfg Config) backend.InferenceBackend {
	switch cfg.Backend {
	case "openai":
		return openai.New(openai.Config{
			BaseURL:        cfg.OpenAIBase,
			APIKey:         cfg.OpenAIAPIKey,
			Model:          cfg.OpenAIModel,
			EmbeddingModel: cfg.OpenAIEmbeddingModel,
		})
	case "", "noop":
		return noop.New()
	default:
		// Unknown backend name: fall back to noop rather than failing.
		// Operators who mistype a name will see "noop" in logs and can
		// fix it without the miner crash-looping.
		return noop.New()
	}
}

// WithBackend swaps the inference backend at runtime. It is safe to call
// before Start; calling it after Start may race with in-flight tasks, so
// callers wanting hot-swap should stop the miner first.
//
// This is primarily intended for tests and for operators wiring custom
// backends (e.g. a direct MLX/CUDA binding) from their own main package.
func (m *Miner) WithBackend(b backend.InferenceBackend) *Miner {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b != nil {
		m.backend = b
	}
	return m
}

// Backend returns the currently configured inference backend. Useful for
// tests and for exposing Capabilities() over the API.
func (m *Miner) Backend() backend.InferenceBackend {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.backend
}

// Start begins mining operations
func (m *Miner) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return ErrAlreadyRunning
	}
	m.running = true
	m.startTime = time.Now()
	m.mu.Unlock()

	// Start task worker
	go m.taskWorker(ctx)

	// Start result handler
	go m.resultHandler(ctx)

	// Start API server
	go m.startAPI()

	// Main mining loop
	go m.miningLoop(ctx)

	return nil
}

// Stop halts mining operations
func (m *Miner) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return ErrNotRunning
	}
	m.running = false
	m.mu.Unlock()

	close(m.stopCh)

	if m.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		m.server.Shutdown(ctx)
	}

	return nil
}

// GPUStatsProvider is an optional hook that populates GPU telemetry into
// Stats. When set, GetStats invokes it and merges the result into the
// returned snapshot. Leave unset for a zero-cost no-op (GPUUtilization stays
// 0.0, which is the current pre-refactor behaviour).
//
// A follow-up PR will wire a real NVML/MLX provider; this hook keeps the
// seam in place without pulling heavy bindings into the main miner binary.
type GPUStatsProvider func() (utilization float64, memoryUsed uint64)

// SetGPUStatsProvider installs a best-effort GPU telemetry source. Safe to
// call before or after Start. Passing nil removes the hook.
func (m *Miner) SetGPUStatsProvider(p GPUStatsProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gpuStatsProvider = p
}

// GetStats returns current mining statistics, including best-effort GPU
// telemetry when a GPUStatsProvider has been installed.
func (m *Miner) GetStats() Stats {
	m.mu.RLock()
	provider := m.gpuStatsProvider
	m.mu.RUnlock()

	m.mu.RLock()
	stats := m.stats
	if m.running {
		stats.Uptime = time.Since(m.startTime)
	}
	m.mu.RUnlock()

	// Populate GPU telemetry outside the lock so a slow provider can't
	// stall unrelated callers.
	if provider != nil {
		util, mem := provider()
		stats.GPUUtilization = util
		if mem != 0 {
			stats.MemoryUsed = mem
		}
	}
	return stats
}

// SubmitTask submits a new task for processing
func (m *Miner) SubmitTask(task *Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return ErrNotRunning
	}

	if task.ID == "" {
		id := make([]byte, 16)
		rand.Read(id)
		task.ID = hex.EncodeToString(id)
	}

	task.Status = "pending"
	task.CreatedAt = time.Now()
	m.tasks[task.ID] = task

	select {
	case m.taskCh <- task:
		return nil
	default:
		return errors.New("task queue full")
	}
}

// GetTask retrieves a task by ID
func (m *Miner) GetTask(id string) (*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[id]
	if !ok {
		return nil, errors.New("task not found")
	}
	return task, nil
}

// miningLoop polls for new tasks from the network
func (m *Miner) miningLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.pollForTasks(ctx)
		}
	}
}

// pollForTasks checks the node for available tasks
func (m *Miner) pollForTasks(ctx context.Context) {
	// In production, this would query the AIVM for pending tasks
	// For now, we just log that we're polling
	m.mu.RLock()
	running := m.running
	m.mu.RUnlock()

	if !running {
		return
	}

	// Query node for tasks
	url := fmt.Sprintf("%s/ext/bc/A/ai/pendingTasks", m.config.NodeURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Parse and submit tasks
	var tasks []*Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return
	}

	for _, task := range tasks {
		m.SubmitTask(task)
	}
}

// taskWorker processes tasks from the queue
func (m *Miner) taskWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case task := <-m.taskCh:
			m.processTask(ctx, task)
		}
	}
}

// processTask executes an AI task
func (m *Miner) processTask(ctx context.Context, task *Task) {
	m.mu.Lock()
	now := time.Now()
	task.StartedAt = &now
	task.Status = "processing"
	m.mu.Unlock()

	// Process based on task type
	var err error
	switch task.Type {
	case TaskInference:
		err = m.runInference(ctx, task)
	case TaskChat:
		err = m.runChat(ctx, task)
	case TaskEmbedding:
		err = m.runEmbedding(ctx, task)
	default:
		err = ErrInvalidTask
	}

	m.mu.Lock()
	endTime := time.Now()
	task.EndedAt = &endTime

	if err != nil {
		task.Status = "failed"
		m.stats.TasksFailed++
	} else {
		task.Status = "completed"
		m.stats.TasksCompleted++
		m.stats.TotalRewards += task.Reward
	}
	m.mu.Unlock()

	m.resultCh <- task
}

// runInference executes an inference task via the configured backend.
func (m *Miner) runInference(ctx context.Context, task *Task) error {
	var input struct {
		Prompt    string `json:"prompt"`
		MaxTokens int    `json:"max_tokens"`
	}
	if err := json.Unmarshal(task.Input, &input); err != nil {
		return err
	}

	// Dispatch to the pluggable backend. The noop default preserves the
	// legacy placeholder shape ("Response to: <prompt>", tokens=10); the
	// openai backend talks to any OpenAI-compatible server (llama.cpp,
	// vllm, ollama, LocalAI, or api.openai.com itself).
	resp, err := m.Backend().Inference(ctx, backend.InferenceRequest{
		Model:     task.Model,
		Prompt:    input.Prompt,
		MaxTokens: input.MaxTokens,
	})
	if err != nil {
		return err
	}

	output := map[string]interface{}{
		"text":   resp.Text,
		"tokens": resp.Tokens,
		"model":  resp.Model,
	}

	outputBytes, err := json.Marshal(output)
	if err != nil {
		return err
	}
	task.Output = outputBytes

	return nil
}

// runChat handles chat-style inference via the configured backend.
func (m *Miner) runChat(ctx context.Context, task *Task) error {
	var input struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		MaxTokens int `json:"max_tokens"`
	}
	if err := json.Unmarshal(task.Input, &input); err != nil {
		return err
	}

	msgs := make([]backend.Message, 0, len(input.Messages))
	for _, m := range input.Messages {
		msgs = append(msgs, backend.Message{Role: m.Role, Content: m.Content})
	}

	resp, err := m.Backend().Chat(ctx, backend.ChatRequest{
		Model:     task.Model,
		Messages:  msgs,
		MaxTokens: input.MaxTokens,
	})
	if err != nil {
		return err
	}

	output := map[string]interface{}{
		"role":    resp.Role,
		"content": resp.Content,
		"model":   resp.Model,
	}

	outputBytes, err := json.Marshal(output)
	if err != nil {
		return err
	}
	task.Output = outputBytes

	return nil
}

// runEmbedding generates embeddings via the configured backend.
func (m *Miner) runEmbedding(ctx context.Context, task *Task) error {
	var input struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(task.Input, &input); err != nil {
		return err
	}

	resp, err := m.Backend().Embed(ctx, backend.EmbedRequest{
		Model: task.Model,
		Text:  input.Text,
	})
	if err != nil {
		return err
	}

	output := map[string]interface{}{
		"embedding": resp.Embedding,
		"model":     resp.Model,
	}

	outputBytes, err := json.Marshal(output)
	if err != nil {
		return err
	}
	task.Output = outputBytes

	return nil
}

// resultHandler processes completed tasks
func (m *Miner) resultHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case task := <-m.resultCh:
			m.submitResult(ctx, task)
		}
	}
}

// submitResult sends completed task back to the network
func (m *Miner) submitResult(ctx context.Context, task *Task) {
	// In production, this would submit the result to the AIVM
	url := fmt.Sprintf("%s/ext/bc/A/ai/submitResult", m.config.NodeURL)

	body, err := json.Marshal(task)
	if err != nil {
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	_ = body // Would be sent in request body
}

// startAPI starts the local API server
func (m *Miner) startAPI() {
	mux := http.NewServeMux()

	mux.HandleFunc("/stats", m.handleStats)
	mux.HandleFunc("/task", m.handleTask)
	mux.HandleFunc("/chat", m.handleChat)
	mux.HandleFunc("/health", m.handleHealth)

	m.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", m.config.APIPort),
		Handler: mux,
	}

	m.server.ListenAndServe()
}

func (m *Miner) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := m.GetStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (m *Miner) handleTask(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		id := r.URL.Query().Get("id")
		task, err := m.GetTask(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(task)

	case "POST":
		var task Task
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := m.SubmitTask(&task); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"task_id": task.ID})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *Miner) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Model string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create chat task
	input, _ := json.Marshal(req)
	task := &Task{
		Type:  TaskChat,
		Model: req.Model,
		Input: input,
	}

	if err := m.SubmitTask(task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Wait for result (with timeout)
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			http.Error(w, "timeout", http.StatusGatewayTimeout)
			return
		case <-ticker.C:
			t, err := m.GetTask(task.ID)
			if err != nil {
				continue
			}
			if t.Status == "completed" {
				w.Header().Set("Content-Type", "application/json")
				w.Write(t.Output)
				return
			}
			if t.Status == "failed" {
				http.Error(w, "task failed", http.StatusInternalServerError)
				return
			}
		}
	}
}

func (m *Miner) handleHealth(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	running := m.running
	m.mu.RUnlock()

	status := "healthy"
	if !running {
		status = "stopped"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  status,
		"running": running,
	})
}

// MinerStatus represents the current status of the miner
type MinerStatus struct {
	Wallet  string `json:"wallet"`
	Running bool   `json:"running"`
	Stats   Stats  `json:"stats"`
}

// Status returns the current miner status
func (m *Miner) Status() MinerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return MinerStatus{
		Wallet:  m.config.WalletAddress,
		Running: m.running,
		Stats:   m.stats,
	}
}

// IsRunning returns whether the miner is running
func (m *Miner) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// OpenAI compatible types

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest represents an OpenAI-compatible chat request
type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage tracks token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletionResponse represents an OpenAI-compatible chat response
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}
