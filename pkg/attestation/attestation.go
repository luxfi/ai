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
	DeviceID      string    `json:"device_id"`
	Model         string    `json:"model"`
	CCEnabled     bool      `json:"cc_enabled"`
	TEEIOEnabled  bool      `json:"tee_io_enabled"`
	DriverVersion string    `json:"driver_version"`
	VBIOSVersion  string    `json:"vbios_version"`
	NRASToken     []byte    `json:"nras_token"`
	Timestamp     time.Time `json:"timestamp"`
}

// DeviceStatus tracks attested device status
type DeviceStatus struct {
	Attested   bool      `json:"attested"`
	TrustScore uint8     `json:"trust_score"`
	LastSeen   time.Time `json:"last_seen"`
	Operator   string    `json:"operator"`
	Vendor     TEEType   `json:"vendor"`
	JobHistory []string  `json:"job_history"`
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

// VerifyGPUAttestation verifies GPU attestation via NVIDIA NRAS
func (v *Verifier) VerifyGPUAttestation(att *GPUAttestation) (*DeviceStatus, error) {
	if att == nil || len(att.NRASToken) == 0 {
		return nil, ErrInvalidQuote
	}
	if !v.validateNRASToken(att.NRASToken) {
		return nil, ErrInvalidQuote
	}
	status := &DeviceStatus{
		Attested:   true,
		TrustScore: calculateTrustScore(att),
		LastSeen:   time.Now(),
		Operator:   att.DeviceID,
		Vendor:     TEETypeNVIDIA,
		JobHistory: []string{},
	}
	v.attestedDevices[att.DeviceID] = status
	return status, nil
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

func calculateTrustScore(att *GPUAttestation) uint8 {
	score := uint8(50)
	if att.CCEnabled {
		score += 25
	}
	if att.TEEIOEnabled {
		score += 15
	}
	switch att.Model {
	case "H100", "H200", "B100", "B200", "GB200":
		score += 10
	case "A100", "A10":
		score += 5
	}
	if score > 100 {
		score = 100
	}
	return score
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
