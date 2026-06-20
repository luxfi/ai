use std::path::PathBuf;

use super::external_node_detector::{is_external_node_running, are_ports_available};
// Ollama has been retired for chat/embeddings. Inference is served by the native
// hanzo-engine on :36900 (chat + /v1/embeddings from the same process), detected
// and reused across apps via an HTTP /health probe. The desktop must never spawn
// or kill an `ollama` process. The OllamaProcessHandler import is retained ONLY so
// the legacy `get_ollama_version` Tauri command keeps compiling — it no longer
// participates in startup/shutdown.
use super::process_handlers::ollama_process_handler::OllamaProcessHandler;
use super::process_handlers::hanzo_node_process_handler::ZooNodeProcessHandler;
use crate::local_zoo_node::hanzo_node_options::ZooNodeOptions;
use anyhow::Result;
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
    hanzo_node_process: ZooNodeProcessHandler,
    event_broadcaster: broadcast::Sender<ZooNodeManagerEvent>,
    app_resource_dir: PathBuf,
    llm_models_path: PathBuf,
    external_node_running: bool,
    managed_by_app: bool,
    /// Child handle for the daemon-managed inference engine (hanzo-engine).
    /// `None` when no engine was spawned by us (e.g. one was already running and
    /// got reused). This single process serves both chat and /v1/embeddings.
    engine_child: Option<std::process::Child>,
}

impl ZooNodeManager {
    pub(crate) fn new(app: AppHandle, app_resource_dir: PathBuf, app_data_dir: PathBuf) -> Self {
        let (hanzo_node_sender, _hanzo_node_receiver) = channel(100);
        let (event_broadcaster, _) = broadcast::channel(10);
        let llm_models_path = app
            .path()
            .resolve("llm-models", BaseDirectory::Resource)
            .unwrap();
        // No OllamaProcessHandler is constructed: ollama is retired and the
        // desktop never starts or kills it.
        ZooNodeManager {
            hanzo_node_process: ZooNodeProcessHandler::new(
                app,
                hanzo_node_sender,
                app_resource_dir.clone(),
                app_data_dir,
            ),
            event_broadcaster,
            app_resource_dir,
            llm_models_path,
            external_node_running: false,
            managed_by_app: false,
            engine_child: None,
        }
    }

    pub async fn get_hanzo_node_options(&self) -> ZooNodeOptions {
        let options = self.hanzo_node_process.get_options();
        options.clone()
    }

    pub async fn is_running(&self) -> bool {
        // Zoo Node (with Hanzo Engine) is the primary requirement.
        self.hanzo_node_process.is_running().await
    }

    /// True if the inference engine's OpenAI API is responding.
    async fn engine_up(models_url: &str) -> bool {
        reqwest::Client::new()
            .get(models_url)
            .timeout(std::time::Duration::from_secs(2))
            .send()
            .await
            .map(|r| r.status().is_success())
            .unwrap_or(false)
    }

    /// Spawn (or reuse) the daemon-managed inference engine — hanzo-engine
    /// (mistral.rs, CUDA on GPU). Returns the child if we started it, or `None`
    /// if an engine was already listening on the port (detect-and-reuse).
    ///
    /// Carmack-correct: one engine process per GPU; every app probes :36900 and
    /// reuses the live one. Knobs come from OS env vars with hardcoded fallbacks
    /// (ENGINE_PORT / ENGINE_BIN / ENGINE_GGUF_REPO / ENGINE_MODEL) so no new
    /// fields are required on ZooNodeOptions.
    async fn spawn_engine(&self) -> Result<Option<std::process::Child>, String> {
        let pick = |env: &str, fallback: &str| -> String {
            std::env::var(env).unwrap_or_else(|_| fallback.to_string())
        };
        let port = pick("ENGINE_PORT", "36900");
        let models_url = format!("http://127.0.0.1:{}/health", port);

        if Self::engine_up(&models_url).await {
            log::info!("hanzo-engine already listening on :{}, reusing it", port);
            return Ok(None);
        }

        let bin = pick(
            "ENGINE_BIN",
            "/home/z/work/hanzo/engine/target/release/hanzo",
        );
        // Model id for `--model`. Prefer the configured GGUF repo (the agent-tuned
        // zen chat model); fall back to a safetensors model id. hanzo-engine
        // resolves either via its Ollama-style loader.
        let gguf_repo = pick("ENGINE_GGUF_REPO", "zenlm/zen-eco-4b-agent-gguf");
        let model = if !gguf_repo.is_empty() {
            gguf_repo
        } else {
            pick("ENGINE_MODEL", "Qwen/Qwen3-0.6B")
        };

        // hanzo-engine uses an Ollama-style CLI: `serve` boots an OpenAI-compatible
        // server (chat + /v1/embeddings from the same process) bound to host:port.
        log::info!("starting hanzo-engine `{}` (model {}) on :{}", bin, model, port);
        let mut cmd = std::process::Command::new(&bin);
        cmd.arg("serve")
            .arg("-p")
            .arg(&port)
            .arg("auto")
            .arg("-m")
            .arg(&model);

        let child = cmd
            .spawn()
            .map_err(|e| format!("failed to spawn hanzo-engine ({}): {}", bin, e))?;

        // Non-blocking: don't stall node startup on model load.
        log::info!(
            "hanzo-engine spawned on :{} (loading model in background)",
            port
        );
        Ok(Some(child))
    }

    pub async fn spawn(&mut self) -> Result<(), String> {
        // Check if external nodes are already running
        let (hanzo_node_external, ollama_external) = is_external_node_running(2000, 11435).await;

        if hanzo_node_external || ollama_external {
            self.external_node_running = true;
            self.emit_event(ZooNodeManagerEvent::ExternalNodeDetected {
                zoo_node: hanzo_node_external,
                ollama: ollama_external,
            });

            // If external nodes are running, don't spawn our own
            if hanzo_node_external && ollama_external {
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

        // Daemon-managed inference engine (hanzo-engine / mistral.rs on the GPU)
        // replaces Ollama. If one is already listening on the engine port we
        // detect and reuse it. We keep the legacy Ollama-named events so the
        // frontend startup gating keeps working unchanged.
        self.emit_event(ZooNodeManagerEvent::StartingOllama);
        match self.spawn_engine().await {
            Ok(child) => {
                self.engine_child = child;
                self.emit_event(ZooNodeManagerEvent::OllamaStarted);
            }
            Err(e) => {
                log::error!("failed to start hanzo-engine: {}", e);
                self.emit_event(ZooNodeManagerEvent::OllamaStartError { error: e.clone() });
                // Continue starting the node; chat is unavailable until the engine
                // is reachable, but the node and its API should still come up.
            }
        }

        self.emit_event(ZooNodeManagerEvent::StartingZooNode);
        match self.hanzo_node_process.spawn().await {
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
        self.hanzo_node_process.kill().await;
        self.emit_event(ZooNodeManagerEvent::ZooNodeStopped);

        // Stop the daemon-managed engine (only if we started it). This one process
        // served both chat and embeddings, so there is nothing else to stop. Ollama
        // is retired: nothing to kill, no port sweep.
        self.emit_event(ZooNodeManagerEvent::StoppingOllama);
        if let Some(mut child) = self.engine_child.take() {
            let _ = child.kill();
            let _ = child.wait();
        }
        self.emit_event(ZooNodeManagerEvent::OllamaStopped);
    }

    pub async fn remove_storage(&self, preserve_keys: bool) -> Result<(), String> {
        self.hanzo_node_process
            .remove_storage(preserve_keys)
            .await
    }

    pub fn open_storage_location(&self) -> Result<(), String> {
        self.hanzo_node_process.open_storage_location()
    }

    pub fn open_storage_location_with_path(&self, relative_path: &str) -> Result<(), String> {
        self.hanzo_node_process
            .open_storage_location_with_path(relative_path)
    }

    pub fn open_chat_folder(
        &self,
        storage_location: &str,
        chat_folder_name: &str,
    ) -> Result<(), String> {
        self.hanzo_node_process
            .open_chat_folder(storage_location, chat_folder_name)
    }

    pub async fn set_default_hanzo_node_options(&mut self) -> ZooNodeOptions {
        self.hanzo_node_process.set_default_options()
    }

    pub async fn set_hanzo_node_options(
        &mut self,
        options: ZooNodeOptions,
    ) -> ZooNodeOptions {
        self.hanzo_node_process.set_options(options)
    }

    fn emit_event(&mut self, new_event: ZooNodeManagerEvent) {
        let _ = self.event_broadcaster.send(new_event);
    }

    pub fn subscribe_to_events(
        &mut self,
    ) -> tokio::sync::broadcast::Receiver<ZooNodeManagerEvent> {
        self.event_broadcaster.subscribe()
    }

    /// Retained for the legacy `hanzo_node_get_ollama_api_url` Tauri command.
    /// Ollama is retired; we report the native engine's OpenAI-compatible URL so
    /// any remaining frontend caller talks to hanzo-engine, never to ollama.
    pub fn get_ollama_api_url(&self) -> String {
        let port = std::env::var("ENGINE_PORT").unwrap_or_else(|_| "36900".to_string());
        format!("http://127.0.0.1:{}", port)
    }

    pub async fn get_ollama_version(app: AppHandle) -> Result<String> {
        OllamaProcessHandler::version(app).await
    }

    pub async fn check_external_nodes(&mut self) -> (bool, bool) {
        let (hanzo_node_external, ollama_external) = is_external_node_running(2000, 11435).await;
        self.external_node_running = hanzo_node_external || ollama_external;
        (hanzo_node_external, ollama_external)
    }

    pub fn is_managed_by_app(&self) -> bool {
        self.managed_by_app
    }

    pub fn is_external_node(&self) -> bool {
        self.external_node_running
    }
}
