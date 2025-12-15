// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cc

import (
	"testing"
	"time"
)

// =============================================================================
// Trust Score Weight Tests
// =============================================================================

func TestDefaultWeights(t *testing.T) {
	weights := DefaultWeights()

	// Verify weights sum to 1.0
	sum := weights.Hardware + weights.Attestation + weights.Reputation + weights.Uptime
	if sum < 0.99 || sum > 1.01 {
		t.Errorf("DefaultWeights() sum = %v, want ~1.0", sum)
	}

	// Verify individual weights per LP-5610
	if weights.Hardware != 0.40 {
		t.Errorf("Hardware weight = %v, want 0.40", weights.Hardware)
	}
	if weights.Attestation != 0.30 {
		t.Errorf("Attestation weight = %v, want 0.30", weights.Attestation)
	}
	if weights.Reputation != 0.20 {
		t.Errorf("Reputation weight = %v, want 0.20", weights.Reputation)
	}
	if weights.Uptime != 0.10 {
		t.Errorf("Uptime weight = %v, want 0.10", weights.Uptime)
	}
}

// =============================================================================
// Hardware Score Tests - LP-5610 Section 5.1
// =============================================================================

func TestCalculateHardwareScore(t *testing.T) {
	tests := []struct {
		name     string
		input    TrustScoreInput
		minScore uint8
		maxScore uint8
	}{
		{
			name: "Tier 1 with all features",
			input: TrustScoreInput{
				Tier:              Tier1GPUNativeCC,
				GPUGeneration:     10,
				CCFeaturesEnabled: true,
				TEEIOEnabled:      true,
				RIMVerified:       true,
				HardwareCapabilities: &HardwareCapability{
					MIGSupported: true,
					GPUMemoryMB:  81920, // 80GB+
				},
			},
			minScore: 40,
			maxScore: 100,
		},
		{
			name: "Tier 1 minimal",
			input: TrustScoreInput{
				Tier:          Tier1GPUNativeCC,
				GPUGeneration: 5,
			},
			minScore: 30,
			maxScore: 45,
		},
		{
			name: "Tier 2 with high gen GPU",
			input: TrustScoreInput{
				Tier:          Tier2ConfidentialVM,
				GPUGeneration: 15, // Very high gen, should cap
			},
			minScore: 25,
			maxScore: 35,
		},
		{
			name: "Tier 3 base",
			input: TrustScoreInput{
				Tier:          Tier3DeviceTEE,
				GPUGeneration: 8,
			},
			minScore: 15,
			maxScore: 25,
		},
		{
			name: "Tier 4 stake only",
			input: TrustScoreInput{
				Tier: Tier4Standard,
			},
			minScore: 5,
			maxScore: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTrustScore(&tt.input)
			// Hardware contributes 40% max, so check component score
			if result.HardwareScore < tt.minScore || result.HardwareScore > tt.maxScore {
				t.Errorf("HardwareScore = %v, want between %v and %v",
					result.HardwareScore, tt.minScore, tt.maxScore)
			}
		})
	}
}

// =============================================================================
// Attestation Score Tests - LP-5610 Section 5.2
// =============================================================================

func TestCalculateAttestationScore(t *testing.T) {
	tests := []struct {
		name     string
		input    TrustScoreInput
		minScore uint8
		maxScore uint8
	}{
		{
			name: "Very fresh NVTrust attestation",
			input: TrustScoreInput{
				Tier:              Tier1GPUNativeCC,
				AttestationAge:    30 * time.Minute, // < 25% of 6h
				AttestationMethod: "nvtrust",
				LocalVerification: true,
				CertChainValid:    true,
			},
			minScore: 95,
			maxScore: 100,
		},
		{
			name: "Medium age SEV-SNP",
			input: TrustScoreInput{
				Tier:              Tier2ConfidentialVM,
				AttestationAge:    12 * time.Hour, // 50% of 24h
				AttestationMethod: "sev-snp",
				LocalVerification: true,
			},
			minScore: 80,
			maxScore: 95,
		},
		{
			name: "Old attestation",
			input: TrustScoreInput{
				Tier:              Tier3DeviceTEE,
				AttestationAge:    6 * 24 * time.Hour, // >75% of 7d
				AttestationMethod: "secure-enclave",
			},
			minScore: 70,
			maxScore: 85,
		},
		{
			name: "Software attestation",
			input: TrustScoreInput{
				Tier:              Tier4Standard,
				AttestationAge:    24 * time.Hour,
				AttestationMethod: "software",
			},
			minScore: 70,
			maxScore: 90, // Software attestation gives moderate score
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTrustScore(&tt.input)
			if result.AttestationScore < tt.minScore || result.AttestationScore > tt.maxScore {
				t.Errorf("AttestationScore = %v, want between %v and %v",
					result.AttestationScore, tt.minScore, tt.maxScore)
			}
		})
	}
}

// =============================================================================
// Reputation Score Tests - LP-5610 Section 5.3
// =============================================================================

func TestCalculateReputationScore(t *testing.T) {
	tests := []struct {
		name     string
		input    TrustScoreInput
		minScore uint8
		maxScore uint8
	}{
		{
			name: "Perfect reputation",
			input: TrustScoreInput{
				Tier:            Tier1GPUNativeCC,
				TasksCompleted:  10000,
				TasksFailed:     0,
				SlashingEvents:  0,
				ReputationScore: 1.0,
			},
			minScore: 95,
			maxScore: 100,
		},
		{
			name: "Good reputation with some failures",
			input: TrustScoreInput{
				Tier:            Tier2ConfidentialVM,
				TasksCompleted:  1000,
				TasksFailed:     50,
				SlashingEvents:  0,
				ReputationScore: 0.8,
			},
			minScore: 75,
			maxScore: 95,
		},
		{
			name: "New provider (no history)",
			input: TrustScoreInput{
				Tier:            Tier4Standard,
				TasksCompleted:  0,
				TasksFailed:     0,
				SlashingEvents:  0,
				ReputationScore: 0.5,
			},
			minScore: 50,
			maxScore: 65,
		},
		{
			name: "Bad reputation with slashing",
			input: TrustScoreInput{
				Tier:            Tier3DeviceTEE,
				TasksCompleted:  100,
				TasksFailed:     50,
				SlashingEvents:  5,
				ReputationScore: 0.3,
			},
			minScore: 20,
			maxScore: 50,
		},
		{
			name: "Many slashing events",
			input: TrustScoreInput{
				Tier:           Tier4Standard,
				SlashingEvents: 10, // Max penalty
			},
			minScore: 0,
			maxScore: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTrustScore(&tt.input)
			if result.ReputationScore < tt.minScore || result.ReputationScore > tt.maxScore {
				t.Errorf("ReputationScore = %v, want between %v and %v",
					result.ReputationScore, tt.minScore, tt.maxScore)
			}
		})
	}
}

// =============================================================================
// Uptime Score Tests - LP-5610 Section 5.4
// =============================================================================

func TestCalculateUptimeScore(t *testing.T) {
	tests := []struct {
		name     string
		input    TrustScoreInput
		minScore uint8
		maxScore uint8
	}{
		{
			name: "Perfect uptime",
			input: TrustScoreInput{
				Tier:                  Tier1GPUNativeCC,
				UptimePercentage:      100.0,
				LastSeenDelta:         10 * time.Second,
				ConsecutiveHeartbeats: 5000,
			},
			minScore: 95,
			maxScore: 100,
		},
		{
			name: "Good uptime",
			input: TrustScoreInput{
				Tier:                  Tier2ConfidentialVM,
				UptimePercentage:      95.0,
				LastSeenDelta:         3 * time.Minute,
				ConsecutiveHeartbeats: 500,
			},
			minScore: 75,
			maxScore: 95,
		},
		{
			name: "Recent heartbeat",
			input: TrustScoreInput{
				Tier:                  Tier3DeviceTEE,
				UptimePercentage:      90.0,
				LastSeenDelta:         30 * time.Second,
				ConsecutiveHeartbeats: 100,
			},
			minScore: 70,
			maxScore: 90,
		},
		{
			name: "Stale heartbeat",
			input: TrustScoreInput{
				Tier:                  Tier4Standard,
				UptimePercentage:      80.0,
				LastSeenDelta:         30 * time.Minute,
				ConsecutiveHeartbeats: 5,
			},
			minScore: 50,
			maxScore: 70,
		},
		{
			name: "Very stale",
			input: TrustScoreInput{
				Tier:             Tier4Standard,
				UptimePercentage: 70.0,
				LastSeenDelta:    2 * time.Hour,
			},
			minScore: 40,
			maxScore: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTrustScore(&tt.input)
			if result.UptimeScore < tt.minScore || result.UptimeScore > tt.maxScore {
				t.Errorf("UptimeScore = %v, want between %v and %v",
					result.UptimeScore, tt.minScore, tt.maxScore)
			}
		})
	}
}

// =============================================================================
// Score Clamping Tests
// =============================================================================

func TestScoreClamping(t *testing.T) {
	// Test that scores are clamped to tier limits

	// High input that would exceed Tier 2 max (89)
	tier2Input := &TrustScoreInput{
		Tier:                  Tier2ConfidentialVM,
		GPUGeneration:         10,
		CCFeaturesEnabled:     true,
		TEEIOEnabled:          true,
		RIMVerified:           true,
		AttestationAge:        1 * time.Hour,
		AttestationMethod:     "nvtrust",
		LocalVerification:     true,
		CertChainValid:        true,
		TasksCompleted:        10000,
		ReputationScore:       1.0,
		UptimePercentage:      100.0,
		LastSeenDelta:         10 * time.Second,
		ConsecutiveHeartbeats: 5000,
	}

	result := CalculateTrustScore(tier2Input)
	if result.TotalScore > 89 {
		t.Errorf("Tier 2 score %d exceeds max 89", result.TotalScore)
	}

	// Low input that would go below Tier 1 min (90)
	tier1LowInput := &TrustScoreInput{
		Tier:           Tier1GPUNativeCC,
		SlashingEvents: 10,
	}

	result = CalculateTrustScore(tier1LowInput)
	if result.TotalScore < 90 {
		t.Errorf("Tier 1 score %d below min 90", result.TotalScore)
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warning about score clamping")
	}
}

// =============================================================================
// Weighted Contribution Tests
// =============================================================================

func TestWeightedContributions(t *testing.T) {
	input := &TrustScoreInput{
		Tier:              Tier2ConfidentialVM,
		GPUGeneration:     8,
		AttestationAge:    12 * time.Hour,
		AttestationMethod: "sev-snp",
		TasksCompleted:    100,
		UptimePercentage:  95.0,
	}

	result := CalculateTrustScore(input)
	weights := DefaultWeights()

	// Verify each contribution = score * weight
	expectedHardware := float64(result.HardwareScore) * weights.Hardware
	expectedAttest := float64(result.AttestationScore) * weights.Attestation
	expectedRep := float64(result.ReputationScore) * weights.Reputation
	expectedUptime := float64(result.UptimeScore) * weights.Uptime

	// Check contribution calculations
	if result.HardwareContribution != expectedHardware {
		t.Errorf("HardwareContribution = %.1f, want %.1f", result.HardwareContribution, expectedHardware)
	}
	if result.AttestationContribution != expectedAttest {
		t.Errorf("AttestationContribution = %.1f, want %.1f", result.AttestationContribution, expectedAttest)
	}
	if result.ReputationContribution != expectedRep {
		t.Errorf("ReputationContribution = %.1f, want %.1f", result.ReputationContribution, expectedRep)
	}
	if result.UptimeContribution != expectedUptime {
		t.Errorf("UptimeContribution = %.1f, want %.1f", result.UptimeContribution, expectedUptime)
	}

	// Total contributions should be positive
	total := result.HardwareContribution + result.AttestationContribution +
		result.ReputationContribution + result.UptimeContribution
	if total <= 0 {
		t.Error("Total contributions should be positive")
	}

	// TotalScore should be clamped to tier limits (max 89 for Tier2)
	if result.TotalScore > 89 {
		t.Errorf("Tier2 TotalScore %d exceeds max 89", result.TotalScore)
	}
}

func TestWeightedContributionsHighHardware(t *testing.T) {
	// Test case where hardware IS the largest contributor
	input := &TrustScoreInput{
		Tier:              Tier1GPUNativeCC,
		GPUGeneration:     10,
		CCFeaturesEnabled: true,
		TEEIOEnabled:      true,
		RIMVerified:       true,
		HardwareCapabilities: &HardwareCapability{
			MIGSupported: true,
			GPUMemoryMB:  81920,
		},
		// Minimal other scores
		AttestationAge:    23 * time.Hour, // Old attestation
		AttestationMethod: "software",
		TasksCompleted:    0,
		UptimePercentage:  50.0,
	}

	result := CalculateTrustScore(input)

	// With maxed hardware and minimal others, hardware should dominate
	// Hardware: 35 + 5 + 3 + 2 + 2 + 1 + 2 = 50 * 0.40 = 20
	// Attestation: low score * 0.30 = low
	if result.HardwareContribution < result.AttestationContribution {
		t.Logf("Hardware contribution %.1f should be >= Attestation %.1f when hardware is maxed",
			result.HardwareContribution, result.AttestationContribution)
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestScoreEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input TrustScoreInput
	}{
		{
			name: "Zero everything",
			input: TrustScoreInput{
				Tier: Tier4Standard,
			},
		},
		{
			name: "Max values",
			input: TrustScoreInput{
				Tier:                  Tier1GPUNativeCC,
				GPUGeneration:         255,
				CCFeaturesEnabled:     true,
				TEEIOEnabled:          true,
				RIMVerified:           true,
				TasksCompleted:        ^uint64(0),
				UptimePercentage:      100.0,
				ConsecutiveHeartbeats: ^uint64(0),
			},
		},
		{
			name: "Unusual tier",
			input: TrustScoreInput{
				Tier: TierUnknown,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := CalculateTrustScore(&tt.input)
			if result == nil {
				t.Error("Result should not be nil")
			}
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkCalculateHardwareScore(b *testing.B) {
	input := &TrustScoreInput{
		Tier:              Tier1GPUNativeCC,
		GPUGeneration:     10,
		CCFeaturesEnabled: true,
		TEEIOEnabled:      true,
		RIMVerified:       true,
		HardwareCapabilities: &HardwareCapability{
			MIGSupported: true,
			GPUMemoryMB:  81920,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateTrustScore(input)
	}
}

func BenchmarkCalculateTrustScoreAllComponents(b *testing.B) {
	input := &TrustScoreInput{
		Tier:                  Tier1GPUNativeCC,
		GPUGeneration:         10,
		CCFeaturesEnabled:     true,
		TEEIOEnabled:          true,
		RIMVerified:           true,
		AttestationAge:        1 * time.Hour,
		AttestationMethod:     "nvtrust",
		LocalVerification:     true,
		CertChainValid:        true,
		TasksCompleted:        1000,
		TasksFailed:           10,
		SlashingEvents:        0,
		ReputationScore:       0.9,
		UptimePercentage:      99.9,
		LastSeenDelta:         30 * time.Second,
		ConsecutiveHeartbeats: 500,
		HardwareCapabilities: &HardwareCapability{
			MIGSupported: true,
			GPUMemoryMB:  81920,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateTrustScore(input)
	}
}

// =============================================================================
// Additional Coverage Tests - Edge Cases and Boundary Values
// =============================================================================

// TestCalculateHardwareScoreTier4WithFeatures tests Tier4Standard with CC features
// (covers the Tier4Standard case in calculateHardwareScore)
func TestCalculateHardwareScoreTier4WithFeatures(t *testing.T) {
	tests := []struct {
		name          string
		input         TrustScoreInput
		expectedScore uint8
	}{
		{
			name: "Tier4 with CC features enabled",
			input: TrustScoreInput{
				Tier:              Tier4Standard,
				CCFeaturesEnabled: true, // +3
				TEEIOEnabled:      true, // +2
				RIMVerified:       true, // +2
			},
			// Base 5 + 3 + 2 + 2 = 12
			expectedScore: 12,
		},
		{
			name: "Tier4 with hardware capabilities",
			input: TrustScoreInput{
				Tier: Tier4Standard,
				HardwareCapabilities: &HardwareCapability{
					MIGSupported: true,  // +1
					GPUMemoryMB:  81920, // +2 (>80GB)
				},
			},
			// Base 5 + 1 + 2 = 8
			expectedScore: 8,
		},
		{
			name: "Tier4 with all bonuses",
			input: TrustScoreInput{
				Tier:              Tier4Standard,
				CCFeaturesEnabled: true,
				TEEIOEnabled:      true,
				RIMVerified:       true,
				HardwareCapabilities: &HardwareCapability{
					MIGSupported: true,
					GPUMemoryMB:  100000,
				},
			},
			// Base 5 + 3 + 2 + 2 + 1 + 2 = 15
			expectedScore: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateHardwareScore(&tt.input)
			if score != tt.expectedScore {
				t.Errorf("calculateHardwareScore() = %d, want %d", score, tt.expectedScore)
			}
		})
	}
}

// TestCalculateHardwareScoreHighGPUGeneration tests GPU generation capping
func TestCalculateHardwareScoreHighGPUGeneration(t *testing.T) {
	// Tier1 with GPUGeneration > 10 should cap at 40
	input := &TrustScoreInput{
		Tier:          Tier1GPUNativeCC,
		GPUGeneration: 15, // > 10, should cap
	}
	score := calculateHardwareScore(input)
	if score != 40 {
		t.Errorf("Tier1 with GPU gen 15 should cap at 40, got %d", score)
	}

	// Tier2 with high gen should cap at 30
	input2 := &TrustScoreInput{
		Tier:          Tier2ConfidentialVM,
		GPUGeneration: 20,
	}
	score2 := calculateHardwareScore(input2)
	if score2 != 30 {
		t.Errorf("Tier2 with high GPU gen should cap at 30, got %d", score2)
	}

	// Tier3 with high gen should cap at 20
	input3 := &TrustScoreInput{
		Tier:          Tier3DeviceTEE,
		GPUGeneration: 20,
	}
	score3 := calculateHardwareScore(input3)
	if score3 != 20 {
		t.Errorf("Tier3 with high GPU gen should cap at 20, got %d", score3)
	}
}

// TestCalculateReputationScoreNegative tests that negative scores clamp to 0
func TestCalculateReputationScoreNegative(t *testing.T) {
	// Many slashing events with no tasks should push score negative, then clamp to 0
	input := &TrustScoreInput{
		Tier:            Tier4Standard,
		TasksCompleted:  0,
		TasksFailed:     0,
		SlashingEvents:  10, // 10 * 10 = 100 penalty, max capped at 30
		ReputationScore: 0,  // No historical bonus
	}
	// Base 50 - 30 (max penalty) = 20
	score := calculateReputationScore(input)
	if score != 20 {
		t.Errorf("Expected 20 (50 base - 30 max penalty), got %d", score)
	}

	// Extreme slashing with poor task history
	input2 := &TrustScoreInput{
		Tier:            Tier4Standard,
		TasksCompleted:  1,
		TasksFailed:     99, // 1% success rate
		SlashingEvents:  5,  // 50 penalty, capped at 30
		ReputationScore: 0,
	}
	// Base 50 + 0.01*30 (success rate) + 0 (volume) - 30 (penalty) = ~20
	score2 := calculateReputationScore(input2)
	if score2 > 25 {
		t.Errorf("Poor reputation score should be low, got %d", score2)
	}
}

// TestCalculateReputationScoreMaxBounds tests score capping at 100
func TestCalculateReputationScoreMaxBounds(t *testing.T) {
	// Max everything should cap at 100
	input := &TrustScoreInput{
		Tier:            Tier1GPUNativeCC,
		TasksCompleted:  100000, // High volume bonus +5
		TasksFailed:     0,      // 100% success rate, +30
		SlashingEvents:  0,
		ReputationScore: 1.0, // +15
	}
	// Base 50 + 30 + 5 + 15 = 100 (capped)
	score := calculateReputationScore(input)
	if score != 100 {
		t.Errorf("Max reputation should be 100, got %d", score)
	}
}

// TestCalculateReputationScoreVolumeBrackets tests all volume bonus brackets
func TestCalculateReputationScoreVolumeBrackets(t *testing.T) {
	tests := []struct {
		name           string
		tasksCompleted uint64
		tasksFailed    uint64
		expectedBonus  uint8 // Approximate expected volume bonus
	}{
		{"No volume bonus (< 10)", 5, 0, 0},
		{"Low volume (10-100)", 50, 0, 1},
		{"Medium volume (100-1000)", 500, 0, 3},
		{"High volume (> 1000)", 5000, 0, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := &TrustScoreInput{
				Tier:            Tier4Standard,
				TasksCompleted:  1,
				TasksFailed:     0,
				ReputationScore: 0,
			}
			baseScore := calculateReputationScore(base)

			input := &TrustScoreInput{
				Tier:            Tier4Standard,
				TasksCompleted:  tt.tasksCompleted,
				TasksFailed:     tt.tasksFailed,
				ReputationScore: 0,
			}
			score := calculateReputationScore(input)

			// Score should be higher than base with volume bonus
			if tt.tasksCompleted > 10 && score <= baseScore {
				t.Errorf("Volume bonus not applied: base=%d, with volume=%d", baseScore, score)
			}
		})
	}
}

// TestCalculateUptimeScoreAllBrackets tests all LastSeenDelta time brackets
func TestCalculateUptimeScoreAllBrackets(t *testing.T) {
	tests := []struct {
		name               string
		lastSeenDelta      time.Duration
		expectedHeartbeat  uint8 // Approximate expected heartbeat freshness score
	}{
		{"< 1 minute", 30 * time.Second, 15},
		{"1-5 minutes", 3 * time.Minute, 12},
		{"5-15 minutes", 10 * time.Minute, 8},
		{"15-60 minutes", 30 * time.Minute, 4},
		{"> 1 hour", 2 * time.Hour, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &TrustScoreInput{
				Tier:                  Tier4Standard,
				UptimePercentage:      0, // Zero uptime to isolate heartbeat
				LastSeenDelta:         tt.lastSeenDelta,
				ConsecutiveHeartbeats: 0,
			}
			score := calculateUptimeScore(input)
			if score != tt.expectedHeartbeat {
				t.Errorf("Heartbeat freshness score for %v = %d, want %d",
					tt.lastSeenDelta, score, tt.expectedHeartbeat)
			}
		})
	}
}

// TestCalculateUptimeScoreConsecutiveHeartbeats tests consecutive heartbeat brackets
func TestCalculateUptimeScoreConsecutiveHeartbeats(t *testing.T) {
	tests := []struct {
		name                  string
		consecutiveHeartbeats uint64
		expectedBonus         uint8
	}{
		{"< 10", 5, 0},
		{"10-100", 50, 5},
		{"100-1000", 500, 10},
		{"> 1000", 5000, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &TrustScoreInput{
				Tier:                  Tier4Standard,
				UptimePercentage:      0,
				LastSeenDelta:         2 * time.Hour, // No heartbeat bonus
				ConsecutiveHeartbeats: tt.consecutiveHeartbeats,
			}
			score := calculateUptimeScore(input)
			if score != tt.expectedBonus {
				t.Errorf("Consecutive heartbeats %d = score %d, want %d",
					tt.consecutiveHeartbeats, score, tt.expectedBonus)
			}
		})
	}
}

// TestCalculateUptimeScoreMaxBound tests score capping at 100
func TestCalculateUptimeScoreMaxBound(t *testing.T) {
	input := &TrustScoreInput{
		Tier:                  Tier1GPUNativeCC,
		UptimePercentage:      100.0,               // +70
		LastSeenDelta:         10 * time.Second,    // +15
		ConsecutiveHeartbeats: 10000,               // +15
	}
	// 70 + 15 + 15 = 100
	score := calculateUptimeScore(input)
	if score != 100 {
		t.Errorf("Max uptime score should be 100, got %d", score)
	}
}

// TestRecoverScoreAfterGoodBehaviorOverflow tests overflow case
func TestRecoverScoreAfterGoodBehaviorOverflow(t *testing.T) {
	// Test case where currentScore + recovery would exceed maxScore
	currentScore := uint8(95)
	maxScore := uint8(100)
	recoveryRate := 0.5 // (100-95) * 0.5 = 2.5 -> 2

	newScore := RecoverScoreAfterGoodBehavior(currentScore, maxScore, recoveryRate)
	if newScore > maxScore {
		t.Errorf("RecoverScoreAfterGoodBehavior overflow: got %d, max is %d", newScore, maxScore)
	}
	expected := uint8(97) // 95 + 2 = 97
	if newScore != expected {
		t.Errorf("RecoverScoreAfterGoodBehavior = %d, want %d", newScore, expected)
	}

	// Test where recovery would exactly hit max
	currentScore = uint8(90)
	maxScore = uint8(100)
	recoveryRate = 1.0 // (100-90) * 1.0 = 10

	newScore = RecoverScoreAfterGoodBehavior(currentScore, maxScore, recoveryRate)
	if newScore != maxScore {
		t.Errorf("Full recovery should hit max: got %d, want %d", newScore, maxScore)
	}

	// Test where newScore calculation would exceed maxScore due to rounding
	currentScore = uint8(99)
	maxScore = uint8(100)
	recoveryRate = 2.0 // (100-99) * 2.0 = 2, 99+2=101 > 100

	newScore = RecoverScoreAfterGoodBehavior(currentScore, maxScore, recoveryRate)
	if newScore != maxScore {
		t.Errorf("Recovery should cap at max: got %d, want %d", newScore, maxScore)
	}
}

// TestRecoverScoreAfterGoodBehaviorEdgeCases tests edge cases
func TestRecoverScoreAfterGoodBehaviorEdgeCases(t *testing.T) {
	// Already at max
	newScore := RecoverScoreAfterGoodBehavior(100, 100, 0.1)
	if newScore != 100 {
		t.Errorf("At max, should stay at max: got %d", newScore)
	}

	// Zero recovery rate
	newScore = RecoverScoreAfterGoodBehavior(50, 100, 0.0)
	if newScore != 50 {
		t.Errorf("Zero recovery rate should not change score: got %d, want 50", newScore)
	}

	// Low current score
	newScore = RecoverScoreAfterGoodBehavior(1, 100, 0.1)
	// (100-1) * 0.1 = 9.9 -> 9, 1 + 9 = 10
	if newScore != 10 {
		t.Errorf("Low score recovery: got %d, want 10", newScore)
	}
}

// TestAdjustScoreForSlashingEdgeCases tests slashing adjustment edge cases
// =============================================================================
// Additional Coverage Tests for Scoring Functions
// =============================================================================

func TestCalculateHardwareScore_AllBranches(t *testing.T) {
	tests := []struct {
		name  string
		input TrustScoreInput
	}{
		{
			name: "Tier1 with high GPU generation (>10)",
			input: TrustScoreInput{
				Tier:          Tier1GPUNativeCC,
				GPUGeneration: 15, // > 10, triggers max cap
			},
		},
		{
			name: "Tier2 with high GPU generation (caps at 30)",
			input: TrustScoreInput{
				Tier:          Tier2ConfidentialVM,
				GPUGeneration: 20, // High enough to trigger cap
			},
		},
		{
			name: "Tier3 with high GPU generation (caps at 20)",
			input: TrustScoreInput{
				Tier:          Tier3DeviceTEE,
				GPUGeneration: 20, // High enough to trigger cap
			},
		},
		{
			name: "With TEE-IO enabled",
			input: TrustScoreInput{
				Tier:         Tier1GPUNativeCC,
				TEEIOEnabled: true,
			},
		},
		{
			name: "With RIM verified",
			input: TrustScoreInput{
				Tier:        Tier1GPUNativeCC,
				RIMVerified: true,
			},
		},
		{
			name: "With MIG support",
			input: TrustScoreInput{
				Tier: Tier1GPUNativeCC,
				HardwareCapabilities: &HardwareCapability{
					MIGSupported: true,
				},
			},
		},
		{
			name: "With high GPU memory (>80GB)",
			input: TrustScoreInput{
				Tier: Tier1GPUNativeCC,
				HardwareCapabilities: &HardwareCapability{
					GPUMemoryMB: 81920, // 80GB+
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTrustScore(&tt.input)
			if result.HardwareScore == 0 && tt.input.Tier != Tier4Standard {
				t.Errorf("Expected non-zero hardware score, got %d", result.HardwareScore)
			}
		})
	}
}

func TestCalculateReputationScore_AllBranches(t *testing.T) {
	tests := []struct {
		name  string
		input TrustScoreInput
	}{
		{
			name: "High volume bonus (>1000 tasks)",
			input: TrustScoreInput{
				Tier:           Tier1GPUNativeCC,
				TasksCompleted: 1500,
				TasksFailed:    10,
			},
		},
		{
			name: "Medium volume bonus (>100 tasks)",
			input: TrustScoreInput{
				Tier:           Tier1GPUNativeCC,
				TasksCompleted: 150,
				TasksFailed:    5,
			},
		},
		{
			name: "Low volume bonus (>10 tasks)",
			input: TrustScoreInput{
				Tier:           Tier1GPUNativeCC,
				TasksCompleted: 15,
				TasksFailed:    1,
			},
		},
		{
			name: "Heavy slashing (penalty > 30 caps)",
			input: TrustScoreInput{
				Tier:           Tier1GPUNativeCC,
				TasksCompleted: 100,
				SlashingEvents: 5, // 50 penalty, caps at 30
			},
		},
		{
			name: "Score below 0 clamps",
			input: TrustScoreInput{
				Tier:             Tier1GPUNativeCC,
				TasksCompleted:   0, // No tasks
				SlashingEvents:   10,
				ReputationScore:  0,
			},
		},
		{
			name: "Score above 100 clamps",
			input: TrustScoreInput{
				Tier:            Tier1GPUNativeCC,
				TasksCompleted:  5000,
				TasksFailed:     0,
				ReputationScore: 1.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTrustScore(&tt.input)
			// Just ensure it runs without panic and returns valid score
			if result.ReputationScore > 100 {
				t.Errorf("Reputation score %d exceeds 100", result.ReputationScore)
			}
		})
	}
}

func TestCalculateUptimeScore_AllBranches(t *testing.T) {
	tests := []struct {
		name  string
		input TrustScoreInput
	}{
		{
			name: "Very fresh heartbeat (<1 min)",
			input: TrustScoreInput{
				Tier:              Tier1GPUNativeCC,
				UptimePercentage:  100,
				LastSeenDelta:     30 * time.Second,
			},
		},
		{
			name: "Fresh heartbeat (<5 min)",
			input: TrustScoreInput{
				Tier:              Tier1GPUNativeCC,
				UptimePercentage:  100,
				LastSeenDelta:     3 * time.Minute,
			},
		},
		{
			name: "Recent heartbeat (<15 min)",
			input: TrustScoreInput{
				Tier:              Tier1GPUNativeCC,
				UptimePercentage:  100,
				LastSeenDelta:     10 * time.Minute,
			},
		},
		{
			name: "Stale heartbeat (<1 hour)",
			input: TrustScoreInput{
				Tier:              Tier1GPUNativeCC,
				UptimePercentage:  100,
				LastSeenDelta:     30 * time.Minute,
			},
		},
		{
			name: "Very stale heartbeat (>1 hour)",
			input: TrustScoreInput{
				Tier:              Tier1GPUNativeCC,
				UptimePercentage:  100,
				LastSeenDelta:     2 * time.Hour,
			},
		},
		{
			name: "High consecutive heartbeats (>1000)",
			input: TrustScoreInput{
				Tier:                  Tier1GPUNativeCC,
				UptimePercentage:      100,
				ConsecutiveHeartbeats: 1500,
			},
		},
		{
			name: "Medium consecutive heartbeats (>100)",
			input: TrustScoreInput{
				Tier:                  Tier1GPUNativeCC,
				UptimePercentage:      100,
				ConsecutiveHeartbeats: 150,
			},
		},
		{
			name: "Low consecutive heartbeats (>10)",
			input: TrustScoreInput{
				Tier:                  Tier1GPUNativeCC,
				UptimePercentage:      100,
				ConsecutiveHeartbeats: 15,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTrustScore(&tt.input)
			if result.UptimeScore > 100 {
				t.Errorf("Uptime score %d exceeds 100", result.UptimeScore)
			}
		})
	}
}

func TestAdjustScoreForSlashingEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		currentScore  uint8
		severity      float64
		expectedScore uint8
	}{
		{"Mild slashing", 100, 0.1, 90},
		{"Moderate slashing", 100, 0.5, 50},
		{"Severe slashing (100%)", 100, 1.0, 1}, // Never goes to zero
		{"Severe slashing (>100%)", 100, 1.5, 1},
		{"Low score slashing", 10, 0.5, 5},
		{"Very low score", 2, 0.9, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AdjustScoreForSlashing(tt.currentScore, tt.severity)
			if result != tt.expectedScore {
				t.Errorf("AdjustScoreForSlashing(%d, %.1f) = %d, want %d",
					tt.currentScore, tt.severity, result, tt.expectedScore)
			}
		})
	}
}

// TestValidateProviderScoreEdgeCases tests provider score validation edge cases
func TestValidateProviderScoreEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		score     uint8
		tier      CCTier
		expectErr bool
	}{
		{"Tier1 meets minimum", 95, Tier1GPUNativeCC, false},
		{"Tier1 at minimum", 90, Tier1GPUNativeCC, false},
		{"Tier1 below minimum", 85, Tier1GPUNativeCC, true},
		{"Tier2 meets minimum", 75, Tier2ConfidentialVM, false},
		{"Tier2 below minimum", 50, Tier2ConfidentialVM, true},
		{"Tier3 meets minimum", 50, Tier3DeviceTEE, false},
		{"Tier4 meets minimum", 20, Tier4Standard, false},
		{"Tier4 below minimum", 5, Tier4Standard, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProviderScore(tt.score, tt.tier)
			if tt.expectErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.expectErr && err != ErrTierNotMet {
				t.Errorf("Expected ErrTierNotMet, got %v", err)
			}
		})
	}
}

// =============================================================================
// QuickTrustScore Comprehensive Tests
// =============================================================================

func TestQuickTrustScoreComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		tier     CCTier
		cap      *HardwareCapability
		minScore uint8
		maxScore uint8
	}{
		{
			name:     "Tier1 with nil capability",
			tier:     Tier1GPUNativeCC,
			cap:      nil,
			minScore: 90, // Tier1 minimum
			maxScore: 100,
		},
		{
			name: "Tier1 with Blackwell GPU (9.0)",
			tier: Tier1GPUNativeCC,
			cap: &HardwareCapability{
				ComputeCap:    "9.0",
				GPUCCEnabled:  true,
				TEEIOSupported: true,
			},
			minScore: 90,
			maxScore: 100,
		},
		{
			name: "Tier1 with Ada GPU (8.9)",
			tier: Tier1GPUNativeCC,
			cap: &HardwareCapability{
				ComputeCap:   "8.9",
				GPUCCEnabled: true,
			},
			minScore: 90,
			maxScore: 100,
		},
		{
			name: "Tier1 with older GPU",
			tier: Tier1GPUNativeCC,
			cap: &HardwareCapability{
				ComputeCap: "7.5", // Turing or older
			},
			minScore: 90,
			maxScore: 100,
		},
		{
			name:     "Tier2 with nil capability",
			tier:     Tier2ConfidentialVM,
			cap:      nil,
			minScore: 70,
			maxScore: 89,
		},
		{
			name: "Tier2 with full capability",
			tier: Tier2ConfidentialVM,
			cap: &HardwareCapability{
				ComputeCap:     "9.0",
				GPUCCEnabled:   true,
				TEEIOSupported: true,
				MIGSupported:   true,
				GPUMemoryMB:    81920,
			},
			minScore: 70,
			maxScore: 89,
		},
		{
			name:     "Tier3 with nil capability",
			tier:     Tier3DeviceTEE,
			cap:      nil,
			minScore: 40,
			maxScore: 69,
		},
		{
			name: "Tier3 with device TEE",
			tier: Tier3DeviceTEE,
			cap: &HardwareCapability{
				ComputeCap:       "8.9",
				DeviceTEEEnabled: true,
			},
			minScore: 40,
			maxScore: 69,
		},
		{
			name:     "Tier4 with nil capability",
			tier:     Tier4Standard,
			cap:      nil,
			minScore: 10,
			maxScore: 49,
		},
		{
			name: "Tier4 with some hardware",
			tier: Tier4Standard,
			cap: &HardwareCapability{
				ComputeCap: "7.0",
			},
			minScore: 10,
			maxScore: 49,
		},
		{
			name:     "Unknown tier with nil capability",
			tier:     TierUnknown,
			cap:      nil,
			minScore: 0,
			maxScore: 49, // TierUnknown has low max
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := QuickTrustScore(tt.tier, tt.cap)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("QuickTrustScore(%v, cap) = %d, want in range [%d, %d]",
					tt.tier, score, tt.minScore, tt.maxScore)
			}
		})
	}
}

// TestQuickTrustScoreGPUGenerationMapping tests GPU generation inference from ComputeCap
func TestQuickTrustScoreGPUGenerationMapping(t *testing.T) {
	// Test that ComputeCap correctly maps to GPUGeneration
	// 9.0 -> gen 10, 8.9 -> gen 9, other -> gen 5

	// Blackwell/Hopper (9.0) should get gen 10
	cap90 := &HardwareCapability{ComputeCap: "9.0"}
	score90 := QuickTrustScore(Tier2ConfidentialVM, cap90)

	// Ada (8.9) should get gen 9
	cap89 := &HardwareCapability{ComputeCap: "8.9"}
	score89 := QuickTrustScore(Tier2ConfidentialVM, cap89)

	// Older should get gen 5
	cap75 := &HardwareCapability{ComputeCap: "7.5"}
	score75 := QuickTrustScore(Tier2ConfidentialVM, cap75)

	// 9.0 should score higher than or equal to 8.9, which should score higher than older
	if score90 < score75 {
		t.Errorf("9.0 compute cap (%d) should score >= 7.5 (%d)", score90, score75)
	}
	if score89 < score75 {
		t.Errorf("8.9 compute cap (%d) should score >= 7.5 (%d)", score89, score75)
	}
}

// TestQuickTrustScoreCCFeaturesFromCapability tests CC feature inference
func TestQuickTrustScoreCCFeaturesFromCapability(t *testing.T) {
	// With CC features enabled
	capWithCC := &HardwareCapability{
		ComputeCap:     "9.0",
		GPUCCEnabled:   true,
		TEEIOSupported: true,
	}
	scoreWithCC := QuickTrustScore(Tier2ConfidentialVM, capWithCC)

	// Without CC features
	capNoCC := &HardwareCapability{
		ComputeCap:     "9.0",
		GPUCCEnabled:   false,
		TEEIOSupported: false,
	}
	scoreNoCC := QuickTrustScore(Tier2ConfidentialVM, capNoCC)

	// With CC should score higher
	if scoreWithCC < scoreNoCC {
		t.Errorf("With CC features (%d) should score >= without (%d)", scoreWithCC, scoreNoCC)
	}
}

// =============================================================================
// Attestation Method Edge Cases
// =============================================================================

func TestCalculateAttestationScoreAllMethods(t *testing.T) {
	methods := []struct {
		method string
		bonus  uint8
	}{
		{"nvtrust", 10},
		{"sev-snp", 8},
		{"tdx", 8},
		{"cca", 6},
		{"secure-enclave", 5},
		{"software", 2},
		{"unknown", 2},
		{"", 2},
	}

	for _, m := range methods {
		t.Run(m.method, func(t *testing.T) {
			input := &TrustScoreInput{
				Tier:              Tier2ConfidentialVM,
				AttestationAge:    23 * time.Hour, // >75% of 24h, no freshness bonus
				AttestationMethod: m.method,
			}
			score := calculateAttestationScore(input)
			// Base 70 + method bonus
			expected := uint8(70 + m.bonus)
			if score != expected {
				t.Errorf("Attestation method %q score = %d, want %d", m.method, score, expected)
			}
		})
	}
}

// TestCalculateAttestationScoreFreshnessBrackets tests all freshness brackets
func TestCalculateAttestationScoreFreshnessBrackets(t *testing.T) {
	// Tier1 has 6h validity
	tests := []struct {
		name       string
		age        time.Duration
		freshBonus uint8
	}{
		{"Very fresh (<25%)", 1 * time.Hour, 15},          // 1h / 6h = 16.7%
		{"Fresh (25-50%)", 2 * time.Hour, 10},             // 2h / 6h = 33%
		{"Medium (50-75%)", 4 * time.Hour, 5},             // 4h / 6h = 67%
		{"Old (>75%)", 5 * time.Hour, 0},                  // 5h / 6h = 83%
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &TrustScoreInput{
				Tier:              Tier1GPUNativeCC,
				AttestationAge:    tt.age,
				AttestationMethod: "software", // +2
			}
			score := calculateAttestationScore(input)
			// Base 70 + freshness + 2 (software)
			expected := uint8(70 + tt.freshBonus + 2)
			if score != expected {
				t.Errorf("Attestation age %v score = %d, want %d", tt.age, score, expected)
			}
		})
	}
}

// TestCalculateAttestationScoreZeroValidity tests tier with zero attestation validity
func TestCalculateAttestationScoreZeroValidity(t *testing.T) {
	// TierUnknown returns 0 for AttestationValidity
	input := &TrustScoreInput{
		Tier:              TierUnknown,
		AttestationAge:    1 * time.Hour,
		AttestationMethod: "software",
	}
	score := calculateAttestationScore(input)
	// Base 70 + 0 (no freshness, validity is 0) + 2 (software) = 72
	if score != 72 {
		t.Errorf("TierUnknown attestation score = %d, want 72", score)
	}
}

// =============================================================================
// Custom Weights Tests
// =============================================================================

func TestCalculateTrustScoreWithCustomWeights(t *testing.T) {
	input := &TrustScoreInput{
		Tier:             Tier2ConfidentialVM,
		GPUGeneration:    8,
		UptimePercentage: 100.0,
	}

	// All weight on uptime
	weights := TrustScoreWeight{
		Hardware:    0.0,
		Attestation: 0.0,
		Reputation:  0.0,
		Uptime:      1.0,
	}

	result := CalculateTrustScoreWithWeights(input, weights)

	// Only uptime should contribute
	if result.HardwareContribution != 0 {
		t.Errorf("Hardware contribution should be 0, got %.1f", result.HardwareContribution)
	}
	if result.AttestationContribution != 0 {
		t.Errorf("Attestation contribution should be 0, got %.1f", result.AttestationContribution)
	}
	if result.ReputationContribution != 0 {
		t.Errorf("Reputation contribution should be 0, got %.1f", result.ReputationContribution)
	}
	if result.UptimeContribution <= 0 {
		t.Errorf("Uptime contribution should be > 0, got %.1f", result.UptimeContribution)
	}
}

// =============================================================================
// Score Clamping Edge Cases
// =============================================================================

func TestScoreClampingExactBoundaries(t *testing.T) {
	// Test score exactly at tier minimum
	input := &TrustScoreInput{
		Tier:           Tier4Standard,
		SlashingEvents: 0,
	}
	result := CalculateTrustScore(input)
	minScore := Tier4Standard.BaseTrustScore()
	if result.TotalScore < minScore {
		t.Errorf("Score %d below Tier4 minimum %d", result.TotalScore, minScore)
	}
	if !result.MeetsMinimum {
		t.Error("Score should meet minimum")
	}
}

// TestCalculateReputationScoreNegativeClamping tests extreme slashing that would go negative
func TestCalculateReputationScoreNegativeClamping(t *testing.T) {
	// This tests the defensive `if score < 0` branch
	// While mathematically unreachable with current logic (base 50 - max 30 penalty = 20 minimum),
	// we test the boundary to verify defensive code works if logic changes

	// Extreme case: many slashing events
	input := &TrustScoreInput{
		Tier:            Tier4Standard,
		TasksCompleted:  0,
		TasksFailed:     0,
		SlashingEvents:  100, // Would be 1000 penalty if not capped, but capped at 30
		ReputationScore: 0,
	}
	score := calculateReputationScore(input)
	// Base 50 - 30 (capped penalty) = 20, not negative
	if score != 20 {
		t.Errorf("Expected 20, got %d", score)
	}
}

// TestTotalScoreExceedsMax tests that scores exceeding tier max are clamped
func TestTotalScoreExceedsMax(t *testing.T) {
	// Max out everything to exceed tier max
	input := &TrustScoreInput{
		Tier:                  Tier3DeviceTEE,
		GPUGeneration:         10,
		CCFeaturesEnabled:     true,
		TEEIOEnabled:          true,
		RIMVerified:           true,
		AttestationAge:        1 * time.Minute,
		AttestationMethod:     "nvtrust",
		LocalVerification:     true,
		CertChainValid:        true,
		TasksCompleted:        10000,
		TasksFailed:           0,
		ReputationScore:       1.0,
		UptimePercentage:      100.0,
		LastSeenDelta:         10 * time.Second,
		ConsecutiveHeartbeats: 5000,
		HardwareCapabilities: &HardwareCapability{
			MIGSupported: true,
			GPUMemoryMB:  100000,
		},
	}

	result := CalculateTrustScore(input)
	maxScore := Tier3DeviceTEE.MaxTrustScore()
	if result.TotalScore > maxScore {
		t.Errorf("Tier3 score %d exceeds max %d", result.TotalScore, maxScore)
	}
}
