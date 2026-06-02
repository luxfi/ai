# AI Reward / Receipt Conformance Spec

**Status:** reference = Go (`github.com/luxfi/ai/pkg/rewards`). Any other implementation
(the native **Rust** `hanzod` port, the C++ `aivm`, etc.) MUST reproduce
[`vectors.json`](./vectors.json) **byte‑for‑byte**: identical receipt hashes (hex) and
identical reward amounts (decimal wei strings). This is how "Go and Rust work identically"
is *proven* — both run the same vectors in CI and assert equality.

Regenerate vectors from the reference impl:

```
go run ./conformance/gen > conformance/vectors.json
```

---

## 1. `Receipt.Hash() -> [32]byte`

`SHA‑256` over the concatenation (no separators, no length prefixes) of exactly **five**
fields, in this order:

```
sha256( utf8(JobID) || utf8(ProviderID) || ModelHash[32] || InputHash[32] || OutputHash[32] )
```

`ComputeTime`, `GPUModel`, `Timestamp`, `Proof`, `Signature` are **NOT** hashed. Strings are
raw UTF‑8 bytes. Output is the full 32‑byte digest.

## 2. `CalculateReward(receipt, providerStats) -> big.Int` (wei)

Constants (from `NewRewardCalculator`):

| name | value |
|------|-------|
| `baseReward` | `1_000_000_000_000_000` wei (1e15, "0.001 coin") |
| `uptimeBonus` | `0.10` (applied iff `providerStats != nil && Uptime >= 0.999`) |
| `speedBonus` | `0.05` (applied iff `receipt.ComputeTime < 100`) |
| `complexityFactor` (default) | `1.0` |

Factor tables:

- **model complexity** = `1.0` (constant today; future: model-registry lookup keyed by `ModelHash`).
- **compute factor** by `ComputeTime` (ms): `<100 → 1.0`, `<1000 → 1.5`, `<10000 → 2.0`, else `3.0`.
- **gpu bonus** by `GPUModel`: `GB200|B200 → 0.20`, `H200|H100 → 0.15`, `A100 → 0.10`, `RTX 4090 → 0.05`, else `0.0`.

### PINNED ROUNDING RULE (the cross‑language determinism contract)

Every float factor `F` is applied with this exact sequence using **arbitrary‑precision
integers** (Go `math/big`, Rust e.g. `num-bigint`/`U256`):

```
scaled = int64( F * 100.0 )      // IEEE-754 double multiply, then TRUNCATE toward zero
reward = reward * scaled / 100   // integer (big) multiply then truncating integer division
```

`int64(F * 100.0)` uses IEEE‑754 binary64 — identical on Go and Rust — so the truncation is
deterministic across languages. For the current constants this yields clean integers
(`0.20→20, 0.15→15, 0.10→10, 0.05→5, 1.5→150, …`), but implementations MUST keep the
truncation semantics so future non‑representable factors stay in lock‑step.

### Exact order of operations

```
reward = baseReward
reward = reward * int64(complexity*100) / 100          // multiplicative
reward = reward * int64(computeFactor*100) / 100        // multiplicative
bonus  = reward * int64(gpuBonus*100) / 100 ; reward += bonus
if stats != nil && stats.Uptime >= 0.999:
    bonus = reward * int64(0.10*100) / 100 ; reward += bonus   // compounds on post-gpu reward
if receipt.ComputeTime < 100:
    bonus = reward * int64(0.05*100) / 100 ; reward += bonus   // compounds on post-uptime reward
return reward
```

Note the bonuses **compound** (each is a percentage of the running total, not of base), and
order matters: gpu → uptime → speed.

## 3. Conformance test (both impls)

1. Load `vectors.json`.
2. For each `hash_vectors[i]`: build the receipt from the hex fields, compute `Hash`, assert hex == `expected_hash`.
3. For each `reward_vectors[i]`: build receipt + (optional) stats, compute reward, assert decimal == `expected_reward_wei`.

CI runs this for Go **and** Rust against the *same* committed `vectors.json`. Divergence = fail.
