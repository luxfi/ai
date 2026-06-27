// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package attestation

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"

	ccnvidia "github.com/luxfi/cc/attest/nvidia"
)

// signedRIM builds an ed25519-signed NVIDIA RIM in the wire format the
// canonical cc/attest/nvidia verifier accepts (signature over the body
// without the signature fields), plus a matching GPU evidence envelope.
func nvFixture(t *testing.T) (evidence, rim []byte, roots []ccnvidia.TrustRoot) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	roots = []ccnvidia.TrustRoot{{KeyID: "nvidia-rim-1", Public: pub}}
	entries := []ccnvidia.RIMEntry{
		{Name: "FW_RUNTIME", ValueHex: "deadbeef00112233"},
		{Name: "VBIOS_RT", ValueHex: "cafebabe44556677"},
	}
	body := struct {
		Architecture  string              `json:"architecture"`
		DriverVersion string              `json:"driver_version"`
		VBIOSVersion  string              `json:"vbios_version"`
		Entries       []ccnvidia.RIMEntry `json:"entries"`
	}{"Hopper", "535.104.05", "96.00.74.00.01", entries}
	canon, _ := json.Marshal(body)
	sig := ed25519.Sign(priv, canon)
	signed := struct {
		Architecture  string              `json:"architecture"`
		DriverVersion string              `json:"driver_version"`
		VBIOSVersion  string              `json:"vbios_version"`
		Entries       []ccnvidia.RIMEntry `json:"entries"`
		SignerKeyID   string              `json:"signer_key_id"`
		SignatureB64  string              `json:"signature"`
	}{"Hopper", "535.104.05", "96.00.74.00.01", entries, "nvidia-rim-1", base64.StdEncoding.EncodeToString(sig)}
	rim, _ = json.Marshal(signed)
	evidence = []byte(`{
  "evidence_version": "2.1",
  "gpu_uuid": "GPU-1234-5678",
  "architecture": "Hopper",
  "driver_version": "535.104.05",
  "vbios_version": "96.00.74.00.01",
  "nonce": "` + hex.EncodeToString(make([]byte, 32)) + `",
  "measurements": [
    {"pcr_index": 0, "name": "FW_RUNTIME", "value": "deadbeef00112233"},
    {"pcr_index": 1, "name": "VBIOS_RT",   "value": "cafebabe44556677"}
  ],
  "attestation_quote": "AAAA",
  "nvswitch_present": false
}`)
	return evidence, rim, roots
}

// TestVerifyGPULocalEvidence_ConsumesCCAttest proves the GPU hardware-CC verify
// runs through the canonical cc/attest KindNVTrust primitive (no parallel
// verifier in this package) and that the AI trust-score POLICY then scores the
// verified result.
func TestVerifyGPULocalEvidence_ConsumesCCAttest(t *testing.T) {
	evidence, rim, roots := nvFixture(t)
	v := NewVerifier()

	att, err := v.VerifyGPULocalEvidence(context.Background(), "GPU-001", "H100", evidence, rim, roots)
	if err != nil {
		t.Fatalf("VerifyGPULocalEvidence: %v", err)
	}
	if att == nil || !att.LocalEvidence.RIMVerified {
		t.Fatal("expected a RIM-verified attestation from cc/attest")
	}
	if att.DriverVersion != "535.104.05" || att.VBIOSVersion != "96.00.74.00.01" {
		t.Fatalf("verified driver/vbios not propagated: %s / %s", att.DriverVersion, att.VBIOSVersion)
	}
	if att.Mode != ModeLocal {
		t.Fatalf("mode = %v, want ModeLocal", att.Mode)
	}

	// Trust-score POLICY scores the cc/attest-verified attestation.
	status, err := v.VerifyGPUAttestation(att)
	if err != nil {
		t.Fatalf("policy scoring: %v", err)
	}
	if !status.Attested || status.TrustScore == 0 {
		t.Fatalf("expected attested with non-zero trust score, got %+v", status)
	}
	if !status.HardwareCC {
		t.Error("RIM-verified attestation should carry HardwareCC")
	}
}

func TestVerifyGPULocalEvidence_RejectsTamperedRIM(t *testing.T) {
	evidence, rim, roots := nvFixture(t)
	v := NewVerifier()
	// Break the signed RIM: the measurement no longer matches the signature.
	bad := make([]byte, len(rim))
	copy(bad, rim)
	for i := 0; i+15 < len(bad); i++ {
		if string(bad[i:i+16]) == "deadbeef00112233" {
			bad[i+15] = '4'
			break
		}
	}
	if _, err := v.VerifyGPULocalEvidence(context.Background(), "GPU-001", "H100", evidence, bad, roots); err == nil {
		t.Fatal("expected verification failure on tampered RIM")
	}
}

func TestVerifyGPULocalEvidence_RefusesWithoutTrustRoots(t *testing.T) {
	evidence, rim, _ := nvFixture(t)
	v := NewVerifier()
	// No trust roots ⇒ cc/attest refuses (no insecure mode).
	if _, err := v.VerifyGPULocalEvidence(context.Background(), "GPU-001", "H100", evidence, rim, nil); err == nil {
		t.Fatal("expected refusal with no trust roots")
	}
}
