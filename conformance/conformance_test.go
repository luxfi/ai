// Package conformance asserts the Go reference reproduces the committed golden
// vectors. The native Rust (hanzod) port MUST have an equivalent test loading the
// SAME vectors.json and producing identical results. See SPEC.md.
package conformance

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/luxfi/ai/pkg/rewards"
)

type vectorsFile struct {
	HashVectors []struct {
		Name       string `json:"name"`
		JobID      string `json:"job_id"`
		ProviderID string `json:"provider_id"`
		ModelHash  string `json:"model_hash"`
		InputHash  string `json:"input_hash"`
		OutputHash string `json:"output_hash"`
		Expected   string `json:"expected_hash"`
	} `json:"hash_vectors"`
	RewardVectors []struct {
		Name        string  `json:"name"`
		ModelHash   string  `json:"model_hash"`
		ComputeTime uint64  `json:"compute_time_ms"`
		GPUModel    string  `json:"gpu_model"`
		HasStats    bool    `json:"has_stats"`
		Uptime      float64 `json:"uptime"`
		Expected    string  `json:"expected_reward_wei"`
	} `json:"reward_vectors"`
}

func hex32(t *testing.T, s string) [32]byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 32 {
		t.Fatalf("bad hex32 %q: %v", s, err)
	}
	var a [32]byte
	copy(a[:], b)
	return a
}

func load(t *testing.T) vectorsFile {
	t.Helper()
	raw, err := os.ReadFile("vectors.json")
	if err != nil {
		t.Fatalf("read vectors.json: %v", err)
	}
	var v vectorsFile
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("parse vectors.json: %v", err)
	}
	return v
}

func TestReceiptHashConformance(t *testing.T) {
	v := load(t)
	if len(v.HashVectors) == 0 {
		t.Fatal("no hash vectors")
	}
	for _, c := range v.HashVectors {
		r := &rewards.Receipt{
			JobID: c.JobID, ProviderID: c.ProviderID,
			ModelHash: hex32(t, c.ModelHash), InputHash: hex32(t, c.InputHash), OutputHash: hex32(t, c.OutputHash),
		}
		got := r.Hash()
		if h := hex.EncodeToString(got[:]); h != c.Expected {
			t.Errorf("%s: hash = %s, want %s", c.Name, h, c.Expected)
		}
	}
}

func TestCalculateRewardConformance(t *testing.T) {
	v := load(t)
	if len(v.RewardVectors) == 0 {
		t.Fatal("no reward vectors")
	}
	rc := rewards.NewRewardCalculator()
	for _, c := range v.RewardVectors {
		var ps *rewards.ProviderStats
		if c.HasStats {
			ps = &rewards.ProviderStats{Uptime: c.Uptime}
		}
		r := &rewards.Receipt{ModelHash: hex32(t, c.ModelHash), ComputeTime: c.ComputeTime, GPUModel: c.GPUModel}
		if got := rc.CalculateReward(r, ps).String(); got != c.Expected {
			t.Errorf("%s: reward = %s, want %s", c.Name, got, c.Expected)
		}
	}
}
