use std::path::PathBuf;

use crate::hardware::{hardware_get_summary, RequirementsStatus};
use serde::{Deserialize, Serialize};

/// It matches ENV variables names from ZooNode
#[derive(Serialize, Deserialize, Clone)]
pub struct ZooNodeOptions {
    pub node_api_ip: Option<String>,
    pub node_api_port: Option<String>,
    pub node_ws_port: Option<String>,
    pub node_https_port: Option<String>,
    pub node_ip: Option<String>,
    pub node_port: Option<String>,
    pub global_identity_name: Option<String>,
    pub node_storage_path: Option<String>,
    pub embeddings_server_url: Option<String>,
    pub first_device_needs_registration_code: Option<String>,
    pub initial_agent_names: Option<String>,
    pub initial_agent_urls: Option<String>,
    pub initial_agent_models: Option<String>,
    pub initial_agent_api_keys: Option<String>,
    pub starting_num_qr_devices: Option<String>,
    pub log_all: Option<String>,
    pub proxy_identity: Option<String>,
    pub rpc_url: Option<String>,
    pub default_embedding_model: Option<String>,
    pub supported_embedding_models: Option<String>,
    pub hanzo_tools_runner_deno_binary_path: Option<String>,
    pub hanzo_tools_runner_uv_binary_path: Option<String>,
    pub zoo_store_url: Option<String>,
    pub secret_desktop_installation_proof_key: Option<String>,
    pub enable_ollama: Option<bool>,
    pub auto_update_runtimes: Option<bool>,
    pub auto_delete_old_runtimes: Option<bool>,
    pub node_version: Option<String>,
    pub ollama_version: Option<String>,
    // Engine selection fields
    pub gguf_engine_version: Option<String>,
    pub mlx_engine_version: Option<String>,
    pub preload_other_versions: Option<bool>,
}

impl ZooNodeOptions {
    pub fn with_app_options(
        app_resource_dir: PathBuf,
        app_data_dir: PathBuf,
    ) -> ZooNodeOptions {
        let default_node_storage_path = app_data_dir
            .join("node_storage")
            .to_string_lossy()
            .to_string();
        log::debug!("Node storage path: {:?}", default_node_storage_path);
        ZooNodeOptions {
            node_storage_path: Some(default_node_storage_path),
            ..Default::default()
        }
    }

    pub fn default_initial_model() -> String {
        "zoo-backend:FREE_TEXT_INFERENCE".to_string()
    }

    pub fn from_merge(
        base_options: ZooNodeOptions,
        options: ZooNodeOptions,
    ) -> ZooNodeOptions {
        let default_options = ZooNodeOptions::default();
        ZooNodeOptions {
            node_api_ip: Some(
                options
                    .node_api_ip
                    .or(base_options.node_api_ip)
                    .unwrap_or_default(),
            ),
            node_api_port: Some(
                options
                    .node_api_port
                    .or(base_options.node_api_port)
                    .unwrap_or_default(),
            ),
            node_ws_port: Some(
                options
                    .node_ws_port
                    .or(base_options.node_ws_port)
                    .unwrap_or_default(),
            ),
            node_ip: Some(options.node_ip.or(base_options.node_ip).unwrap_or_default()),
            node_port: Some(
                options
                    .node_port
                    .or(base_options.node_port)
                    .unwrap_or_default(),
            ),
            node_https_port: Some(
                options
                    .node_https_port
                    .or(base_options.node_https_port)
                    .unwrap_or_default(),
            ),
            global_identity_name: Some(
                options
                    .global_identity_name
                    .or(base_options.global_identity_name)
                    .unwrap_or_default(),
            ),
            node_storage_path: Some(
                options
                    .node_storage_path
                    .or(base_options.node_storage_path)
                    .unwrap_or_default(),
            ),
            embeddings_server_url: Some(
                options
                    .embeddings_server_url
                    .or(base_options.embeddings_server_url)
                    .unwrap_or_default(),
            ),
            first_device_needs_registration_code: Some(
                options
                    .first_device_needs_registration_code
                    .or(base_options.first_device_needs_registration_code)
                    .unwrap_or_default(),
            ),
            // Start with NO agents by default - users can add them in settings
            // This prevents startup crashes and allows configuration after onboarding
            initial_agent_names: Some(
                options
                    .initial_agent_names
                    .or(base_options.initial_agent_names)
                    .unwrap_or_else(|| "".to_string()),
            ),
            initial_agent_urls: Some(
                options
                    .initial_agent_urls
                    .or(base_options.initial_agent_urls)
                    .unwrap_or_else(|| "".to_string()),
            ),
            initial_agent_models: Some(
                options
                    .initial_agent_models
                    .or(base_options.initial_agent_models)
                    .unwrap_or_else(|| "".to_string()),
            ),
            initial_agent_api_keys: Some(
                options
                    .initial_agent_api_keys
                    .or(base_options.initial_agent_api_keys)
                    .unwrap_or_else(|| "".to_string()),
            ),
            starting_num_qr_devices: Some(
                options
                    .starting_num_qr_devices
                    .or(base_options.starting_num_qr_devices)
                    .unwrap_or_default(),
            ),
            log_all: Some(options.log_all.or(base_options.log_all).unwrap_or_default()),
            rpc_url: Some(options.rpc_url.or(base_options.rpc_url).unwrap_or_default()),
            default_embedding_model: Some(
                options
                    .default_embedding_model
                    .or(base_options.default_embedding_model)
                    .unwrap_or_else(|| default_options.default_embedding_model.unwrap_or_default()),
            ),
            supported_embedding_models: Some(
                options
                    .supported_embedding_models
                    .or(base_options.supported_embedding_models)
                    .unwrap_or_else(|| default_options.supported_embedding_models.unwrap_or_default()),
            ),
            hanzo_tools_runner_deno_binary_path: Some(
                options
                    .hanzo_tools_runner_deno_binary_path
                    .or(base_options.hanzo_tools_runner_deno_binary_path)
                    .unwrap_or_default(),
            ),
            hanzo_tools_runner_uv_binary_path: Some(
                options
                    .hanzo_tools_runner_uv_binary_path
                    .or(base_options.hanzo_tools_runner_uv_binary_path)
                    .unwrap_or_default(),
            ),
            zoo_store_url: Some(
                options
                    .zoo_store_url
                    .or(base_options.zoo_store_url)
                    .unwrap_or_default(),
            ),
            secret_desktop_installation_proof_key: default_options
                .secret_desktop_installation_proof_key,
            proxy_identity: default_options.proxy_identity,
            enable_ollama: options
                .enable_ollama
                .or(base_options.enable_ollama)
                .or(default_options.enable_ollama),
            auto_update_runtimes: options
                .auto_update_runtimes
                .or(base_options.auto_update_runtimes)
                .or(default_options.auto_update_runtimes),
            auto_delete_old_runtimes: options
                .auto_delete_old_runtimes
                .or(base_options.auto_delete_old_runtimes)
                .or(default_options.auto_delete_old_runtimes),
            node_version: options
                .node_version
                .or(base_options.node_version)
                .or(default_options.node_version),
            ollama_version: options
                .ollama_version
                .or(base_options.ollama_version)
                .or(default_options.ollama_version),
            gguf_engine_version: options
                .gguf_engine_version
                .or(base_options.gguf_engine_version)
                .or(default_options.gguf_engine_version),
            mlx_engine_version: options
                .mlx_engine_version
                .or(base_options.mlx_engine_version)
                .or(default_options.mlx_engine_version),
            preload_other_versions: options
                .preload_other_versions
                .or(base_options.preload_other_versions)
                .or(default_options.preload_other_versions),
        }
    }
}
impl Default for ZooNodeOptions {
    fn default() -> ZooNodeOptions {
        let hanzo_tools_runner_deno_binary_path = std::env::current_exe()
            .unwrap()
            .parent()
            .unwrap()
            .join(if cfg!(target_os = "windows") {
                "deno.exe"
            } else {
                "deno"
            })
            .to_string_lossy()
            .to_string();

        let hanzo_tools_runner_uv_binary_path = std::env::current_exe()
            .unwrap()
            .parent()
            .unwrap()
            .join(if cfg!(target_os = "windows") {
                "uv.exe"
            } else {
                "uv"
            })
            .to_string_lossy()
            .to_string();

        ZooNodeOptions {
            node_api_ip: Some("127.0.0.1".to_string()),
            node_api_port: Some("2000".to_string()),
            node_ws_port: Some("2001".to_string()),
            // P2P listens on ALL interfaces so the node is reachable + shareable
            // on the network (LAN peers connect directly; the relay advertises it
            // to the wider network). The HTTP API above stays on 127.0.0.1
            // (local-only) for safety.
            node_ip: Some("0.0.0.0".to_string()),
            node_port: Some("2002".to_string()),
            node_https_port: Some("2003".to_string()),
            // Chain-only DID prefix — node derives did:lux:0x<addr> from its key.
            global_identity_name: Some("did:lux:auto".to_string()),
            node_storage_path: Some("./".to_string()),
            // Use Hanzo Engine for embeddings by default (Ollama is opt-in)
            embeddings_server_url: Some("zoo-backend:FREE_TEXT_INFERENCE".to_string()),
            first_device_needs_registration_code: Some("false".to_string()),
            // Start with NO agents - let users configure in settings after onboarding
            initial_agent_urls: Some("".to_string()),
            initial_agent_names: Some("".to_string()),
            initial_agent_models: Some("".to_string()),
            initial_agent_api_keys: Some("".to_string()),
            starting_num_qr_devices: Some("0".to_string()),
            log_all: Some("1".to_string()),
            proxy_identity: Some("@@libp2p_relayer.sepolia-zoo".to_string()),
            rpc_url: Some("https://sepolia.base.org".to_string()),
            default_embedding_model: Some("snowflake-arctic-embed:xs".to_string()),
            supported_embedding_models: Some(
                "snowflake-arctic-embed:xs,embeddinggemma:300m,jina/jina-embeddings-v2-base-es:latest".to_string(),
            ),
            hanzo_tools_runner_deno_binary_path: Some(hanzo_tools_runner_deno_binary_path),
            hanzo_tools_runner_uv_binary_path: Some(hanzo_tools_runner_uv_binary_path),
            zoo_store_url: Some("https://gateway.hanzo.ai".to_string()), // Free tier uses Hanzo gateway
            secret_desktop_installation_proof_key: option_env!(
                "SECRET_DESKTOP_INSTALLATION_PROOF_KEY"
            )
            .and_then(|s| Some(s.to_string())),
            enable_ollama: Some(false),  // Ollama is OPT-IN, not default
            auto_update_runtimes: Some(true),  // Auto-update enabled by default
            auto_delete_old_runtimes: Some(false),  // Don't delete old versions by default
            node_version: Some("1.0.0".to_string()),  // Default version, will be updated dynamically
            ollama_version: None,  // Will be fetched when Ollama is enabled
            gguf_engine_version: Some("v1.32.0".to_string()),  // Default GGUF engine version
            mlx_engine_version: Some("v0.8.0".to_string()),  // Default MLX engine version
            preload_other_versions: Some(false),  // Don't preload other versions by default
        }
    }
}
