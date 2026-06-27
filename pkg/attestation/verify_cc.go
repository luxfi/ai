// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// verify_cc.go is the bridge from this package's attestation POLICY (trust
// tiers, CC-capable vs software mode, device tracking — see attestation.go)
// to the one canonical hardware-attestation VERIFY primitive,
// github.com/luxfi/cc/attest. There is no parallel verifier here: every
// cryptographic check (AMD SEV-SNP vendor chain, NVIDIA GPU local-RIM) runs
// inside cc/attest. This package only scores the verified result.
//
// Depending on the leaf cc module (not luxfi/mpc) keeps the confidential-AI
// system decoupled from the MPC custody system — attestation is an orthogonal
// concern both consume.

package attestation

import (
	"context"
	"errors"
	"fmt"
	"time"

	ccattest "github.com/luxfi/cc/attest"
	ccnvidia "github.com/luxfi/cc/attest/nvidia"
)

// VerifyCPUAttestation verifies a CPU TEE attestation quote by delegating to
// the canonical cc/attest verifier.
//
//   - SEV-SNP: fully verified (VCEK chain to the AMD root + report signature).
//   - TDX:     routed to cc/attest's TDX verifier.
//   - SGX/ARM/unknown: no cc/attest backend, refused with ErrUnsupportedTEE.
//     This package never had real SGX/ARM crypto; refusing is fail-closed,
//     not a regression.
func (v *Verifier) VerifyCPUAttestation(quote *AttestationQuote, expectedMeasurement []byte) error {
	if quote == nil || len(quote.Quote) == 0 {
		return ErrInvalidQuote
	}
	if time.Since(quote.Timestamp) > time.Hour {
		return ErrQuoteExpired
	}
	kind, ok := ccKindForTEE(quote.Type)
	if !ok {
		return ErrUnsupportedTEE
	}

	opts := make([]ccattest.Option, 0, 2)
	if len(expectedMeasurement) > 0 {
		opts = append(opts, ccattest.WithExpectedMeasurement(expectedMeasurement))
	}
	if len(quote.ReportData) > 0 {
		opts = append(opts, ccattest.WithExpectedReportData(quote.ReportData))
	}

	if _, err := ccattest.Dispatch(context.Background(), kind, quote.Quote, opts...); err != nil {
		// A verified-chain-but-policy-rejected result (e.g. the supplied
		// expected measurement did not match) maps to the measurement error;
		// everything else is an invalid quote.
		if errors.Is(err, ccattest.ErrPolicy) {
			return ErrInvalidMeasurement
		}
		return fmt.Errorf("%w: %v", ErrInvalidQuote, err)
	}
	return nil
}

// ccKindForTEE maps an AI TEEType to the cc/attest hardware Kind. SGX, ARM,
// and unknown types have no cc/attest backend.
func ccKindForTEE(t TEEType) (ccattest.Kind, bool) {
	switch t {
	case TEETypeSEVSNP:
		return ccattest.KindSEVSNP, true
	case TEETypeTDX:
		return ccattest.KindTDX, true
	default:
		return "", false
	}
}

// VerifyGPULocalEvidence runs the cloud-free NVIDIA GPU attestation through the
// canonical cc/attest KindNVTrust verifier: it parses the GPU evidence
// envelope, verifies the operator-pinned signed RIM, and matches every
// reported measurement to the signed golden value. On success it returns a
// GPUAttestation populated from the verified report (RIMVerified true, verified
// driver/VBIOS), ready for the trust-score POLICY in VerifyGPUAttestation.
//
// model is the provider's declared GPU model (e.g. "H100"); the policy layer
// scores trust from it (IsHardwareCCCapable + tier bonuses). The cryptographic
// guarantee here is "this evidence matches an NVIDIA-signed RIM bound to a
// fresh nonce" — see cc/attest NVTrust for the honest scope.
//
// This is the ONLY hardware-GPU attestation path in lux/ai. There is no
// parallel verifier.
func (v *Verifier) VerifyGPULocalEvidence(
	ctx context.Context,
	deviceID, model string,
	gpuEvidence, signedRIM []byte,
	rimTrustRoots []ccnvidia.TrustRoot,
) (*GPUAttestation, error) {
	rep, err := ccattest.Dispatch(ctx, ccattest.KindNVTrust, gpuEvidence,
		ccattest.WithNVTrustRIM(signedRIM),
		ccattest.WithNVTrustTrustRoots(rimTrustRoots),
	)
	if err != nil {
		return nil, err
	}
	return &GPUAttestation{
		DeviceID:      deviceID,
		Model:         model,
		CCEnabled:     true, // passed hardware CC (RIM-matched) attestation
		DriverVersion: rep.Extra["nvtrust.driver_version"],
		VBIOSVersion:  rep.Extra["nvtrust.vbios_version"],
		Timestamp:     rep.IssuedAt,
		Mode:          ModeLocal,
		LocalEvidence: &LocalGPUEvidence{
			SPDMReport:  gpuEvidence,
			RIMVerified: true,
		},
	}, nil
}
