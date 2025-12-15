// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cc

import (
	"time"
)

// TrustScoreWeight defines the weights for trust score components
// per LP-5610 Section 5: Trust Score Calculation
type TrustScoreWeight struct {
	Hardware    float64 // 40% - Hardware tier and features
	Attestation float64 // 30% - Attestation freshness and verification
	Reputation  float64 // 20% - Historical performance
	Uptime      float64 // 10% - Availability
}

// DefaultWeights returns the default trust score weights per LP-5610
func DefaultWeights() TrustScoreWeight {
	return TrustScoreWeight{
		Hardware:    0.40,
		Attestation: 0.30,
		Reputation:  0.20,
		Uptime:      0.10,
	}
}

// TrustScoreInput contains all inputs needed to calculate trust score
type TrustScoreInput struct {
	// Hardware-based inputs
	Tier                  CCTier
	GPUGeneration         uint8           // 1-10, higher = newer
	CCFeaturesEnabled     bool            // GPU CC mode enabled
	TEEIOEnabled          bool            // TEE-IO for Blackwell
	RIMVerified           bool            // Reference Integrity Manifest verified
	HardwareCapabilities  *HardwareCapability

	// Attestation-based inputs
	AttestationAge        time.Duration   // Time since last attestation
	AttestationMethod     string          // "nvtrust", "sev-snp", "tdx", "software"
	LocalVerification     bool            // True if locally verified (no cloud)
	CertChainValid        bool            // Certificate chain validated

	// Reputation-based inputs
	TasksCompleted        uint64          // Total tasks completed
	TasksFailed           uint64          // Total tasks failed
	SlashingEvents        uint64          // Number of slashing events
	ReputationScore       float64         // 0.0-1.0 historical reputation

	// Uptime-based inputs
	UptimePercentage      float64         // 0.0-100.0 uptime percentage
	LastSeenDelta         time.Duration   // Time since last heartbeat
	ConsecutiveHeartbeats uint64          // Consecutive successful heartbeats
}

// TrustScoreResult contains the calculated trust score and breakdown
type TrustScoreResult struct {
	// Final trust score (0-100)
	TotalScore uint8 `json:"total_score"`

	// Component scores (0-100 each, weighted to total)
	HardwareScore    uint8 `json:"hardware_score"`
	AttestationScore uint8 `json:"attestation_score"`
	ReputationScore  uint8 `json:"reputation_score"`
	UptimeScore      uint8 `json:"uptime_score"`

	// Weighted contributions
	HardwareContribution    float64 `json:"hardware_contribution"`
	AttestationContribution float64 `json:"attestation_contribution"`
	ReputationContribution  float64 `json:"reputation_contribution"`
	UptimeContribution      float64 `json:"uptime_contribution"`

	// Tier information
	Tier            CCTier `json:"tier"`
	MeetsMinimum    bool   `json:"meets_minimum"`
	MinimumRequired uint8  `json:"minimum_required"`

	// Warnings or issues
	Warnings []string `json:"warnings,omitempty"`
}

// CalculateTrustScore calculates the trust score based on all inputs
// per LP-5610 Section 5: Trust Score Calculation
func CalculateTrustScore(input *TrustScoreInput) *TrustScoreResult {
	return CalculateTrustScoreWithWeights(input, DefaultWeights())
}

// CalculateTrustScoreWithWeights calculates trust score with custom weights
func CalculateTrustScoreWithWeights(input *TrustScoreInput, weights TrustScoreWeight) *TrustScoreResult {
	result := &TrustScoreResult{
		Tier:     input.Tier,
		Warnings: []string{},
	}

	// Calculate each component score
	result.HardwareScore = calculateHardwareScore(input)
	result.AttestationScore = calculateAttestationScore(input)
	result.ReputationScore = calculateReputationScore(input)
	result.UptimeScore = calculateUptimeScore(input)

	// Calculate weighted contributions
	result.HardwareContribution = float64(result.HardwareScore) * weights.Hardware
	result.AttestationContribution = float64(result.AttestationScore) * weights.Attestation
	result.ReputationContribution = float64(result.ReputationScore) * weights.Reputation
	result.UptimeContribution = float64(result.UptimeScore) * weights.Uptime

	// Calculate total score
	total := result.HardwareContribution +
		result.AttestationContribution +
		result.ReputationContribution +
		result.UptimeContribution

	// Clamp to tier limits
	minScore := input.Tier.BaseTrustScore()
	maxScore := input.Tier.MaxTrustScore()

	if total < float64(minScore) {
		result.TotalScore = minScore
		result.Warnings = append(result.Warnings, "Score clamped to tier minimum")
	} else if total > float64(maxScore) {
		result.TotalScore = maxScore
	} else {
		result.TotalScore = uint8(total)
	}

	// Check if meets minimum requirement
	result.MinimumRequired = minScore
	result.MeetsMinimum = result.TotalScore >= minScore

	return result
}

// calculateHardwareScore calculates the hardware component of trust score
// Hardware = 40 points max per LP-5610
func calculateHardwareScore(input *TrustScoreInput) uint8 {
	score := float64(0)

	// Base score by tier (0-35 points)
	switch input.Tier {
	case Tier1GPUNativeCC:
		score = 35 + float64(input.GPUGeneration)*0.5 // 35-40 points
		if input.GPUGeneration > 10 {
			score = 40
		}
	case Tier2ConfidentialVM:
		score = 25 + float64(input.GPUGeneration)*0.5 // 25-30 points
		if score > 30 {
			score = 30
		}
	case Tier3DeviceTEE:
		score = 15 + float64(input.GPUGeneration)*0.5 // 15-20 points
		if score > 20 {
			score = 20
		}
	case Tier4Standard:
		score = 5 // Base stake-only score
	}

	// CC feature bonuses
	if input.CCFeaturesEnabled {
		score += 3 // +3 for CC mode enabled
	}
	if input.TEEIOEnabled {
		score += 2 // +2 for TEE-IO (Blackwell)
	}
	if input.RIMVerified {
		score += 2 // +2 for RIM verification
	}

	// Hardware capability bonuses
	if input.HardwareCapabilities != nil {
		if input.HardwareCapabilities.MIGSupported {
			score += 1 // +1 for MIG support
		}
		if input.HardwareCapabilities.GPUMemoryMB > 80000 { // >80GB
			score += 2 // +2 for high memory
		}
	}

	// Cap at 100 (will be weighted to 40%)
	if score > 100 {
		score = 100
	}

	return uint8(score)
}

// calculateAttestationScore calculates the attestation component
// Attestation = 30 points max per LP-5610
func calculateAttestationScore(input *TrustScoreInput) uint8 {
	score := float64(70) // Base score for valid attestation

	// Freshness bonus (0-15 points)
	maxAge := input.Tier.AttestationValidity()
	if maxAge > 0 {
		ageRatio := float64(input.AttestationAge) / float64(maxAge)
		if ageRatio < 0.25 {
			score += 15 // Very fresh
		} else if ageRatio < 0.50 {
			score += 10
		} else if ageRatio < 0.75 {
			score += 5
		}
		// No bonus if >75% age
	}

	// Verification method bonus
	switch input.AttestationMethod {
	case "nvtrust":
		score += 10 // Best: local GPU attestation
	case "sev-snp", "tdx":
		score += 8 // Good: CPU TEE attestation
	case "cca":
		score += 6 // ARM CCA
	case "secure-enclave":
		score += 5 // Apple Secure Enclave
	default:
		score += 2 // Software attestation
	}

	// Local verification bonus (blockchain requirement)
	if input.LocalVerification {
		score += 5 // +5 for no cloud dependency
	}

	// Certificate chain validation
	if input.CertChainValid {
		score += 3 // +3 for valid cert chain
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return uint8(score)
}

// calculateReputationScore calculates the reputation component
// Reputation = 20 points max per LP-5610
func calculateReputationScore(input *TrustScoreInput) uint8 {
	score := float64(50) // Base score

	// Task completion rate
	if input.TasksCompleted > 0 {
		totalTasks := input.TasksCompleted + input.TasksFailed
		successRate := float64(input.TasksCompleted) / float64(totalTasks)
		score += successRate * 30 // 0-30 points for success rate

		// Volume bonus
		if totalTasks > 1000 {
			score += 5
		} else if totalTasks > 100 {
			score += 3
		} else if totalTasks > 10 {
			score += 1
		}
	}

	// Slashing penalty
	if input.SlashingEvents > 0 {
		penalty := float64(input.SlashingEvents) * 10
		if penalty > 30 {
			penalty = 30
		}
		score -= penalty
	}

	// Historical reputation score contribution
	if input.ReputationScore > 0 {
		score += input.ReputationScore * 15 // 0-15 points from history
	}

	// Clamp to valid range
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return uint8(score)
}

// calculateUptimeScore calculates the uptime component
// Uptime = 10 points max per LP-5610
func calculateUptimeScore(input *TrustScoreInput) uint8 {
	score := float64(0)

	// Uptime percentage (0-70 points)
	score += input.UptimePercentage * 0.7

	// Heartbeat freshness (0-15 points)
	switch {
	case input.LastSeenDelta < 1*time.Minute:
		score += 15
	case input.LastSeenDelta < 5*time.Minute:
		score += 12
	case input.LastSeenDelta < 15*time.Minute:
		score += 8
	case input.LastSeenDelta < 1*time.Hour:
		score += 4
	}

	// Consecutive heartbeats bonus (0-15 points)
	if input.ConsecutiveHeartbeats > 1000 {
		score += 15
	} else if input.ConsecutiveHeartbeats > 100 {
		score += 10
	} else if input.ConsecutiveHeartbeats > 10 {
		score += 5
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return uint8(score)
}

// QuickTrustScore calculates a quick trust score with minimal inputs
// Useful for initial tier classification before full attestation
func QuickTrustScore(tier CCTier, cap *HardwareCapability) uint8 {
	input := &TrustScoreInput{
		Tier:                 tier,
		HardwareCapabilities: cap,
		AttestationAge:       0,
		LocalVerification:    true,
		UptimePercentage:     100.0,
		ReputationScore:      0.5,
	}

	// Set GPU generation based on model
	if cap != nil {
		switch {
		case cap.ComputeCap == "9.0": // Blackwell/Hopper
			input.GPUGeneration = 10
		case cap.ComputeCap == "8.9": // Ada
			input.GPUGeneration = 9
		default:
			input.GPUGeneration = 5
		}
		input.CCFeaturesEnabled = cap.GPUCCEnabled
		input.TEEIOEnabled = cap.TEEIOSupported
	}

	result := CalculateTrustScore(input)
	return result.TotalScore
}

// ValidateProviderScore checks if a provider meets minimum score requirements
func ValidateProviderScore(score uint8, tier CCTier) error {
	minScore := tier.BaseTrustScore()
	if score < minScore {
		return ErrTierNotMet
	}
	return nil
}

// AdjustScoreForSlashing reduces trust score after a slashing event
func AdjustScoreForSlashing(currentScore uint8, slashingSeverity float64) uint8 {
	reduction := uint8(float64(currentScore) * slashingSeverity)
	if reduction >= currentScore {
		return 1 // Never go to zero, allow recovery
	}
	return currentScore - reduction
}

// RecoverScoreAfterGoodBehavior increases trust score after good behavior
func RecoverScoreAfterGoodBehavior(currentScore, maxScore uint8, recoveryRate float64) uint8 {
	recovery := uint8(float64(maxScore-currentScore) * recoveryRate)
	newScore := currentScore + recovery
	if newScore > maxScore {
		return maxScore
	}
	return newScore
}
