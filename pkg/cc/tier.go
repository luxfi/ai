// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package cc implements Confidential Compute tier classification for the Lux AI Network.
// This package defines the 3-tier CC system per LP-5610:
//
//   Tier 1 — "GPU-native CC": NVIDIA Blackwell, Hopper, RTX PRO 6000 with NVTrust
//   Tier 2 — "Confidential VM + GPU": AMD SEV-SNP, Intel TDX, Arm CCA + GPU
//   Tier 3 — "Device TEE + AI engine": Qualcomm TrustZone/SPU, Apple Secure Enclave
//   Tier 4 — "Standard" (non-CC): Consumer GPUs, stake-based soft attestation
//
// All attestation is LOCAL - no cloud dependencies (blockchain requirement).
// See: https://github.com/luxfi/lps/blob/main/LPs/lp-5610-ai-confidential-compute-tiers.md
package cc

import (
	"errors"
	"fmt"
	"time"
)

// CCTier represents the Confidential Compute tier classification
// per LP-5610: AI Confidential Compute Tier Specification
type CCTier uint8

const (
	// Tier1GPUNativeCC is the highest trust tier: GPU-native confidential compute
	// Hardware: NVIDIA Blackwell (B100/B200/GB200), Hopper (H100/H200), RTX PRO 6000
	// Attestation: Hardware GPU Quote via NVTrust (local verification)
	// Trust Score: 90-100
	Tier1GPUNativeCC CCTier = 1

	// Tier2ConfidentialVM is CPU-level VM isolation with GPU passthrough
	// Hardware: AMD EPYC + SEV-SNP, Intel Xeon + TDX, Arm Neoverse + CCA
	// Attestation: CPU TEE Report + GPU passthrough
	// Trust Score: 70-89
	Tier2ConfidentialVM CCTier = 2

	// Tier3DeviceTEE is edge device TEE with integrated AI accelerator
	// Hardware: Qualcomm Snapdragon (TrustZone/SPU), Apple Silicon (Secure Enclave)
	// Attestation: Device TEE Quote
	// Trust Score: 50-69
	Tier3DeviceTEE CCTier = 3

	// Tier4Standard is non-CC hardware with software/stake-based attestation
	// Hardware: Consumer GPUs (RTX 4090/5090), Cloud VMs without CC
	// Attestation: Software + stake-based
	// Trust Score: 10-49
	Tier4Standard CCTier = 4

	// TierUnknown indicates unclassified or invalid tier
	TierUnknown CCTier = 0
)

// String returns the human-readable name for the CC tier
func (t CCTier) String() string {
	switch t {
	case Tier1GPUNativeCC:
		return "GPU-Native-CC"
	case Tier2ConfidentialVM:
		return "Confidential-VM"
	case Tier3DeviceTEE:
		return "Device-TEE"
	case Tier4Standard:
		return "Standard"
	default:
		return "Unknown"
	}
}

// Description returns a detailed description of the tier
func (t CCTier) Description() string {
	switch t {
	case Tier1GPUNativeCC:
		return "Full GPU-level hardware confidential compute with NVTrust attestation"
	case Tier2ConfidentialVM:
		return "CPU-level VM isolation (SEV-SNP/TDX/CCA) with GPU passthrough"
	case Tier3DeviceTEE:
		return "Edge device TEE with integrated AI accelerator"
	case Tier4Standard:
		return "Software/stake-based attestation without hardware CC"
	default:
		return "Unknown tier classification"
	}
}

// BaseTrustScore returns the base trust score for the tier
func (t CCTier) BaseTrustScore() uint8 {
	switch t {
	case Tier1GPUNativeCC:
		return 90
	case Tier2ConfidentialVM:
		return 70
	case Tier3DeviceTEE:
		return 50
	case Tier4Standard:
		return 10
	default:
		return 0
	}
}

// MaxTrustScore returns the maximum trust score achievable for the tier
func (t CCTier) MaxTrustScore() uint8 {
	switch t {
	case Tier1GPUNativeCC:
		return 100
	case Tier2ConfidentialVM:
		return 89
	case Tier3DeviceTEE:
		return 69
	case Tier4Standard:
		return 49
	default:
		return 0
	}
}

// MinStakeLUX returns the minimum stake required for the tier in LUX tokens
// Tier 1: 100,000 LUX, Tier 2: 50,000 LUX, Tier 3: 10,000 LUX, Tier 4: 1,000 LUX
// Note: For wei conversion, multiply by 1e18 using big.Int
func (t CCTier) MinStakeLUX() uint64 {
	switch t {
	case Tier1GPUNativeCC:
		return 100_000 // 100,000 LUX
	case Tier2ConfidentialVM:
		return 50_000 // 50,000 LUX
	case Tier3DeviceTEE:
		return 10_000 // 10,000 LUX
	case Tier4Standard:
		return 1_000 // 1,000 LUX
	default:
		return 0
	}
}

// RewardMultiplier returns the reward multiplier for the tier
// Tier 1: 1.5x, Tier 2: 1.0x, Tier 3: 0.75x, Tier 4: 0.5x
func (t CCTier) RewardMultiplier() float64 {
	switch t {
	case Tier1GPUNativeCC:
		return 1.5
	case Tier2ConfidentialVM:
		return 1.0
	case Tier3DeviceTEE:
		return 0.75
	case Tier4Standard:
		return 0.5
	default:
		return 0.0
	}
}

// AttestationValidity returns the attestation validity period for the tier
func (t CCTier) AttestationValidity() time.Duration {
	switch t {
	case Tier1GPUNativeCC:
		return 6 * time.Hour // Re-attest every 6 hours for Tier 1
	case Tier2ConfidentialVM:
		return 24 * time.Hour // Re-attest every 24 hours for Tier 2
	case Tier3DeviceTEE:
		return 7 * 24 * time.Hour // Re-attest every 7 days for Tier 3
	case Tier4Standard:
		return 30 * 24 * time.Hour // Re-attest every 30 days for Tier 4
	default:
		return 0
	}
}

// MeetsTierRequirement checks if this tier meets or exceeds the required tier
func (t CCTier) MeetsTierRequirement(required CCTier) bool {
	// Lower tier number = higher security
	// Tier 1 meets all requirements, Tier 4 only meets Tier 4 requirement
	return t != TierUnknown && t <= required
}

// Errors for tier operations
var (
	ErrInvalidTier         = errors.New("invalid CC tier")
	ErrTierNotMet          = errors.New("provider tier does not meet requirement")
	ErrAttestationExpired  = errors.New("attestation has expired")
	ErrInvalidAttestation  = errors.New("invalid attestation evidence")
	ErrInsufficientStake   = errors.New("insufficient stake for tier")
	ErrHardwareNotSupported = errors.New("hardware does not support required CC tier")
)

// TierAttestation represents an attestation bound to a specific CC tier
type TierAttestation struct {
	// Tier is the CC tier classification
	Tier CCTier `json:"tier"`

	// ProviderID is the unique identifier of the compute provider
	ProviderID string `json:"provider_id"`

	// HardwareID is the unique hardware identifier (GPU serial, etc.)
	HardwareID string `json:"hardware_id"`

	// EvidenceHash is the hash of the attestation evidence (for on-chain anchoring)
	EvidenceHash [32]byte `json:"evidence_hash"`

	// TrustScore is the calculated trust score (0-100)
	TrustScore uint8 `json:"trust_score"`

	// IssuedAt is when the attestation was issued
	IssuedAt time.Time `json:"issued_at"`

	// ExpiresAt is when the attestation expires
	ExpiresAt time.Time `json:"expires_at"`

	// ChainID is the chain where this attestation is registered
	ChainID uint64 `json:"chain_id"`

	// BlockHeight is the block height at registration
	BlockHeight uint64 `json:"block_height"`

	// HardwareInfo contains hardware-specific information
	HardwareInfo *HardwareInfo `json:"hardware_info,omitempty"`
}

// HardwareInfo contains hardware-specific information for attestation
type HardwareInfo struct {
	// Vendor is the hardware vendor (NVIDIA, AMD, Intel, Apple, Qualcomm)
	Vendor string `json:"vendor"`

	// Model is the hardware model (H100, B200, MI300X, M4, etc.)
	Model string `json:"model"`

	// Serial is the hardware serial number
	Serial string `json:"serial"`

	// DriverVersion is the driver version
	DriverVersion string `json:"driver_version"`

	// FirmwareVersion is the firmware/VBIOS version
	FirmwareVersion string `json:"firmware_version"`

	// CCEnabled indicates if CC features are enabled
	CCEnabled bool `json:"cc_enabled"`

	// TEEIOEnabled indicates if TEE-IO features are enabled (for GPUs)
	TEEIOEnabled bool `json:"tee_io_enabled"`

	// ComputeCapability for GPUs (e.g., "9.0" for Blackwell)
	ComputeCapability string `json:"compute_capability,omitempty"`

	// MemorySize in bytes
	MemorySize uint64 `json:"memory_size"`
}

// IsValid checks if the attestation is currently valid
func (a *TierAttestation) IsValid() bool {
	if a.Tier == TierUnknown {
		return false
	}
	now := time.Now()
	return now.After(a.IssuedAt) && now.Before(a.ExpiresAt)
}

// IsExpired checks if the attestation has expired
func (a *TierAttestation) IsExpired() bool {
	return time.Now().After(a.ExpiresAt)
}

// TimeUntilExpiry returns the duration until the attestation expires
func (a *TierAttestation) TimeUntilExpiry() time.Duration {
	return time.Until(a.ExpiresAt)
}

// MeetsTierRequirement checks if this attestation meets the required tier
func (a *TierAttestation) MeetsTierRequirement(required CCTier) error {
	if !a.IsValid() {
		return ErrAttestationExpired
	}
	if !a.Tier.MeetsTierRequirement(required) {
		return fmt.Errorf("%w: have %s, need %s", ErrTierNotMet, a.Tier, required)
	}
	return nil
}

// ParseTier parses a tier from a string or uint8
func ParseTier(value interface{}) (CCTier, error) {
	switch v := value.(type) {
	case uint8:
		return parseTierUint8(v)
	case int:
		return parseTierUint8(uint8(v))
	case string:
		return parseTierString(v)
	default:
		return TierUnknown, ErrInvalidTier
	}
}

func parseTierUint8(v uint8) (CCTier, error) {
	if v >= 1 && v <= 4 {
		return CCTier(v), nil
	}
	return TierUnknown, ErrInvalidTier
}

func parseTierString(v string) (CCTier, error) {
	switch v {
	case "1", "tier1", "Tier1", "GPU-Native-CC", "gpu-native-cc":
		return Tier1GPUNativeCC, nil
	case "2", "tier2", "Tier2", "Confidential-VM", "confidential-vm":
		return Tier2ConfidentialVM, nil
	case "3", "tier3", "Tier3", "Device-TEE", "device-tee":
		return Tier3DeviceTEE, nil
	case "4", "tier4", "Tier4", "Standard", "standard":
		return Tier4Standard, nil
	default:
		return TierUnknown, ErrInvalidTier
	}
}

// TierRequirement defines minimum requirements for task execution
type TierRequirement struct {
	// MinTier is the minimum CC tier required
	MinTier CCTier `json:"min_tier"`

	// RequireValidAttestation requires the attestation to be currently valid
	RequireValidAttestation bool `json:"require_valid_attestation"`

	// MaxAttestationAge is the maximum age of the attestation
	MaxAttestationAge time.Duration `json:"max_attestation_age"`

	// MinTrustScore is the minimum trust score required
	MinTrustScore uint8 `json:"min_trust_score"`

	// RequireSpecificVendor requires a specific hardware vendor
	RequireSpecificVendor string `json:"require_specific_vendor,omitempty"`

	// RequireMinMemory is the minimum GPU memory required (in bytes)
	RequireMinMemory uint64 `json:"require_min_memory,omitempty"`
}

// DefaultTierRequirement returns default requirements for a tier
func DefaultTierRequirement(tier CCTier) *TierRequirement {
	return &TierRequirement{
		MinTier:                 tier,
		RequireValidAttestation: true,
		MaxAttestationAge:       tier.AttestationValidity(),
		MinTrustScore:           tier.BaseTrustScore(),
	}
}

// IsMet checks if the requirement is met by the given attestation
func (r *TierRequirement) IsMet(attestation *TierAttestation) error {
	if attestation == nil {
		return ErrInvalidAttestation
	}

	// Check tier requirement
	if err := attestation.MeetsTierRequirement(r.MinTier); err != nil {
		return err
	}

	// Check attestation validity
	if r.RequireValidAttestation && !attestation.IsValid() {
		return ErrAttestationExpired
	}

	// Check attestation age
	if r.MaxAttestationAge > 0 {
		age := time.Since(attestation.IssuedAt)
		if age > r.MaxAttestationAge {
			return fmt.Errorf("%w: attestation age %v exceeds max %v", ErrAttestationExpired, age, r.MaxAttestationAge)
		}
	}

	// Check trust score
	if attestation.TrustScore < r.MinTrustScore {
		return fmt.Errorf("%w: trust score %d below minimum %d", ErrTierNotMet, attestation.TrustScore, r.MinTrustScore)
	}

	// Check vendor requirement
	if r.RequireSpecificVendor != "" && attestation.HardwareInfo != nil {
		if attestation.HardwareInfo.Vendor != r.RequireSpecificVendor {
			return fmt.Errorf("%w: requires vendor %s, have %s", ErrHardwareNotSupported, r.RequireSpecificVendor, attestation.HardwareInfo.Vendor)
		}
	}

	// Check memory requirement
	if r.RequireMinMemory > 0 && attestation.HardwareInfo != nil {
		if attestation.HardwareInfo.MemorySize < r.RequireMinMemory {
			return fmt.Errorf("%w: requires %d bytes memory, have %d", ErrHardwareNotSupported, r.RequireMinMemory, attestation.HardwareInfo.MemorySize)
		}
	}

	return nil
}
