// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cc

import (
	"testing"
	"time"
)

// =============================================================================
// CCTier Tests - LP-5610 Section 3: Tier Classification
// =============================================================================

func TestTierString(t *testing.T) {
	tests := []struct {
		tier     CCTier
		expected string
	}{
		{Tier1GPUNativeCC, "GPU-Native-CC"},
		{Tier2ConfidentialVM, "Confidential-VM"},
		{Tier3DeviceTEE, "Device-TEE"},
		{Tier4Standard, "Standard"},
		{TierUnknown, "Unknown"},
		{CCTier(99), "Unknown"}, // Invalid tier
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.tier.String(); got != tt.expected {
				t.Errorf("CCTier.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTierDescription(t *testing.T) {
	tests := []struct {
		tier        CCTier
		containsStr string
	}{
		{Tier1GPUNativeCC, "GPU-level hardware"},
		{Tier2ConfidentialVM, "CPU-level VM isolation"},
		{Tier3DeviceTEE, "Edge device TEE"},
		{Tier4Standard, "Software/stake-based"},
		{TierUnknown, "Unknown tier"},
		{CCTier(99), "Unknown tier"}, // Invalid tier
	}

	for _, tt := range tests {
		t.Run(tt.tier.String(), func(t *testing.T) {
			desc := tt.tier.Description()
			if desc == "" {
				t.Error("Description() should not return empty string")
			}
		})
	}
}

func TestTierBaseTrustScore(t *testing.T) {
	// LP-5610 Section 5: Trust Score ranges
	tests := []struct {
		tier     CCTier
		expected uint8
	}{
		{Tier1GPUNativeCC, 90},    // 90-100 range
		{Tier2ConfidentialVM, 70}, // 70-89 range
		{Tier3DeviceTEE, 50},      // 50-69 range
		{Tier4Standard, 10},       // 10-49 range
		{TierUnknown, 0},
		{CCTier(99), 0}, // Invalid tier
	}

	for _, tt := range tests {
		t.Run(tt.tier.String(), func(t *testing.T) {
			if got := tt.tier.BaseTrustScore(); got != tt.expected {
				t.Errorf("CCTier.BaseTrustScore() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTierMaxTrustScore(t *testing.T) {
	tests := []struct {
		tier     CCTier
		expected uint8
	}{
		{Tier1GPUNativeCC, 100},
		{Tier2ConfidentialVM, 89},
		{Tier3DeviceTEE, 69},
		{Tier4Standard, 49},
		{TierUnknown, 0},
		{CCTier(99), 0}, // Invalid tier
	}

	for _, tt := range tests {
		t.Run(tt.tier.String(), func(t *testing.T) {
			if got := tt.tier.MaxTrustScore(); got != tt.expected {
				t.Errorf("CCTier.MaxTrustScore() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTierMinStakeLUX(t *testing.T) {
	// LP-5610 Section 6: Stake requirements
	tests := []struct {
		tier     CCTier
		expected uint64
	}{
		{Tier1GPUNativeCC, 100_000},   // 100,000 LUX
		{Tier2ConfidentialVM, 50_000}, // 50,000 LUX
		{Tier3DeviceTEE, 10_000},      // 10,000 LUX
		{Tier4Standard, 1_000},        // 1,000 LUX
		{TierUnknown, 0},
		{CCTier(99), 0}, // Invalid tier
	}

	for _, tt := range tests {
		t.Run(tt.tier.String(), func(t *testing.T) {
			if got := tt.tier.MinStakeLUX(); got != tt.expected {
				t.Errorf("CCTier.MinStakeLUX() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTierRewardMultiplier(t *testing.T) {
	// LP-5610 Section 7: Reward multipliers
	tests := []struct {
		tier     CCTier
		expected float64
	}{
		{Tier1GPUNativeCC, 1.5},
		{Tier2ConfidentialVM, 1.0},
		{Tier3DeviceTEE, 0.75},
		{Tier4Standard, 0.5},
		{TierUnknown, 0.0},
		{CCTier(99), 0.0}, // Invalid tier
	}

	for _, tt := range tests {
		t.Run(tt.tier.String(), func(t *testing.T) {
			if got := tt.tier.RewardMultiplier(); got != tt.expected {
				t.Errorf("CCTier.RewardMultiplier() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTierAttestationValidity(t *testing.T) {
	// LP-5610 Section 4: Attestation refresh intervals
	tests := []struct {
		tier     CCTier
		expected time.Duration
	}{
		{Tier1GPUNativeCC, 6 * time.Hour},
		{Tier2ConfidentialVM, 24 * time.Hour},
		{Tier3DeviceTEE, 7 * 24 * time.Hour},
		{Tier4Standard, 30 * 24 * time.Hour},
		{TierUnknown, 0},
		{CCTier(99), 0}, // Invalid tier
	}

	for _, tt := range tests {
		t.Run(tt.tier.String(), func(t *testing.T) {
			if got := tt.tier.AttestationValidity(); got != tt.expected {
				t.Errorf("CCTier.AttestationValidity() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTierMeetsTierRequirement(t *testing.T) {
	tests := []struct {
		tier     CCTier
		required CCTier
		expected bool
	}{
		// Tier 1 meets all requirements (lower number = higher security)
		{Tier1GPUNativeCC, Tier1GPUNativeCC, true},
		{Tier1GPUNativeCC, Tier2ConfidentialVM, true},
		{Tier1GPUNativeCC, Tier3DeviceTEE, true},
		{Tier1GPUNativeCC, Tier4Standard, true},

		// Tier 2 meets Tier 2-4 requirements
		{Tier2ConfidentialVM, Tier1GPUNativeCC, false},
		{Tier2ConfidentialVM, Tier2ConfidentialVM, true},
		{Tier2ConfidentialVM, Tier3DeviceTEE, true},
		{Tier2ConfidentialVM, Tier4Standard, true},

		// Tier 3 meets Tier 3-4 requirements
		{Tier3DeviceTEE, Tier1GPUNativeCC, false},
		{Tier3DeviceTEE, Tier2ConfidentialVM, false},
		{Tier3DeviceTEE, Tier3DeviceTEE, true},
		{Tier3DeviceTEE, Tier4Standard, true},

		// Tier 4 only meets Tier 4 requirement
		{Tier4Standard, Tier1GPUNativeCC, false},
		{Tier4Standard, Tier2ConfidentialVM, false},
		{Tier4Standard, Tier3DeviceTEE, false},
		{Tier4Standard, Tier4Standard, true},

		// Unknown tier meets nothing
		{TierUnknown, Tier4Standard, false},
		{TierUnknown, TierUnknown, false},
	}

	for _, tt := range tests {
		name := tt.tier.String() + "_meets_" + tt.required.String()
		t.Run(name, func(t *testing.T) {
			if got := tt.tier.MeetsTierRequirement(tt.required); got != tt.expected {
				t.Errorf("CCTier.MeetsTierRequirement() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// ParseTier Tests
// =============================================================================

func TestParseTier(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected CCTier
		wantErr  bool
	}{
		// Uint8 inputs
		{"uint8(1)", uint8(1), Tier1GPUNativeCC, false},
		{"uint8(2)", uint8(2), Tier2ConfidentialVM, false},
		{"uint8(3)", uint8(3), Tier3DeviceTEE, false},
		{"uint8(4)", uint8(4), Tier4Standard, false},
		{"uint8(0)", uint8(0), TierUnknown, true},
		{"uint8(5)", uint8(5), TierUnknown, true},
		{"uint8(255)", uint8(255), TierUnknown, true},

		// Int inputs
		{"int(1)", int(1), Tier1GPUNativeCC, false},
		{"int(2)", int(2), Tier2ConfidentialVM, false},
		{"int(3)", int(3), Tier3DeviceTEE, false},
		{"int(4)", int(4), Tier4Standard, false},
		{"int(0)", int(0), TierUnknown, true},
		{"int(5)", int(5), TierUnknown, true},

		// String inputs - numeric
		{"string 1", "1", Tier1GPUNativeCC, false},
		{"string 2", "2", Tier2ConfidentialVM, false},
		{"string 3", "3", Tier3DeviceTEE, false},
		{"string 4", "4", Tier4Standard, false},

		// String inputs - tier names
		{"tier1", "tier1", Tier1GPUNativeCC, false},
		{"Tier1", "Tier1", Tier1GPUNativeCC, false},
		{"tier2", "tier2", Tier2ConfidentialVM, false},
		{"Tier2", "Tier2", Tier2ConfidentialVM, false},
		{"tier3", "tier3", Tier3DeviceTEE, false},
		{"Tier3", "Tier3", Tier3DeviceTEE, false},
		{"tier4", "tier4", Tier4Standard, false},
		{"Tier4", "Tier4", Tier4Standard, false},

		// String inputs - display names
		{"GPU-Native-CC", "GPU-Native-CC", Tier1GPUNativeCC, false},
		{"gpu-native-cc", "gpu-native-cc", Tier1GPUNativeCC, false},
		{"Confidential-VM", "Confidential-VM", Tier2ConfidentialVM, false},
		{"confidential-vm", "confidential-vm", Tier2ConfidentialVM, false},
		{"Device-TEE", "Device-TEE", Tier3DeviceTEE, false},
		{"device-tee", "device-tee", Tier3DeviceTEE, false},
		{"Standard", "Standard", Tier4Standard, false},
		{"standard", "standard", Tier4Standard, false},

		// Invalid strings
		{"invalid", "invalid", TierUnknown, true},
		{"empty", "", TierUnknown, true},
		{"tier5", "tier5", TierUnknown, true},

		// Invalid types
		{"float64", float64(1.0), TierUnknown, true},
		{"bool", true, TierUnknown, true},
		{"nil", nil, TierUnknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTier(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseTier() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// TierAttestation Tests - LP-5610 Section 4: Attestation
// =============================================================================

func TestTierAttestation_IsValid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		attestation TierAttestation
		expected    bool
	}{
		{
			name: "valid attestation",
			attestation: TierAttestation{
				Tier:      Tier1GPUNativeCC,
				IssuedAt:  now.Add(-1 * time.Hour),
				ExpiresAt: now.Add(5 * time.Hour),
			},
			expected: true,
		},
		{
			name: "expired attestation",
			attestation: TierAttestation{
				Tier:      Tier1GPUNativeCC,
				IssuedAt:  now.Add(-24 * time.Hour),
				ExpiresAt: now.Add(-1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "future attestation (not yet valid)",
			attestation: TierAttestation{
				Tier:      Tier1GPUNativeCC,
				IssuedAt:  now.Add(1 * time.Hour),
				ExpiresAt: now.Add(7 * time.Hour),
			},
			expected: false,
		},
		{
			name: "unknown tier",
			attestation: TierAttestation{
				Tier:      TierUnknown,
				IssuedAt:  now.Add(-1 * time.Hour),
				ExpiresAt: now.Add(5 * time.Hour),
			},
			expected: false,
		},
		{
			name: "just expired (edge case)",
			attestation: TierAttestation{
				Tier:      Tier2ConfidentialVM,
				IssuedAt:  now.Add(-25 * time.Hour),
				ExpiresAt: now.Add(-1 * time.Millisecond),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.attestation.IsValid(); got != tt.expected {
				t.Errorf("TierAttestation.IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTierAttestation_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		attestation TierAttestation
		expected    bool
	}{
		{
			name: "not expired",
			attestation: TierAttestation{
				ExpiresAt: now.Add(5 * time.Hour),
			},
			expected: false,
		},
		{
			name: "expired",
			attestation: TierAttestation{
				ExpiresAt: now.Add(-1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "just expired",
			attestation: TierAttestation{
				ExpiresAt: now.Add(-1 * time.Millisecond),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.attestation.IsExpired(); got != tt.expected {
				t.Errorf("TierAttestation.IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTierAttestation_TimeUntilExpiry(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		attestation TierAttestation
		minDuration time.Duration
		maxDuration time.Duration
	}{
		{
			name: "5 hours until expiry",
			attestation: TierAttestation{
				ExpiresAt: now.Add(5 * time.Hour),
			},
			minDuration: 4*time.Hour + 59*time.Minute,
			maxDuration: 5*time.Hour + 1*time.Minute,
		},
		{
			name: "1 minute until expiry",
			attestation: TierAttestation{
				ExpiresAt: now.Add(1 * time.Minute),
			},
			minDuration: 50 * time.Second,
			maxDuration: 70 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.attestation.TimeUntilExpiry()
			if got < tt.minDuration || got > tt.maxDuration {
				t.Errorf("TierAttestation.TimeUntilExpiry() = %v, want between %v and %v",
					got, tt.minDuration, tt.maxDuration)
			}
		})
	}
}

func TestTierAttestation_MeetsTierRequirement(t *testing.T) {
	now := time.Now()
	validAttestation := TierAttestation{
		Tier:      Tier2ConfidentialVM,
		IssuedAt:  now.Add(-1 * time.Hour),
		ExpiresAt: now.Add(23 * time.Hour),
	}
	expiredAttestation := TierAttestation{
		Tier:      Tier1GPUNativeCC,
		IssuedAt:  now.Add(-24 * time.Hour),
		ExpiresAt: now.Add(-1 * time.Hour),
	}

	tests := []struct {
		name        string
		attestation TierAttestation
		required    CCTier
		wantErr     bool
	}{
		{"valid tier 2 meets tier 2", validAttestation, Tier2ConfidentialVM, false},
		{"valid tier 2 meets tier 3", validAttestation, Tier3DeviceTEE, false},
		{"valid tier 2 meets tier 4", validAttestation, Tier4Standard, false},
		{"valid tier 2 does not meet tier 1", validAttestation, Tier1GPUNativeCC, true},
		{"expired attestation fails even for lower tier", expiredAttestation, Tier4Standard, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.attestation.MeetsTierRequirement(tt.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("TierAttestation.MeetsTierRequirement() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// TierRequirement Tests
// =============================================================================

func TestDefaultTierRequirement(t *testing.T) {
	tiers := []CCTier{Tier1GPUNativeCC, Tier2ConfidentialVM, Tier3DeviceTEE, Tier4Standard}

	for _, tier := range tiers {
		t.Run(tier.String(), func(t *testing.T) {
			req := DefaultTierRequirement(tier)
			if req.MinTier != tier {
				t.Errorf("DefaultTierRequirement().MinTier = %v, want %v", req.MinTier, tier)
			}
			if !req.RequireValidAttestation {
				t.Error("DefaultTierRequirement().RequireValidAttestation should be true")
			}
			if req.MinTrustScore != tier.BaseTrustScore() {
				t.Errorf("DefaultTierRequirement().MinTrustScore = %v, want %v",
					req.MinTrustScore, tier.BaseTrustScore())
			}
			if req.MaxAttestationAge != tier.AttestationValidity() {
				t.Errorf("DefaultTierRequirement().MaxAttestationAge = %v, want %v",
					req.MaxAttestationAge, tier.AttestationValidity())
			}
		})
	}
}

func TestTierRequirement_IsMet(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		requirement TierRequirement
		attestation *TierAttestation
		wantErr     bool
		errType     error
	}{
		{
			name: "meets all requirements",
			requirement: TierRequirement{
				MinTier:                 Tier2ConfidentialVM,
				RequireValidAttestation: true,
				MinTrustScore:           70,
			},
			attestation: &TierAttestation{
				Tier:       Tier1GPUNativeCC,
				TrustScore: 95,
				IssuedAt:   now.Add(-1 * time.Hour),
				ExpiresAt:  now.Add(5 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "tier too low",
			requirement: TierRequirement{
				MinTier: Tier1GPUNativeCC,
			},
			attestation: &TierAttestation{
				Tier:       Tier2ConfidentialVM,
				TrustScore: 85,
				IssuedAt:   now.Add(-1 * time.Hour),
				ExpiresAt:  now.Add(5 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "trust score too low",
			requirement: TierRequirement{
				MinTier:       Tier3DeviceTEE,
				MinTrustScore: 60,
			},
			attestation: &TierAttestation{
				Tier:       Tier3DeviceTEE,
				TrustScore: 55,
				IssuedAt:   now.Add(-1 * time.Hour),
				ExpiresAt:  now.Add(5 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "nil attestation",
			requirement: TierRequirement{
				MinTier: Tier4Standard,
			},
			attestation: nil,
			wantErr:     true,
		},
		{
			name: "attestation too old",
			requirement: TierRequirement{
				MinTier:           Tier2ConfidentialVM,
				MaxAttestationAge: 1 * time.Hour,
			},
			attestation: &TierAttestation{
				Tier:       Tier2ConfidentialVM,
				TrustScore: 80,
				IssuedAt:   now.Add(-2 * time.Hour),
				ExpiresAt:  now.Add(22 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "requires valid attestation but expired",
			requirement: TierRequirement{
				MinTier:                 Tier4Standard,
				RequireValidAttestation: true,
			},
			attestation: &TierAttestation{
				Tier:       Tier4Standard,
				TrustScore: 40,
				IssuedAt:   now.Add(-31 * 24 * time.Hour),
				ExpiresAt:  now.Add(-1 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "vendor requirement not met",
			requirement: TierRequirement{
				MinTier:               Tier1GPUNativeCC,
				RequireSpecificVendor: "NVIDIA",
			},
			attestation: &TierAttestation{
				Tier:       Tier1GPUNativeCC,
				TrustScore: 95,
				IssuedAt:   now.Add(-1 * time.Hour),
				ExpiresAt:  now.Add(5 * time.Hour),
				HardwareInfo: &HardwareInfo{
					Vendor: "AMD",
				},
			},
			wantErr: true,
		},
		{
			name: "vendor requirement met",
			requirement: TierRequirement{
				MinTier:               Tier1GPUNativeCC,
				RequireSpecificVendor: "NVIDIA",
			},
			attestation: &TierAttestation{
				Tier:       Tier1GPUNativeCC,
				TrustScore: 95,
				IssuedAt:   now.Add(-1 * time.Hour),
				ExpiresAt:  now.Add(5 * time.Hour),
				HardwareInfo: &HardwareInfo{
					Vendor: "NVIDIA",
				},
			},
			wantErr: false,
		},
		{
			name: "memory requirement not met",
			requirement: TierRequirement{
				MinTier:          Tier1GPUNativeCC,
				RequireMinMemory: 80 * 1024 * 1024 * 1024, // 80GB
			},
			attestation: &TierAttestation{
				Tier:       Tier1GPUNativeCC,
				TrustScore: 95,
				IssuedAt:   now.Add(-1 * time.Hour),
				ExpiresAt:  now.Add(5 * time.Hour),
				HardwareInfo: &HardwareInfo{
					Vendor:     "NVIDIA",
					MemorySize: 40 * 1024 * 1024 * 1024, // 40GB
				},
			},
			wantErr: true,
		},
		{
			name: "memory requirement met",
			requirement: TierRequirement{
				MinTier:          Tier1GPUNativeCC,
				RequireMinMemory: 80 * 1024 * 1024 * 1024, // 80GB
			},
			attestation: &TierAttestation{
				Tier:       Tier1GPUNativeCC,
				TrustScore: 95,
				IssuedAt:   now.Add(-1 * time.Hour),
				ExpiresAt:  now.Add(5 * time.Hour),
				HardwareInfo: &HardwareInfo{
					Vendor:     "NVIDIA",
					MemorySize: 96 * 1024 * 1024 * 1024, // 96GB
				},
			},
			wantErr: false,
		},
		// Additional coverage: vendor requirement with nil HardwareInfo (should pass - no hardware info to check)
		{
			name: "vendor requirement with nil hardware info passes",
			requirement: TierRequirement{
				MinTier:               Tier2ConfidentialVM,
				RequireSpecificVendor: "NVIDIA",
			},
			attestation: &TierAttestation{
				Tier:         Tier2ConfidentialVM,
				TrustScore:   80,
				IssuedAt:     now.Add(-1 * time.Hour),
				ExpiresAt:    now.Add(23 * time.Hour),
				HardwareInfo: nil, // No hardware info
			},
			wantErr: false,
		},
		// Additional coverage: memory requirement with nil HardwareInfo (should pass - no hardware info to check)
		{
			name: "memory requirement with nil hardware info passes",
			requirement: TierRequirement{
				MinTier:          Tier2ConfidentialVM,
				RequireMinMemory: 80 * 1024 * 1024 * 1024, // 80GB
			},
			attestation: &TierAttestation{
				Tier:         Tier2ConfidentialVM,
				TrustScore:   80,
				IssuedAt:     now.Add(-1 * time.Hour),
				ExpiresAt:    now.Add(23 * time.Hour),
				HardwareInfo: nil, // No hardware info
			},
			wantErr: false,
		},
		// Additional coverage: MaxAttestationAge = 0 (no age check)
		{
			name: "no attestation age check when max is zero",
			requirement: TierRequirement{
				MinTier:           Tier4Standard,
				MaxAttestationAge: 0, // No age limit
			},
			attestation: &TierAttestation{
				Tier:       Tier4Standard,
				TrustScore: 40,
				IssuedAt:   now.Add(-365 * 24 * time.Hour), // 1 year old
				ExpiresAt:  now.Add(1 * time.Hour),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.requirement.IsMet(tt.attestation)
			if (err != nil) != tt.wantErr {
				t.Errorf("TierRequirement.IsMet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// Trust Score Tests - LP-5610 Section 5: Trust Score Calculation
// =============================================================================

func TestCalculateTrustScore(t *testing.T) {
	tests := []struct {
		name     string
		input    TrustScoreInput
		minScore uint8
		maxScore uint8
	}{
		{
			name: "tier 1 with all features enabled",
			input: TrustScoreInput{
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
				ReputationScore:       0.9,
				UptimePercentage:      99.9,
				LastSeenDelta:         30 * time.Second,
				ConsecutiveHeartbeats: 500,
				HardwareCapabilities: &HardwareCapability{
					MIGSupported: true,
					GPUMemoryMB:  81920, // 80GB+
				},
			},
			minScore: 90,
			maxScore: 100,
		},
		{
			name: "tier 1 minimal features",
			input: TrustScoreInput{
				Tier:              Tier1GPUNativeCC,
				GPUGeneration:     8,
				CCFeaturesEnabled: false,
				AttestationAge:    5 * time.Hour,
				AttestationMethod: "nvtrust",
				UptimePercentage:  95.0,
			},
			minScore: 90,
			maxScore: 100,
		},
		{
			name: "tier 2 with SEV-SNP",
			input: TrustScoreInput{
				Tier:              Tier2ConfidentialVM,
				GPUGeneration:     8,
				CCFeaturesEnabled: false,
				AttestationAge:    12 * time.Hour,
				AttestationMethod: "sev-snp",
				LocalVerification: true,
				TasksCompleted:    100,
				UptimePercentage:  95.0,
				LastSeenDelta:     5 * time.Minute,
			},
			minScore: 70,
			maxScore: 89,
		},
		{
			name: "tier 2 with TDX",
			input: TrustScoreInput{
				Tier:              Tier2ConfidentialVM,
				GPUGeneration:     7,
				AttestationAge:    20 * time.Hour,
				AttestationMethod: "tdx",
				LocalVerification: true,
				UptimePercentage:  90.0,
				LastSeenDelta:     10 * time.Minute,
			},
			minScore: 70,
			maxScore: 89,
		},
		{
			name: "tier 2 with CCA",
			input: TrustScoreInput{
				Tier:              Tier2ConfidentialVM,
				AttestationAge:    18 * time.Hour,
				AttestationMethod: "cca",
				UptimePercentage:  85.0,
			},
			minScore: 70,
			maxScore: 89,
		},
		{
			name: "tier 3 with secure enclave",
			input: TrustScoreInput{
				Tier:              Tier3DeviceTEE,
				AttestationAge:    3 * 24 * time.Hour,
				AttestationMethod: "secure-enclave",
				UptimePercentage:  80.0,
			},
			minScore: 50,
			maxScore: 69,
		},
		{
			name: "tier 4 minimal",
			input: TrustScoreInput{
				Tier:              Tier4Standard,
				GPUGeneration:     5,
				AttestationAge:    24 * time.Hour,
				AttestationMethod: "software",
				UptimePercentage:  90.0,
				LastSeenDelta:     10 * time.Minute,
			},
			minScore: 10,
			maxScore: 49,
		},
		{
			name: "tier 4 with slashing events",
			input: TrustScoreInput{
				Tier:           Tier4Standard,
				SlashingEvents: 5,
				UptimePercentage: 80.0,
			},
			minScore: 10,
			maxScore: 49,
		},
		{
			name: "tier 3 high volume tasks",
			input: TrustScoreInput{
				Tier:              Tier3DeviceTEE,
				TasksCompleted:    5000,
				TasksFailed:       50,
				ReputationScore:   0.95,
				UptimePercentage:  99.5,
				LastSeenDelta:     30 * time.Second,
				ConsecutiveHeartbeats: 2000,
			},
			minScore: 50,
			maxScore: 69,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTrustScore(&tt.input)
			if result.TotalScore < tt.minScore || result.TotalScore > tt.maxScore {
				t.Errorf("CalculateTrustScore() score = %v, want between %v and %v",
					result.TotalScore, tt.minScore, tt.maxScore)
			}
			if result.Tier != tt.input.Tier {
				t.Errorf("CalculateTrustScore() tier = %v, want %v", result.Tier, tt.input.Tier)
			}
			if result.MeetsMinimum != (result.TotalScore >= tt.input.Tier.BaseTrustScore()) {
				t.Error("MeetsMinimum inconsistent with TotalScore")
			}
		})
	}
}

func TestCalculateTrustScoreWithWeights(t *testing.T) {
	input := &TrustScoreInput{
		Tier:              Tier2ConfidentialVM,
		GPUGeneration:     8,
		AttestationAge:    12 * time.Hour,
		AttestationMethod: "sev-snp",
		UptimePercentage:  95.0,
	}

	// Custom weights - heavily favor hardware
	customWeights := TrustScoreWeight{
		Hardware:    0.80,
		Attestation: 0.10,
		Reputation:  0.05,
		Uptime:      0.05,
	}

	result := CalculateTrustScoreWithWeights(input, customWeights)
	if result.TotalScore < 70 || result.TotalScore > 89 {
		t.Errorf("Custom weights score = %v, want between 70 and 89", result.TotalScore)
	}
}

func TestQuickTrustScoreTiers(t *testing.T) {
	tests := []struct {
		name     string
		tier     CCTier
		cap      *HardwareCapability
		minScore uint8
		maxScore uint8
	}{
		{
			name: "Tier 1 Blackwell GPU",
			tier: Tier1GPUNativeCC,
			cap: &HardwareCapability{
				GPUVendor:      VendorNVIDIA,
				GPUModel:       "B200",
				GPUCCEnabled:   true,
				TEEIOSupported: true,
				ComputeCap:     "9.0",
			},
			minScore: 90,
			maxScore: 100,
		},
		{
			name: "Tier 1 H100 GPU",
			tier: Tier1GPUNativeCC,
			cap: &HardwareCapability{
				GPUVendor:    VendorNVIDIA,
				GPUModel:     "H100",
				GPUCCEnabled: true,
				ComputeCap:   "9.0",
			},
			minScore: 90,
			maxScore: 100,
		},
		{
			name: "Tier 2 with Ada GPU",
			tier: Tier2ConfidentialVM,
			cap: &HardwareCapability{
				GPUVendor:  VendorNVIDIA,
				GPUModel:   "RTX 6000",
				ComputeCap: "8.9",
			},
			minScore: 70,
			maxScore: 89,
		},
		{
			name: "Tier 4 consumer GPU",
			tier: Tier4Standard,
			cap: &HardwareCapability{
				GPUVendor:    VendorNVIDIA,
				GPUModel:     "RTX 4090",
				GPUCCEnabled: false,
				ComputeCap:   "8.9",
			},
			minScore: 10,
			maxScore: 49,
		},
		{
			name:     "Tier 4 no capability",
			tier:     Tier4Standard,
			cap:      nil,
			minScore: 10,
			maxScore: 49,
		},
		{
			name: "Tier 3 older GPU",
			tier: Tier3DeviceTEE,
			cap: &HardwareCapability{
				GPUVendor:  VendorNVIDIA,
				GPUModel:   "RTX 3090",
				ComputeCap: "8.0",
			},
			minScore: 50,
			maxScore: 69,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := QuickTrustScore(tt.tier, tt.cap)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("QuickTrustScore() = %v, want between %v and %v",
					score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestValidateProviderScoreTiers(t *testing.T) {
	tests := []struct {
		score   uint8
		tier    CCTier
		wantErr bool
	}{
		{95, Tier1GPUNativeCC, false},
		{90, Tier1GPUNativeCC, false},
		{89, Tier1GPUNativeCC, true},
		{85, Tier2ConfidentialVM, false},
		{70, Tier2ConfidentialVM, false},
		{69, Tier2ConfidentialVM, true},
		{60, Tier3DeviceTEE, false},
		{50, Tier3DeviceTEE, false},
		{49, Tier3DeviceTEE, true},
		{40, Tier4Standard, false},
		{10, Tier4Standard, false},
		{9, Tier4Standard, true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			err := ValidateProviderScore(tt.score, tt.tier)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProviderScore(%d, %s) error = %v, wantErr %v",
					tt.score, tt.tier, err, tt.wantErr)
			}
		})
	}
}

func TestAdjustScoreForSlashingTiers(t *testing.T) {
	tests := []struct {
		current  uint8
		severity float64
		expected uint8
	}{
		{100, 0.1, 90},   // 10% slash
		{100, 0.5, 50},   // 50% slash
		{50, 0.5, 25},    // 50% of 50
		{10, 1.0, 1},     // 100% slash -> minimum 1
		{100, 0.0, 100},  // No slash
		{1, 0.5, 1},      // Already at minimum
		{100, 2.0, 1},    // Over 100% slash -> minimum 1
	}

	for _, tt := range tests {
		got := AdjustScoreForSlashing(tt.current, tt.severity)
		if got != tt.expected {
			t.Errorf("AdjustScoreForSlashing(%v, %v) = %v, want %v",
				tt.current, tt.severity, got, tt.expected)
		}
	}
}

func TestRecoverScoreAfterGoodBehavior(t *testing.T) {
	tests := []struct {
		current  uint8
		max      uint8
		rate     float64
		expected uint8
	}{
		{50, 100, 0.1, 55},   // 10% recovery of remaining 50
		{90, 100, 0.5, 95},   // 50% of remaining 10
		{99, 100, 0.5, 99},   // Small recovery (rounds to 0)
		{100, 100, 0.5, 100}, // Already at max
		{0, 100, 0.1, 10},    // Recovery from 0
		{50, 50, 0.5, 50},    // Already at max for tier
	}

	for _, tt := range tests {
		got := RecoverScoreAfterGoodBehavior(tt.current, tt.max, tt.rate)
		if got != tt.expected {
			t.Errorf("RecoverScoreAfterGoodBehavior(%v, %v, %v) = %v, want %v",
				tt.current, tt.max, tt.rate, got, tt.expected)
		}
	}
}

// =============================================================================
// Hardware Capability Tests
// =============================================================================

func TestHardwareCapability_CanAchieveTier(t *testing.T) {
	tests := []struct {
		name     string
		cap      HardwareCapability
		tier     CCTier
		expected bool
	}{
		{
			name:     "Tier1 max can achieve Tier1",
			cap:      HardwareCapability{MaxTier: Tier1GPUNativeCC},
			tier:     Tier1GPUNativeCC,
			expected: true,
		},
		{
			name:     "Tier1 max can achieve Tier2",
			cap:      HardwareCapability{MaxTier: Tier1GPUNativeCC},
			tier:     Tier2ConfidentialVM,
			expected: true,
		},
		{
			name:     "Tier2 max cannot achieve Tier1",
			cap:      HardwareCapability{MaxTier: Tier2ConfidentialVM},
			tier:     Tier1GPUNativeCC,
			expected: false,
		},
		{
			name:     "Tier4 max can only achieve Tier4",
			cap:      HardwareCapability{MaxTier: Tier4Standard},
			tier:     Tier3DeviceTEE,
			expected: false,
		},
		{
			name:     "Unknown max achieves tier 4 (0 <= 4)",
			cap:      HardwareCapability{MaxTier: TierUnknown},
			tier:     Tier4Standard,
			expected: true, // TierUnknown (0) <= Tier4Standard (4) is true
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cap.CanAchieveTier(tt.tier); got != tt.expected {
				t.Errorf("HardwareCapability.CanAchieveTier(%v) = %v, want %v",
					tt.tier, got, tt.expected)
			}
		})
	}
}

func TestHardwareCapability_GetSupportedTiers(t *testing.T) {
	tests := []struct {
		name        string
		cap         HardwareCapability
		expectedLen int
	}{
		{
			name:        "Tier1 supports all 4 tiers",
			cap:         HardwareCapability{MaxTier: Tier1GPUNativeCC},
			expectedLen: 4,
		},
		{
			name:        "Tier2 supports 3 tiers",
			cap:         HardwareCapability{MaxTier: Tier2ConfidentialVM},
			expectedLen: 3,
		},
		{
			name:        "Tier3 supports 2 tiers",
			cap:         HardwareCapability{MaxTier: Tier3DeviceTEE},
			expectedLen: 2,
		},
		{
			name:        "Tier4 supports 1 tier",
			cap:         HardwareCapability{MaxTier: Tier4Standard},
			expectedLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tiers := tt.cap.GetSupportedTiers()
			if len(tiers) != tt.expectedLen {
				t.Errorf("GetSupportedTiers() returned %d tiers, want %d",
					len(tiers), tt.expectedLen)
			}
		})
	}
}

func TestHardwareCapability_IsMethods(t *testing.T) {
	tests := []struct {
		name           string
		cap            HardwareCapability
		isGPUCC        bool
		isCPUTEE       bool
		isDeviceTEE    bool
		requiresSetup  bool
	}{
		{
			name: "Full GPU CC capable",
			cap: HardwareCapability{
				GPUCCSupported: true,
				GPUCCEnabled:   true,
				NVTrustAvail:   true,
				CPUTEEType:     TEENone, // Explicitly set to None
			},
			isGPUCC:       true,
			isCPUTEE:      false,
			isDeviceTEE:   false,
			requiresSetup: false,
		},
		{
			name: "GPU CC supported but not enabled",
			cap: HardwareCapability{
				GPUCCSupported: true,
				GPUCCEnabled:   false,
				NVTrustAvail:   true,
				CPUTEEType:     TEENone,
			},
			isGPUCC:       true,
			isCPUTEE:      false,
			isDeviceTEE:   false,
			requiresSetup: true,
		},
		{
			name: "GPU CC supported but nvtrust missing",
			cap: HardwareCapability{
				GPUCCSupported: true,
				GPUCCEnabled:   true,
				NVTrustAvail:   false,
				CPUTEEType:     TEENone,
			},
			isGPUCC:       true,
			isCPUTEE:      false,
			isDeviceTEE:   false,
			requiresSetup: true,
		},
		{
			name: "CPU TEE capable - SEV-SNP",
			cap: HardwareCapability{
				CPUTEEType: TEESEVSNP,
			},
			isGPUCC:       false,
			isCPUTEE:      true,
			isDeviceTEE:   false,
			requiresSetup: false,
		},
		{
			name: "CPU TEE capable - TDX",
			cap: HardwareCapability{
				CPUTEEType: TEETDX,
			},
			isGPUCC:       false,
			isCPUTEE:      true,
			isDeviceTEE:   false,
			requiresSetup: false,
		},
		{
			name: "Device TEE capable",
			cap: HardwareCapability{
				DeviceTEEEnabled: true,
				CPUTEEType:       TEENone,
			},
			isGPUCC:       false,
			isCPUTEE:      false,
			isDeviceTEE:   true,
			requiresSetup: false,
		},
		{
			name: "No CC capability",
			cap: HardwareCapability{
				GPUCCSupported: false,
				CPUTEEType:     TEENone,
			},
			isGPUCC:       false,
			isCPUTEE:      false,
			isDeviceTEE:   false,
			requiresSetup: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cap.IsGPUCCCapable(); got != tt.isGPUCC {
				t.Errorf("IsGPUCCCapable() = %v, want %v", got, tt.isGPUCC)
			}
			if got := tt.cap.IsCPUTEECapable(); got != tt.isCPUTEE {
				t.Errorf("IsCPUTEECapable() = %v, want %v", got, tt.isCPUTEE)
			}
			if got := tt.cap.IsDeviceTEECapable(); got != tt.isDeviceTEE {
				t.Errorf("IsDeviceTEECapable() = %v, want %v", got, tt.isDeviceTEE)
			}
			needsSetup, _ := tt.cap.RequiresSetup()
			if needsSetup != tt.requiresSetup {
				t.Errorf("RequiresSetup() = %v, want %v", needsSetup, tt.requiresSetup)
			}
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkCalculateTrustScore(b *testing.B) {
	input := &TrustScoreInput{
		Tier:              Tier1GPUNativeCC,
		GPUGeneration:     10,
		CCFeaturesEnabled: true,
		AttestationAge:    1 * time.Hour,
		AttestationMethod: "nvtrust",
		LocalVerification: true,
		TasksCompleted:    1000,
		UptimePercentage:  99.9,
		LastSeenDelta:     30 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateTrustScore(input)
	}
}

func BenchmarkQuickTrustScore(b *testing.B) {
	cap := &HardwareCapability{
		GPUVendor:    VendorNVIDIA,
		GPUModel:     "H100",
		GPUCCEnabled: true,
		ComputeCap:   "9.0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		QuickTrustScore(Tier1GPUNativeCC, cap)
	}
}

func BenchmarkParseTier(b *testing.B) {
	inputs := []interface{}{
		uint8(1),
		int(2),
		"GPU-Native-CC",
		"tier3",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseTier(inputs[i%len(inputs)])
	}
}
