//! Runs the SHARED golden vectors (../../vectors.json) against the Rust impl.
//! This is the proof that the Go reference and the Rust port are identical.

use lux_ai_conformance::{calculate_reward, receipt_hash};
use serde::Deserialize;

#[derive(Deserialize)]
struct Vectors {
    hash_vectors: Vec<HashVec>,
    reward_vectors: Vec<RewardVec>,
}
#[derive(Deserialize)]
struct HashVec {
    name: String,
    job_id: String,
    provider_id: String,
    model_hash: String,
    input_hash: String,
    output_hash: String,
    expected_hash: String,
}
#[derive(Deserialize)]
struct RewardVec {
    name: String,
    #[allow(dead_code)]
    model_hash: String,
    compute_time_ms: u64,
    gpu_model: String,
    has_stats: bool,
    uptime: f64,
    expected_reward_wei: String,
}

fn arr32(s: &str) -> [u8; 32] {
    let v = hex::decode(s).expect("hex");
    assert_eq!(v.len(), 32);
    let mut a = [0u8; 32];
    a.copy_from_slice(&v);
    a
}

fn load() -> Vectors {
    // crate root = conformance/rust ; vectors live at conformance/vectors.json
    let raw = std::fs::read_to_string(concat!(env!("CARGO_MANIFEST_DIR"), "/../vectors.json"))
        .expect("read ../vectors.json");
    serde_json::from_str(&raw).expect("parse vectors.json")
}

#[test]
fn receipt_hash_conformance() {
    let v = load();
    assert!(!v.hash_vectors.is_empty());
    for c in &v.hash_vectors {
        let got = receipt_hash(&c.job_id, &c.provider_id, &arr32(&c.model_hash), &arr32(&c.input_hash), &arr32(&c.output_hash));
        assert_eq!(hex::encode(got), c.expected_hash, "hash mismatch: {}", c.name);
    }
}

#[test]
fn calculate_reward_conformance() {
    let v = load();
    assert!(!v.reward_vectors.is_empty());
    for c in &v.reward_vectors {
        let stats = if c.has_stats { Some(c.uptime) } else { None };
        let got = calculate_reward(1.0, c.compute_time_ms, &c.gpu_model, stats);
        assert_eq!(got.to_string(), c.expected_reward_wei, "reward mismatch: {}", c.name);
    }
}
