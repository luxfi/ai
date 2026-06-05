use crate::mining::{MiningManager, MiningState};
use std::sync::Arc;
use tauri::{AppHandle, State};

pub type MiningManagerState = Arc<MiningManager>;

#[tauri::command]
pub async fn mining_get_state(
    mining_manager: State<'_, MiningManagerState>,
) -> Result<MiningState, String> {
    Ok(mining_manager.get_state().await)
}

#[tauri::command]
pub async fn mining_start(
    app: AppHandle,
    wallet_address: String,
    mining_manager: State<'_, MiningManagerState>,
) -> Result<(), String> {
    mining_manager.start_mining(app, wallet_address).await;
    Ok(())
}

#[tauri::command]
pub async fn mining_stop(mining_manager: State<'_, MiningManagerState>) -> Result<(), String> {
    mining_manager.stop_mining().await;
    Ok(())
}

#[tauri::command]
pub async fn mining_get_balance(
    mining_manager: State<'_, MiningManagerState>,
) -> Result<f64, String> {
    Ok(mining_manager.get_balance().await)
}

#[tauri::command]
pub async fn mining_get_wallet_address(
    mining_manager: State<'_, MiningManagerState>,
) -> Result<String, String> {
    Ok(mining_manager.get_wallet_address().await)
}
