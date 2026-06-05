use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;
use std::sync::Arc;
use tauri::{AppHandle, Emitter};
use tokio::sync::RwLock;
use tokio::time::{interval, Duration};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MiningState {
    pub wallet_address: String,
    pub balance: f64,
    pub is_mining: bool,
    pub last_updated: u64,
}

impl Default for MiningState {
    fn default() -> Self {
        Self {
            wallet_address: String::new(), // Empty by default, set when mining starts
            balance: 0.0,
            is_mining: false,
            last_updated: 0,
        }
    }
}

pub struct MiningManager {
    state: Arc<RwLock<MiningState>>,
    storage_path: PathBuf,
}

impl MiningManager {
    pub fn new(storage_path: PathBuf) -> Self {
        let state = if storage_path.exists() {
            match fs::read_to_string(&storage_path) {
                Ok(content) => serde_json::from_str(&content).unwrap_or_default(),
                Err(_) => MiningState::default(),
            }
        } else {
            MiningState::default()
        };

        Self {
            state: Arc::new(RwLock::new(state)),
            storage_path,
        }
    }

    pub async fn get_state(&self) -> MiningState {
        self.state.read().await.clone()
    }

    pub async fn start_mining(&self, app: AppHandle, wallet_address: String) {
        let mut state = self.state.write().await;
        if state.is_mining {
            return; // Already mining
        }

        // Use provided wallet address or generate if not provided
        if !wallet_address.is_empty() {
            state.wallet_address = wallet_address;
        } else if state.wallet_address.is_empty() {
            // Fallback: generate one if none provided
            state.wallet_address = generate_wallet_address();
        }

        state.is_mining = true;
        self.save_state(&state).await;
        drop(state);

        // Spawn mining task
        let state_clone = self.state.clone();
        let storage_path = self.storage_path.clone();

        tauri::async_runtime::spawn(async move {
            let mut ticker = interval(Duration::from_secs(60)); // Mine 1 $AI per minute

            loop {
                ticker.tick().await;

                let mut state = state_clone.write().await;
                if !state.is_mining {
                    break; // Stop mining
                }

                // Increment balance
                state.balance += 1.0;
                state.last_updated = std::time::SystemTime::now()
                    .duration_since(std::time::UNIX_EPOCH)
                    .unwrap_or(Duration::from_secs(0))
                    .as_secs();

                // Save state
                if let Ok(json) = serde_json::to_string(&*state) {
                    let _ = fs::write(&storage_path, json);
                }

                // Emit event to frontend
                let _ = app.emit("mining-update", state.clone());
            }
        });
    }

    pub async fn stop_mining(&self) {
        let mut state = self.state.write().await;
        state.is_mining = false;
        self.save_state(&state).await;
    }

    pub async fn get_balance(&self) -> f64 {
        self.state.read().await.balance
    }

    pub async fn get_wallet_address(&self) -> String {
        self.state.read().await.wallet_address.clone()
    }

    async fn save_state(&self, state: &MiningState) {
        if let Ok(json) = serde_json::to_string(state) {
            let _ = fs::write(&self.storage_path, json);
        }
    }
}

/// Generate a random Ethereum-style wallet address
fn generate_wallet_address() -> String {
    use rand::Rng;
    let mut rng = rand::thread_rng();
    let bytes: Vec<u8> = (0..20).map(|_| rng.gen()).collect();
    format!(
        "0x{}",
        bytes.iter().map(|b| format!("{:02x}", b)).collect::<String>()
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_generate_wallet_address() {
        let addr = generate_wallet_address();
        assert!(addr.starts_with("0x"));
        assert_eq!(addr.len(), 42); // 0x + 40 hex chars
    }
}
