// Command gen produces the canonical AI-reward/receipt conformance vectors from
// the Go reference implementation (github.com/luxfi/ai/pkg/rewards). Any other
// implementation (the native Rust hanzod port) MUST reproduce vectors.json
// byte-for-byte. See ../SPEC.md for the pinned semantics.
//
//	go run ./conformance/gen > conformance/vectors.json
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/luxfi/ai/pkg/rewards"
)

// fill returns a deterministic 32-byte array seeded by b (so vectors are stable).
func fill(b byte) [32]byte {
	var a [32]byte
	for i := range a {
		a[i] = b + byte(i)
	}
	return a
}

type hashVector struct {
	Name       string `json:"name"`
	JobID      string `json:"job_id"`
	ProviderID string `json:"provider_id"`
	ModelHash  string `json:"model_hash"`  // hex(32)
	InputHash  string `json:"input_hash"`  // hex(32)
	OutputHash string `json:"output_hash"` // hex(32)
	Expected   string `json:"expected_hash"`
}

type rewardVector struct {
	Name        string `json:"name"`
	ModelHash   string `json:"model_hash"` // hex(32)
	ComputeTime uint64 `json:"compute_time_ms"`
	GPUModel    string `json:"gpu_model"`
	HasStats    bool   `json:"has_stats"`
	Uptime      float64 `json:"uptime"`     // only if has_stats
	Expected    string `json:"expected_reward_wei"`
}

func main() {
	rc := rewards.NewRewardCalculator()

	// ---- Receipt.Hash vectors (covers the 5 hashed fields) ----
	var hv []hashVector
	hashCases := []struct {
		name              string
		job, prov         string
		mSeed, iSeed, oSeed byte
	}{
		{"zeros", "", "", 0, 0, 0},
		{"basic", "job-1", "prov-1", 1, 2, 3},
		{"unicode", "jöb-π", "prov-本", 9, 8, 7},
		{"long-ids", "job-0123456789abcdef", "provider-fedcba9876543210", 0x40, 0x80, 0xc0},
	}
	for _, c := range hashCases {
		m, i, o := fill(c.mSeed), fill(c.iSeed), fill(c.oSeed)
		r := &rewards.Receipt{JobID: c.job, ProviderID: c.prov, ModelHash: m, InputHash: i, OutputHash: o}
		h := r.Hash()
		hv = append(hv, hashVector{
			Name: c.name, JobID: c.job, ProviderID: c.prov,
			ModelHash: hex.EncodeToString(m[:]), InputHash: hex.EncodeToString(i[:]), OutputHash: hex.EncodeToString(o[:]),
			Expected: hex.EncodeToString(h[:]),
		})
	}

	// ---- CalculateReward vectors (covers every branch) ----
	var rv []rewardVector
	gpus := []string{
		"GB200", "B200", "H200", "H100", "RTX PRO 6000", "GB10", "DGX Spark", // frontier
		"A100", "MI300X", "L40S", // datacenter
		"RTX 5090", "RTX 4090", "M4 Ultra", "RTX A6000", // prosumer
		"RTX 4080", "Radeon RX 7900 XTX", "Strix Halo", "Ryzen AI Max+ 395", "Apple M4 Max", // consumer/APU
		"Intel Arc A770", "Apple M4 Pro", "Apple M4", // integrated/entry
		"CPU", "unknown", "", // baseline
	}
	times := []uint64{50, 99, 100, 999, 1000, 9999, 10000, 50000}
	model := fill(1)
	add := func(name, gpu string, t uint64, hasStats bool, uptime float64) {
		var ps *rewards.ProviderStats
		if hasStats {
			ps = &rewards.ProviderStats{Uptime: uptime}
		}
		r := &rewards.Receipt{ModelHash: model, ComputeTime: t, GPUModel: gpu}
		got := rc.CalculateReward(r, ps)
		rv = append(rv, rewardVector{
			Name: name, ModelHash: hex.EncodeToString(model[:]), ComputeTime: t,
			GPUModel: gpu, HasStats: hasStats, Uptime: uptime, Expected: got.String(),
		})
	}
	for _, g := range gpus {
		for _, t := range times {
			add(fmt.Sprintf("gpu=%q/t=%d/nostats", g, t), g, t, false, 0)
		}
	}
	// uptime bonus boundary (0.999) and speed-bonus interplay
	for _, up := range []float64{0.0, 0.998, 0.999, 1.0} {
		add(fmt.Sprintf("uptime=%g/t=50", up), "H100", 50, true, up)
		add(fmt.Sprintf("uptime=%g/t=5000", up), "H100", 5000, true, up)
	}

	out := map[string]any{
		"_spec":          "../SPEC.md",
		"_reference":     "github.com/luxfi/ai/pkg/rewards",
		"base_reward_wei": "1000000000000000",
		"hash_vectors":   hv,
		"reward_vectors": rv,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
