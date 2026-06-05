use std::fs;
use std::path::PathBuf;

use super::external_node_detector::{is_external_node_running, are_ports_available};
use super::ollama_api::ollama_api_client::OllamaApiClient;
use super::ollama_api::ollama_api_types::OllamaApiPullResponse;
use super::process_handlers::ollama_process_handler::OllamaProcessHandler;
use super::process_handlers::zoo_node_process_handler::ZooNodeProcessHandler;
use crate::local_zoo_node::zoo_node_options::ZooNodeOptions;
use crate::models::embedding_model;
use anyhow::Result;
use futures_util::StreamExt;
use log::error;
use serde::{Deserialize, Serialize};
use tauri::path::BaseDirectory;
use tauri::AppHandle;
use tauri::Manager;
use tokio::sync::broadcast;
use tokio::sync::mpsc::channel;

#[derive(Serialize, Deserialize, Clone)]
pub enum ZooNodeManagerEvent {
    StartingZooNode,
    ZooNodeStarted,
    ZooNodeStartError { error: String },

    StartingOllama,
    OllamaStarted,
    OllamaStartError { error: String },

    PullingModelStart { model: String },
    PullingModelProgress { model: String, progress: u32 },
    PullingModelDone { model: String },
    PullingModelError { model: String, error: String },

    CreatingModelStart { model: String },
    CreatingModelProgress { model: String, progress: u32 },
    CreatingModelDone { model: String },
    CreatingModelError { model: String, error: String },

    StoppingZooNode,
    ZooNodeStopped,
    ZooNodeStopError { error: String },

    StoppingOllama,
    OllamaStopped,
    OllamaStopError { error: String },
    
    ExternalNodeDetected { zoo_node: bool, ollama: bool },
    PortsUnavailable { ports: Vec<u16> },
}

pub struct ZooNodeManager {
    ollama_process: OllamaProcessHandler,
    zoo_node_process: ZooNodeProcessHandler,
    event_broadcaster: broadcast::Sender<ZooNodeManagerEvent>,
    app_resource_dir: PathBuf,
    llm_models_path: PathBuf,
    external_node_running: bool,
    managed_by_app: bool,
}

impl ZooNodeManager {
    pub(crate) fn new(app: AppHandle, app_resource_dir: PathBuf, app_data_dir: PathBuf) -> Self {
        let (ollama_sender, _ollama_receiver) = channel(100);
        let (zoo_node_sender, _zoo_node_receiver) = channel(100);
        let (event_broadcaster, _) = broadcast::channel(10);
        let llm_models_path = app
            .path()
            .resolve("llm-models", BaseDirectory::Resource)
            .unwrap();
        ZooNodeManager {
            ollama_process: OllamaProcessHandler::new(
                app.clone(),
                ollama_sender,
                app_resource_dir.clone(),
            ),
            zoo_node_process: ZooNodeProcessHandler::new(
                app,
                zoo_node_sender,
                app_resource_dir.clone(),
                app_data_dir,
            ),
            event_broadcaster,
            app_resource_dir,
            llm_models_path,
            external_node_running: false,
            managed_by_app: false,
        }
    }

    pub async fn get_zoo_node_options(&self) -> ZooNodeOptions {
        let options = self.zoo_node_process.get_options();
        options.clone()
    }

    pub async fn is_running(&self) -> bool {
        // Zoo Node (with Hanzo Engine) is the primary requirement
        // Ollama is optional
        self.zoo_node_process.is_running().await
    }

    pub async fn is_ollama_running(&self) -> bool {
        self.ollama_process.is_running().await
    }

    pub async fn spawn(&mut self) -> Result<(), String> {
        // Check if external nodes are already running
        let (zoo_node_external, ollama_external) = is_external_node_running(2000, 11435).await;
        
        if zoo_node_external || ollama_external {
            self.external_node_running = true;
            self.emit_event(ZooNodeManagerEvent::ExternalNodeDetected {
                zoo_node: zoo_node_external,
                ollama: ollama_external,
            });
            
            // If external nodes are running, don't spawn our own
            if zoo_node_external && ollama_external {
                log::info!("External Zoo node and Ollama detected, using external services");
                return Ok(());
            }
        }
        
        // Check if required ports are available
        let ports_to_check = vec![2000, 2001, 2002, 11435];
        let port_availability = are_ports_available(&ports_to_check).await;
        let unavailable_ports: Vec<u16> = port_availability
            .iter()
            .filter(|(_, available)| !available)
            .map(|(port, _)| *port)
            .collect();
        
        if !unavailable_ports.is_empty() && !self.external_node_running {
            self.emit_event(ZooNodeManagerEvent::PortsUnavailable {
                ports: unavailable_ports.clone(),
            });
            return Err(format!("Ports unavailable: {:?}", unavailable_ports));
        }
        
        self.managed_by_app = true;

        // Check if Ollama is enabled in options
        let enable_ollama = self.zoo_node_process.get_options().enable_ollama.unwrap_or(false);

        if enable_ollama {
            log::info!("Ollama is enabled, starting Ollama process");
            self.emit_event(ZooNodeManagerEvent::StartingOllama);
            match self.ollama_process.spawn(None).await {
                Ok(_) => {
                    self.emit_event(ZooNodeManagerEvent::OllamaStarted);
                }
                Err(e) => {
                    log::warn!("failed spawning ollama process, continuing without Ollama: {:?}", e);
                    // Don't return error - Ollama is optional
                    self.emit_event(ZooNodeManagerEvent::OllamaStartError { error: e.clone() });
                }
            }

            // Only set up embedding models if Ollama started successfully
            if self.ollama_process.is_running().await {
                let ollama_api_url = self.ollama_process.get_ollama_api_base_url();
                let ollama_api = OllamaApiClient::new(ollama_api_url);

                let installed_models = match ollama_api.tags().await {
                    Ok(response) => response.models.iter().map(|m| m.model.clone()).collect(),
                    Err(e) => {
                        log::warn!("ollama api tags request failed, fallback assuming there are not local models {:?}", e);
                        vec![]
                    }
                };

                let default_embedding_model = self
                    .zoo_node_process
                    .get_options()
                    .default_embedding_model
                    .unwrap();
                if !installed_models.contains(&default_embedding_model.to_string()) {
                    log::info!(
                        "default embedding model {} not found in local models list [{}], creating it",
                        default_embedding_model,
                        installed_models.join(", ")
                    );
                    self.emit_event(ZooNodeManagerEvent::CreatingModelStart {
                        model: default_embedding_model.to_string(),
                    });

                    // Use the embedded GGUF model
                    let gguf_data = embedding_model::get_model_data(&self.llm_models_path);

                    match ollama_api
                        .create_model_from_gguf(&default_embedding_model, gguf_data)
                        .await
                    {
                        Ok(_) => {
                            self.emit_event(ZooNodeManagerEvent::CreatingModelDone {
                                model: default_embedding_model.to_string(),
                            });
                        }
                        Err(e) => {
                            log::warn!("failed to create model from gguf, continuing without embedding model: {}", e);
                            self.emit_event(ZooNodeManagerEvent::CreatingModelError {
                                model: default_embedding_model.to_string(),
                                error: e.to_string(),
                            });
                            // Don't return error - continue without embedding model
                        }
                    }
                }
            }
        } else {
            log::info!("Ollama is disabled, skipping Ollama startup (Hanzo Engine will be used instead)");
        }

        self.emit_event(ZooNodeManagerEvent::StartingZooNode);
        match self.zoo_node_process.spawn().await {
            Ok(_) => {
                self.emit_event(ZooNodeManagerEvent::ZooNodeStarted);
            }
            Err(e) => {
                self.kill().await;
                self.emit_event(ZooNodeManagerEvent::ZooNodeStartError {
                    error: e.clone(),
                });
                return Err(e);
            }
        }
        Ok(())
    }

    pub async fn kill(&mut self) {
        // Only kill processes if they are managed by the app
        if !self.managed_by_app {
            log::info!("Skipping kill - nodes are not managed by this app");
            return;
        }
        self.emit_event(ZooNodeManagerEvent::StoppingZooNode);
        self.zoo_node_process.kill().await;
        self.emit_event(ZooNodeManagerEvent::ZooNodeStopped);

        // Only kill Ollama if it was started
        if self.ollama_process.is_running().await {
            self.emit_event(ZooNodeManagerEvent::StoppingOllama);
            self.ollama_process.kill().await;
            self.emit_event(ZooNodeManagerEvent::OllamaStopped);
        }
    }

    pub async fn remove_storage(&self, preserve_keys: bool) -> Result<(), String> {
        self.zoo_node_process
            .remove_storage(preserve_keys)
            .await
    }

    pub fn open_storage_location(&self) -> Result<(), String> {
        self.zoo_node_process.open_storage_location()
    }

    pub fn open_storage_location_with_path(&self, relative_path: &str) -> Result<(), String> {
        self.zoo_node_process
            .open_storage_location_with_path(relative_path)
    }

    pub fn open_chat_folder(
        &self,
        storage_location: &str,
        chat_folder_name: &str,
    ) -> Result<(), String> {
        self.zoo_node_process
            .open_chat_folder(storage_location, chat_folder_name)
    }

    pub async fn set_default_zoo_node_options(&mut self) -> ZooNodeOptions {
        self.zoo_node_process.set_default_options()
    }

    pub async fn set_zoo_node_options(
        &mut self,
        options: ZooNodeOptions,
    ) -> ZooNodeOptions {
        self.zoo_node_process.set_options(options)
    }

    fn emit_event(&mut self, new_event: ZooNodeManagerEvent) {
        let _ = self.event_broadcaster.send(new_event);
    }

    pub fn subscribe_to_events(
        &mut self,
    ) -> tokio::sync::broadcast::Receiver<ZooNodeManagerEvent> {
        self.event_broadcaster.subscribe()
    }

    pub fn get_ollama_api_url(&self) -> String {
        self.ollama_process.get_ollama_api_base_url()
    }

    pub async fn get_ollama_version(app: AppHandle) -> Result<String> {
        OllamaProcessHandler::version(app).await
    }
    
    pub async fn check_external_nodes(&mut self) -> (bool, bool) {
        let (zoo_node_external, ollama_external) = is_external_node_running(2000, 11435).await;
        self.external_node_running = zoo_node_external || ollama_external;
        (zoo_node_external, ollama_external)
    }
    
    pub fn is_managed_by_app(&self) -> bool {
        self.managed_by_app
    }
    
    pub fn is_external_node(&self) -> bool {
        self.external_node_running
    }
}
