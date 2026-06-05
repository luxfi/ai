use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct EngineVersion {
    pub version: String,
    pub display_name: String,
    pub release_date: String,
    pub is_installed: bool,
    pub is_default: bool,
    pub size_mb: u64,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct EngineInfo {
    pub engine_type: String, // "gguf" or "mlx"
    pub available_versions: Vec<EngineVersion>,
    pub current_version: String,
}

#[tauri::command]
pub async fn get_available_engine_versions(
    engine_type: String,
) -> Result<Vec<EngineVersion>, String> {
    // Mock data for available engine versions
    // In production, this would query a remote server or local registry
    let versions = match engine_type.as_str() {
        "gguf" => vec![
            EngineVersion {
                version: "v1.32.0".to_string(),
                display_name: "GGUF Engine v1.32.0 (Latest)".to_string(),
                release_date: "2025-01-15".to_string(),
                is_installed: true,
                is_default: true,
                size_mb: 245,
            },
            EngineVersion {
                version: "v1.31.0".to_string(),
                display_name: "GGUF Engine v1.31.0".to_string(),
                release_date: "2024-12-20".to_string(),
                is_installed: false,
                is_default: false,
                size_mb: 240,
            },
            EngineVersion {
                version: "v1.30.0".to_string(),
                display_name: "GGUF Engine v1.30.0".to_string(),
                release_date: "2024-11-15".to_string(),
                is_installed: false,
                is_default: false,
                size_mb: 238,
            },
        ],
        "mlx" => vec![
            EngineVersion {
                version: "v0.8.0".to_string(),
                display_name: "MLX Engine v0.8.0 (Latest)".to_string(),
                release_date: "2025-01-10".to_string(),
                is_installed: true,
                is_default: true,
                size_mb: 180,
            },
            EngineVersion {
                version: "v0.7.0".to_string(),
                display_name: "MLX Engine v0.7.0".to_string(),
                release_date: "2024-12-01".to_string(),
                is_installed: false,
                is_default: false,
                size_mb: 175,
            },
            EngineVersion {
                version: "v0.6.0".to_string(),
                display_name: "MLX Engine v0.6.0".to_string(),
                release_date: "2024-10-15".to_string(),
                is_installed: false,
                is_default: false,
                size_mb: 170,
            },
        ],
        _ => vec![],
    };

    Ok(versions)
}

#[tauri::command]
pub async fn get_engine_info() -> Result<HashMap<String, EngineInfo>, String> {
    let mut engines = HashMap::new();

    // Get GGUF engine info
    let gguf_versions = get_available_engine_versions("gguf".to_string()).await?;
    engines.insert(
        "gguf".to_string(),
        EngineInfo {
            engine_type: "gguf".to_string(),
            available_versions: gguf_versions,
            current_version: "v1.32.0".to_string(),
        },
    );

    // Get MLX engine info
    let mlx_versions = get_available_engine_versions("mlx".to_string()).await?;
    engines.insert(
        "mlx".to_string(),
        EngineInfo {
            engine_type: "mlx".to_string(),
            available_versions: mlx_versions,
            current_version: "v0.8.0".to_string(),
        },
    );

    Ok(engines)
}

#[tauri::command]
pub async fn download_engine_version(
    engine_type: String,
    version: String,
) -> Result<String, String> {
    // Mock implementation - in production this would download the engine
    log::info!("Downloading {} engine version {}", engine_type, version);

    // Simulate download progress
    tokio::time::sleep(tokio::time::Duration::from_secs(2)).await;

    Ok(format!("Successfully downloaded {} version {}", engine_type, version))
}

#[tauri::command]
pub async fn switch_engine_version(
    engine_type: String,
    version: String,
) -> Result<String, String> {
    // Mock implementation - in production this would switch the active engine
    log::info!("Switching {} engine to version {}", engine_type, version);

    // Update the configuration
    // This would normally update the hanzo_node_options

    Ok(format!("Successfully switched {} to version {}", engine_type, version))
}