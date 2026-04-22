package miner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luxfi/ai/pkg/miner/backend"
	"github.com/luxfi/ai/pkg/miner/backend/noop"
)

func TestNewMiner(t *testing.T) {
	cfg := Config{
		NodeURL:       "http://localhost:9090",
		WalletAddress: "0x1234567890abcdef",
		ModelDir:      "/tmp/models",
		CacheSize:     1024 * 1024 * 1024,
		GPUEnabled:    false,
		APIPort:       8888,
		MaxTasks:      10,
	}

	m := New(cfg)
	if m == nil {
		t.Fatal("expected non-nil miner")
	}
	if m.config.NodeURL != cfg.NodeURL {
		t.Errorf("expected node URL %s, got %s", cfg.NodeURL, m.config.NodeURL)
	}
	if m.config.WalletAddress != cfg.WalletAddress {
		t.Errorf("expected wallet %s, got %s", cfg.WalletAddress, m.config.WalletAddress)
	}
}

func TestMinerStatus(t *testing.T) {
	cfg := Config{
		NodeURL:       "http://localhost:9090",
		WalletAddress: "0xtest",
		ModelDir:      "/tmp/models",
		CacheSize:     1024 * 1024 * 1024,
		GPUEnabled:    false,
		APIPort:       8889,
		MaxTasks:      10,
	}

	m := New(cfg)
	status := m.Status()

	if status.Wallet != cfg.WalletAddress {
		t.Errorf("expected wallet %s, got %s", cfg.WalletAddress, status.Wallet)
	}
	if status.Running {
		t.Error("expected miner to not be running initially")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.NodeURL != "http://localhost:9650" {
		t.Errorf("expected default node URL http://localhost:9650, got %s", cfg.NodeURL)
	}
	if !cfg.GPUEnabled {
		t.Error("expected GPU to be enabled by default")
	}
	if cfg.MaxTasks != 10 {
		t.Errorf("expected max tasks 10, got %d", cfg.MaxTasks)
	}
	if cfg.CacheSize != 10*1024*1024*1024 {
		t.Errorf("expected cache size 10GB, got %d", cfg.CacheSize)
	}
}

func TestHealthHandler(t *testing.T) {
	cfg := Config{
		NodeURL:       "http://localhost:9090",
		WalletAddress: "0xtest",
		ModelDir:      "/tmp/models",
		CacheSize:     1024 * 1024 * 1024,
		GPUEnabled:    false,
		APIPort:       8890,
		MaxTasks:      10,
	}

	m := New(cfg)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	m.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["status"] != "stopped" {
		t.Errorf("expected status 'stopped', got %v", resp["status"])
	}
}

func TestStatsHandler(t *testing.T) {
	cfg := Config{
		NodeURL:       "http://localhost:9090",
		WalletAddress: "0xtest",
		ModelDir:      "/tmp/models",
		CacheSize:     1024 * 1024 * 1024,
		GPUEnabled:    false,
		APIPort:       8891,
		MaxTasks:      10,
	}

	m := New(cfg)

	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()

	m.handleStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var stats Stats
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if stats.TasksCompleted != 0 {
		t.Errorf("expected 0 tasks completed, got %d", stats.TasksCompleted)
	}
}

func TestChatCompletionRequest(t *testing.T) {
	reqBody := `{
		"model": "zen-mini-0.5b",
		"messages": [
			{"role": "user", "content": "Hello"}
		]
	}`

	var req ChatCompletionRequest
	if err := json.Unmarshal([]byte(reqBody), &req); err != nil {
		t.Fatalf("failed to unmarshal request: %v", err)
	}

	if req.Model != "zen-mini-0.5b" {
		t.Errorf("expected model zen-mini-0.5b, got %s", req.Model)
	}
	if len(req.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(req.Messages))
	}
	if req.Messages[0].Role != "user" {
		t.Errorf("expected role user, got %s", req.Messages[0].Role)
	}
}

func TestChatCompletionResponse(t *testing.T) {
	resp := ChatCompletionResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "zen-mini-0.5b",
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "Hello! How can I help?",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 8,
			TotalTokens:      18,
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var decoded ChatCompletionResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if decoded.ID != resp.ID {
		t.Errorf("expected ID %s, got %s", resp.ID, decoded.ID)
	}
	if len(decoded.Choices) != 1 {
		t.Errorf("expected 1 choice, got %d", len(decoded.Choices))
	}
	if decoded.Usage.TotalTokens != 18 {
		t.Errorf("expected 18 total tokens, got %d", decoded.Usage.TotalTokens)
	}
}

func TestTaskTypes(t *testing.T) {
	if TaskInference != "inference" {
		t.Errorf("expected TaskInference to be 'inference', got %s", TaskInference)
	}
	if TaskChat != "chat" {
		t.Errorf("expected TaskChat to be 'chat', got %s", TaskChat)
	}
	if TaskEmbedding != "embedding" {
		t.Errorf("expected TaskEmbedding to be 'embedding', got %s", TaskEmbedding)
	}
}

func TestErrors(t *testing.T) {
	if ErrNotRunning.Error() != "miner not running" {
		t.Errorf("unexpected error message: %s", ErrNotRunning.Error())
	}
	if ErrAlreadyRunning.Error() != "miner already running" {
		t.Errorf("unexpected error message: %s", ErrAlreadyRunning.Error())
	}
}

func TestTaskJSON(t *testing.T) {
	task := Task{
		ID:     "test-123",
		Type:   TaskChat,
		Model:  "zen-mini-0.5b",
		Status: "pending",
		Reward: 100,
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("failed to marshal task: %v", err)
	}

	if !strings.Contains(string(data), "test-123") {
		t.Error("expected JSON to contain task ID")
	}
	if !strings.Contains(string(data), "chat") {
		t.Error("expected JSON to contain task type")
	}
}

// TestDefaultBackendIsNoop locks in the zero-config behaviour: a Miner built
// from DefaultConfig() uses the deterministic noop backend, so existing
// deployments see no change.
func TestDefaultBackendIsNoop(t *testing.T) {
	m := New(DefaultConfig())
	if name := m.Backend().Name(); name != "noop" {
		t.Errorf("default backend: got %q want %q", name, "noop")
	}
}

// TestBackendSelectionViaConfig confirms Config.Backend wires through.
func TestBackendSelectionViaConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Backend = "openai"
	m := New(cfg)
	if name := m.Backend().Name(); name != "openai" {
		t.Errorf("configured backend: got %q want %q", name, "openai")
	}
}

// TestUnknownBackendFallsBackToNoop guards the operator-mistype path.
func TestUnknownBackendFallsBackToNoop(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Backend = "not-a-real-backend"
	m := New(cfg)
	if name := m.Backend().Name(); name != "noop" {
		t.Errorf("unknown backend fallback: got %q want %q", name, "noop")
	}
}

// TestWithBackendHotSwap checks the runtime override used by tests and by
// operators wiring a custom backend from main.
func TestWithBackendHotSwap(t *testing.T) {
	m := New(DefaultConfig()).WithBackend(noop.New())
	if name := m.Backend().Name(); name != "noop" {
		t.Errorf("WithBackend: got %q want %q", name, "noop")
	}
}

// TestRunChatUsesBackend end-to-end confirms the wiring: swapping the
// backend changes the output of runChat.
func TestRunChatUsesBackend(t *testing.T) {
	m := New(DefaultConfig()).WithBackend(&recordingBackend{
		chatContent: "custom response from swapped backend",
	})

	input, _ := json.Marshal(map[string]any{
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	task := &Task{Type: TaskChat, Model: "m", Input: input}

	if err := m.runChat(context.Background(), task); err != nil {
		t.Fatalf("runChat: %v", err)
	}
	if !strings.Contains(string(task.Output), "custom response from swapped backend") {
		t.Errorf("runChat should route through backend, got: %s", task.Output)
	}
}

// TestRunEmbeddingUsesBackend mirrors TestRunChatUsesBackend for embeddings.
func TestRunEmbeddingUsesBackend(t *testing.T) {
	m := New(DefaultConfig()).WithBackend(&recordingBackend{
		embedding: []float64{1, 2, 3},
	})

	input, _ := json.Marshal(map[string]string{"text": "hello"})
	task := &Task{Type: TaskEmbedding, Model: "m", Input: input}

	if err := m.runEmbedding(context.Background(), task); err != nil {
		t.Fatalf("runEmbedding: %v", err)
	}
	if !strings.Contains(string(task.Output), "[1,2,3]") {
		t.Errorf("runEmbedding should route through backend, got: %s", task.Output)
	}
}

// TestGPUStatsProviderPopulatesStats checks the sibling GPU-utilization
// hook: GetStats merges provider output into the returned Stats without
// breaking callers that don't install a provider.
func TestGPUStatsProviderPopulatesStats(t *testing.T) {
	m := New(DefaultConfig())

	// No provider: GPUUtilization stays 0.0 (pre-refactor behaviour).
	if got := m.GetStats().GPUUtilization; got != 0.0 {
		t.Errorf("no provider: GPUUtilization = %v want 0.0", got)
	}

	m.SetGPUStatsProvider(func() (float64, uint64) { return 0.42, 12345 })
	stats := m.GetStats()
	if stats.GPUUtilization != 0.42 {
		t.Errorf("GPUUtilization: got %v want 0.42", stats.GPUUtilization)
	}
	if stats.MemoryUsed != 12345 {
		t.Errorf("MemoryUsed: got %d want 12345", stats.MemoryUsed)
	}

	// Removing the provider restores the zero-cost default.
	m.SetGPUStatsProvider(nil)
	if got := m.GetStats().GPUUtilization; got != 0.0 {
		t.Errorf("provider removed: GPUUtilization = %v want 0.0", got)
	}
}

// recordingBackend is a test double implementing backend.InferenceBackend.
type recordingBackend struct {
	chatContent string
	embedding   []float64
}

func (*recordingBackend) Name() string { return "recording" }
func (*recordingBackend) Capabilities() backend.Capabilities {
	return backend.Capabilities{Chat: true, Inference: true, Embedding: true}
}
func (r *recordingBackend) Chat(_ context.Context, req backend.ChatRequest) (backend.ChatResponse, error) {
	return backend.ChatResponse{Role: "assistant", Content: r.chatContent, Model: req.Model}, nil
}
func (r *recordingBackend) Inference(_ context.Context, req backend.InferenceRequest) (backend.InferenceResponse, error) {
	return backend.InferenceResponse{Text: r.chatContent, Model: req.Model}, nil
}
func (r *recordingBackend) Embed(_ context.Context, req backend.EmbedRequest) (backend.EmbedResponse, error) {
	return backend.EmbedResponse{Embedding: r.embedding, Model: req.Model}, nil
}
