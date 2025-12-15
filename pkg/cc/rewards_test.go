// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cc

import (
	"math/big"
	"testing"
	"time"
)

func TestModelingLevelString(t *testing.T) {
	tests := []struct {
		level    ModelingLevel
		expected string
	}{
		{ModelingLevelInferenceLight, "Inference-Light"},
		{ModelingLevelInferenceStandard, "Inference-Standard"},
		{ModelingLevelInferenceHeavy, "Inference-Heavy"},
		{ModelingLevelTraining, "Training"},
		{ModelingLevelSpecialized, "Specialized"},
		{ModelingLevel(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("ModelingLevel(%d).String() = %s, want %s", tt.level, got, tt.expected)
		}
	}
}

func TestModelingLevelMultipliers(t *testing.T) {
	// Verify multipliers increase with level
	levels := []ModelingLevel{
		ModelingLevelInferenceLight,
		ModelingLevelInferenceStandard,
		ModelingLevelInferenceHeavy,
		ModelingLevelTraining,
		ModelingLevelSpecialized,
	}

	prevMult := 0.0
	for _, level := range levels {
		mult := level.BaseRewardMultiplier()
		if mult <= prevMult {
			t.Errorf("Level %s multiplier %f not greater than previous %f",
				level, mult, prevMult)
		}
		prevMult = mult
	}
}

func TestBlockRewardSplit(t *testing.T) {
	// 100 LUX block reward
	totalReward := new(big.Int).Mul(big.NewInt(100), big.NewInt(1e18))

	validatorReward, aiPoolReward := CalculateBlockRewardSplit(totalReward)

	// Validator should get 90 LUX
	expectedValidator := new(big.Int).Mul(big.NewInt(90), big.NewInt(1e18))
	if validatorReward.Cmp(expectedValidator) != 0 {
		t.Errorf("Validator reward = %s, want %s", validatorReward, expectedValidator)
	}

	// AI pool should get 10 LUX
	expectedAI := new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18))
	if aiPoolReward.Cmp(expectedAI) != 0 {
		t.Errorf("AI pool reward = %s, want %s", aiPoolReward, expectedAI)
	}

	// Total should equal original
	total := new(big.Int).Add(validatorReward, aiPoolReward)
	if total.Cmp(totalReward) != 0 {
		t.Errorf("Total rewards = %s, want %s", total, totalReward)
	}
}

func TestAIProviderRewardWeight(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		provider *AIProvider
		minWeight float64
		maxWeight float64
	}{
		{
			name: "Tier1 with high stake",
			provider: &AIProvider{
				ProviderID: "test-1",
				Attestation: &TierAttestation{
					Tier:      Tier1GPUNativeCC,
					IssuedAt:  now.Add(-1 * time.Hour),
					ExpiresAt: now.Add(5 * time.Hour),
				},
				MaxModelingLevel: ModelingLevelSpecialized,
				StakeLUX:         100_000,
				ConsecutiveEpochs: 500,
				ReputationScore:   0.9,
			},
			minWeight: 10.0, // High tier, high level, high stake
			maxWeight: 100.0,
		},
		{
			name: "Tier4 with minimum stake",
			provider: &AIProvider{
				ProviderID: "test-2",
				Attestation: &TierAttestation{
					Tier:      Tier4Standard,
					IssuedAt:  now.Add(-1 * time.Hour),
					ExpiresAt: now.Add(29 * 24 * time.Hour),
				},
				MaxModelingLevel: ModelingLevelInferenceLight,
				StakeLUX:         1_000,
				ConsecutiveEpochs: 10,
				ReputationScore:   0.5,
			},
			minWeight: 0.1, // Low tier, low level, low stake
			maxWeight: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weight := tt.provider.RewardWeight()
			if weight < tt.minWeight || weight > tt.maxWeight {
				t.Errorf("RewardWeight() = %f, want between %f and %f",
					weight, tt.minWeight, tt.maxWeight)
			}
		})
	}
}

func TestAIRewardPoolRegistration(t *testing.T) {
	pool := NewAIRewardPool(1 * time.Hour)

	now := time.Now()
	provider := &AIProvider{
		ProviderID: "test-provider",
		Attestation: &TierAttestation{
			Tier:      Tier2ConfidentialVM,
			IssuedAt:  now.Add(-1 * time.Hour),
			ExpiresAt: now.Add(23 * time.Hour),
		},
		MaxModelingLevel: ModelingLevelInferenceStandard,
		StakeLUX:         50_000,
		LastHeartbeat:    now,
	}

	// Should register successfully
	err := pool.RegisterProvider(provider)
	if err != nil {
		t.Errorf("RegisterProvider() error = %v", err)
	}

	// Verify provider is in pool
	if _, exists := pool.Providers[provider.ProviderID]; !exists {
		t.Error("Provider not found in pool after registration")
	}

	// Should fail with insufficient stake
	lowStakeProvider := &AIProvider{
		ProviderID: "low-stake",
		StakeLUX:   100, // Below minimum
	}
	err = pool.RegisterProvider(lowStakeProvider)
	if err != ErrInsufficientStake {
		t.Errorf("RegisterProvider() with low stake error = %v, want %v", err, ErrInsufficientStake)
	}
}

func TestParticipationRewards(t *testing.T) {
	pool := NewAIRewardPool(1 * time.Hour)
	now := time.Now()

	// Add some providers
	providers := []*AIProvider{
		{
			ProviderID: "tier1-provider",
			Attestation: &TierAttestation{
				Tier:      Tier1GPUNativeCC,
				IssuedAt:  now.Add(-1 * time.Hour),
				ExpiresAt: now.Add(5 * time.Hour),
			},
			MaxModelingLevel: ModelingLevelInferenceHeavy,
			StakeLUX:         100_000,
			LastHeartbeat:    now,
			ReputationScore:  0.9,
		},
		{
			ProviderID: "tier2-provider",
			Attestation: &TierAttestation{
				Tier:      Tier2ConfidentialVM,
				IssuedAt:  now.Add(-1 * time.Hour),
				ExpiresAt: now.Add(23 * time.Hour),
			},
			MaxModelingLevel: ModelingLevelInferenceStandard,
			StakeLUX:         50_000,
			LastHeartbeat:    now,
			ReputationScore:  0.8,
		},
		{
			ProviderID: "tier4-provider",
			Attestation: &TierAttestation{
				Tier:      Tier4Standard,
				IssuedAt:  now.Add(-1 * time.Hour),
				ExpiresAt: now.Add(29 * 24 * time.Hour),
			},
			MaxModelingLevel: ModelingLevelInferenceLight,
			StakeLUX:         1_000,
			LastHeartbeat:    now,
			ReputationScore:  0.5,
		},
	}

	for _, p := range providers {
		pool.RegisterProvider(p)
	}

	// Set pool amount (10 LUX)
	pool.TotalPoolLUX = new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18))

	// Calculate participation rewards
	rewards := pool.CalculateParticipationRewards(5 * time.Minute)

	if len(rewards) != 3 {
		t.Errorf("Expected 3 rewards, got %d", len(rewards))
	}

	// Tier1 should get highest reward
	var tier1Reward, tier4Reward *big.Int
	for _, r := range rewards {
		if r.ProviderID == "tier1-provider" {
			tier1Reward = r.RewardLUX
		}
		if r.ProviderID == "tier4-provider" {
			tier4Reward = r.RewardLUX
		}
	}

	if tier1Reward == nil || tier4Reward == nil {
		t.Fatal("Missing rewards for tier1 or tier4 provider")
	}

	if tier1Reward.Cmp(tier4Reward) <= 0 {
		t.Errorf("Tier1 reward %s should be greater than Tier4 reward %s",
			tier1Reward, tier4Reward)
	}
}

func TestTaskReward(t *testing.T) {
	pool := NewAIRewardPool(1 * time.Hour)
	now := time.Now()

	provider := &AIProvider{
		ProviderID: "task-provider",
		Attestation: &TierAttestation{
			Tier:      Tier1GPUNativeCC,
			IssuedAt:  now.Add(-1 * time.Hour),
			ExpiresAt: now.Add(5 * time.Hour),
		},
		MaxModelingLevel: ModelingLevelInferenceHeavy,
		StakeLUX:         100_000,
	}

	// Calculate reward for 1000 compute units at Level 3
	reward := pool.CalculateTaskReward(
		provider,
		"task-123",
		ModelingLevelInferenceHeavy,
		1000,
	)

	if reward.RewardLUX.Cmp(big.NewInt(0)) <= 0 {
		t.Error("Task reward should be positive")
	}

	if reward.TaskID != "task-123" {
		t.Errorf("TaskID = %s, want task-123", reward.TaskID)
	}

	if reward.ComputeUnits != 1000 {
		t.Errorf("ComputeUnits = %d, want 1000", reward.ComputeUnits)
	}

	// Higher level should give higher reward
	lowLevelReward := pool.CalculateTaskReward(
		provider,
		"task-456",
		ModelingLevelInferenceLight,
		1000,
	)

	if reward.RewardLUX.Cmp(lowLevelReward.RewardLUX) <= 0 {
		t.Error("Higher modeling level should give higher reward")
	}
}

func TestRandomMiningEligibility(t *testing.T) {
	now := time.Now()
	maxAge := 5 * time.Minute

	tests := []struct {
		name     string
		provider *AIProvider
		eligible bool
		reason   string
	}{
		{
			name: "Eligible provider",
			provider: &AIProvider{
				ProviderID: "eligible",
				Attestation: &TierAttestation{
					Tier:      Tier2ConfidentialVM,
					IssuedAt:  now.Add(-1 * time.Hour),
					ExpiresAt: now.Add(23 * time.Hour),
				},
				StakeLUX:      50_000,
				LastHeartbeat: now,
			},
			eligible: true,
			reason:   "eligible",
		},
		{
			name:     "Nil provider",
			provider: nil,
			eligible: false,
			reason:   "provider is nil",
		},
		{
			name: "Offline provider",
			provider: &AIProvider{
				ProviderID: "offline",
				Attestation: &TierAttestation{
					Tier:      Tier2ConfidentialVM,
					IssuedAt:  now.Add(-1 * time.Hour),
					ExpiresAt: now.Add(23 * time.Hour),
				},
				StakeLUX:      50_000,
				LastHeartbeat: now.Add(-10 * time.Minute), // Too old
			},
			eligible: false,
			reason:   "provider offline",
		},
		{
			name: "No attestation",
			provider: &AIProvider{
				ProviderID:    "no-attest",
				Attestation:   nil,
				StakeLUX:      50_000,
				LastHeartbeat: now,
			},
			eligible: false,
			reason:   "no attestation",
		},
		{
			name: "Expired attestation",
			provider: &AIProvider{
				ProviderID: "expired",
				Attestation: &TierAttestation{
					Tier:      Tier2ConfidentialVM,
					IssuedAt:  now.Add(-25 * time.Hour),
					ExpiresAt: now.Add(-1 * time.Hour), // Expired
				},
				StakeLUX:      50_000,
				LastHeartbeat: now,
			},
			eligible: false,
			reason:   "attestation expired",
		},
		{
			name: "Insufficient stake",
			provider: &AIProvider{
				ProviderID: "low-stake",
				Attestation: &TierAttestation{
					Tier:      Tier1GPUNativeCC, // Requires 100k LUX
					IssuedAt:  now.Add(-1 * time.Hour),
					ExpiresAt: now.Add(5 * time.Hour),
				},
				StakeLUX:      10_000, // Only 10k
				LastHeartbeat: now,
			},
			eligible: false,
			reason:   "insufficient stake",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eligible, reason := RandomMiningEligibility(tt.provider, maxAge)
			if eligible != tt.eligible {
				t.Errorf("RandomMiningEligibility() eligible = %v, want %v", eligible, tt.eligible)
			}
			if reason != tt.reason {
				t.Errorf("RandomMiningEligibility() reason = %s, want %s", reason, tt.reason)
			}
		})
	}
}

func TestEpochRewardSummary(t *testing.T) {
	pool := NewAIRewardPool(1 * time.Hour)
	now := time.Now()

	// Add providers across tiers
	providers := []*AIProvider{
		{
			ProviderID: "t1",
			Attestation: &TierAttestation{
				Tier:      Tier1GPUNativeCC,
				IssuedAt:  now.Add(-1 * time.Hour),
				ExpiresAt: now.Add(5 * time.Hour),
			},
			MaxModelingLevel: ModelingLevelInferenceHeavy,
			StakeLUX:         100_000,
			LastHeartbeat:    now,
			ReputationScore:  0.9,
		},
		{
			ProviderID: "t2",
			Attestation: &TierAttestation{
				Tier:      Tier2ConfidentialVM,
				IssuedAt:  now.Add(-1 * time.Hour),
				ExpiresAt: now.Add(23 * time.Hour),
			},
			MaxModelingLevel: ModelingLevelInferenceStandard,
			StakeLUX:         50_000,
			LastHeartbeat:    now,
			ReputationScore:  0.8,
		},
	}

	for _, p := range providers {
		pool.RegisterProvider(p)
	}

	// 1000 LUX total block rewards
	totalRewards := new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18))

	summary := pool.CalculateEpochRewards(totalRewards, 5*time.Minute)

	// Validator should get 900 LUX
	expectedValidator := new(big.Int).Mul(big.NewInt(900), big.NewInt(1e18))
	if summary.ValidatorRewardsLUX.Cmp(expectedValidator) != 0 {
		t.Errorf("Validator rewards = %s, want %s", summary.ValidatorRewardsLUX, expectedValidator)
	}

	// AI pool should get 100 LUX
	expectedAI := new(big.Int).Mul(big.NewInt(100), big.NewInt(1e18))
	if summary.AIPoolRewardsLUX.Cmp(expectedAI) != 0 {
		t.Errorf("AI pool rewards = %s, want %s", summary.AIPoolRewardsLUX, expectedAI)
	}

	// Should have 2 online providers
	if summary.OnlineProviders != 2 {
		t.Errorf("OnlineProviders = %d, want 2", summary.OnlineProviders)
	}

	// Check tier distribution
	if summary.TierDistribution[Tier1GPUNativeCC] != 1 {
		t.Errorf("Tier1 count = %d, want 1", summary.TierDistribution[Tier1GPUNativeCC])
	}
	if summary.TierDistribution[Tier2ConfidentialVM] != 1 {
		t.Errorf("Tier2 count = %d, want 1", summary.TierDistribution[Tier2ConfidentialVM])
	}
}

func BenchmarkParticipationRewards(b *testing.B) {
	pool := NewAIRewardPool(1 * time.Hour)
	now := time.Now()

	// Add 100 providers
	for i := 0; i < 100; i++ {
		tier := CCTier((i % 4) + 1)
		pool.Providers[string(rune('A'+i))] = &AIProvider{
			ProviderID: string(rune('A' + i)),
			Attestation: &TierAttestation{
				Tier:      tier,
				IssuedAt:  now.Add(-1 * time.Hour),
				ExpiresAt: now.Add(tier.AttestationValidity()),
			},
			MaxModelingLevel:  ModelingLevel((i % 5) + 1),
			StakeLUX:          uint64((i + 1) * 10_000),
			LastHeartbeat:     now,
			ConsecutiveEpochs: uint64(i * 10),
			ReputationScore:   float64(i%10) / 10.0,
		}
	}

	pool.TotalPoolLUX = new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.CalculateParticipationRewards(5 * time.Minute)
	}
}

// ============================================================================
// Additional tests for coverage improvement
// ============================================================================

// TestMinVRAMGB tests all ModelingLevel.MinVRAMGB() cases
func TestMinVRAMGB(t *testing.T) {
	tests := []struct {
		level    ModelingLevel
		expected uint64
	}{
		{ModelingLevelInferenceLight, 8},
		{ModelingLevelInferenceStandard, 24},
		{ModelingLevelInferenceHeavy, 80},
		{ModelingLevelTraining, 48},
		{ModelingLevelSpecialized, 16},
		{ModelingLevel(0), 0},   // Invalid/unknown level
		{ModelingLevel(99), 0},  // Invalid level
		{ModelingLevel(255), 0}, // Max uint8 invalid level
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			got := tt.level.MinVRAMGB()
			if got != tt.expected {
				t.Errorf("ModelingLevel(%d).MinVRAMGB() = %d, want %d",
					tt.level, got, tt.expected)
			}
		})
	}

	// Verify ordering makes sense (training requires less than heavy inference)
	if ModelingLevelTraining.MinVRAMGB() >= ModelingLevelInferenceHeavy.MinVRAMGB() {
		t.Error("Training VRAM should be less than InferenceHeavy (which needs 80GB for 70B+ models)")
	}
}

// TestBaseRewardMultiplierDefault tests the default case for invalid levels
func TestBaseRewardMultiplierDefault(t *testing.T) {
	invalidLevels := []ModelingLevel{0, 6, 99, 255}

	for _, level := range invalidLevels {
		mult := level.BaseRewardMultiplier()
		if mult != 0.0 {
			t.Errorf("ModelingLevel(%d).BaseRewardMultiplier() = %f, want 0.0",
				level, mult)
		}
	}
}

// TestEffectiveTierAllCases tests all EffectiveTier code paths
func TestEffectiveTierAllCases(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		provider *AIProvider
		expected CCTier
	}{
		{
			name: "Valid Tier1 attestation",
			provider: &AIProvider{
				ProviderID: "t1",
				Attestation: &TierAttestation{
					Tier:      Tier1GPUNativeCC,
					IssuedAt:  now.Add(-1 * time.Hour),
					ExpiresAt: now.Add(5 * time.Hour),
				},
			},
			expected: Tier1GPUNativeCC,
		},
		{
			name: "Valid Tier2 attestation",
			provider: &AIProvider{
				ProviderID: "t2",
				Attestation: &TierAttestation{
					Tier:      Tier2ConfidentialVM,
					IssuedAt:  now.Add(-1 * time.Hour),
					ExpiresAt: now.Add(23 * time.Hour),
				},
			},
			expected: Tier2ConfidentialVM,
		},
		{
			name: "Valid Tier3 attestation",
			provider: &AIProvider{
				ProviderID: "t3",
				Attestation: &TierAttestation{
					Tier:      Tier3DeviceTEE,
					IssuedAt:  now.Add(-1 * time.Hour),
					ExpiresAt: now.Add(6 * 24 * time.Hour),
				},
			},
			expected: Tier3DeviceTEE,
		},
		{
			name: "Valid Tier4 attestation",
			provider: &AIProvider{
				ProviderID: "t4",
				Attestation: &TierAttestation{
					Tier:      Tier4Standard,
					IssuedAt:  now.Add(-1 * time.Hour),
					ExpiresAt: now.Add(29 * 24 * time.Hour),
				},
			},
			expected: Tier4Standard,
		},
		{
			name: "Nil attestation defaults to Tier4",
			provider: &AIProvider{
				ProviderID:  "nil-attest",
				Attestation: nil,
			},
			expected: Tier4Standard,
		},
		{
			name: "Expired attestation defaults to Tier4",
			provider: &AIProvider{
				ProviderID: "expired",
				Attestation: &TierAttestation{
					Tier:      Tier1GPUNativeCC,
					IssuedAt:  now.Add(-10 * time.Hour),
					ExpiresAt: now.Add(-1 * time.Hour), // Expired
				},
			},
			expected: Tier4Standard,
		},
		{
			name: "Future attestation (not yet valid) defaults to Tier4",
			provider: &AIProvider{
				ProviderID: "future",
				Attestation: &TierAttestation{
					Tier:      Tier1GPUNativeCC,
					IssuedAt:  now.Add(1 * time.Hour), // Issued in future
					ExpiresAt: now.Add(7 * time.Hour),
				},
			},
			expected: Tier4Standard,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.provider.EffectiveTier()
			if got != tt.expected {
				t.Errorf("EffectiveTier() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestSqrtEdgeCases tests sqrt helper function edge cases
func TestSqrtEdgeCases(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
		tolerance float64
	}{
		{0, 0, 0},
		{-1, 0, 0},          // Negative returns 0
		{-100, 0, 0},        // Large negative returns 0
		{1, 1, 0.0001},
		{4, 2, 0.0001},
		{9, 3, 0.0001},
		{16, 4, 0.0001},
		{100, 10, 0.0001},
		{2, 1.4142, 0.001},
		{0.25, 0.5, 0.0001},
		{10000, 100, 0.001},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := sqrt(tt.input)
			diff := got - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("sqrt(%f) = %f, want %f (tolerance %f)",
					tt.input, got, tt.expected, tt.tolerance)
			}
		})
	}
}

// TestMinEdgeCases tests min helper function edge cases
func TestMinEdgeCases(t *testing.T) {
	tests := []struct {
		a, b     float64
		expected float64
	}{
		{1, 2, 1},
		{2, 1, 1},
		{0, 0, 0},
		{-1, 1, -1},
		{1, -1, -1},
		{-5, -3, -5},
		{0.5, 0.25, 0.25},
		{1e10, 1e9, 1e9},
		{-1e10, 1e10, -1e10},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := min(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("min(%f, %f) = %f, want %f",
					tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

// TestRegisterProviderErrorCases tests RegisterProvider error paths
func TestRegisterProviderErrorCases(t *testing.T) {
	pool := NewAIRewardPool(1 * time.Hour)

	// Test empty provider ID
	emptyIDProvider := &AIProvider{
		ProviderID: "",
		StakeLUX:   100_000,
	}
	err := pool.RegisterProvider(emptyIDProvider)
	if err != ErrInvalidAttestation {
		t.Errorf("RegisterProvider with empty ID: got %v, want %v",
			err, ErrInvalidAttestation)
	}

	// Test insufficient stake for Tier4 minimum
	lowStakeProvider := &AIProvider{
		ProviderID: "low-stake",
		StakeLUX:   500, // Below Tier4 minimum (1000)
	}
	err = pool.RegisterProvider(lowStakeProvider)
	if err != ErrInsufficientStake {
		t.Errorf("RegisterProvider with low stake: got %v, want %v",
			err, ErrInsufficientStake)
	}

	// Test exact minimum stake succeeds
	minStakeProvider := &AIProvider{
		ProviderID: "min-stake",
		StakeLUX:   1_000, // Exactly Tier4 minimum
	}
	err = pool.RegisterProvider(minStakeProvider)
	if err != nil {
		t.Errorf("RegisterProvider with exact min stake: unexpected error %v", err)
	}

	// Test overwriting existing provider (should succeed)
	updatedProvider := &AIProvider{
		ProviderID: "min-stake",
		StakeLUX:   50_000, // Updated stake
	}
	err = pool.RegisterProvider(updatedProvider)
	if err != nil {
		t.Errorf("RegisterProvider overwrite: unexpected error %v", err)
	}
	if pool.Providers["min-stake"].StakeLUX != 50_000 {
		t.Error("Provider was not updated")
	}
}

// TestCalculateParticipationRewardsAllPaths tests all code paths
func TestCalculateParticipationRewardsAllPaths(t *testing.T) {
	now := time.Now()
	maxAge := 5 * time.Minute

	t.Run("Empty pool returns nil", func(t *testing.T) {
		pool := NewAIRewardPool(1 * time.Hour)
		pool.TotalPoolLUX = big.NewInt(1e18)
		rewards := pool.CalculateParticipationRewards(maxAge)
		if rewards != nil {
			t.Errorf("Expected nil for empty pool, got %v", rewards)
		}
	})

	t.Run("All providers offline returns nil", func(t *testing.T) {
		pool := NewAIRewardPool(1 * time.Hour)
		pool.TotalPoolLUX = big.NewInt(1e18)
		pool.Providers["offline"] = &AIProvider{
			ProviderID: "offline",
			Attestation: &TierAttestation{
				Tier:      Tier2ConfidentialVM,
				IssuedAt:  now.Add(-1 * time.Hour),
				ExpiresAt: now.Add(23 * time.Hour),
			},
			StakeLUX:      50_000,
			LastHeartbeat: now.Add(-10 * time.Minute), // Offline
		}
		rewards := pool.CalculateParticipationRewards(maxAge)
		if rewards != nil {
			t.Errorf("Expected nil for all offline, got %v", rewards)
		}
	})

	t.Run("Provider with nil attestation skipped", func(t *testing.T) {
		pool := NewAIRewardPool(1 * time.Hour)
		pool.TotalPoolLUX = big.NewInt(1e18)
		pool.Providers["nil-attest"] = &AIProvider{
			ProviderID:    "nil-attest",
			Attestation:   nil,
			StakeLUX:      50_000,
			LastHeartbeat: now,
		}
		rewards := pool.CalculateParticipationRewards(maxAge)
		if rewards != nil {
			t.Errorf("Expected nil for nil attestation, got %v", rewards)
		}
	})

	t.Run("Provider with invalid attestation skipped", func(t *testing.T) {
		pool := NewAIRewardPool(1 * time.Hour)
		pool.TotalPoolLUX = big.NewInt(1e18)
		pool.Providers["expired"] = &AIProvider{
			ProviderID: "expired",
			Attestation: &TierAttestation{
				Tier:      Tier2ConfidentialVM,
				IssuedAt:  now.Add(-25 * time.Hour),
				ExpiresAt: now.Add(-1 * time.Hour), // Expired
			},
			StakeLUX:      50_000,
			LastHeartbeat: now,
		}
		rewards := pool.CalculateParticipationRewards(maxAge)
		if rewards != nil {
			t.Errorf("Expected nil for expired attestation, got %v", rewards)
		}
	})

	t.Run("Mixed online/offline providers", func(t *testing.T) {
		pool := NewAIRewardPool(1 * time.Hour)
		pool.TotalPoolLUX = new(big.Int).Mul(big.NewInt(100), big.NewInt(1e18))

		// Online provider
		pool.Providers["online"] = &AIProvider{
			ProviderID: "online",
			Attestation: &TierAttestation{
				Tier:      Tier2ConfidentialVM,
				IssuedAt:  now.Add(-1 * time.Hour),
				ExpiresAt: now.Add(23 * time.Hour),
			},
			MaxModelingLevel: ModelingLevelInferenceStandard,
			StakeLUX:         50_000,
			LastHeartbeat:    now,
			ReputationScore:  0.8,
		}

		// Offline provider
		pool.Providers["offline"] = &AIProvider{
			ProviderID: "offline",
			Attestation: &TierAttestation{
				Tier:      Tier1GPUNativeCC,
				IssuedAt:  now.Add(-1 * time.Hour),
				ExpiresAt: now.Add(5 * time.Hour),
			},
			MaxModelingLevel: ModelingLevelSpecialized,
			StakeLUX:         100_000,
			LastHeartbeat:    now.Add(-10 * time.Minute), // Offline
			ReputationScore:  0.9,
		}

		rewards := pool.CalculateParticipationRewards(maxAge)
		if len(rewards) != 1 {
			t.Errorf("Expected 1 reward (only online), got %d", len(rewards))
		}
		if rewards[0].ProviderID != "online" {
			t.Errorf("Expected online provider, got %s", rewards[0].ProviderID)
		}
	})

	t.Run("Single provider gets entire participation pool", func(t *testing.T) {
		pool := NewAIRewardPool(1 * time.Hour)
		pool.TotalPoolLUX = new(big.Int).Mul(big.NewInt(100), big.NewInt(1e18))
		pool.ParticipationShare = 0.30

		pool.Providers["solo"] = &AIProvider{
			ProviderID: "solo",
			Attestation: &TierAttestation{
				Tier:      Tier2ConfidentialVM,
				IssuedAt:  now.Add(-1 * time.Hour),
				ExpiresAt: now.Add(23 * time.Hour),
			},
			MaxModelingLevel: ModelingLevelInferenceStandard,
			StakeLUX:         50_000,
			LastHeartbeat:    now,
			ReputationScore:  0.5,
		}

		rewards := pool.CalculateParticipationRewards(maxAge)
		if len(rewards) != 1 {
			t.Fatalf("Expected 1 reward, got %d", len(rewards))
		}

		// Single provider should get 100% weight share
		if rewards[0].WeightShare != 1.0 {
			t.Errorf("Single provider WeightShare = %f, want 1.0", rewards[0].WeightShare)
		}
	})
}

// TestRewardWeightEdgeCases tests RewardWeight calculation edge cases
func TestRewardWeightEdgeCases(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		provider    *AIProvider
		description string
	}{
		{
			name: "Zero stake",
			provider: &AIProvider{
				ProviderID: "zero-stake",
				Attestation: &TierAttestation{
					Tier:      Tier4Standard,
					IssuedAt:  now.Add(-1 * time.Hour),
					ExpiresAt: now.Add(29 * 24 * time.Hour),
				},
				MaxModelingLevel:  ModelingLevelInferenceLight,
				StakeLUX:          0, // Zero stake
				ConsecutiveEpochs: 0,
				ReputationScore:   0.5,
			},
			description: "StakeWeight should be 1.0 when stake <= 1000",
		},
		{
			name: "Stake exactly 1000",
			provider: &AIProvider{
				ProviderID: "exact-1000",
				Attestation: &TierAttestation{
					Tier:      Tier4Standard,
					IssuedAt:  now.Add(-1 * time.Hour),
					ExpiresAt: now.Add(29 * 24 * time.Hour),
				},
				MaxModelingLevel:  ModelingLevelInferenceLight,
				StakeLUX:          1000,
				ConsecutiveEpochs: 0,
				ReputationScore:   0.5,
			},
			description: "StakeWeight should be 1.0 when stake == 1000",
		},
		{
			name: "Stake just over 1000",
			provider: &AIProvider{
				ProviderID: "over-1000",
				Attestation: &TierAttestation{
					Tier:      Tier4Standard,
					IssuedAt:  now.Add(-1 * time.Hour),
					ExpiresAt: now.Add(29 * 24 * time.Hour),
				},
				MaxModelingLevel:  ModelingLevelInferenceLight,
				StakeLUX:          1001,
				ConsecutiveEpochs: 0,
				ReputationScore:   0.5,
			},
			description: "StakeWeight should be > 1.0 when stake > 1000",
		},
		{
			name: "Very high stake (capped at 10x)",
			provider: &AIProvider{
				ProviderID: "high-stake",
				Attestation: &TierAttestation{
					Tier:      Tier1GPUNativeCC,
					IssuedAt:  now.Add(-1 * time.Hour),
					ExpiresAt: now.Add(5 * time.Hour),
				},
				MaxModelingLevel:  ModelingLevelSpecialized,
				StakeLUX:          100_000_000, // 100M stake
				ConsecutiveEpochs: 0,
				ReputationScore:   0.5,
			},
			description: "StakeWeight should be capped at 10x",
		},
		{
			name: "Max consecutive epochs (1000+)",
			provider: &AIProvider{
				ProviderID: "veteran",
				Attestation: &TierAttestation{
					Tier:      Tier2ConfidentialVM,
					IssuedAt:  now.Add(-1 * time.Hour),
					ExpiresAt: now.Add(23 * time.Hour),
				},
				MaxModelingLevel:  ModelingLevelInferenceStandard,
				StakeLUX:          50_000,
				ConsecutiveEpochs: 2000, // Over 1000
				ReputationScore:   0.5,
			},
			description: "UptimeBonus should be capped at 1.5x",
		},
		{
			name: "Zero reputation",
			provider: &AIProvider{
				ProviderID: "new-provider",
				Attestation: &TierAttestation{
					Tier:      Tier3DeviceTEE,
					IssuedAt:  now.Add(-1 * time.Hour),
					ExpiresAt: now.Add(6 * 24 * time.Hour),
				},
				MaxModelingLevel:  ModelingLevelInferenceStandard,
				StakeLUX:          25_000,
				ConsecutiveEpochs: 0,
				ReputationScore:   0.0, // Zero reputation
			},
			description: "RepBonus should be 0.8x at zero reputation",
		},
		{
			name: "Max reputation",
			provider: &AIProvider{
				ProviderID: "trusted",
				Attestation: &TierAttestation{
					Tier:      Tier3DeviceTEE,
					IssuedAt:  now.Add(-1 * time.Hour),
					ExpiresAt: now.Add(6 * 24 * time.Hour),
				},
				MaxModelingLevel:  ModelingLevelInferenceStandard,
				StakeLUX:          25_000,
				ConsecutiveEpochs: 0,
				ReputationScore:   1.0, // Max reputation
			},
			description: "RepBonus should be 1.2x at max reputation",
		},
		{
			name: "Nil attestation falls back to Tier4",
			provider: &AIProvider{
				ProviderID:        "no-attest",
				Attestation:       nil,
				MaxModelingLevel:  ModelingLevelInferenceLight,
				StakeLUX:          5_000,
				ConsecutiveEpochs: 0,
				ReputationScore:   0.5,
			},
			description: "Should use Tier4 multiplier when no attestation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weight := tt.provider.RewardWeight()
			// Just verify it doesn't panic and returns a positive value
			if weight <= 0 {
				t.Errorf("RewardWeight() = %f, want positive value. %s",
					weight, tt.description)
			}
		})
	}

	// Verify stake cap works
	t.Run("Verify stake cap at 10x", func(t *testing.T) {
		highStake := &AIProvider{
			ProviderID: "high",
			Attestation: &TierAttestation{
				Tier:      Tier4Standard,
				IssuedAt:  now.Add(-1 * time.Hour),
				ExpiresAt: now.Add(29 * 24 * time.Hour),
			},
			MaxModelingLevel:  ModelingLevelInferenceLight,
			StakeLUX:          1_000_000_000, // 1 billion
			ConsecutiveEpochs: 0,
			ReputationScore:   0.5,
		}
		moderateStake := &AIProvider{
			ProviderID: "moderate",
			Attestation: &TierAttestation{
				Tier:      Tier4Standard,
				IssuedAt:  now.Add(-1 * time.Hour),
				ExpiresAt: now.Add(29 * 24 * time.Hour),
			},
			MaxModelingLevel:  ModelingLevelInferenceLight,
			StakeLUX:          100_000_000, // 100M (both should hit 10x cap)
			ConsecutiveEpochs: 0,
			ReputationScore:   0.5,
		}

		highWeight := highStake.RewardWeight()
		moderateWeight := moderateStake.RewardWeight()

		// Both should have same weight since stake is capped at 10x
		if highWeight != moderateWeight {
			t.Errorf("Stake cap not working: 1B stake weight=%f, 100M stake weight=%f",
				highWeight, moderateWeight)
		}
	})
}

// TestIsOnline tests the AIProvider.IsOnline method
func TestIsOnline(t *testing.T) {
	now := time.Now()
	maxAge := 5 * time.Minute

	tests := []struct {
		name          string
		lastHeartbeat time.Time
		expected      bool
	}{
		{"Just now", now, true},
		{"4 minutes ago", now.Add(-4 * time.Minute), true},
		{"Exactly 5 minutes ago", now.Add(-5 * time.Minute), false},
		{"6 minutes ago", now.Add(-6 * time.Minute), false},
		{"1 hour ago", now.Add(-1 * time.Hour), false},
		{"Future heartbeat", now.Add(1 * time.Minute), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &AIProvider{
				ProviderID:    "test",
				LastHeartbeat: tt.lastHeartbeat,
			}
			got := provider.IsOnline(maxAge)
			if got != tt.expected {
				t.Errorf("IsOnline() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestNewAIRewardPool verifies pool initialization
func TestNewAIRewardPool(t *testing.T) {
	duration := 2 * time.Hour
	pool := NewAIRewardPool(duration)

	if pool.EpochDuration != duration {
		t.Errorf("EpochDuration = %v, want %v", pool.EpochDuration, duration)
	}
	if pool.Providers == nil {
		t.Error("Providers map should be initialized")
	}
	if len(pool.Providers) != 0 {
		t.Errorf("Providers should be empty, got %d", len(pool.Providers))
	}
	if pool.TotalPoolLUX == nil || pool.TotalPoolLUX.Cmp(big.NewInt(0)) != 0 {
		t.Error("TotalPoolLUX should be initialized to 0")
	}
	if pool.ParticipationShare != 0.30 {
		t.Errorf("ParticipationShare = %f, want 0.30", pool.ParticipationShare)
	}
	if pool.TaskShare != 0.70 {
		t.Errorf("TaskShare = %f, want 0.70", pool.TaskShare)
	}
}
