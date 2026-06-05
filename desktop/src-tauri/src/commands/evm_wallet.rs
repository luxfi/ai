use crate::wallet::{EvmWalletManager, NetworkConfig};
use serde::{Deserialize, Serialize};
use tauri::State;
use std::sync::Arc;
use tokio::sync::RwLock;

pub type EvmWalletState = Arc<RwLock<EvmWalletManager>>;

#[derive(Debug, Serialize, Deserialize)]
pub struct NetworkSelection {
    pub name: String,
    pub chain_id: u64,
    pub rpc_url: String,
    pub block_explorer: String,
}

#[tauri::command]
pub async fn evm_get_balance(
    address: String,
    wallet: State<'_, EvmWalletState>,
) -> Result<String, String> {
    let manager = wallet.read().await;
    manager.get_balance(&address).await
}

#[tauri::command]
pub async fn evm_get_token_balance(
    wallet_address: String,
    token_address: String,
    wallet: State<'_, EvmWalletState>,
) -> Result<String, String> {
    let manager = wallet.read().await;
    manager.get_token_balance(&wallet_address, &token_address).await
}

#[tauri::command]
pub async fn evm_send_transaction(
    from_private_key: String,
    to_address: String,
    amount_eth: String,
    wallet: State<'_, EvmWalletState>,
) -> Result<String, String> {
    let manager = wallet.read().await;
    manager.send_transaction(&from_private_key, &to_address, &amount_eth).await
}

#[tauri::command]
pub async fn evm_estimate_gas(
    from_address: String,
    to_address: String,
    amount_eth: String,
    wallet: State<'_, EvmWalletState>,
) -> Result<String, String> {
    let manager = wallet.read().await;
    manager.estimate_gas(&from_address, &to_address, &amount_eth).await
}

#[tauri::command]
pub async fn evm_get_transaction_status(
    tx_hash: String,
    wallet: State<'_, EvmWalletState>,
) -> Result<String, String> {
    let manager = wallet.read().await;
    manager.get_transaction_status(&tx_hash).await
}

#[tauri::command]
pub async fn evm_switch_network(
    network: NetworkSelection,
    wallet: State<'_, EvmWalletState>,
) -> Result<(), String> {
    let new_config = NetworkConfig {
        name: network.name,
        chain_id: network.chain_id,
        rpc_url: network.rpc_url,
        block_explorer: network.block_explorer,
    };

    let new_manager = EvmWalletManager::new(new_config);
    let mut manager = wallet.write().await;
    *manager = new_manager;

    Ok(())
}

pub fn init_evm_wallet() -> EvmWalletState {
    // Start with Hanzo Mainnet by default
    let manager = EvmWalletManager::new(NetworkConfig::hanzo_mainnet());
    Arc::new(RwLock::new(manager))
}
