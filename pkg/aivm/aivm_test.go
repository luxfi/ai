// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package aivm

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/luxfi/ai/pkg/attestation"
)

func TestNewVM(t *testing.T) {
	vm := NewVM()
	if vm == nil {
		t.Fatal("NewVM() returned nil")
	}
	if vm.tasks == nil {
		t.Error("tasks map not initialized")
	}
	if vm.providers == nil {
		t.Error("providers map not initialized")
	}
	if vm.models == nil {
		t.Error("models map not initialized")
	}
	if vm.verifier == nil {
		t.Error("verifier not initialized")
	}
	if vm.distributor == nil {
		t.Error("distributor not initialized")
	}
}

func TestDefaultModels(t *testing.T) {
	models := defaultModels()
	if len(models) == 0 {
		t.Error("no default models defined")
	}

	expectedModels := []string{"zen-coder-1.5b", "zen-mini-0.5b", "qwen3-8b"}
	for _, name := range expectedModels {
		if _, ok := models[name]; !ok {
			t.Errorf("expected model %s not found", name)
		}
	}
}

func TestStartStop(t *testing.T) {
	vm := NewVM()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := vm.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Starting again should fail
	err = vm.Start(ctx)
	if err == nil {
		t.Error("expected error starting already running VM")
	}

	err = vm.Stop()
	if err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
}

func TestRegisterProvider(t *testing.T) {
	vm := NewVM()

	provider := &Provider{
		ID:            "provider-001",
		WalletAddress: "0x1234567890abcdef",
		Endpoint:      "http://localhost:8080",
		GPUs: []GPUInfo{
			{Model: "H100", Memory: 80, TFLOPS: 1979, Available: true},
		},
	}

	err := vm.RegisterProvider(provider)
	if err != nil {
		t.Fatalf("RegisterProvider() error: %v", err)
	}

	providers := vm.GetProviders()
	if len(providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(providers))
	}

	if providers[0].ID != "provider-001" {
		t.Errorf("provider ID = %s, want provider-001", providers[0].ID)
	}
}

func TestRegisterProviderWithGPUAttestation(t *testing.T) {
	vm := NewVM()

	provider := &Provider{
		ID:            "provider-001",
		WalletAddress: "0x1234567890abcdef",
		Endpoint:      "http://localhost:8080",
		GPUs: []GPUInfo{
			{Model: "H100", Memory: 80, TFLOPS: 1979, Available: true},
		},
		GPUAttestation: &attestation.GPUAttestation{
			DeviceID:     "GPU-001",
			Model:        "H100",
			CCEnabled:    true,
			TEEIOEnabled: true,
			NRASToken:    make([]byte, 256),
		},
	}

	err := vm.RegisterProvider(provider)
	if err != nil {
		t.Fatalf("RegisterProvider() error: %v", err)
	}

	if provider.Reputation == 0 {
		t.Error("reputation should be set from attestation trust score")
	}
}

func TestSubmitTask(t *testing.T) {
	vm := NewVM()

	task := &Task{
		ID:    "task-001",
		Type:  TaskTypeInference,
		Model: "zen-coder-1.5b",
		Input: json.RawMessage(`{"prompt": "Hello"}`),
		Fee:   1000,
	}

	err := vm.SubmitTask(task)
	if err != nil {
		t.Fatalf("SubmitTask() error: %v", err)
	}

	retrieved, err := vm.GetTask("task-001")
	if err != nil {
		t.Fatalf("GetTask() error: %v", err)
	}

	if retrieved.Status != TaskStatusPending {
		t.Errorf("task status = %s, want pending", retrieved.Status)
	}
}

func TestSubmitTask_InvalidTask(t *testing.T) {
	vm := NewVM()

	task := &Task{
		ID: "", // Invalid - empty ID
	}

	err := vm.SubmitTask(task)
	if err != ErrInvalidTask {
		t.Errorf("expected ErrInvalidTask, got %v", err)
	}
}

func TestGetTask_NotFound(t *testing.T) {
	vm := NewVM()

	_, err := vm.GetTask("nonexistent")
	if err != ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestSubmitResult(t *testing.T) {
	vm := NewVM()

	// Register a provider
	provider := &Provider{
		ID:       "provider-001",
		Endpoint: "http://localhost:8080",
		GPUs:     []GPUInfo{{Model: "H100", Available: true}},
	}
	vm.RegisterProvider(provider)

	// Submit a task
	task := &Task{
		ID:    "task-001",
		Type:  TaskTypeInference,
		Model: "zen-coder-1.5b",
		Input: json.RawMessage(`{"prompt": "Hello"}`),
	}
	vm.SubmitTask(task)

	// Submit result
	result := &TaskResult{
		TaskID:      "task-001",
		ProviderID:  "provider-001",
		Output:      json.RawMessage(`{"response": "Hi there!"}`),
		ComputeTime: 500,
		Proof:       make([]byte, 64),
	}

	err := vm.SubmitResult(result)
	if err != nil {
		t.Fatalf("SubmitResult() error: %v", err)
	}

	retrieved, _ := vm.GetTask("task-001")
	if retrieved.Status != TaskStatusCompleted {
		t.Errorf("task status = %s, want completed", retrieved.Status)
	}
}

func TestSubmitResult_TaskNotFound(t *testing.T) {
	vm := NewVM()

	result := &TaskResult{
		TaskID:     "nonexistent",
		ProviderID: "provider-001",
	}

	err := vm.SubmitResult(result)
	if err != ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestSubmitResult_WithError(t *testing.T) {
	vm := NewVM()

	task := &Task{
		ID:    "task-001",
		Type:  TaskTypeInference,
		Model: "zen-coder-1.5b",
		Input: json.RawMessage(`{"prompt": "Hello"}`),
	}
	vm.SubmitTask(task)

	result := &TaskResult{
		TaskID:     "task-001",
		ProviderID: "provider-001",
		Error:      "model not available",
	}

	err := vm.SubmitResult(result)
	if err != nil {
		t.Fatalf("SubmitResult() error: %v", err)
	}

	retrieved, _ := vm.GetTask("task-001")
	if retrieved.Status != TaskStatusFailed {
		t.Errorf("task status = %s, want failed", retrieved.Status)
	}
}

func TestGetPendingTasks(t *testing.T) {
	vm := NewVM()

	for i := 0; i < 5; i++ {
		task := &Task{
			ID:    "task-" + string(rune('0'+i)),
			Type:  TaskTypeInference,
			Model: "zen-coder-1.5b",
			Input: json.RawMessage(`{}`),
		}
		vm.SubmitTask(task)
	}

	pending := vm.GetPendingTasks("provider-001")
	if len(pending) != 5 {
		t.Errorf("expected 5 pending tasks, got %d", len(pending))
	}
}

func TestGetModels(t *testing.T) {
	vm := NewVM()

	models := vm.GetModels()
	if len(models) == 0 {
		t.Error("expected models, got none")
	}

	hasZenCoder := false
	for _, m := range models {
		if m.ID == "zen-coder-1.5b" {
			hasZenCoder = true
			break
		}
	}
	if !hasZenCoder {
		t.Error("zen-coder-1.5b not found in models")
	}
}

func TestGetStats(t *testing.T) {
	vm := NewVM()

	// Submit some tasks
	for i := 0; i < 3; i++ {
		task := &Task{
			ID:    "task-" + string(rune('0'+i)),
			Type:  TaskTypeInference,
			Model: "zen-coder-1.5b",
			Input: json.RawMessage(`{}`),
		}
		vm.SubmitTask(task)
	}

	stats := vm.GetStats()

	if stats["tasks_pending"].(int) != 3 {
		t.Errorf("tasks_pending = %v, want 3", stats["tasks_pending"])
	}
	if stats["models_available"].(int) == 0 {
		t.Error("models_available should be > 0")
	}
}

func TestGetRewardStats(t *testing.T) {
	vm := NewVM()

	// Register provider and submit task with result
	provider := &Provider{
		ID:       "provider-001",
		Endpoint: "http://localhost:8080",
		GPUs:     []GPUInfo{{Model: "H100", Available: true}},
	}
	vm.RegisterProvider(provider)

	task := &Task{
		ID:    "task-001",
		Type:  TaskTypeInference,
		Model: "zen-coder-1.5b",
		Input: json.RawMessage(`{}`),
	}
	vm.SubmitTask(task)

	result := &TaskResult{
		TaskID:      "task-001",
		ProviderID:  "provider-001",
		Output:      json.RawMessage(`{}`),
		ComputeTime: 500,
		Proof:       make([]byte, 64),
	}
	vm.SubmitResult(result)

	stats, err := vm.GetRewardStats("provider-001")
	if err != nil {
		t.Fatalf("GetRewardStats() error: %v", err)
	}

	if stats["tasks_completed"].(uint64) != 1 {
		t.Errorf("tasks_completed = %v, want 1", stats["tasks_completed"])
	}
}

func TestGetRewardStats_NotFound(t *testing.T) {
	vm := NewVM()

	_, err := vm.GetRewardStats("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent provider")
	}
}

func TestClaimRewards(t *testing.T) {
	vm := NewVM()

	// Register provider and submit task with result
	provider := &Provider{
		ID:       "provider-001",
		Endpoint: "http://localhost:8080",
		GPUs:     []GPUInfo{{Model: "H100", Available: true}},
	}
	vm.RegisterProvider(provider)

	task := &Task{
		ID:    "task-001",
		Type:  TaskTypeInference,
		Model: "zen-coder-1.5b",
		Input: json.RawMessage(`{}`),
	}
	vm.SubmitTask(task)

	result := &TaskResult{
		TaskID:      "task-001",
		ProviderID:  "provider-001",
		Output:      json.RawMessage(`{}`),
		ComputeTime: 500,
		Proof:       make([]byte, 64),
	}
	vm.SubmitResult(result)

	claimed, err := vm.ClaimRewards("provider-001")
	if err != nil {
		t.Fatalf("ClaimRewards() error: %v", err)
	}

	if claimed == "" || claimed == "0" {
		t.Error("claimed rewards should be positive")
	}

	// Second claim should return 0
	claimed2, _ := vm.ClaimRewards("provider-001")
	if claimed2 != "0" {
		t.Errorf("second claim = %s, want 0", claimed2)
	}
}

func TestGetMerkleRoot(t *testing.T) {
	vm := NewVM()

	// Empty merkle root
	emptyRoot := vm.GetMerkleRoot()
	if emptyRoot != [32]byte{} {
		t.Error("empty VM should have empty merkle root")
	}

	// Submit task with result
	provider := &Provider{
		ID:       "provider-001",
		Endpoint: "http://localhost:8080",
		GPUs:     []GPUInfo{{Model: "H100", Available: true}},
	}
	vm.RegisterProvider(provider)

	task := &Task{
		ID:    "task-001",
		Type:  TaskTypeInference,
		Model: "zen-coder-1.5b",
		Input: json.RawMessage(`{}`),
	}
	vm.SubmitTask(task)

	result := &TaskResult{
		TaskID:      "task-001",
		ProviderID:  "provider-001",
		Output:      json.RawMessage(`{}`),
		ComputeTime: 500,
		Proof:       make([]byte, 64),
	}
	vm.SubmitResult(result)

	root := vm.GetMerkleRoot()
	if root == [32]byte{} {
		t.Error("merkle root should not be empty after task completion")
	}
}

func TestTaskTypes(t *testing.T) {
	types := []TaskType{
		TaskTypeInference,
		TaskTypeTraining,
		TaskTypeEmbedding,
		TaskTypeMining,
	}

	for _, tt := range types {
		if tt == "" {
			t.Error("task type should not be empty")
		}
	}
}

func TestTaskStatuses(t *testing.T) {
	statuses := []TaskStatus{
		TaskStatusPending,
		TaskStatusAssigned,
		TaskStatusProcessing,
		TaskStatusCompleted,
		TaskStatusFailed,
	}

	for _, s := range statuses {
		if s == "" {
			t.Error("task status should not be empty")
		}
	}
}

func TestAssignTask(t *testing.T) {
	vm := NewVM()

	// Register a provider
	provider := &Provider{
		ID:       "provider-001",
		Endpoint: "http://localhost:8080",
		GPUs:     []GPUInfo{{Model: "H100", Available: true}},
		Status: ProviderStatus{
			Online:   true,
			LastSeen: time.Now(),
			MaxTasks: 10,
		},
		Reputation: 90,
	}
	vm.providers[provider.ID] = provider

	// Create and assign task
	task := &Task{
		ID:     "task-001",
		Type:   TaskTypeInference,
		Model:  "zen-coder-1.5b",
		Input:  json.RawMessage(`{}`),
		Status: TaskStatusPending,
	}
	vm.tasks[task.ID] = task

	vm.assignTask(task)

	if task.Status != TaskStatusAssigned {
		t.Errorf("task status = %s, want assigned", task.Status)
	}
	if task.AssignedTo != "provider-001" {
		t.Errorf("task assigned to = %s, want provider-001", task.AssignedTo)
	}
}

func TestAssignTask_NoAvailableProvider(t *testing.T) {
	vm := NewVM()

	// Register an offline provider
	provider := &Provider{
		ID:       "provider-001",
		Endpoint: "http://localhost:8080",
		Status: ProviderStatus{
			Online: false,
		},
	}
	vm.providers[provider.ID] = provider

	task := &Task{
		ID:     "task-001",
		Type:   TaskTypeInference,
		Status: TaskStatusPending,
	}
	vm.tasks[task.ID] = task

	vm.assignTask(task)

	// Task should remain pending since no provider available
	if task.Status != TaskStatusPending {
		t.Errorf("task status = %s, want pending (no available provider)", task.Status)
	}
}

func TestAssignTask_BestProviderSelection(t *testing.T) {
	vm := NewVM()

	// Register multiple providers with different reputations
	providers := []*Provider{
		{
			ID:         "low-rep",
			Endpoint:   "http://localhost:8080",
			Reputation: 50,
			Status:     ProviderStatus{Online: true, MaxTasks: 10},
		},
		{
			ID:         "high-rep",
			Endpoint:   "http://localhost:8081",
			Reputation: 95,
			Status:     ProviderStatus{Online: true, MaxTasks: 10},
		},
		{
			ID:         "med-rep",
			Endpoint:   "http://localhost:8082",
			Reputation: 75,
			Status:     ProviderStatus{Online: true, MaxTasks: 10},
		},
	}

	for _, p := range providers {
		vm.providers[p.ID] = p
	}

	task := &Task{
		ID:     "task-001",
		Type:   TaskTypeInference,
		Status: TaskStatusPending,
	}
	vm.tasks[task.ID] = task

	vm.assignTask(task)

	if task.AssignedTo != "high-rep" {
		t.Errorf("task assigned to %s, want high-rep (highest reputation)", task.AssignedTo)
	}
}

func TestProviderStatusTracking(t *testing.T) {
	vm := NewVM()

	provider := &Provider{
		ID:       "provider-001",
		Endpoint: "http://localhost:8080",
		GPUs:     []GPUInfo{{Model: "H100", Available: true}},
		Status: ProviderStatus{
			Online:   true,
			LastSeen: time.Now(),
			MaxTasks: 10,
		},
	}
	vm.RegisterProvider(provider)

	// Submit and complete multiple tasks
	for i := 0; i < 5; i++ {
		task := &Task{
			ID:    "task-" + string(rune('0'+i)),
			Type:  TaskTypeInference,
			Model: "zen-coder-1.5b",
			Input: json.RawMessage(`{}`),
		}
		vm.SubmitTask(task)

		result := &TaskResult{
			TaskID:      task.ID,
			ProviderID:  "provider-001",
			Output:      json.RawMessage(`{}`),
			ComputeTime: 500,
			Proof:       make([]byte, 64),
		}
		vm.SubmitResult(result)
	}

	// Check provider stats
	providers := vm.GetProviders()
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}

	if providers[0].TasksHandled != 5 {
		t.Errorf("TasksHandled = %d, want 5", providers[0].TasksHandled)
	}
}

func TestGetModelHash(t *testing.T) {
	vm := NewVM()

	// Known model
	hash := vm.getModelHash("zen-coder-1.5b")
	if hash == [32]byte{} {
		t.Error("hash for known model should not be empty")
	}

	// Unknown model - should still return a hash
	unknownHash := vm.getModelHash("unknown-model")
	if unknownHash == [32]byte{} {
		t.Error("hash for unknown model should not be empty")
	}

	// Different models should have different hashes
	if hash == unknownHash {
		t.Error("different models should have different hashes")
	}
}

func TestConcurrentTaskSubmission(t *testing.T) {
	vm := NewVM()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vm.Start(ctx)
	defer vm.Stop()

	// Submit tasks concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			task := &Task{
				ID:    "task-" + string(rune('A'+idx)),
				Type:  TaskTypeInference,
				Model: "zen-coder-1.5b",
				Input: json.RawMessage(`{}`),
			}
			vm.SubmitTask(task)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	stats := vm.GetStats()
	pendingOrAssigned := stats["tasks_pending"].(int)
	if pendingOrAssigned == 0 {
		t.Error("expected tasks to be submitted")
	}
}
