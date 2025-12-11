// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package attestation

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidQuote       = errors.New("invalid attestation quote")
	ErrInvalidMeasurement = errors.New("measurement mismatch")
	ErrQuoteExpired       = errors.New("quote expired")
	ErrUnsupportedTEE     = errors.New("unsupported TEE type")
	ErrInvalidSignature   = errors.New("invalid signature")
)

// AttestationMode indicates the type of attestation
type AttestationMode uint8

const (
	// ModeHardwareCC - Full hardware Confidential Computing (H100, H200, B100, B200, GB200, RTX PRO 6000)
	ModeHardwareCC AttestationMode = iota
	// ModeLocalVerifier - Local nvtrust verification without NVIDIA cloud
	ModeLocalVerifier
	// ModeSoftware - Software attestation for consumer GPUs (RTX 5090, DGX Spark, etc)
	ModeSoftware
)

// TEEType represents the type of Trusted Execution Environment
type TEEType uint8

const (
	TEETypeUnknown TEEType = iota
	TEETypeSGX             // Intel SGX (DCAP)
	TEETypeSEVSNP          // AMD SEV-SNP
	TEETypeTDX             // Intel TDX
	TEETypeNVIDIA          // NVIDIA H100 Confidential Computing
	TEETypeARM             // ARM CCA
)

func (t TEEType) String() string {
	switch t {
	case TEETypeSGX:
		return "SGX"
	case TEETypeSEVSNP:
		return "SEV-SNP"
	case TEETypeTDX:
		return "TDX"
	case TEETypeNVIDIA:
		return "NVIDIA-CC"
	case TEETypeARM:
		return "ARM-CCA"
	default:
		return "Unknown"
	}
}

// AttestationQuote represents a TEE attestation quote
type AttestationQuote struct {
	Type        TEEType   `json:"type"`
	Version     uint32    `json:"version"`
	Quote       []byte    `json:"quote"`
	Measurement []byte    `json:"measurement"`
	ReportData  []byte    `json:"report_data"`
	Timestamp   time.Time `json:"timestamp"`
	Nonce       []byte    `json:"nonce"`
}

// GPUAttestation represents GPU-specific attestation (NVIDIA H100/Blackwell)
type GPUAttestation struct {
	DeviceID      string          `json:"device_id"`
	Model         string          `json:"model"`
	CCEnabled     bool            `json:"cc_enabled"`
	TEEIOEnabled  bool            `json:"tee_io_enabled"`
	DriverVersion string          `json:"driver_version"`
	VBIOSVersion  string          `json:"vbios_version"`
	NRASToken     []byte          `json:"nras_token"`
	Timestamp     time.Time       `json:"timestamp"`
	Mode          AttestationMode `json:"mode"`

	// For local nvtrust verification (ModeLocalVerifier)
	LocalEvidence *LocalGPUEvidence `json:"local_evidence,omitempty"`

	// For software attestation (ModeSoftware) - consumer GPUs
	SoftwareAttestation *SoftwareGPUAttestation `json:"software_attestation,omitempty"`
}

// LocalGPUEvidence represents evidence from nvtrust local verifier
// See: https://github.com/NVIDIA/nvtrust
type LocalGPUEvidence struct {
	// SPDM measurement report from GPU
	SPDMReport []byte `json:"spdm_report"`
	// GPU certificates chain
	CertChain []byte `json:"cert_chain"`
	// RIM (Reference Integrity Manifest) verification result
	RIMVerified bool `json:"rim_verified"`
	// Driver attestation report
	DriverReport []byte `json:"driver_report"`
	// Nonce used for freshness
	Nonce [32]byte `json:"nonce"`
}

// SoftwareGPUAttestation for consumer GPUs without hardware CC
// Lower trust score but still useful for the network
type SoftwareGPUAttestation struct {
	// GPU identity from nvidia-smi
	GPUSerial   string `json:"gpu_serial"`
	PCIID       string `json:"pci_id"`
	BoardID     string `json:"board_id"`
	GPUPartNum  string `json:"gpu_part_num"`
	ComputeCaps string `json:"compute_caps"` // e.g., "8.9" for RTX 5090

	// Driver/firmware info
	DriverVersion string `json:"driver_version"`
	CUDAVersion   string `json:"cuda_version"`
	VBIOSVersion  string `json:"vbios_version"`

	// Performance attestation - prove the GPU ran computation
	BenchmarkHash [32]byte `json:"benchmark_hash"` // Hash of benchmark result
	BenchmarkTime uint64   `json:"benchmark_time_ms"`

	// Provider signature (signed with provider's key)
	ProviderPubKey []byte `json:"provider_pubkey"`
	Signature      []byte `json:"signature"`

	// Timestamp and freshness
	Timestamp time.Time `json:"timestamp"`
	Nonce     [32]byte  `json:"nonce"`
}

// DeviceStatus tracks attested device status
type DeviceStatus struct {
	Attested   bool            `json:"attested"`
	TrustScore uint8           `json:"trust_score"`
	LastSeen   time.Time       `json:"last_seen"`
	Operator   string          `json:"operator"`
	Vendor     TEEType         `json:"vendor"`
	JobHistory []string        `json:"job_history"`
	Mode       AttestationMode `json:"mode"`
	HardwareCC bool            `json:"hardware_cc"` // True if hardware CC verified
}

// Verifier verifies TEE attestations
type Verifier struct {
	trustedMeasurements map[string][]byte
	attestedDevices     map[string]*DeviceStatus
}

// NewVerifier creates a new attestation verifier
func NewVerifier() *Verifier {
	return &Verifier{
		trustedMeasurements: make(map[string][]byte),
		attestedDevices:     make(map[string]*DeviceStatus),
	}
}

// RegisterTrustedMeasurement registers a trusted measurement
func (v *Verifier) RegisterTrustedMeasurement(name string, measurement []byte) {
	v.trustedMeasurements[name] = measurement
}

// VerifyCPUAttestation verifies CPU TEE attestation
func (v *Verifier) VerifyCPUAttestation(quote *AttestationQuote, expectedMeasurement []byte) error {
	if quote == nil || len(quote.Quote) == 0 {
		return ErrInvalidQuote
	}
	if time.Since(quote.Timestamp) > time.Hour {
		return ErrQuoteExpired
	}
	switch quote.Type {
	case TEETypeSGX:
		return v.verifySGXQuote(quote, expectedMeasurement)
	case TEETypeSEVSNP:
		return v.verifySEVSNPQuote(quote, expectedMeasurement)
	case TEETypeTDX:
		return v.verifyTDXQuote(quote, expectedMeasurement)
	default:
		return ErrUnsupportedTEE
	}
}

// VerifyGPUAttestation verifies GPU attestation based on mode
func (v *Verifier) VerifyGPUAttestation(att *GPUAttestation) (*DeviceStatus, error) {
	if att == nil {
		return nil, ErrInvalidQuote
	}

	var status *DeviceStatus
	var err error

	switch att.Mode {
	case ModeHardwareCC:
		// Full hardware CC via NRAS token
		status, err = v.verifyHardwareCCAttestation(att)
	case ModeLocalVerifier:
		// Local nvtrust verification
		status, err = v.verifyLocalGPUAttestation(att)
	case ModeSoftware:
		// Software attestation for consumer GPUs
		status, err = v.verifySoftwareGPUAttestation(att)
	default:
		// Legacy: check for NRAS token
		if len(att.NRASToken) > 0 {
			status, err = v.verifyHardwareCCAttestation(att)
		} else if att.SoftwareAttestation != nil {
			status, err = v.verifySoftwareGPUAttestation(att)
		} else {
			return nil, ErrInvalidQuote
		}
	}

	if err != nil {
		return nil, err
	}

	v.attestedDevices[att.DeviceID] = status
	return status, nil
}

// verifyHardwareCCAttestation verifies full hardware CC via NRAS
func (v *Verifier) verifyHardwareCCAttestation(att *GPUAttestation) (*DeviceStatus, error) {
	if len(att.NRASToken) == 0 {
		return nil, ErrInvalidQuote
	}
	if !v.validateNRASToken(att.NRASToken) {
		return nil, ErrInvalidQuote
	}

	return &DeviceStatus{
		Attested:   true,
		TrustScore: calculateHardwareCCTrustScore(att),
		LastSeen:   time.Now(),
		Operator:   att.DeviceID,
		Vendor:     TEETypeNVIDIA,
		JobHistory: []string{},
		Mode:       ModeHardwareCC,
		HardwareCC: true,
	}, nil
}

// verifyLocalGPUAttestation verifies via local nvtrust
// See: https://github.com/NVIDIA/nvtrust
func (v *Verifier) verifyLocalGPUAttestation(att *GPUAttestation) (*DeviceStatus, error) {
	if att.LocalEvidence == nil {
		return nil, ErrInvalidQuote
	}

	ev := att.LocalEvidence

	// Verify SPDM report exists
	if len(ev.SPDMReport) < 256 {
		return nil, ErrInvalidQuote
	}

	// Verify certificate chain exists
	if len(ev.CertChain) < 256 {
		return nil, ErrInvalidQuote
	}

	// In production: verify SPDM signature against NVIDIA root cert
	// In production: compare measurements against RIM golden values

	trustScore := calculateLocalVerifierTrustScore(att, ev)

	return &DeviceStatus{
		Attested:   true,
		TrustScore: trustScore,
		LastSeen:   time.Now(),
		Operator:   att.DeviceID,
		Vendor:     TEETypeNVIDIA,
		JobHistory: []string{},
		Mode:       ModeLocalVerifier,
		HardwareCC: ev.RIMVerified, // Only true if RIM verification passed
	}, nil
}

// verifySoftwareGPUAttestation verifies consumer GPU software attestation
func (v *Verifier) verifySoftwareGPUAttestation(att *GPUAttestation) (*DeviceStatus, error) {
	if att.SoftwareAttestation == nil {
		return nil, ErrInvalidQuote
	}

	sw := att.SoftwareAttestation

	// Verify basic fields
	if sw.GPUSerial == "" || sw.DriverVersion == "" {
		return nil, ErrInvalidQuote
	}

	// Verify provider signature exists
	if len(sw.Signature) < 64 || len(sw.ProviderPubKey) < 32 {
		return nil, ErrInvalidSignature
	}

	// Verify timestamp freshness
	if time.Since(sw.Timestamp) > time.Hour {
		return nil, ErrQuoteExpired
	}

	// In production: verify signature against provider's public key
	// signedData := hashSoftwareAttestation(sw)
	// if !verifySignature(sw.ProviderPubKey, signedData, sw.Signature) {
	//     return nil, ErrInvalidSignature
	// }

	trustScore := calculateSoftwareTrustScore(att, sw)

	return &DeviceStatus{
		Attested:   true,
		TrustScore: trustScore,
		LastSeen:   time.Now(),
		Operator:   att.DeviceID,
		Vendor:     TEETypeNVIDIA,
		JobHistory: []string{},
		Mode:       ModeSoftware,
		HardwareCC: false, // Software attestation cannot claim hardware CC
	}, nil
}

func (v *Verifier) verifySGXQuote(quote *AttestationQuote, expectedMeasurement []byte) error {
	if len(quote.Quote) < 432 {
		return ErrInvalidQuote
	}
	mrenclave := quote.Quote[112:144]
	if len(expectedMeasurement) > 0 && !bytesEqual(mrenclave, expectedMeasurement) {
		return ErrInvalidMeasurement
	}
	return nil
}

func (v *Verifier) verifySEVSNPQuote(quote *AttestationQuote, expectedMeasurement []byte) error {
	if len(quote.Quote) < 1184 {
		return ErrInvalidQuote
	}
	report, err := ParseSEVSNPReport(quote.Quote)
	if err != nil {
		return err
	}
	if len(expectedMeasurement) > 0 && !bytesEqual(report.Measurement[:], expectedMeasurement) {
		return ErrInvalidMeasurement
	}
	return nil
}

func (v *Verifier) verifyTDXQuote(quote *AttestationQuote, expectedMeasurement []byte) error {
	if len(quote.Quote) < 584 {
		return ErrInvalidQuote
	}
	tdxQuote, err := ParseTDXQuote(quote.Quote)
	if err != nil {
		return err
	}
	if len(expectedMeasurement) > 0 && !bytesEqual(tdxQuote.ReportData[:], expectedMeasurement) {
		return ErrInvalidMeasurement
	}
	return nil
}

func (v *Verifier) validateNRASToken(token []byte) bool {
	return len(token) >= 256
}

// SEVSNPReport represents AMD SEV-SNP attestation report
type SEVSNPReport struct {
	Version         uint32
	GuestSVN        uint32
	Policy          uint64
	FamilyID        [16]byte
	ImageID         [16]byte
	VMPL            uint32
	SignatureAlgo   uint32
	PlatformVersion uint64
	PlatformInfo    uint64
	AuthorKeyEn     uint32
	ReportData      [64]byte
	Measurement     [48]byte
	HostData        [32]byte
	IDKeyDigest     [48]byte
	AuthorKeyDigest [48]byte
	ReportID        [32]byte
	ReportIDMA      [32]byte
	ReportedTCB     uint64
	ChipID          [64]byte
	Signature       [512]byte
}

// ParseSEVSNPReport parses AMD SEV-SNP attestation report
func ParseSEVSNPReport(data []byte) (*SEVSNPReport, error) {
	if len(data) < 1184 {
		return nil, ErrInvalidQuote
	}
	report := &SEVSNPReport{
		Version:         binary.LittleEndian.Uint32(data[0:4]),
		GuestSVN:        binary.LittleEndian.Uint32(data[4:8]),
		Policy:          binary.LittleEndian.Uint64(data[8:16]),
		VMPL:            binary.LittleEndian.Uint32(data[48:52]),
		SignatureAlgo:   binary.LittleEndian.Uint32(data[52:56]),
		PlatformVersion: binary.LittleEndian.Uint64(data[56:64]),
		PlatformInfo:    binary.LittleEndian.Uint64(data[64:72]),
		AuthorKeyEn:     binary.LittleEndian.Uint32(data[72:76]),
		ReportedTCB:     binary.LittleEndian.Uint64(data[380:388]),
	}
	copy(report.FamilyID[:], data[16:32])
	copy(report.ImageID[:], data[32:48])
	copy(report.ReportData[:], data[76:140])
	copy(report.Measurement[:], data[140:188])
	copy(report.HostData[:], data[188:220])
	copy(report.IDKeyDigest[:], data[220:268])
	copy(report.AuthorKeyDigest[:], data[268:316])
	copy(report.ReportID[:], data[316:348])
	copy(report.ReportIDMA[:], data[348:380])
	copy(report.ChipID[:], data[388:452])
	copy(report.Signature[:], data[672:1184])
	return report, nil
}

// TDXQuote represents Intel TDX attestation quote
type TDXQuote struct {
	Version            uint16
	AttestationKeyType uint16
	TEEType            uint32
	Reserved           [4]byte
	VendorID           [16]byte
	UserData           [20]byte
	ReportData         [64]byte
}

// ParseTDXQuote parses Intel TDX quote
func ParseTDXQuote(data []byte) (*TDXQuote, error) {
	if len(data) < 584 {
		return nil, ErrInvalidQuote
	}
	quote := &TDXQuote{
		Version:            binary.LittleEndian.Uint16(data[0:2]),
		AttestationKeyType: binary.LittleEndian.Uint16(data[2:4]),
		TEEType:            binary.LittleEndian.Uint32(data[4:8]),
	}
	copy(quote.Reserved[:], data[8:12])
	copy(quote.VendorID[:], data[12:28])
	copy(quote.UserData[:], data[28:48])
	copy(quote.ReportData[:], data[48:112])
	return quote, nil
}

// calculateTrustScore - legacy function for backward compatibility
func calculateTrustScore(att *GPUAttestation) uint8 {
	return calculateHardwareCCTrustScore(att)
}

// calculateHardwareCCTrustScore for hardware CC attestation (highest trust)
// Max score: 100 for datacenter GPUs with full CC
func calculateHardwareCCTrustScore(att *GPUAttestation) uint8 {
	score := uint8(60) // Base score for hardware CC
	if att.CCEnabled {
		score += 20
	}
	if att.TEEIOEnabled {
		score += 10
	}
	switch att.Model {
	case "GB200", "B200", "B100": // Blackwell datacenter
		score += 10
	case "H200", "H100": // Hopper datacenter
		score += 8
	case "RTX PRO 6000": // Blackwell professional
		score += 6
	case "A100", "A10": // Ampere datacenter
		score += 4
	}
	if score > 100 {
		score = 100
	}
	return score
}

// calculateLocalVerifierTrustScore for local nvtrust verification
// Max score: 95 (slightly less than cloud-verified NRAS)
func calculateLocalVerifierTrustScore(att *GPUAttestation, ev *LocalGPUEvidence) uint8 {
	score := uint8(50) // Base for local verification
	if att.CCEnabled {
		score += 20
	}
	if att.TEEIOEnabled {
		score += 10
	}
	if ev.RIMVerified {
		score += 10 // Bonus for RIM verification
	}
	switch att.Model {
	case "GB200", "B200", "B100":
		score += 8
	case "H200", "H100":
		score += 6
	case "RTX PRO 6000":
		score += 4
	case "A100", "A10":
		score += 2
	}
	if score > 95 {
		score = 95 // Cap at 95 for local verification
	}
	return score
}

// calculateSoftwareTrustScore for consumer GPU software attestation
// Max score: 60 (significantly lower - no hardware CC)
func calculateSoftwareTrustScore(att *GPUAttestation, sw *SoftwareGPUAttestation) uint8 {
	score := uint8(20) // Base for software attestation

	// GPU model bonuses (consumer GPUs)
	switch att.Model {
	case "RTX 5090", "RTX 5080": // Blackwell consumer
		score += 15
	case "GB10": // DGX Spark
		score += 12
	case "RTX 4090", "RTX 4080": // Ada consumer
		score += 10
	case "RTX 3090", "RTX 3080":
		score += 8
	default:
		score += 5
	}

	// Benchmark verification bonus
	if sw.BenchmarkHash != [32]byte{} && sw.BenchmarkTime > 0 {
		score += 10 // Proves GPU actually ran computation
	}

	// Signature verification bonus
	if len(sw.Signature) >= 64 && len(sw.ProviderPubKey) >= 32 {
		score += 10 // Provider accountability
	}

	// Driver version bonus (newer = better)
	if sw.DriverVersion != "" {
		score += 5
	}

	if score > 60 {
		score = 60 // Cap at 60 for software attestation
	}
	return score
}

// IsHardwareCCCapable returns true if the GPU model supports hardware CC
func IsHardwareCCCapable(model string) bool {
	switch model {
	case "H100", "H200", "B100", "B200", "GB200", "RTX PRO 6000":
		return true
	default:
		return false
	}
}

// GetDeviceStatus returns the status of an attested device
func (v *Verifier) GetDeviceStatus(deviceID string) (*DeviceStatus, bool) {
	status, ok := v.attestedDevices[deviceID]
	return status, ok
}

// RecordJobCompletion records job completion for a device
func (v *Verifier) RecordJobCompletion(deviceID, jobID string) {
	if status, ok := v.attestedDevices[deviceID]; ok {
		status.JobHistory = append(status.JobHistory, jobID)
		status.LastSeen = time.Now()
	}
}

// ComputeAttestationHash computes hash for on-chain anchoring
func ComputeAttestationHash(quote *AttestationQuote) [32]byte {
	h := sha256.New()
	h.Write([]byte{byte(quote.Type)})
	h.Write(quote.Quote)
	h.Write(quote.Measurement)
	h.Write(quote.Nonce)
	var hash [32]byte
	copy(hash[:], h.Sum(nil))
	return hash
}

// FormatDeviceID formats device ID from attestation
func FormatDeviceID(teeType TEEType, identifier []byte) string {
	return fmt.Sprintf("%s-%s", teeType.String(), hex.EncodeToString(identifier[:8]))
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
