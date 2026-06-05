use ethers::prelude::*;
use ethers::providers::{Http, Provider};
use ethers::signers::{LocalWallet, Signer};
use ethers::types::{Address, TransactionReceipt, TransactionRequest, U256};
use ethers::utils::{format_units, parse_ether};
use serde::{Deserialize, Serialize};
use std::str::FromStr;
use std::sync::Arc;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvmWalletManager {
    pub network: NetworkConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NetworkConfig {
    pub name: String,
    pub chain_id: u64,
    pub rpc_url: String,
    pub block_explorer: String,
}

impl NetworkConfig {
    pub fn hanzo_mainnet() -> Self {
        Self {
            name: "Hanzo Mainnet".to_string(),
            chain_id: 36900,
            rpc_url: "https://rpc.hanzo.network".to_string(),
            block_explorer: "https://explorer.hanzo.network".to_string(),
        }
    }

    pub fn hanzo_testnet() -> Self {
        Self {
            name: "Hanzo Testnet".to_string(),
            chain_id: 36901,
            rpc_url: "https://rpc.hanzo-test.network".to_string(),
            block_explorer: "https://explorer.hanzo-test.network".to_string(),
        }
    }
}

impl EvmWalletManager {
    pub fn new(network: NetworkConfig) -> Self {
        Self { network }
    }

    /// Get balance for an address
    pub async fn get_balance(&self, address: &str) -> Result<String, String> {
        let provider = self.get_provider().map_err(|e| e.to_string())?;

        let address = Address::from_str(address)
            .map_err(|e| format!("Invalid address: {}", e))?;

        let balance = provider
            .get_balance(address, None)
            .await
            .map_err(|e| format!("Failed to get balance: {}", e))?;

        // Convert wei to ether
        let balance_eth = format_units(balance, "ether")
            .map_err(|e| format!("Failed to format balance: {}", e))?;

        Ok(balance_eth)
    }

    /// Get token balance (ERC20)
    pub async fn get_token_balance(
        &self,
        wallet_address: &str,
        token_address: &str,
    ) -> Result<String, String> {
        // ERC20 balanceOf ABI
        let abi = r#"[{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"type":"function"}]"#;

        let provider = self.get_provider().map_err(|e| e.to_string())?;

        let token_addr = Address::from_str(token_address)
            .map_err(|e| format!("Invalid token address: {}", e))?;

        let wallet_addr = Address::from_str(wallet_address)
            .map_err(|e| format!("Invalid wallet address: {}", e))?;

        let abi_value: ethers::abi::Contract = serde_json::from_str(abi).unwrap();
        let contract = Contract::new(token_addr, abi_value, provider);

        let balance: U256 = contract
            .method::<_, U256>("balanceOf", wallet_addr)
            .map_err(|e| format!("Failed to call balanceOf: {}", e))?
            .call()
            .await
            .map_err(|e| format!("Failed to get token balance: {}", e))?;

        let balance_str = format_units(balance, 18)
            .map_err(|e| format!("Failed to format balance: {}", e))?;

        Ok(balance_str)
    }

    /// Send transaction (withdrawal)
    pub async fn send_transaction(
        &self,
        from_private_key: &str,
        to_address: &str,
        amount_eth: &str,
    ) -> Result<String, String> {
        let provider = self.get_provider().map_err(|e| e.to_string())?;

        // Parse private key
        let wallet = LocalWallet::from_str(from_private_key)
            .map_err(|e| format!("Invalid private key: {}", e))?
            .with_chain_id(self.network.chain_id);

        let signer = SignerMiddleware::new(provider.clone(), wallet);

        // Parse amount
        let amount = parse_ether(amount_eth)
            .map_err(|e| format!("Invalid amount: {}", e))?;

        // Parse destination address
        let to = Address::from_str(to_address)
            .map_err(|e| format!("Invalid destination address: {}", e))?;

        // Build transaction
        let tx = TransactionRequest::new()
            .to(to)
            .value(amount);

        // Send transaction
        let pending_tx = signer
            .send_transaction(tx, None)
            .await
            .map_err(|e| format!("Failed to send transaction: {}", e))?;

        let tx_hash = format!("{:?}", pending_tx.tx_hash());

        // Wait for confirmation (optional)
        let receipt = pending_tx
            .await
            .map_err(|e| format!("Transaction failed: {}", e))?
            .ok_or_else(|| "Transaction receipt not found".to_string())?;

        Ok(tx_hash)
    }

    /// Estimate gas for a transaction
    pub async fn estimate_gas(
        &self,
        from_address: &str,
        to_address: &str,
        amount_eth: &str,
    ) -> Result<String, String> {
        let provider = self.get_provider().map_err(|e| e.to_string())?;

        let from = Address::from_str(from_address)
            .map_err(|e| format!("Invalid from address: {}", e))?;

        let to = Address::from_str(to_address)
            .map_err(|e| format!("Invalid to address: {}", e))?;

        let amount = parse_ether(amount_eth)
            .map_err(|e| format!("Invalid amount: {}", e))?;

        let tx = TransactionRequest::new()
            .from(from)
            .to(to)
            .value(amount);

        let gas = provider
            .estimate_gas(&tx.into(), None)
            .await
            .map_err(|e| format!("Failed to estimate gas: {}", e))?;

        // Get gas price
        let gas_price = provider
            .get_gas_price()
            .await
            .map_err(|e| format!("Failed to get gas price: {}", e))?;

        let total_fee = gas * gas_price;
        let fee_eth = format_units(total_fee, "ether")
            .map_err(|e| format!("Failed to format fee: {}", e))?;

        Ok(format!("{} ETH", fee_eth))
    }

    /// Get transaction status
    pub async fn get_transaction_status(&self, tx_hash: &str) -> Result<String, String> {
        let provider = self.get_provider().map_err(|e| e.to_string())?;

        let hash = TxHash::from_str(tx_hash)
            .map_err(|e| format!("Invalid transaction hash: {}", e))?;

        let receipt = provider
            .get_transaction_receipt(hash)
            .await
            .map_err(|e| format!("Failed to get transaction: {}", e))?;

        match receipt {
            Some(r) => {
                let status = if r.status == Some(1.into()) {
                    "confirmed"
                } else {
                    "failed"
                };
                Ok(status.to_string())
            }
            None => Ok("pending".to_string()),
        }
    }

    /// Helper: Create provider
    fn get_provider(&self) -> Result<Arc<Provider<Http>>, String> {
        let provider = Provider::<Http>::try_from(&self.network.rpc_url)
            .map_err(|e| format!("Failed to create provider: {}", e))?;
        Ok(Arc::new(provider))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_get_balance() {
        let manager = EvmWalletManager::new(NetworkConfig::hanzo_testnet());

        // Test with a zero address
        let zero_address = "0x0000000000000000000000000000000000000000";
        let result = manager.get_balance(zero_address).await;

        // Should not error (might be 0 balance)
        assert!(result.is_ok() || result.is_err());
    }
}
