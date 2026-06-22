// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rewards

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
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

	// Node capability bonus: compute class + confidential-compute trust (integer %).
	pct := nodeCapabilityBonusPct(receipt.GPUModel)
	bonusAmount := new(big.Int).Mul(reward, big.NewInt(pct))
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

// Two orthogonal axes price a node: ComputeClass (raw throughput) and
// ConfidentialCompute (TEE trust). Both are deterministic keyword maps that MUST
// stay identical in the Rust port (see conformance/SPEC.md). These tiers are the
// durable structure; the integer percents are BOOTSTRAP tokenomics — once the HMM
// compute market is live they become a reserve/quality signal feeding the market,
// not the final price (the market prices fair value from supply/demand).

// ComputeClass ranks raw AI-compute capability of a device.
type ComputeClass int

const (
	ClassCpuOrUnknown      ComputeClass = iota // CPU-only / unrecognized — still earns a baseline
	ClassIntegratedOrEntry                     // integrated / entry GPU, base Apple Silicon, Intel Arc
	ClassConsumerDiscrete                      // mainstream discrete GPU / AI APU / Apple M*Max
	ClassProsumerHighEnd                       // RTX 5090/4090, Apple M*Ultra, DGX Spark/GB10 desktop appliance
	ClassWorkstationAi                         // RTX PRO 6000 Blackwell / 6000 Ada / A6000 (~7x a Spark)
	ClassDatacenter                            // A100 / L40S / A40 / Instinct MI250
	ClassPremiumDatacenter                     // H100 / H200 / GH200 / MI300
	ClassFrontierDatacenter                    // GB200 / B200
)

// BaseBonusPct is an INTEGER percent (no float, so no truncation artifacts).
func (c ComputeClass) BaseBonusPct() int64 {
	switch c {
	case ClassFrontierDatacenter:
		return 20
	case ClassPremiumDatacenter:
		return 15
	case ClassDatacenter:
		return 10
	case ClassWorkstationAi:
		return 8
	case ClassProsumerHighEnd:
		return 6
	case ClassConsumerDiscrete:
		return 4
	case ClassIntegratedOrEntry:
		return 2
	default:
		return 1 // CPU / unknown — every device earns something
	}
}

// ConfidentialCompute ranks trusted-execution capability — ORTHOGONAL to throughput.
// NVIDIA GPU confidential computing began with Hopper/H100; Blackwell strengthens
// attestation. A model string yields at most CapableGpuTee; runtime attestation
// (attested GPU TEE + IO path) upgrades to VerifiedGpuTee. DGX Spark/GB10 are NOT
// GPU-TEE class regardless of Blackwell branding.
type ConfidentialCompute int

const (
	CCNone           ConfidentialCompute = iota // no GPU TEE
	CCUnknown                                   // undetermined
	CCCpuTeeOnly                                // CPU TEE (SEV/TDX) but GPU unprotected
	CCCapableGpuTee                             // hardware family supports GPU TEE, not yet attested
	CCVerifiedGpuTee                            // attested CC mode with verified IO path
)

// CapabilityBonusPct is the trust premium (integer percent).
func (cc ConfidentialCompute) CapabilityBonusPct() int64 {
	switch cc {
	case CCVerifiedGpuTee:
		return 8
	case CCCapableGpuTee:
		return 4
	case CCCpuTeeOnly:
		return 1
	default:
		return 0 // None / Unknown
	}
}

func hasAny(m string, subs ...string) bool {
	for _, s := range subs {
		if strings.Contains(m, s) {
			return true
		}
	}
	return false
}

// ClassifyCompute maps any vendor/model string to a ComputeClass. Keep IDENTICAL in Rust.
func ClassifyCompute(model string) ComputeClass {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case hasAny(m, "gb200", "b200", "b100"):
		return ClassFrontierDatacenter
	case hasAny(m, "h100", "h200", "gh200", "mi300"):
		return ClassPremiumDatacenter
	case hasAny(m, "a100", "l40", "a40", "a30", "mi250", "mi210", "instinct"):
		return ClassDatacenter
	// RTX PRO 6000 Blackwell is ~7x a DGX Spark (GDDR7 ~1.8TB/s, far more FLOPs) -> workstation, above Spark.
	case hasAny(m, "rtx pro 6000", "rtx 6000", "6000 blackwell", "6000 ada", "a6000"):
		return ClassWorkstationAi
	case hasAny(m, "5090", "4090", "ultra", "gb10", "dgx spark", "spark"):
		return ClassProsumerHighEnd
	case hasAny(m, "rtx 40", "rtx 30", "4080", "3090", "3080", "radeon rx", "rx 7", "rx 9", "strix halo", "ryzen ai max", "evo x2", " max"):
		return ClassConsumerDiscrete
	case hasAny(m, "arc", "iris", "vega", "apple m", "m1", "m2", "m3", "m4", " pro", "radeon"):
		return ClassIntegratedOrEntry
	default:
		return ClassCpuOrUnknown
	}
}

// ClassifyConfidentialComputeByModel maps a model string to at most CapableGpuTee.
// DGX Spark/GB10 are explicitly excluded. Keep IDENTICAL in Rust.
func ClassifyConfidentialComputeByModel(model string) ConfidentialCompute {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case hasAny(m, "dgx spark", "gb10", "spark"): // local desktop AI — NOT GPU-TEE class
		return CCNone
	case hasAny(m, "gb200", "b200", "b100", "h100", "h200", "gh200"): // Hopper/Blackwell datacenter GPU CC
		return CCCapableGpuTee
	case hasAny(m, "rtx pro 6000 blackwell", "rtx pro 6000", "rtx 6000 blackwell"): // Blackwell workstation CC (exact, not generic rtx 6000)
		return CCCapableGpuTee
	default:
		return CCNone
	}
}

// nodeCapabilityBonusPct = raw compute class + confidential-compute trust (integer %).
// Runtime attestation can add the Verified delta on top of the model-derived value.
func nodeCapabilityBonusPct(gpuModel string) int64 {
	return ClassifyCompute(gpuModel).BaseBonusPct() + ClassifyConfidentialComputeByModel(gpuModel).CapabilityBonusPct()
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

// QuorumOutcome is the result of settling a quorum task through the reward
// distributor: who was paid, who was slashed, and the totals. It is the legacy
// in-memory mirror of the canonical A-Chain settlement (chains/aivm), kept so the
// reward ledger reflects quorum outcomes and the slash path is exercised in the
// same module that owns provider stats.
type QuorumOutcome struct {
	Reached      bool     // did the winning group reach threshold
	Paid         []string // winners credited rewardPerWinner
	Slashed      []string // withholders slashed
	TotalPaid    *big.Int
	TotalSlashed *big.Int
}

// SettleQuorum is the REAL settlement consequence on the reward ledger:
//   - rewards are credited to winners ONLY when the quorum is reached
//     (len(winners) >= threshold); a sub-threshold call pays NO ONE.
//   - every withholder (selected, committed, but never revealed) is slashed via
//     SlashProvider — the slash path is now driven by settlement, not left as a
//     never-triggered stub.
//
// rewardPerWinner is the per-winner credit; slashing zeroes a withholder's
// pending rewards and marks it slashed (SlashProvider's existing semantics).
// Idempotency / replay protection for the authoritative path lives in
// chains/aivm (the settled marker); this ledger mirror is additive and is called
// once per task by the settlement driver.
func (rd *RewardDistributor) SettleQuorum(winners, withholders []string, threshold int, rewardPerWinner *big.Int) QuorumOutcome {
	out := QuorumOutcome{TotalPaid: big.NewInt(0), TotalSlashed: big.NewInt(0)}

	// Slash withholders regardless of quorum outcome (they withheld either way).
	// SlashProvider takes its own lock, so call it BEFORE taking ours.
	for _, w := range withholders {
		before := rd.GetPendingRewards(w)
		if err := rd.SlashProvider(w, "withheld reveal in quorum task"); err == nil {
			out.Slashed = append(out.Slashed, w)
			out.TotalSlashed.Add(out.TotalSlashed, before)
		}
	}

	// Pay winners ONLY on quorum.
	if len(winners) < threshold {
		return out // no quorum -> no payment
	}
	out.Reached = true

	rd.mu.Lock()
	defer rd.mu.Unlock()
	for _, w := range winners {
		stats, ok := rd.providers[w]
		if !ok {
			stats = &ProviderStats{ProviderID: w, TotalRewards: big.NewInt(0), Uptime: 1.0}
			rd.providers[w] = stats
		}
		if stats.Slashed {
			continue // a slashed provider is never paid
		}
		stats.TotalRewards.Add(stats.TotalRewards, rewardPerWinner)
		stats.TasksCompleted++
		if _, ok := rd.pendingRewards[w]; !ok {
			rd.pendingRewards[w] = big.NewInt(0)
		}
		rd.pendingRewards[w].Add(rd.pendingRewards[w], rewardPerWinner)
		rd.totalMinted.Add(rd.totalMinted, rewardPerWinner)
		rd.epochRewards.Add(rd.epochRewards, rewardPerWinner)
		out.Paid = append(out.Paid, w)
		out.TotalPaid.Add(out.TotalPaid, rewardPerWinner)
	}
	return out
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
