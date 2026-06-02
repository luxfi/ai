//! Native Rust port of `github.com/luxfi/ai/pkg/rewards` (receipt hash + reward
//! calculation), implementing `../SPEC.md` exactly. Proven identical to the Go
//! reference by the conformance test (`tests/conformance.rs`) which runs the
//! shared `../vectors.json`. This is the seed for the `hanzod` Rust port.

use sha2::{Digest, Sha256};

/// Base reward in wei (Go: `big.NewInt(1e15)`).
pub const BASE_REWARD_WEI: u128 = 1_000_000_000_000_000;

/// `Receipt.Hash()` — SHA-256 over exactly five fields, concatenated raw
/// (utf8 strings, then the three 32-byte arrays). No separators/length prefixes.
pub fn receipt_hash(job_id: &str, provider_id: &str, model_hash: &[u8; 32], input_hash: &[u8; 32], output_hash: &[u8; 32]) -> [u8; 32] {
    let mut h = Sha256::new();
    h.update(job_id.as_bytes());
    h.update(provider_id.as_bytes());
    h.update(model_hash);
    h.update(input_hash);
    h.update(output_hash);
    let out = h.finalize();
    let mut a = [0u8; 32];
    a.copy_from_slice(&out);
    a
}

fn compute_factor(compute_time_ms: u64) -> f64 {
    if compute_time_ms < 100 { 1.0 }
    else if compute_time_ms < 1000 { 1.5 }
    else if compute_time_ms < 10000 { 2.0 }
    else { 3.0 }
}

// Two orthogonal axes (keyword-identical to the Go `ClassifyCompute` +
// `ClassifyConfidentialComputeByModel`, see ../SPEC.md): raw compute class and
// confidential-compute trust. Integer percent throughout — no float truncation.
// Order matters; keep in lockstep with the Go source.

fn compute_base_bonus_pct(gpu_model: &str) -> u128 {
    let m = gpu_model.trim().to_lowercase();
    let has = |subs: &[&str]| subs.iter().any(|s| m.contains(s));
    if has(&["gb200", "b200", "b100"]) {
        20 // frontier datacenter
    } else if has(&["h100", "h200", "gh200", "mi300"]) {
        15 // premium datacenter
    } else if has(&["a100", "l40", "a40", "a30", "mi250", "mi210", "instinct"]) {
        10 // datacenter
    } else if has(&["rtx pro 6000", "rtx 6000", "6000 blackwell", "6000 ada", "a6000"]) {
        8 // workstation-AI (RTX PRO 6000 ~7x a Spark)
    } else if has(&["5090", "4090", "ultra", "gb10", "dgx spark", "spark"]) {
        6 // prosumer high-end / Apple M*Ultra / desktop appliance
    } else if has(&["rtx 40", "rtx 30", "4080", "3090", "3080", "radeon rx", "rx 7", "rx 9", "strix halo", "ryzen ai max", "evo x2", " max"]) {
        4 // consumer discrete / AI APU / Apple M*Max
    } else if has(&["arc", "iris", "vega", "apple m", "m1", "m2", "m3", "m4", " pro", "radeon"]) {
        2 // integrated / entry / base Apple / Intel
    } else {
        1 // CPU / unknown — every device earns something
    }
}

// model string -> at most CapableGpuTee (4%). DGX Spark/GB10 explicitly excluded.
fn confidential_compute_bonus_pct(gpu_model: &str) -> u128 {
    let m = gpu_model.trim().to_lowercase();
    let has = |subs: &[&str]| subs.iter().any(|s| m.contains(s));
    if has(&["dgx spark", "gb10", "spark"]) {
        0 // local desktop AI — NOT GPU-TEE class
    } else if has(&["gb200", "b200", "b100", "h100", "h200", "gh200"]) {
        4 // Hopper/Blackwell datacenter GPU CC (CapableGpuTee)
    } else if has(&["rtx pro 6000 blackwell", "rtx pro 6000", "rtx 6000 blackwell"]) {
        4 // Blackwell workstation CC (exact, not generic rtx 6000)
    } else {
        0
    }
}

fn node_capability_bonus_pct(gpu_model: &str) -> u128 {
    compute_base_bonus_pct(gpu_model) + confidential_compute_bonus_pct(gpu_model)
}

/// PINNED ROUNDING RULE (see SPEC.md): scale a float factor to an integer
/// percentage via IEEE-754 multiply then **truncate toward zero** — identical
/// to Go's `int64(F * 100.0)`.
#[inline]
fn pct(factor: f64) -> u128 {
    ((factor * 100.0) as i64) as u128
}

/// Port of `RewardCalculator.CalculateReward`. Integer (u128) math throughout;
/// bonuses compound in fixed order gpu -> uptime -> speed.
pub fn calculate_reward(model_complexity: f64, compute_time_ms: u64, gpu_model: &str, stats: Option<f64>) -> u128 {
    let mut reward = BASE_REWARD_WEI;

    // model complexity (default 1.0) — multiplicative
    reward = reward * pct(model_complexity) / 100;
    // compute time factor — multiplicative
    reward = reward * pct(compute_factor(compute_time_ms)) / 100;
    // node capability bonus (compute class + confidential-compute trust), integer %
    let bonus = reward * node_capability_bonus_pct(gpu_model) / 100;
    reward += bonus;
    // uptime bonus (>= 0.999)
    if let Some(uptime) = stats {
        if uptime >= 0.999 {
            let b = reward * pct(0.10) / 100;
            reward += b;
        }
    }
    // speed bonus (sub-100ms)
    if compute_time_ms < 100 {
        let b = reward * pct(0.05) / 100;
        reward += b;
    }
    reward
}
