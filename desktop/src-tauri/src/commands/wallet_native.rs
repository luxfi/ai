use crate::commands::secure_storage::SecureStore;
use bip39::{Mnemonic, Language};
use ethers::core::k256::ecdsa::SigningKey;
use ethers::signers::coins_bip39::English;
use ethers::signers::{LocalWallet, Signer, MnemonicBuilder, Wallet};
use ethers::types::Address;
use serde::{Deserialize, Serialize};
use tauri::State;

#[derive(Debug, Serialize, Deserialize)]
pub struct WalletCreationResponse {
    pub address: String,
    pub mnemonic: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct WalletImportResponse {
    pub address: String,
}

/// Create a new wallet with a random mnemonic
#[tauri::command]
pub async fn wallet_create(
    secure_store: State<'_, SecureStore>,
) -> Result<WalletCreationResponse, String> {
    // Generate random entropy for 12-word mnemonic (16 bytes = 128 bits)
    let mut entropy = [0u8; 16];
    rand::Rng::fill(&mut rand::thread_rng(), &mut entropy);
    
    // Generate BIP39 mnemonic from entropy
    let mnemonic = Mnemonic::from_entropy_in(Language::English, &entropy)
        .map_err(|e| format!("Failed to generate mnemonic: {}", e))?;
    
    let mnemonic_phrase = mnemonic.to_string();

    // Create wallet from mnemonic
    let wallet = MnemonicBuilder::<English>::default()
        .phrase(mnemonic_phrase.as_str())
        .build()
        .map_err(|e| format!("Failed to build wallet from mnemonic: {}", e))?;

    // Get the address
    let address = format!("{:?}", wallet.address());

    // Store mnemonic in secure storage using the secure_storage command
    crate::commands::secure_storage::secure_storage_set(
        "wallet_mnemonic".to_string(),
        mnemonic_phrase.clone(),
        secure_store.clone(),
    )
    .await?;

    // Store address
    crate::commands::secure_storage::secure_storage_set(
        "wallet_address".to_string(),
        address.clone(),
        secure_store,
    )
    .await?;

    Ok(WalletCreationResponse {
        address,
        mnemonic: mnemonic_phrase,
    })
}

/// Import wallet from mnemonic
#[tauri::command]
pub async fn wallet_import_mnemonic(
    mnemonic: String,
    secure_store: State<'_, SecureStore>,
) -> Result<WalletImportResponse, String> {
    // Validate and create wallet from mnemonic
    let wallet = MnemonicBuilder::<English>::default()
        .phrase(mnemonic.as_str())
        .build()
        .map_err(|e| format!("Invalid mnemonic: {}", e))?;

    // Get the address
    let address = format!("{:?}", wallet.address());

    // Store mnemonic in secure storage
    crate::commands::secure_storage::secure_storage_set(
        "wallet_mnemonic".to_string(),
        mnemonic.clone(),
        secure_store.clone(),
    )
    .await?;

    // Store address
    crate::commands::secure_storage::secure_storage_set(
        "wallet_address".to_string(),
        address.clone(),
        secure_store,
    )
    .await?;

    Ok(WalletImportResponse { address })
}

/// Import wallet from private key
#[tauri::command]
pub async fn wallet_import_private_key(
    private_key: String,
    secure_store: State<'_, SecureStore>,
) -> Result<WalletImportResponse, String> {
    // Create wallet from private key
    let wallet = private_key
        .parse::<LocalWallet>()
        .map_err(|e| format!("Invalid private key: {}", e))?;

    // Get the address
    let address = format!("{:?}", wallet.address());

    // Store private key in secure storage
    crate::commands::secure_storage::secure_storage_set(
        "wallet_private_key".to_string(),
        private_key.clone(),
        secure_store.clone(),
    )
    .await?;

    // Store address
    crate::commands::secure_storage::secure_storage_set(
        "wallet_address".to_string(),
        address.clone(),
        secure_store,
    )
    .await?;

    Ok(WalletImportResponse { address })
}

/// Get wallet address from secure storage
#[tauri::command]
pub async fn wallet_get_address(
    secure_store: State<'_, SecureStore>,
) -> Result<String, String> {
    crate::commands::secure_storage::secure_storage_get(
        "wallet_address".to_string(),
        secure_store,
    )
    .await
}

/// Get wallet mnemonic from secure storage
#[tauri::command]
pub async fn wallet_get_mnemonic(
    secure_store: State<'_, SecureStore>,
) -> Result<String, String> {
    crate::commands::secure_storage::secure_storage_get(
        "wallet_mnemonic".to_string(),
        secure_store,
    )
    .await
}

/// Check if wallet exists
#[tauri::command]
pub async fn wallet_exists(
    secure_store: State<'_, SecureStore>,
) -> Result<bool, String> {
    match crate::commands::secure_storage::secure_storage_get(
        "wallet_address".to_string(),
        secure_store,
    )
    .await
    {
        Ok(_) => Ok(true),
        Err(_) => Ok(false),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_wallet_creation() {
        // Note: This test requires secure storage to be initialized
        // In practice, we'd mock the secure_store
    }
}
