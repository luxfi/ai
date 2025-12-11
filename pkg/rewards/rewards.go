// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rewards

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"math/big"
	"sync"
	"time"
)

var (
	ErrInvalidReceipt    = errors.New("invalid receipt")
	ErrReceiptExists     = errors.New("receipt already exists")
	ErrInsufficientProof = errors.New("insufficient proof")
	ErrSlashed           = errors.New("provider slashed")
)

// Receipt represents an AI task completion receipt
type Receipt struct {
	JobID       string    `json:"job_id"`
	ProviderID  string    `json:"provider_id"`
	ModelHash   [32]byte  `json:"model_hash"`
	InputHash   [32]byte  `json:"input_hash"`
	OutputHash  [32]byte  `json:"output_hash"`
	ComputeTime uint64    `json:"compute_time_ms"`
	GPUModel    string    `json:"gpu_model"`
	Timestamp   time.Time `json:"timestamp"`
	Proof       []byte    `json:"proof"`
	Signature   []byte    `json:"signature"`
}

// ReceiptHash computes the hash of a receipt for verification
func (r *Receipt) Hash() [32]byte {
	h := sha256.New()
	h.Write([]byte(r.JobID))
	h.Write([]byte(r.ProviderID))
	h.Write(r.ModelHash[:])
	h.Write(r.InputHash[:])
	h.Write(r.OutputHash[:])
	var hash [32]byte
	copy(hash[:], h.Sum(nil))
	return hash
}

// RewardCalculator calculates mining rewards
type RewardCalculator struct {
	baseReward       *big.Int // Base reward per task
	uptimeBonus      float64  // 10% bonus for 99.9% uptime
	speedBonus       float64  // 5% bonus for sub-100ms latency
	complexityFactor float64  // Multiplier based on model complexity
}

// NewRewardCalculator creates a new reward calculator
func NewRewardCalculator() *RewardCalculator {
	return &RewardCalculator{
		baseReward:       big.NewInt(1e15), // 0.001 AI coin per task
		uptimeBonus:      0.10,
		speedBonus:       0.05,
		complexityFactor: 1.0,
	}
}

// CalculateReward calculates reward for a completed task
func (rc *RewardCalculator) CalculateReward(receipt *Receipt, providerStats *ProviderStats) *big.Int {
	reward := new(big.Int).Set(rc.baseReward)

	// Model complexity multiplier
	complexityMultiplier := rc.getModelComplexity(receipt.ModelHash)
	reward.Mul(reward, big.NewInt(int64(complexityMultiplier*100)))
	reward.Div(reward, big.NewInt(100))

	// Compute time factor (more compute = more reward)
	computeFactor := rc.getComputeFactor(receipt.ComputeTime)
	reward.Mul(reward, big.NewInt(int64(computeFactor*100)))
	reward.Div(reward, big.NewInt(100))

	// GPU tier bonus
	gpuBonus := rc.getGPUBonus(receipt.GPUModel)
	bonusAmount := new(big.Int).Mul(reward, big.NewInt(int64(gpuBonus*100)))
	bonusAmount.Div(bonusAmount, big.NewInt(100))
	reward.Add(reward, bonusAmount)

	// Uptime bonus
	if providerStats != nil && providerStats.Uptime >= 0.999 {
		uptimeAmount := new(big.Int).Mul(reward, big.NewInt(int64(rc.uptimeBonus*100)))
		uptimeAmount.Div(uptimeAmount, big.NewInt(100))
		reward.Add(reward, uptimeAmount)
	}

	// Speed bonus (sub-100ms)
	if receipt.ComputeTime < 100 {
		speedAmount := new(big.Int).Mul(reward, big.NewInt(int64(rc.speedBonus*100)))
		speedAmount.Div(speedAmount, big.NewInt(100))
		reward.Add(reward, speedAmount)
	}

	return reward
}

func (rc *RewardCalculator) getModelComplexity(modelHash [32]byte) float64 {
	// Default complexity, would be looked up from model registry
	return 1.0
}

func (rc *RewardCalculator) getComputeFactor(computeTimeMs uint64) float64 {
	// Scale factor based on compute time
	if computeTimeMs < 100 {
		return 1.0
	} else if computeTimeMs < 1000 {
		return 1.5
	} else if computeTimeMs < 10000 {
		return 2.0
	}
	return 3.0
}

func (rc *RewardCalculator) getGPUBonus(gpuModel string) float64 {
	switch gpuModel {
	case "GB200", "B200": // Blackwell
		return 0.20
	case "H200", "H100": // Hopper
		return 0.15
	case "A100":
		return 0.10
	case "RTX 4090":
		return 0.05
	default:
		return 0.0
	}
}

// ProviderStats tracks provider statistics
type ProviderStats struct {
	ProviderID     string    `json:"provider_id"`
	TasksCompleted uint64    `json:"tasks_completed"`
	TotalRewards   *big.Int  `json:"total_rewards"`
	Uptime         float64   `json:"uptime"`
	AvgLatency     uint64    `json:"avg_latency_ms"`
	FailureRate    float64   `json:"failure_rate"`
	LastSeen       time.Time `json:"last_seen"`
	Slashed        bool      `json:"slashed"`
	SlashedAmount  *big.Int  `json:"slashed_amount"`
}

// RewardDistributor manages reward distribution
type RewardDistributor struct {
	mu             sync.RWMutex
	calculator     *RewardCalculator
	providers      map[string]*ProviderStats
	receipts       map[string]*Receipt
	pendingRewards map[string]*big.Int
	totalMinted    *big.Int
	epochRewards   *big.Int
}

// NewRewardDistributor creates a new reward distributor
func NewRewardDistributor() *RewardDistributor {
	return &RewardDistributor{
		calculator:     NewRewardCalculator(),
		providers:      make(map[string]*ProviderStats),
		receipts:       make(map[string]*Receipt),
		pendingRewards: make(map[string]*big.Int),
		totalMinted:    big.NewInt(0),
		epochRewards:   big.NewInt(0),
	}
}

// SubmitReceipt submits a task completion receipt
func (rd *RewardDistributor) SubmitReceipt(receipt *Receipt) (*big.Int, error) {
	if receipt == nil || receipt.JobID == "" {
		return nil, ErrInvalidReceipt
	}

	rd.mu.Lock()
	defer rd.mu.Unlock()

	// Check if receipt already exists
	if _, exists := rd.receipts[receipt.JobID]; exists {
		return nil, ErrReceiptExists
	}

	// Get or create provider stats
	stats, ok := rd.providers[receipt.ProviderID]
	if !ok {
		stats = &ProviderStats{
			ProviderID:   receipt.ProviderID,
			TotalRewards: big.NewInt(0),
			Uptime:       1.0,
		}
		rd.providers[receipt.ProviderID] = stats
	}

	// Check if provider is slashed
	if stats.Slashed {
		return nil, ErrSlashed
	}

	// Verify proof (simplified - would use ZK verification in production)
	if len(receipt.Proof) < 32 {
		return nil, ErrInsufficientProof
	}

	// Calculate reward
	reward := rd.calculator.CalculateReward(receipt, stats)

	// Update provider stats
	stats.TasksCompleted++
	stats.TotalRewards.Add(stats.TotalRewards, reward)
	stats.LastSeen = time.Now()
	stats.AvgLatency = (stats.AvgLatency*(stats.TasksCompleted-1) + receipt.ComputeTime) / stats.TasksCompleted

	// Record receipt
	rd.receipts[receipt.JobID] = receipt

	// Add to pending rewards
	if _, ok := rd.pendingRewards[receipt.ProviderID]; !ok {
		rd.pendingRewards[receipt.ProviderID] = big.NewInt(0)
	}
	rd.pendingRewards[receipt.ProviderID].Add(rd.pendingRewards[receipt.ProviderID], reward)

	// Update totals
	rd.totalMinted.Add(rd.totalMinted, reward)
	rd.epochRewards.Add(rd.epochRewards, reward)

	return reward, nil
}

// ClaimRewards claims pending rewards for a provider
func (rd *RewardDistributor) ClaimRewards(providerID string) (*big.Int, error) {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	pending, ok := rd.pendingRewards[providerID]
	if !ok || pending.Cmp(big.NewInt(0)) == 0 {
		return big.NewInt(0), nil
	}

	claimed := new(big.Int).Set(pending)
	rd.pendingRewards[providerID] = big.NewInt(0)

	return claimed, nil
}

// SlashProvider slashes a provider for invalid attestation
func (rd *RewardDistributor) SlashProvider(providerID string, reason string) error {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	stats, ok := rd.providers[providerID]
	if !ok {
		return errors.New("provider not found")
	}

	stats.Slashed = true

	// Slash 100% of pending rewards
	if pending, ok := rd.pendingRewards[providerID]; ok {
		stats.SlashedAmount = new(big.Int).Set(pending)
		rd.pendingRewards[providerID] = big.NewInt(0)
	}

	return nil
}

// GetProviderStats returns provider statistics
func (rd *RewardDistributor) GetProviderStats(providerID string) (*ProviderStats, bool) {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	stats, ok := rd.providers[providerID]
	return stats, ok
}

// GetPendingRewards returns pending rewards for a provider
func (rd *RewardDistributor) GetPendingRewards(providerID string) *big.Int {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	if pending, ok := rd.pendingRewards[providerID]; ok {
		return new(big.Int).Set(pending)
	}
	return big.NewInt(0)
}

// GetTotalMinted returns total AI coins minted
func (rd *RewardDistributor) GetTotalMinted() *big.Int {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	return new(big.Int).Set(rd.totalMinted)
}

// GetEpochStats returns current epoch statistics
func (rd *RewardDistributor) GetEpochStats() map[string]interface{} {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	activeProviders := 0
	for _, stats := range rd.providers {
		if time.Since(stats.LastSeen) < time.Hour && !stats.Slashed {
			activeProviders++
		}
	}

	return map[string]interface{}{
		"total_minted":     rd.totalMinted.String(),
		"epoch_rewards":    rd.epochRewards.String(),
		"total_receipts":   len(rd.receipts),
		"total_providers":  len(rd.providers),
		"active_providers": activeProviders,
	}
}

// ResetEpoch resets epoch rewards counter
func (rd *RewardDistributor) ResetEpoch() {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	rd.epochRewards = big.NewInt(0)
}

// ExportReceipts exports all receipts for anchoring to Q-Chain
func (rd *RewardDistributor) ExportReceipts() ([]byte, error) {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	receipts := make([]*Receipt, 0, len(rd.receipts))
	for _, r := range rd.receipts {
		receipts = append(receipts, r)
	}

	return json.Marshal(receipts)
}

// ComputeMerkleRoot computes merkle root of all receipts
func (rd *RewardDistributor) ComputeMerkleRoot() [32]byte {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	hashes := make([][32]byte, 0, len(rd.receipts))
	for _, r := range rd.receipts {
		hashes = append(hashes, r.Hash())
	}

	if len(hashes) == 0 {
		return [32]byte{}
	}

	// Build merkle tree (simplified)
	for len(hashes) > 1 {
		var newHashes [][32]byte
		for i := 0; i < len(hashes); i += 2 {
			h := sha256.New()
			h.Write(hashes[i][:])
			if i+1 < len(hashes) {
				h.Write(hashes[i+1][:])
			} else {
				h.Write(hashes[i][:])
			}
			var newHash [32]byte
			copy(newHash[:], h.Sum(nil))
			newHashes = append(newHashes, newHash)
		}
		hashes = newHashes
	}

	return hashes[0]
}
