// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

use serde::{Deserialize, Serialize};
use tauri::Manager;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MinerStatus {
    pub running: bool,
    pub tasks_completed: u64,
    pub total_rewards: f64,
    pub gpu_utilization: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatMessage {
    pub role: String,
    pub content: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatRequest {
    pub model: String,
    pub messages: Vec<ChatMessage>,
    pub max_tokens: Option<u32>,
    pub temperature: Option<f64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatResponse {
    pub id: String,
    pub model: String,
    pub choices: Vec<ChatChoice>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatChoice {
    pub index: u32,
    pub message: ChatMessage,
    pub finish_reason: String,
}

static mut NODE_URL: Option<String> = None;

fn get_node_url() -> String {
    unsafe {
        NODE_URL.clone().unwrap_or_else(|| "http://localhost:9090".to_string())
    }
}

#[tauri::command]
fn set_node_url(url: String) {
    unsafe {
        NODE_URL = Some(url);
    }
}

#[tauri::command]
async fn get_miner_status() -> Result<MinerStatus, String> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/stats", get_node_url());

    match client.get(&url).send().await {
        Ok(resp) => {
            if resp.status().is_success() {
                // Parse response or return defaults
                Ok(MinerStatus {
                    running: true,
                    tasks_completed: 0,
                    total_rewards: 0.0,
                    gpu_utilization: 0.0,
                })
            } else {
                Ok(MinerStatus {
                    running: false,
                    tasks_completed: 0,
                    total_rewards: 0.0,
                    gpu_utilization: 0.0,
                })
            }
        }
        Err(_) => Ok(MinerStatus {
            running: false,
            tasks_completed: 0,
            total_rewards: 0.0,
            gpu_utilization: 0.0,
        }),
    }
}

#[tauri::command]
async fn start_miner(wallet: String) -> Result<String, String> {
    // In production, this would spawn the miner process
    Ok(format!("Miner started for wallet: {}", wallet))
}

#[tauri::command]
async fn stop_miner() -> Result<String, String> {
    Ok("Miner stopped".to_string())
}

#[tauri::command]
async fn chat(request: ChatRequest) -> Result<ChatResponse, String> {
    let client = reqwest::Client::new();
    let url = format!("{}/v1/chat/completions", get_node_url());

    match client.post(&url)
        .json(&request)
        .send()
        .await
    {
        Ok(resp) => {
            if resp.status().is_success() {
                resp.json::<ChatResponse>()
                    .await
                    .map_err(|e| e.to_string())
            } else {
                Err(format!("API error: {}", resp.status()))
            }
        }
        Err(e) => Err(e.to_string()),
    }
}

#[tauri::command]
async fn get_models() -> Result<Vec<String>, String> {
    let client = reqwest::Client::new();
    let url = format!("{}/v1/models", get_node_url());

    match client.get(&url).send().await {
        Ok(resp) => {
            if resp.status().is_success() {
                #[derive(Deserialize)]
                struct ModelsResponse {
                    data: Vec<ModelInfo>,
                }
                #[derive(Deserialize)]
                struct ModelInfo {
                    id: String,
                }

                let models: ModelsResponse = resp.json().await.map_err(|e| e.to_string())?;
                Ok(models.data.into_iter().map(|m| m.id).collect())
            } else {
                Ok(vec!["zen-mini-0.5b".to_string()])
            }
        }
        Err(_) => Ok(vec!["zen-mini-0.5b".to_string()]),
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![
            set_node_url,
            get_miner_status,
            start_miner,
            stop_miner,
            chat,
            get_models,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
