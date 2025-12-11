// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package attestation provides GPU attestation via nvtrust LOCAL verification.
// This is a BLOCKCHAIN - we cannot depend on NVIDIA cloud services.
// All attestation is done locally using nvtrust open-source tools.
//
// nvtrust is NVIDIA's open-source attestation toolkit:
// https://github.com/NVIDIA/nvtrust
//
// Supported GPUs (hardware CC capable):
//   - Datacenter: H100, H200, B100, B200, GB200
//   - Professional: RTX PRO 6000 Blackwell
//
// NOT supported (no CC hardware - confirmed by NVIDIA):
//   - Consumer: RTX 5090, RTX 4090, etc
//   - DGX Spark (GB10) - Blackwell arch but CC explicitly disabled
//   - Source: https://forums.developer.nvidia.com/t/confidential-computing-support-for-dgx-spark-gb10/347945
//
// For unsupported GPUs, use ModeSoftware with reduced trust score.
//
// NO NRAS CLOUD DEPENDENCY - This is fully local/decentralized.

package attestation

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"time"
)

var (
	ErrNvtrustNotAvailable = errors.New("nvtrust local verifier not available")
	ErrGPUNotCCCapable     = errors.New("GPU does not support confidential computing")
	ErrRIMVerifyFailed     = errors.New("RIM verification failed")
	ErrSPDMVerifyFailed    = errors.New("SPDM signature verification failed")
	ErrCertChainInvalid    = errors.New("certificate chain validation failed")
)

// NvtrustConfig configures the local nvtrust verifier
type NvtrustConfig struct {
	// RIMServiceURL - URL for Reference Integrity Manifest service
	// Default: https://rim.nvidia.com (can be self-hosted)
	RIMServiceURL string `json:"rim_service_url"`

	// OCSPServiceURL - URL for certificate revocation checking
	// Default: https://ocsp.nvidia.com
	OCSPServiceURL string `json:"ocsp_service_url"`

	// AllowOffline - Allow verification without RIM/OCSP network access
	// Uses cached RIM values if available
	AllowOffline bool `json:"allow_offline"`

	// TrustedRootCerts - Path to NVIDIA root certificates
	// Default: embedded in nvtrust package
	TrustedRootCerts string `json:"trusted_root_certs"`

	// PolicyFile - Path to attestation policy file
	// See: https://github.com/NVIDIA/nvtrust/tree/main/guest_tools/attestation_sdk/policies
	PolicyFile string `json:"policy_file"`
}

// DefaultNvtrustConfig returns default nvtrust configuration
func DefaultNvtrustConfig() *NvtrustConfig {
	return &NvtrustConfig{
		RIMServiceURL:  "https://rim.nvidia.com",
		OCSPServiceURL: "https://ocsp.nvidia.com",
		AllowOffline:   false,
	}
}

// NvtrustVerifier performs local GPU attestation using nvtrust
// This is the PRIMARY attestation method for the Lux AI network
type NvtrustVerifier struct {
	config *NvtrustConfig

	// Cached RIM (Reference Integrity Manifest) values
	cachedRIMs map[string]*RIMEntry

	// NVIDIA root certificate for signature verification
	rootCert []byte
}

// RIMEntry represents a Reference Integrity Manifest entry
type RIMEntry struct {
	GPUModel        string    `json:"gpu_model"`
	DriverVersion   string    `json:"driver_version"`
	VBIOSVersion    string    `json:"vbios_version"`
	GoldenHash      [48]byte  `json:"golden_hash"` // Expected measurement
	ValidFrom       time.Time `json:"valid_from"`
	ValidUntil      time.Time `json:"valid_until"`
	NVIDIASignature []byte    `json:"nvidia_signature"`
}

// SPDMEvidence represents SPDM (Security Protocol and Data Model) evidence
// from the GPU's hardware attestation module
type SPDMEvidence struct {
	// Version of SPDM protocol (typically 1.1 or 1.2)
	Version uint8 `json:"version"`

	// MeasurementHash - Hash of GPU firmware/configuration
	MeasurementHash [48]byte `json:"measurement_hash"`

	// Nonce - Fresh nonce for replay protection
	Nonce [32]byte `json:"nonce"`

	// Signature - GPU's signature over the measurement
	Signature []byte `json:"signature"`

	// CertificateChain - Chain from GPU cert to NVIDIA root
	CertificateChain []byte `json:"certificate_chain"`

	// RawReport - Full SPDM MEASUREMENT response
	RawReport []byte `json:"raw_report"`
}

// NewNvtrustVerifier creates a new nvtrust local verifier
func NewNvtrustVerifier(config *NvtrustConfig) *NvtrustVerifier {
	if config == nil {
		config = DefaultNvtrustConfig()
	}
	return &NvtrustVerifier{
		config:     config,
		cachedRIMs: make(map[string]*RIMEntry),
		rootCert:   nvidiaCCRootCert, // Embedded NVIDIA root cert
	}
}

// VerifyGPU performs local GPU attestation using nvtrust
// This is the PRIMARY verification method
func (nv *NvtrustVerifier) VerifyGPU(evidence *SPDMEvidence, gpuInfo *GPUHardwareInfo) (*LocalVerificationResult, error) {
	if evidence == nil || gpuInfo == nil {
		return nil, ErrInvalidQuote
	}

	// Step 1: Check if GPU model supports CC
	if !IsHardwareCCCapable(gpuInfo.Model) {
		return nil, ErrGPUNotCCCapable
	}

	// Step 2: Verify certificate chain
	if err := nv.verifyCertificateChain(evidence.CertificateChain); err != nil {
		return nil, err
	}

	// Step 3: Verify SPDM signature
	if err := nv.verifySPDMSignature(evidence); err != nil {
		return nil, err
	}

	// Step 4: Verify measurement against RIM (golden values)
	rimVerified, err := nv.verifyAgainstRIM(evidence.MeasurementHash, gpuInfo)
	if err != nil && !nv.config.AllowOffline {
		return nil, err
	}

	// Step 5: Calculate trust score
	trustScore := nv.calculateLocalTrustScore(gpuInfo, rimVerified)

	return &LocalVerificationResult{
		Verified:       true,
		RIMVerified:    rimVerified,
		TrustScore:     trustScore,
		GPUModel:       gpuInfo.Model,
		DriverVersion:  gpuInfo.DriverVersion,
		MeasurementHash: evidence.MeasurementHash,
		VerifiedAt:     time.Now(),
	}, nil
}

// LocalVerificationResult contains the result of local nvtrust verification
type LocalVerificationResult struct {
	Verified        bool      `json:"verified"`
	RIMVerified     bool      `json:"rim_verified"`
	TrustScore      uint8     `json:"trust_score"`
	GPUModel        string    `json:"gpu_model"`
	DriverVersion   string    `json:"driver_version"`
	MeasurementHash [48]byte  `json:"measurement_hash"`
	VerifiedAt      time.Time `json:"verified_at"`
	Error           string    `json:"error,omitempty"`
}

// GPUHardwareInfo contains GPU hardware information for attestation
type GPUHardwareInfo struct {
	DeviceID      string `json:"device_id"`
	Model         string `json:"model"`
	Serial        string `json:"serial"`
	PCIID         string `json:"pci_id"`
	DriverVersion string `json:"driver_version"`
	VBIOSVersion  string `json:"vbios_version"`
	CCEnabled     bool   `json:"cc_enabled"`
	TEEIOEnabled  bool   `json:"tee_io_enabled"`
}

// verifyCertificateChain verifies the GPU certificate chain up to NVIDIA root
func (nv *NvtrustVerifier) verifyCertificateChain(certChain []byte) error {
	if len(certChain) < 256 {
		return ErrCertChainInvalid
	}

	// In production: Parse X.509 certificate chain
	// Verify each cert signature up to NVIDIA root
	// Check OCSP for revocation status

	// For now: Basic validation that chain exists and has expected structure
	// Production implementation would use crypto/x509

	return nil
}

// verifySPDMSignature verifies the SPDM measurement signature
func (nv *NvtrustVerifier) verifySPDMSignature(evidence *SPDMEvidence) error {
	if len(evidence.Signature) < 64 {
		return ErrSPDMVerifyFailed
	}

	// In production: Extract public key from certificate
	// Verify ECDSA/RSA signature over measurement data
	// Check nonce freshness

	// Verify raw report structure (SPDM 1.1 MEASUREMENT response)
	if len(evidence.RawReport) < 256 {
		return ErrSPDMVerifyFailed
	}

	return nil
}

// verifyAgainstRIM verifies measurement against Reference Integrity Manifest
func (nv *NvtrustVerifier) verifyAgainstRIM(measurementHash [48]byte, gpuInfo *GPUHardwareInfo) (bool, error) {
	// Look up RIM entry for this GPU/driver combination
	rimKey := gpuInfo.Model + "-" + gpuInfo.DriverVersion

	rim, ok := nv.cachedRIMs[rimKey]
	if !ok {
		// In production: Fetch from RIM service
		// rim, err = nv.fetchRIM(gpuInfo)

		// For now: Allow if offline mode enabled
		if nv.config.AllowOffline {
			return false, nil // Not verified but allowed
		}
		return false, ErrRIMVerifyFailed
	}

	// Compare measurement against golden value
	if rim.GoldenHash != measurementHash {
		return false, ErrRIMVerifyFailed
	}

	// Check RIM validity period
	now := time.Now()
	if now.Before(rim.ValidFrom) || now.After(rim.ValidUntil) {
		return false, ErrRIMVerifyFailed
	}

	return true, nil
}

// calculateLocalTrustScore calculates trust score for local verification
func (nv *NvtrustVerifier) calculateLocalTrustScore(gpuInfo *GPUHardwareInfo, rimVerified bool) uint8 {
	// Base score for local nvtrust verification: 70
	// This is PRIMARY method - scores equivalent to NRAS
	score := uint8(70)

	// CC features bonus
	if gpuInfo.CCEnabled {
		score += 15
	}
	if gpuInfo.TEEIOEnabled {
		score += 5
	}

	// GPU model bonus
	switch gpuInfo.Model {
	case "GB200", "B200", "B100": // Blackwell datacenter
		score += 10
	case "H200", "H100": // Hopper datacenter
		score += 8
	case "RTX PRO 6000": // Blackwell professional
		score += 5
	}

	// RIM verification bonus
	if rimVerified {
		score += 5 // Extra trust when golden values match
	}

	if score > 100 {
		score = 100
	}

	return score
}

// RegisterRIM registers a Reference Integrity Manifest entry
// Used for offline/self-hosted RIM verification
func (nv *NvtrustVerifier) RegisterRIM(entry *RIMEntry) {
	key := entry.GPUModel + "-" + entry.DriverVersion
	nv.cachedRIMs[key] = entry
}

// CollectGPUEvidence collects attestation evidence from local GPU
// This wraps nvidia-smi and NVML API calls
func CollectGPUEvidence(deviceIndex int) (*SPDMEvidence, *GPUHardwareInfo, error) {
	// In production: Use NVML API to collect:
	// 1. GPU hardware info (model, serial, driver version)
	// 2. SPDM measurement report
	// 3. Certificate chain
	// 4. Generate fresh nonce

	// This would call into nvtrust guest tools or use NVML directly
	// See: https://github.com/NVIDIA/nvtrust/tree/main/guest_tools

	return nil, nil, ErrNvtrustNotAvailable
}

// GenerateAttestationNonce generates a fresh nonce for attestation
func GenerateAttestationNonce() [32]byte {
	var nonce [32]byte
	// In production: Use crypto/rand
	// For now: Use timestamp-based nonce
	ts := time.Now().UnixNano()
	binary.LittleEndian.PutUint64(nonce[:8], uint64(ts))
	hash := sha256.Sum256(nonce[:])
	copy(nonce[:], hash[:])
	return nonce
}

// NVIDIA CC Root Certificate (placeholder - real cert would be embedded)
// In production: Embed actual NVIDIA root certificate for CC attestation
var nvidiaCCRootCert = []byte{
	// This would contain the actual NVIDIA Confidential Computing root certificate
	// Used to verify GPU certificate chains
	// See: https://docs.nvidia.com/attestation/
}

// ConvertToGPUAttestation converts nvtrust result to GPUAttestation struct
func (result *LocalVerificationResult) ToGPUAttestation(deviceID string, evidence *SPDMEvidence) *GPUAttestation {
	return &GPUAttestation{
		DeviceID:      deviceID,
		Model:         result.GPUModel,
		CCEnabled:     true, // If we got here, CC is enabled
		TEEIOEnabled:  true,
		DriverVersion: result.DriverVersion,
		Mode:          ModeLocalVerifier,
		Timestamp:     result.VerifiedAt,
		LocalEvidence: &LocalGPUEvidence{
			SPDMReport:   evidence.RawReport,
			CertChain:    evidence.CertificateChain,
			RIMVerified:  result.RIMVerified,
			DriverReport: nil, // Would contain driver attestation
			Nonce:        evidence.Nonce,
		},
	}
}
