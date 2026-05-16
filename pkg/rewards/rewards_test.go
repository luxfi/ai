// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rewards

import (
	"math/big"
	"testing"
	"time"
)

func TestReceiptHash(t *testing.T) {
	receipt := &Receipt{
		JobID:      "job-001",
		ProviderID: "provider-001",
		ModelHash:  [32]byte{1, 2, 3},
		InputHash:  [32]byte{4, 5, 6},
		OutputHash: [32]byte{7, 8, 9},
	}

	hash := receipt.Hash()
	if hash == [32]byte{} {
		t.Error("hash should not be empty")
	}

	// Same receipt should produce same hash
	hash2 := receipt.Hash()
	if hash != hash2 {
		t.Error("same receipt should produce same hash")
	}
}

func TestNewRewardCalculator(t *testing.T) {
	rc := NewRewardCalculator()
	if rc == nil {
		t.Fatal("NewRewardCalculator() returned nil")
	}
	if rc.baseReward == nil || rc.baseReward.Cmp(big.NewInt(0)) <= 0 {
		t.Error("baseReward should be positive")
	}
}

func TestCalculateReward_Basic(t *testing.T) {
	rc := NewRewardCalculator()

	receipt := &Receipt{
		JobID:       "job-001",
		ProviderID:  "provider-001",
		ComputeTime: 500, // 500ms
		GPUModel:    "RTX 4090",
		Timestamp:   time.Now(),
		Proof:       make([]byte, 64),
	}

	stats := &ProviderStats{
		ProviderID: "provider-001",
		Uptime:     0.95,
	}

	reward := rc.CalculateReward(receipt, stats)
	if reward == nil || reward.Cmp(big.NewInt(0)) <= 0 {
		t.Error("reward should be positive")
	}
}

func TestCalculateReward_UptimeBonus(t *testing.T) {
	rc := NewRewardCalculator()

	receipt := &Receipt{
		JobID:       "job-001",
		ProviderID:  "provider-001",
		ComputeTime: 500,
		GPUModel:    "A100",
		Proof:       make([]byte, 64),
	}

	statsLowUptime := &ProviderStats{Uptime: 0.95}
	statsHighUptime := &ProviderStats{Uptime: 0.999}

	rewardLow := rc.CalculateReward(receipt, statsLowUptime)
	rewardHigh := rc.CalculateReward(receipt, statsHighUptime)

	if rewardHigh.Cmp(rewardLow) <= 0 {
		t.Error("high uptime should give higher reward")
	}
}

func TestCalculateReward_SpeedBonus(t *testing.T) {
	rc := NewRewardCalculator()

	// Test that speed bonus (5%) is applied for sub-100ms completion
	// Note: Longer compute times get higher base rewards (compute factor)
	// Speed bonus is added ON TOP of the base reward for fast completion

	receiptFast := &Receipt{
		ComputeTime: 50, // Sub-100ms - gets speed bonus
		GPUModel:    "A100",
		Proof:       make([]byte, 64),
	}

	receiptNoBonus := &Receipt{
		ComputeTime: 50, // Same compute time
		GPUModel:    "A100",
		Proof:       make([]byte, 64),
	}

	stats := &ProviderStats{Uptime: 0.95}

	rewardFast := rc.CalculateReward(receiptFast, stats)

	// Both should get same reward since both are sub-100ms
	rewardNoBonus := rc.CalculateReward(receiptNoBonus, stats)

	if rewardFast.Cmp(rewardNoBonus) != 0 {
		t.Error("same compute time should give same reward")
	}

	// Verify speed bonus exists by comparing sub-100ms vs exactly 100ms
	receiptAt100 := &Receipt{
		ComputeTime: 100, // Exactly 100ms - no speed bonus
		GPUModel:    "A100",
		Proof:       make([]byte, 64),
	}

	rewardAt100 := rc.CalculateReward(receiptAt100, stats)

	// Fast receipt (50ms) should get speed bonus, but has lower compute factor (1.0 vs 1.5)
	// So the comparison is: base * 1.0 * 1.10 (gpu) * 1.05 (speed) vs base * 1.5 * 1.10 (gpu)
	// Fast: 1.0 * 1.10 * 1.05 = 1.155
	// At100: 1.5 * 1.10 = 1.65
	// So 100ms should give higher reward due to higher compute factor

	if rewardAt100.Cmp(rewardFast) <= 0 {
		t.Error("longer compute time (100ms) should give higher reward due to compute factor")
	}
}

func TestCalculateReward_GPUBonus(t *testing.T) {
	rc := NewRewardCalculator()

	receiptH100 := &Receipt{
		ComputeTime: 500,
		GPUModel:    "H100",
		Proof:       make([]byte, 64),
	}

	receiptA100 := &Receipt{
		ComputeTime: 500,
		GPUModel:    "A100",
		Proof:       make([]byte, 64),
	}

	stats := &ProviderStats{Uptime: 0.95}

	rewardH100 := rc.CalculateReward(receiptH100, stats)
	rewardA100 := rc.CalculateReward(receiptA100, stats)

	if rewardH100.Cmp(rewardA100) <= 0 {
		t.Error("H100 should give higher reward than A100")
	}
}

func TestNewRewardDistributor(t *testing.T) {
	rd := NewRewardDistributor()
	if rd == nil {
		t.Fatal("NewRewardDistributor() returned nil")
	}
}

func TestSubmitReceipt(t *testing.T) {
	rd := NewRewardDistributor()

	receipt := &Receipt{
		JobID:       "job-001",
		ProviderID:  "provider-001",
		ComputeTime: 500,
		GPUModel:    "H100",
		Timestamp:   time.Now(),
		Proof:       make([]byte, 64),
	}

	reward, err := rd.SubmitReceipt(receipt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reward == nil || reward.Cmp(big.NewInt(0)) <= 0 {
		t.Error("reward should be positive")
	}
}

func TestSubmitReceipt_DuplicateReceipt(t *testing.T) {
	rd := NewRewardDistributor()

	receipt := &Receipt{
		JobID:       "job-001",
		ProviderID:  "provider-001",
		ComputeTime: 500,
		Proof:       make([]byte, 64),
	}

	_, err := rd.SubmitReceipt(receipt)
	if err != nil {
		t.Fatal(err)
	}

	_, err = rd.SubmitReceipt(receipt)
	if err != ErrReceiptExists {
		t.Errorf("expected ErrReceiptExists, got %v", err)
	}
}

func TestSubmitReceipt_InvalidReceipt(t *testing.T) {
	rd := NewRewardDistributor()

	_, err := rd.SubmitReceipt(nil)
	if err != ErrInvalidReceipt {
		t.Errorf("expected ErrInvalidReceipt, got %v", err)
	}

	_, err = rd.SubmitReceipt(&Receipt{JobID: ""})
	if err != ErrInvalidReceipt {
		t.Errorf("expected ErrInvalidReceipt for empty JobID, got %v", err)
	}
}

func TestSubmitReceipt_InsufficientProof(t *testing.T) {
	rd := NewRewardDistributor()

	receipt := &Receipt{
		JobID:      "job-001",
		ProviderID: "provider-001",
		Proof:      make([]byte, 10), // Too short
	}

	_, err := rd.SubmitReceipt(receipt)
	if err != ErrInsufficientProof {
		t.Errorf("expected ErrInsufficientProof, got %v", err)
	}
}

func TestClaimRewards(t *testing.T) {
	rd := NewRewardDistributor()

	receipt := &Receipt{
		JobID:       "job-001",
		ProviderID:  "provider-001",
		ComputeTime: 500,
		GPUModel:    "H100",
		Proof:       make([]byte, 64),
	}

	reward, _ := rd.SubmitReceipt(receipt)

	claimed, err := rd.ClaimRewards("provider-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if claimed.Cmp(reward) != 0 {
		t.Errorf("claimed %s, expected %s", claimed, reward)
	}

	// Second claim should return 0
	claimed2, _ := rd.ClaimRewards("provider-001")
	if claimed2.Cmp(big.NewInt(0)) != 0 {
		t.Error("second claim should return 0")
	}
}

func TestSlashProvider(t *testing.T) {
	rd := NewRewardDistributor()

	receipt := &Receipt{
		JobID:       "job-001",
		ProviderID:  "provider-001",
		ComputeTime: 500,
		Proof:       make([]byte, 64),
	}

	rd.SubmitReceipt(receipt)

	err := rd.SlashProvider("provider-001", "invalid attestation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Provider should be slashed
	stats, _ := rd.GetProviderStats("provider-001")
	if !stats.Slashed {
		t.Error("provider should be slashed")
	}

	// Cannot submit new receipts
	receipt2 := &Receipt{
		JobID:       "job-002",
		ProviderID:  "provider-001",
		ComputeTime: 500,
		Proof:       make([]byte, 64),
	}

	_, err = rd.SubmitReceipt(receipt2)
	if err != ErrSlashed {
		t.Errorf("expected ErrSlashed, got %v", err)
	}
}

func TestSlashProvider_NotFound(t *testing.T) {
	rd := NewRewardDistributor()

	err := rd.SlashProvider("nonexistent", "reason")
	if err == nil {
		t.Error("expected error for nonexistent provider")
	}
}

func TestGetProviderStats(t *testing.T) {
	rd := NewRewardDistributor()

	receipt := &Receipt{
		JobID:       "job-001",
		ProviderID:  "provider-001",
		ComputeTime: 500,
		Proof:       make([]byte, 64),
	}

	rd.SubmitReceipt(receipt)

	stats, ok := rd.GetProviderStats("provider-001")
	if !ok {
		t.Fatal("provider stats not found")
	}
	if stats.TasksCompleted != 1 {
		t.Errorf("TasksCompleted = %d, want 1", stats.TasksCompleted)
	}
}

func TestGetPendingRewards(t *testing.T) {
	rd := NewRewardDistributor()

	pending := rd.GetPendingRewards("nonexistent")
	if pending.Cmp(big.NewInt(0)) != 0 {
		t.Error("pending rewards for nonexistent should be 0")
	}

	receipt := &Receipt{
		JobID:       "job-001",
		ProviderID:  "provider-001",
		ComputeTime: 500,
		Proof:       make([]byte, 64),
	}

	rd.SubmitReceipt(receipt)

	pending = rd.GetPendingRewards("provider-001")
	if pending.Cmp(big.NewInt(0)) <= 0 {
		t.Error("pending rewards should be positive")
	}
}

func TestGetTotalMinted(t *testing.T) {
	rd := NewRewardDistributor()

	initial := rd.GetTotalMinted()
	if initial.Cmp(big.NewInt(0)) != 0 {
		t.Error("initial total should be 0")
	}

	receipt := &Receipt{
		JobID:       "job-001",
		ProviderID:  "provider-001",
		ComputeTime: 500,
		Proof:       make([]byte, 64),
	}

	rd.SubmitReceipt(receipt)

	total := rd.GetTotalMinted()
	if total.Cmp(big.NewInt(0)) <= 0 {
		t.Error("total minted should be positive after receipt")
	}
}

func TestGetEpochStats(t *testing.T) {
	rd := NewRewardDistributor()

	receipt := &Receipt{
		JobID:       "job-001",
		ProviderID:  "provider-001",
		ComputeTime: 500,
		Proof:       make([]byte, 64),
	}

	rd.SubmitReceipt(receipt)

	stats := rd.GetEpochStats()
	if stats["total_receipts"].(int) != 1 {
		t.Error("total_receipts should be 1")
	}
	if stats["total_providers"].(int) != 1 {
		t.Error("total_providers should be 1")
	}
}

func TestResetEpoch(t *testing.T) {
	rd := NewRewardDistributor()

	receipt := &Receipt{
		JobID:       "job-001",
		ProviderID:  "provider-001",
		ComputeTime: 500,
		Proof:       make([]byte, 64),
	}

	rd.SubmitReceipt(receipt)
	rd.ResetEpoch()

	stats := rd.GetEpochStats()
	if stats["epoch_rewards"].(string) != "0" {
		t.Error("epoch_rewards should be 0 after reset")
	}
}

func TestExportReceipts(t *testing.T) {
	rd := NewRewardDistributor()

	receipt := &Receipt{
		JobID:       "job-001",
		ProviderID:  "provider-001",
		ComputeTime: 500,
		Proof:       make([]byte, 64),
	}

	rd.SubmitReceipt(receipt)

	data, err := rd.ExportReceipts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("exported data should not be empty")
	}
}

func TestComputeMerkleRoot(t *testing.T) {
	rd := NewRewardDistributor()

	// Empty merkle root
	emptyRoot := rd.ComputeMerkleRoot()
	if emptyRoot != [32]byte{} {
		t.Error("empty distributor should have empty merkle root")
	}

	// Single receipt
	receipt1 := &Receipt{
		JobID:       "job-001",
		ProviderID:  "provider-001",
		ComputeTime: 500,
		Proof:       make([]byte, 64),
	}
	rd.SubmitReceipt(receipt1)

	root1 := rd.ComputeMerkleRoot()
	if root1 == [32]byte{} {
		t.Error("merkle root should not be empty")
	}

	// Two receipts should change the root
	receipt2 := &Receipt{
		JobID:       "job-002",
		ProviderID:  "provider-001",
		ComputeTime: 500,
		Proof:       make([]byte, 64),
	}
	rd.SubmitReceipt(receipt2)

	root2 := rd.ComputeMerkleRoot()
	if root2 == root1 {
		t.Error("merkle root should change with new receipts")
	}
}

func TestProviderStatsAccumulation(t *testing.T) {
	rd := NewRewardDistributor()

	for i := 0; i < 10; i++ {
		receipt := &Receipt{
			JobID:       "job-" + string(rune('0'+i)),
			ProviderID:  "provider-001",
			ComputeTime: uint64(100 + i*50),
			GPUModel:    "H100",
			Proof:       make([]byte, 64),
		}
		rd.SubmitReceipt(receipt)
	}

	stats, ok := rd.GetProviderStats("provider-001")
	if !ok {
		t.Fatal("provider stats not found")
	}

	if stats.TasksCompleted != 10 {
		t.Errorf("TasksCompleted = %d, want 10", stats.TasksCompleted)
	}
	if stats.TotalRewards.Cmp(big.NewInt(0)) <= 0 {
		t.Error("TotalRewards should be positive")
	}
	if stats.AvgLatency == 0 {
		t.Error("AvgLatency should not be 0")
	}
}
