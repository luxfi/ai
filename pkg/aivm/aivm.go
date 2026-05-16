// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package aivm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/luxfi/ai/pkg/attestation"
	"github.com/luxfi/ai/pkg/rewards"
)

var (
	ErrTaskNotFound    = errors.New("task not found")
	ErrProviderOffline = errors.New("provider offline")
	ErrInvalidTask     = errors.New("invalid task")
)

// TaskType represents types of AI tasks
type TaskType string

const (
	TaskTypeInference TaskType = "inference"
	TaskTypeTraining  TaskType = "training"
	TaskTypeEmbedding TaskType = "embedding"
	TaskTypeMining    TaskType = "mining"
)

// TaskStatus represents task status
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusAssigned   TaskStatus = "assigned"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
)

// Task represents an AI computation task
type Task struct {
	ID          string          `json:"id"`
	Type        TaskType        `json:"type"`
	Model       string          `json:"model"`
	Input       json.RawMessage `json:"input"`
	Output      json.RawMessage `json:"output,omitempty"`
	Status      TaskStatus      `json:"status"`
	AssignedTo  string          `json:"assigned_to,omitempty"`
	Fee         uint64          `json:"fee"`
	CreatedAt   time.Time       `json:"created_at"`
	StartedAt   time.Time       `json:"started_at,omitempty"`
	CompletedAt time.Time       `json:"completed_at,omitempty"`
	ComputeTime uint64          `json:"compute_time_ms,omitempty"`
	Proof       []byte          `json:"proof,omitempty"`
}

// Provider represents an AI compute provider
type Provider struct {
	ID             string                        `json:"id"`
	WalletAddress  string                        `json:"wallet_address"`
	Endpoint       string                        `json:"endpoint"`
	GPUs           []GPUInfo                     `json:"gpus"`
	CPUAttestation *attestation.AttestationQuote `json:"cpu_attestation,omitempty"`
	GPUAttestation *attestation.GPUAttestation   `json:"gpu_attestation,omitempty"`
	Status         ProviderStatus                `json:"status"`
	Reputation     float64                       `json:"reputation"`
	TasksHandled   uint64                        `json:"tasks_handled"`
	JoinedAt       time.Time                     `json:"joined_at"`
}

// GPUInfo describes a GPU
type GPUInfo struct {
	Model       string  `json:"model"`
	Memory      uint64  `json:"memory_gb"`
	TFLOPS      float64 `json:"tflops"`
	Available   bool    `json:"available"`
	Temperature float64 `json:"temperature"`
	Utilization float64 `json:"utilization"`
}

// ProviderStatus represents provider status
type ProviderStatus struct {
	Online       bool      `json:"online"`
	LastSeen     time.Time `json:"last_seen"`
	Uptime       float64   `json:"uptime"`
	FailureRate  float64   `json:"failure_rate"`
	CurrentTasks int       `json:"current_tasks"`
	MaxTasks     int       `json:"max_tasks"`
}

// ModelInfo describes an AI model
type ModelInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Capabilities []string `json:"capabilities"`
	ContextSize  int      `json:"context_size"`
	Parameters   string   `json:"parameters"`
	Hash         [32]byte `json:"hash"`
}

// VM is the AI Virtual Machine
type VM struct {
	mu sync.RWMutex

	// Registries
	tasks     map[string]*Task
	providers map[string]*Provider
	models    map[string]*ModelInfo

	// Attestation and rewards
	verifier    *attestation.Verifier
	distributor *rewards.RewardDistributor

	// State
	running     bool
	taskQueue   chan *Task
	resultQueue chan *TaskResult
}

// TaskResult represents a completed task result
type TaskResult struct {
	TaskID      string          `json:"task_id"`
	ProviderID  string          `json:"provider_id"`
	Output      json.RawMessage `json:"output"`
	ComputeTime uint64          `json:"compute_time_ms"`
	Proof       []byte          `json:"proof"`
	Error       string          `json:"error,omitempty"`
}

// NewVM creates a new AI VM
func NewVM() *VM {
	return &VM{
		tasks:       make(map[string]*Task),
		providers:   make(map[string]*Provider),
		models:      defaultModels(),
		verifier:    attestation.NewVerifier(),
		distributor: rewards.NewRewardDistributor(),
		taskQueue:   make(chan *Task, 1000),
		resultQueue: make(chan *TaskResult, 1000),
	}
}

// defaultModels returns default available models
func defaultModels() map[string]*ModelInfo {
	return map[string]*ModelInfo{
		"zen-coder-1.5b": {
			ID:           "zen-coder-1.5b",
			Name:         "Zen Coder 1.5B",
			Type:         "chat",
			Capabilities: []string{"code", "chat", "completion"},
			ContextSize:  32768,
			Parameters:   "1.5B",
			Hash:         sha256.Sum256([]byte("zen-coder-1.5b")),
		},
		"zen-mini-0.5b": {
			ID:           "zen-mini-0.5b",
			Name:         "Zen Mini 0.5B",
			Type:         "chat",
			Capabilities: []string{"chat", "completion"},
			ContextSize:  8192,
			Parameters:   "0.5B",
			Hash:         sha256.Sum256([]byte("zen-mini-0.5b")),
		},
		"qwen3-8b": {
			ID:           "qwen3-8b",
			Name:         "Qwen3 8B",
			Type:         "chat",
			Capabilities: []string{"chat", "code", "reasoning"},
			ContextSize:  131072,
			Parameters:   "8B",
			Hash:         sha256.Sum256([]byte("qwen3-8b")),
		},
	}
}

// Start starts the AI VM
func (vm *VM) Start(ctx context.Context) error {
	vm.mu.Lock()
	if vm.running {
		vm.mu.Unlock()
		return errors.New("already running")
	}
	vm.running = true
	vm.mu.Unlock()

	// Start task processor
	go vm.processTaskQueue(ctx)
	go vm.processResultQueue(ctx)

	return nil
}

// Stop stops the AI VM
func (vm *VM) Stop() error {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.running = false
	close(vm.taskQueue)
	close(vm.resultQueue)
	return nil
}

// RegisterProvider registers a new compute provider
func (vm *VM) RegisterProvider(provider *Provider) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	// Verify CPU attestation if provided
	if provider.CPUAttestation != nil {
		if err := vm.verifier.VerifyCPUAttestation(provider.CPUAttestation, nil); err != nil {
			return err
		}
	}

	// Verify GPU attestation if provided
	if provider.GPUAttestation != nil {
		status, err := vm.verifier.VerifyGPUAttestation(provider.GPUAttestation)
		if err != nil {
			return err
		}
		provider.Reputation = float64(status.TrustScore)
	}

	provider.JoinedAt = time.Now()
	provider.Status.Online = true
	provider.Status.LastSeen = time.Now()
	provider.Status.Uptime = 1.0
	provider.Status.MaxTasks = len(provider.GPUs) * 2

	vm.providers[provider.ID] = provider
	return nil
}

// SubmitTask submits a new task
func (vm *VM) SubmitTask(task *Task) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if task.ID == "" {
		return ErrInvalidTask
	}

	task.Status = TaskStatusPending
	task.CreatedAt = time.Now()

	vm.tasks[task.ID] = task

	// Send to task queue for processing
	select {
	case vm.taskQueue <- task:
	default:
		// Queue full, task will be processed when space available
	}

	return nil
}

// GetTask returns a task by ID
func (vm *VM) GetTask(taskID string) (*Task, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	task, ok := vm.tasks[taskID]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

// SubmitResult submits a task result
func (vm *VM) SubmitResult(result *TaskResult) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	task, ok := vm.tasks[result.TaskID]
	if !ok {
		return ErrTaskNotFound
	}

	task.Output = result.Output
	task.ComputeTime = result.ComputeTime
	task.Proof = result.Proof
	task.CompletedAt = time.Now()

	if result.Error != "" {
		task.Status = TaskStatusFailed
	} else {
		task.Status = TaskStatusCompleted

		// Create receipt for rewards
		receipt := &rewards.Receipt{
			JobID:       task.ID,
			ProviderID:  result.ProviderID,
			ModelHash:   vm.getModelHash(task.Model),
			ComputeTime: result.ComputeTime,
			Timestamp:   time.Now(),
			Proof:       result.Proof,
		}
		inputHash := sha256.Sum256(task.Input)
		outputHash := sha256.Sum256(result.Output)
		copy(receipt.InputHash[:], inputHash[:])
		copy(receipt.OutputHash[:], outputHash[:])

		// Submit receipt for reward
		vm.distributor.SubmitReceipt(receipt)

		// Update provider stats
		if provider, ok := vm.providers[result.ProviderID]; ok {
			provider.TasksHandled++
			provider.Status.CurrentTasks--
			vm.verifier.RecordJobCompletion(result.ProviderID, task.ID)
		}
	}

	return nil
}

// GetPendingTasks returns pending tasks for a provider
func (vm *VM) GetPendingTasks(providerID string) []*Task {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	var pending []*Task
	for _, task := range vm.tasks {
		if task.Status == TaskStatusPending {
			pending = append(pending, task)
		}
	}
	return pending
}

// GetProviders returns all registered providers
func (vm *VM) GetProviders() []*Provider {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	providers := make([]*Provider, 0, len(vm.providers))
	for _, p := range vm.providers {
		providers = append(providers, p)
	}
	return providers
}

// GetModels returns all available models
func (vm *VM) GetModels() []*ModelInfo {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	models := make([]*ModelInfo, 0, len(vm.models))
	for _, m := range vm.models {
		models = append(models, m)
	}
	return models
}

// GetStats returns VM statistics
func (vm *VM) GetStats() map[string]interface{} {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	var pending, completed, failed int
	for _, task := range vm.tasks {
		switch task.Status {
		case TaskStatusPending:
			pending++
		case TaskStatusCompleted:
			completed++
		case TaskStatusFailed:
			failed++
		}
	}

	activeProviders := 0
	for _, p := range vm.providers {
		if p.Status.Online && time.Since(p.Status.LastSeen) < time.Hour {
			activeProviders++
		}
	}

	epochStats := vm.distributor.GetEpochStats()

	return map[string]interface{}{
		"tasks_pending":      pending,
		"tasks_completed":    completed,
		"tasks_failed":       failed,
		"providers_total":    len(vm.providers),
		"providers_active":   activeProviders,
		"models_available":   len(vm.models),
		"total_minted":       epochStats["total_minted"],
		"epoch_rewards":      epochStats["epoch_rewards"],
		"receipts_processed": epochStats["total_receipts"],
	}
}

// GetRewardStats returns reward statistics for a provider
func (vm *VM) GetRewardStats(providerID string) (map[string]interface{}, error) {
	stats, ok := vm.distributor.GetProviderStats(providerID)
	if !ok {
		return nil, errors.New("provider not found")
	}

	pending := vm.distributor.GetPendingRewards(providerID)

	return map[string]interface{}{
		"provider_id":     stats.ProviderID,
		"tasks_completed": stats.TasksCompleted,
		"total_rewards":   stats.TotalRewards.String(),
		"pending_rewards": pending.String(),
		"uptime":          stats.Uptime,
		"avg_latency_ms":  stats.AvgLatency,
		"slashed":         stats.Slashed,
	}, nil
}

// ClaimRewards claims pending rewards for a provider
func (vm *VM) ClaimRewards(providerID string) (string, error) {
	claimed, err := vm.distributor.ClaimRewards(providerID)
	if err != nil {
		return "", err
	}
	return claimed.String(), nil
}

// GetMerkleRoot returns merkle root of all receipts for Q-Chain anchoring
func (vm *VM) GetMerkleRoot() [32]byte {
	return vm.distributor.ComputeMerkleRoot()
}

// processTaskQueue processes tasks from queue
func (vm *VM) processTaskQueue(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-vm.taskQueue:
			if !ok {
				return
			}
			vm.assignTask(task)
		}
	}
}

// processResultQueue processes results from queue
func (vm *VM) processResultQueue(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case result, ok := <-vm.resultQueue:
			if !ok {
				return
			}
			vm.SubmitResult(result)
		}
	}
}

// assignTask assigns a task to a provider
func (vm *VM) assignTask(task *Task) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	// Find best available provider
	var bestProvider *Provider
	for _, p := range vm.providers {
		if !p.Status.Online || p.Status.CurrentTasks >= p.Status.MaxTasks {
			continue
		}
		if bestProvider == nil || p.Reputation > bestProvider.Reputation {
			bestProvider = p
		}
	}

	if bestProvider != nil {
		task.Status = TaskStatusAssigned
		task.AssignedTo = bestProvider.ID
		task.StartedAt = time.Now()
		bestProvider.Status.CurrentTasks++
	}
}

func (vm *VM) getModelHash(modelID string) [32]byte {
	if model, ok := vm.models[modelID]; ok {
		return model.Hash
	}
	return sha256.Sum256([]byte(modelID))
}
