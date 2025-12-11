// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package attestation

import (
	"testing"
	"time"
)

func TestTEETypeString(t *testing.T) {
	tests := []struct {
		tee      TEEType
		expected string
	}{
		{TEETypeUnknown, "Unknown"},
		{TEETypeSGX, "SGX"},
		{TEETypeSEVSNP, "SEV-SNP"},
		{TEETypeTDX, "TDX"},
		{TEETypeNVIDIA, "NVIDIA-CC"},
		{TEETypeARM, "ARM-CCA"},
	}

	for _, tt := range tests {
		if got := tt.tee.String(); got != tt.expected {
			t.Errorf("TEEType(%d).String() = %s, want %s", tt.tee, got, tt.expected)
		}
	}
}

func TestNewVerifier(t *testing.T) {
	v := NewVerifier()
	if v == nil {
		t.Fatal("NewVerifier() returned nil")
	}
	if v.trustedMeasurements == nil {
		t.Error("trustedMeasurements map not initialized")
	}
	if v.attestedDevices == nil {
		t.Error("attestedDevices map not initialized")
	}
}

func TestRegisterTrustedMeasurement(t *testing.T) {
	v := NewVerifier()
	measurement := []byte("test-measurement")
	v.RegisterTrustedMeasurement("test", measurement)

	if _, ok := v.trustedMeasurements["test"]; !ok {
		t.Error("measurement not registered")
	}
}

func TestVerifyCPUAttestation_NilQuote(t *testing.T) {
	v := NewVerifier()
	err := v.VerifyCPUAttestation(nil, nil)
	if err != ErrInvalidQuote {
		t.Errorf("expected ErrInvalidQuote, got %v", err)
	}
}

func TestVerifyCPUAttestation_EmptyQuote(t *testing.T) {
	v := NewVerifier()
	quote := &AttestationQuote{Quote: []byte{}}
	err := v.VerifyCPUAttestation(quote, nil)
	if err != ErrInvalidQuote {
		t.Errorf("expected ErrInvalidQuote, got %v", err)
	}
}

func TestVerifyCPUAttestation_ExpiredQuote(t *testing.T) {
	v := NewVerifier()
	quote := &AttestationQuote{
		Type:      TEETypeSGX,
		Quote:     make([]byte, 500),
		Timestamp: time.Now().Add(-2 * time.Hour),
	}
	err := v.VerifyCPUAttestation(quote, nil)
	if err != ErrQuoteExpired {
		t.Errorf("expected ErrQuoteExpired, got %v", err)
	}
}

func TestVerifyCPUAttestation_UnsupportedTEE(t *testing.T) {
	v := NewVerifier()
	quote := &AttestationQuote{
		Type:      TEETypeUnknown,
		Quote:     make([]byte, 500),
		Timestamp: time.Now(),
	}
	err := v.VerifyCPUAttestation(quote, nil)
	if err != ErrUnsupportedTEE {
		t.Errorf("expected ErrUnsupportedTEE, got %v", err)
	}
}

func TestVerifySGXQuote(t *testing.T) {
	v := NewVerifier()

	// Create valid SGX quote (432+ bytes)
	quote := &AttestationQuote{
		Type:      TEETypeSGX,
		Quote:     make([]byte, 500),
		Timestamp: time.Now(),
	}

	err := v.VerifyCPUAttestation(quote, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestVerifySGXQuote_MeasurementMismatch(t *testing.T) {
	v := NewVerifier()

	quote := &AttestationQuote{
		Type:      TEETypeSGX,
		Quote:     make([]byte, 500),
		Timestamp: time.Now(),
	}

	expectedMeasurement := make([]byte, 32)
	expectedMeasurement[0] = 0xFF

	err := v.VerifyCPUAttestation(quote, expectedMeasurement)
	if err != ErrInvalidMeasurement {
		t.Errorf("expected ErrInvalidMeasurement, got %v", err)
	}
}

func TestVerifySEVSNPQuote(t *testing.T) {
	v := NewVerifier()

	// Create valid SEV-SNP report (1184 bytes)
	quote := &AttestationQuote{
		Type:      TEETypeSEVSNP,
		Quote:     make([]byte, 1200),
		Timestamp: time.Now(),
	}

	err := v.VerifyCPUAttestation(quote, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestVerifyTDXQuote(t *testing.T) {
	v := NewVerifier()

	// Create valid TDX quote (584+ bytes)
	quote := &AttestationQuote{
		Type:      TEETypeTDX,
		Quote:     make([]byte, 600),
		Timestamp: time.Now(),
	}

	err := v.VerifyCPUAttestation(quote, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestVerifyGPUAttestation(t *testing.T) {
	v := NewVerifier()

	// Local nvtrust attestation - PRIMARY method (no cloud dependency)
	att := &GPUAttestation{
		DeviceID:      "GPU-001",
		Model:         "H100",
		CCEnabled:     true,
		TEEIOEnabled:  true,
		DriverVersion: "535.154.05",
		VBIOSVersion:  "96.00.89.00.01",
		Mode:          ModeLocal,
		LocalEvidence: &LocalGPUEvidence{
			SPDMReport:  make([]byte, 512),
			CertChain:   make([]byte, 1024),
			RIMVerified: true,
			Nonce:       [32]byte{1, 2, 3},
		},
		Timestamp: time.Now(),
	}

	status, err := v.VerifyGPUAttestation(att)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !status.Attested {
		t.Error("device should be attested")
	}
	if status.TrustScore == 0 {
		t.Error("trust score should not be zero")
	}
	if status.Mode != ModeLocal {
		t.Errorf("mode should be ModeLocal, got %v", status.Mode)
	}
}

func TestVerifyGPUAttestation_NilAttestation(t *testing.T) {
	v := NewVerifier()
	_, err := v.VerifyGPUAttestation(nil)
	if err != ErrInvalidQuote {
		t.Errorf("expected ErrInvalidQuote, got %v", err)
	}
}

func TestVerifyGPUAttestation_InvalidEvidence(t *testing.T) {
	v := NewVerifier()
	// No evidence provided - should fail
	att := &GPUAttestation{
		DeviceID: "GPU-001",
		Model:    "H100",
	}
	_, err := v.VerifyGPUAttestation(att)
	if err != ErrInvalidQuote {
		t.Errorf("expected ErrInvalidQuote, got %v", err)
	}
}

func TestCalculateLocalTrustScore(t *testing.T) {
	// Test local nvtrust trust score calculation
	// Base score: 70, max: 100 for datacenter GPUs with full CC
	tests := []struct {
		name     string
		att      *GPUAttestation
		ev       *LocalGPUEvidence
		minScore uint8
		maxScore uint8
	}{
		{
			name: "Base H100 no features",
			att: &GPUAttestation{
				Model: "H100",
			},
			ev:       &LocalGPUEvidence{},
			minScore: 78, // 70 (base) + 8 (H100)
			maxScore: 78,
		},
		{
			name: "H100 with CC enabled",
			att: &GPUAttestation{
				Model:     "H100",
				CCEnabled: true,
			},
			ev:       &LocalGPUEvidence{},
			minScore: 93, // 70 + 15 (CC) + 8 (H100)
			maxScore: 93,
		},
		{
			name: "Full H100 features with RIM",
			att: &GPUAttestation{
				Model:        "H100",
				CCEnabled:    true,
				TEEIOEnabled: true,
			},
			ev:       &LocalGPUEvidence{RIMVerified: true},
			minScore: 100, // 70 + 15 + 5 + 5 (RIM) + 8 = 103 → capped at 100
			maxScore: 100,
		},
		{
			name: "Blackwell datacenter GB200",
			att: &GPUAttestation{
				Model:        "GB200",
				CCEnabled:    true,
				TEEIOEnabled: true,
			},
			ev:       &LocalGPUEvidence{RIMVerified: true},
			minScore: 100, // 70 + 15 + 5 + 5 + 10 = 105 → capped at 100
			maxScore: 100,
		},
		{
			name: "RTX PRO 6000 professional",
			att: &GPUAttestation{
				Model:        "RTX PRO 6000",
				CCEnabled:    true,
				TEEIOEnabled: true,
			},
			ev:       &LocalGPUEvidence{RIMVerified: true},
			minScore: 100, // 70 + 15 + 5 + 5 + 5 = 100
			maxScore: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateLocalTrustScore(tt.att, tt.ev)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("calculateLocalTrustScore() = %d, want between %d and %d",
					score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestParseSEVSNPReport(t *testing.T) {
	// Create minimal valid report
	data := make([]byte, 1200)
	data[0] = 1 // Version

	report, err := ParseSEVSNPReport(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Version != 1 {
		t.Errorf("Version = %d, want 1", report.Version)
	}
}

func TestParseSEVSNPReport_TooShort(t *testing.T) {
	data := make([]byte, 100)
	_, err := ParseSEVSNPReport(data)
	if err != ErrInvalidQuote {
		t.Errorf("expected ErrInvalidQuote, got %v", err)
	}
}

func TestParseTDXQuote(t *testing.T) {
	data := make([]byte, 600)
	data[0] = 4 // Version

	quote, err := ParseTDXQuote(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if quote.Version != 4 {
		t.Errorf("Version = %d, want 4", quote.Version)
	}
}

func TestParseTDXQuote_TooShort(t *testing.T) {
	data := make([]byte, 100)
	_, err := ParseTDXQuote(data)
	if err != ErrInvalidQuote {
		t.Errorf("expected ErrInvalidQuote, got %v", err)
	}
}

func TestGetDeviceStatus(t *testing.T) {
	v := NewVerifier()

	// Local nvtrust attestation
	att := &GPUAttestation{
		DeviceID:  "GPU-001",
		Model:     "H100",
		CCEnabled: true,
		Mode:      ModeLocal,
		LocalEvidence: &LocalGPUEvidence{
			SPDMReport:  make([]byte, 512),
			CertChain:   make([]byte, 1024),
			RIMVerified: true,
		},
	}

	_, err := v.VerifyGPUAttestation(att)
	if err != nil {
		t.Fatal(err)
	}

	status, ok := v.GetDeviceStatus("GPU-001")
	if !ok {
		t.Error("device status not found")
	}
	if !status.Attested {
		t.Error("device should be attested")
	}
}

func TestRecordJobCompletion(t *testing.T) {
	v := NewVerifier()

	// Local nvtrust attestation
	att := &GPUAttestation{
		DeviceID: "GPU-001",
		Model:    "H100",
		Mode:     ModeLocal,
		LocalEvidence: &LocalGPUEvidence{
			SPDMReport: make([]byte, 512),
			CertChain:  make([]byte, 1024),
		},
	}

	v.VerifyGPUAttestation(att)
	v.RecordJobCompletion("GPU-001", "job-001")

	status, _ := v.GetDeviceStatus("GPU-001")
	if len(status.JobHistory) != 1 {
		t.Error("job not recorded")
	}
	if status.JobHistory[0] != "job-001" {
		t.Error("wrong job ID recorded")
	}
}

func TestComputeAttestationHash(t *testing.T) {
	quote := &AttestationQuote{
		Type:        TEETypeSGX,
		Quote:       []byte("test-quote"),
		Measurement: []byte("test-measurement"),
		Nonce:       []byte("test-nonce"),
	}

	hash := ComputeAttestationHash(quote)
	if hash == [32]byte{} {
		t.Error("hash should not be empty")
	}
}

func TestFormatDeviceID(t *testing.T) {
	identifier := []byte("12345678901234567890")
	id := FormatDeviceID(TEETypeSGX, identifier)

	if id == "" {
		t.Error("device ID should not be empty")
	}
	if id[:3] != "SGX" {
		t.Errorf("device ID should start with SGX, got %s", id)
	}
}

func TestBytesEqual(t *testing.T) {
	tests := []struct {
		a, b     []byte
		expected bool
	}{
		{[]byte{1, 2, 3}, []byte{1, 2, 3}, true},
		{[]byte{1, 2, 3}, []byte{1, 2, 4}, false},
		{[]byte{1, 2, 3}, []byte{1, 2}, false},
		{nil, nil, true},
		{[]byte{}, []byte{}, true},
	}

	for i, tt := range tests {
		if got := bytesEqual(tt.a, tt.b); got != tt.expected {
			t.Errorf("test %d: bytesEqual() = %v, want %v", i, got, tt.expected)
		}
	}
}

// Tests for new attestation modes

func TestLocalAttestation(t *testing.T) {
	v := NewVerifier()

	// Local nvtrust attestation - PRIMARY method
	att := &GPUAttestation{
		DeviceID:     "GPU-LOCAL-001",
		Model:        "H100",
		CCEnabled:    true,
		TEEIOEnabled: true,
		Mode:         ModeLocal,
		LocalEvidence: &LocalGPUEvidence{
			SPDMReport:  make([]byte, 512),
			CertChain:   make([]byte, 1024),
			RIMVerified: true,
			Nonce:       [32]byte{1, 2, 3},
		},
	}

	status, err := v.VerifyGPUAttestation(att)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !status.Attested {
		t.Error("device should be attested")
	}
	if status.Mode != ModeLocal {
		t.Errorf("mode = %v, want ModeLocal", status.Mode)
	}
	// Local nvtrust can reach 100 with full features
	if status.TrustScore == 0 {
		t.Error("trust score should not be zero")
	}
	if !status.HardwareCC {
		t.Error("should have HardwareCC true when RIMVerified")
	}
}

func TestLocalAttestation_InvalidEvidence(t *testing.T) {
	v := NewVerifier()

	// Missing local evidence
	att := &GPUAttestation{
		DeviceID: "GPU-LOCAL-001",
		Model:    "H100",
		Mode:     ModeLocal,
	}

	_, err := v.VerifyGPUAttestation(att)
	if err != ErrInvalidQuote {
		t.Errorf("expected ErrInvalidQuote, got %v", err)
	}

	// SPDM report too short
	att.LocalEvidence = &LocalGPUEvidence{
		SPDMReport: make([]byte, 100), // Too short
		CertChain:  make([]byte, 256),
	}

	_, err = v.VerifyGPUAttestation(att)
	if err != ErrInvalidQuote {
		t.Errorf("expected ErrInvalidQuote for short SPDM, got %v", err)
	}
}

func TestSoftwareGPUAttestation(t *testing.T) {
	v := NewVerifier()

	att := &GPUAttestation{
		DeviceID: "GPU-CONSUMER-001",
		Model:    "RTX 5090",
		Mode:     ModeSoftware,
		SoftwareAttestation: &SoftwareGPUAttestation{
			GPUSerial:      "GPU-SERIAL-12345",
			PCIID:          "0000:01:00.0",
			ComputeCaps:    "10.0",
			DriverVersion:  "570.00",
			CUDAVersion:    "13.0",
			BenchmarkHash:  [32]byte{1, 2, 3, 4, 5},
			BenchmarkTime:  1500,
			ProviderPubKey: make([]byte, 64),
			Signature:      make([]byte, 128),
			Timestamp:      time.Now(),
		},
	}

	status, err := v.VerifyGPUAttestation(att)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !status.Attested {
		t.Error("device should be attested")
	}
	if status.Mode != ModeSoftware {
		t.Errorf("mode = %v, want ModeSoftware", status.Mode)
	}
	if status.TrustScore > 60 {
		t.Errorf("software trust score should be capped at 60, got %d", status.TrustScore)
	}
	if status.HardwareCC {
		t.Error("software attestation should not claim HardwareCC")
	}
}

func TestSoftwareGPUAttestation_DGXSpark(t *testing.T) {
	v := NewVerifier()

	att := &GPUAttestation{
		DeviceID: "DGX-SPARK-001",
		Model:    "GB10",
		Mode:     ModeSoftware,
		SoftwareAttestation: &SoftwareGPUAttestation{
			GPUSerial:      "DGX-SERIAL-12345",
			PCIID:          "0000:01:00.0",
			ComputeCaps:    "10.0",
			DriverVersion:  "575.00",
			BenchmarkHash:  [32]byte{1, 2, 3},
			BenchmarkTime:  1000,
			ProviderPubKey: make([]byte, 64),
			Signature:      make([]byte, 128),
			Timestamp:      time.Now(),
		},
	}

	status, err := v.VerifyGPUAttestation(att)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !status.Attested {
		t.Error("DGX Spark should be attested")
	}
	// GB10 score: 20 (base) + 12 (model) + 10 (benchmark) + 10 (signature) + 5 (driver) = 57
	if status.TrustScore < 50 || status.TrustScore > 60 {
		t.Errorf("DGX Spark trust score = %d, expected 50-60", status.TrustScore)
	}
}

func TestSoftwareGPUAttestation_InvalidSignature(t *testing.T) {
	v := NewVerifier()

	att := &GPUAttestation{
		DeviceID: "GPU-CONSUMER-001",
		Model:    "RTX 5090",
		Mode:     ModeSoftware,
		SoftwareAttestation: &SoftwareGPUAttestation{
			GPUSerial:      "GPU-SERIAL-12345",
			DriverVersion:  "570.00",
			ProviderPubKey: make([]byte, 10), // Too short
			Signature:      make([]byte, 10), // Too short
			Timestamp:      time.Now(),
		},
	}

	_, err := v.VerifyGPUAttestation(att)
	if err != ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestSoftwareGPUAttestation_Expired(t *testing.T) {
	v := NewVerifier()

	att := &GPUAttestation{
		DeviceID: "GPU-CONSUMER-001",
		Model:    "RTX 5090",
		Mode:     ModeSoftware,
		SoftwareAttestation: &SoftwareGPUAttestation{
			GPUSerial:      "GPU-SERIAL-12345",
			DriverVersion:  "570.00",
			ProviderPubKey: make([]byte, 64),
			Signature:      make([]byte, 128),
			Timestamp:      time.Now().Add(-2 * time.Hour), // Expired
		},
	}

	_, err := v.VerifyGPUAttestation(att)
	if err != ErrQuoteExpired {
		t.Errorf("expected ErrQuoteExpired, got %v", err)
	}
}

func TestIsHardwareCCCapable(t *testing.T) {
	tests := []struct {
		model   string
		capable bool
	}{
		{"H100", true},
		{"H200", true},
		{"B100", true},
		{"B200", true},
		{"GB200", true},
		{"RTX PRO 6000", true},
		{"RTX 5090", false},
		{"RTX 4090", false},
		{"GB10", false},
		{"A100", false}, // A100 has limited CC, not full
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := IsHardwareCCCapable(tt.model); got != tt.capable {
				t.Errorf("IsHardwareCCCapable(%s) = %v, want %v", tt.model, got, tt.capable)
			}
		})
	}
}

func TestAttestationModes(t *testing.T) {
	// Verify mode constants - ModeLocal is PRIMARY, ModeSoftware for non-CC GPUs
	// ModeHardwareCC and ModeLocalVerifier are legacy aliases for ModeLocal
	if ModeLocal != ModeHardwareCC {
		t.Error("ModeHardwareCC should be alias for ModeLocal")
	}
	if ModeLocal != ModeLocalVerifier {
		t.Error("ModeLocalVerifier should be alias for ModeLocal")
	}
	if ModeLocal == ModeSoftware {
		t.Error("ModeLocal and ModeSoftware should be distinct")
	}
}

func TestLocalAttestationNonCCGPU(t *testing.T) {
	v := NewVerifier()

	// Non-CC GPU (RTX 5090) should NOT pass local verification
	// It should use software attestation instead
	att := &GPUAttestation{
		DeviceID: "GPU-CONSUMER-001",
		Model:    "RTX 5090", // Not CC capable
		Mode:     ModeLocal,
		LocalEvidence: &LocalGPUEvidence{
			SPDMReport: make([]byte, 512),
			CertChain:  make([]byte, 1024),
		},
	}

	_, err := v.VerifyGPUAttestation(att)
	if err == nil {
		t.Error("expected error for non-CC GPU with local attestation")
	}
}
