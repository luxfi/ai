// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::sync::Arc;

use crate::commands::fetch::{get_request, post_request};
use crate::commands::galxe::galxe_generate_proof;
use crate::commands::hardware::hardware_get_summary;
use crate::commands::zoo_node_manager_commands::{
    show_zoo_node_manager_window, zoo_node_get_default_embedding_model,
    zoo_node_get_default_model, zoo_node_get_ollama_api_url,
    zoo_node_get_ollama_version, zoo_node_get_options, zoo_node_is_running,
    zoo_node_kill, zoo_node_open_chat_folder, zoo_node_open_storage_location,
    zoo_node_open_storage_location_with_path, zoo_node_remove_storage,
    zoo_node_set_default_options, zoo_node_set_options, zoo_node_spawn,
    zoo_node_status,
};
use crate::commands::engine_manager::{
    get_available_engine_versions, get_engine_info, download_engine_version, switch_engine_version,
};
use crate::commands::mcp_clients_install::{
    check_claude_installed,
    get_claude_config_help,
    is_server_registered_in_claude,
    register_server_in_claude,
    check_cursor_installed,
    get_cursor_command_config_help,
    get_cursor_sse_config_help,
    is_server_registered_in_cursor,
    register_command_server_in_cursor,
    register_sse_server_in_cursor,
};
use crate::commands::logs::{download_logs, retrieve_logs};
use crate::commands::mining::{
    mining_get_balance, mining_get_state, mining_get_wallet_address, mining_start, mining_stop,
};
use crate::commands::secure_storage::{
    init_secure_storage, secure_storage_delete, secure_storage_get, secure_storage_set,
};
use crate::commands::evm_wallet::{
    init_evm_wallet, evm_get_balance, evm_get_token_balance, evm_send_transaction,
    evm_estimate_gas, evm_get_transaction_status, evm_switch_network,
};
use crate::commands::wallet_native::{
    wallet_create, wallet_import_mnemonic, wallet_import_private_key,
    wallet_get_address, wallet_get_mnemonic, wallet_exists,
};
use crate::commands::spotlight_commands::{hide_spotlight_window_app, show_spotlight_window_app, open_main_window_with_path_app};
use crate::commands::debug_commands::{write_debug_log, save_debug_logs, get_debug_logs_path, open_debug_logs_folder, is_dev_build};
// use crate::commands::runtime_updates::{check_hanzo_update, check_ollama_update, download_and_install_update};
use crate::commands::store::{get_store_response, store_api_proxy};
use deep_links::setup_deep_links;
use global_shortcuts::global_shortcut_handler;
use globals::ZOO_NODE_MANAGER_INSTANCE;
use local_zoo_node::zoo_node_manager::ZooNodeManager;
use mining::MiningManager;
use tauri::{Emitter, WindowEvent};
use tauri::{Manager, RunEvent};
use tokio::sync::RwLock;
use tray::{create_tray, wire_node_events};
use windows::{recreate_window, Window};
mod commands;
mod deep_links;
mod galxe;
mod global_shortcuts;
mod globals;
mod hardware;
mod local_zoo_node;
mod mining;
mod models;
mod tray;
mod wallet;
mod windows;

#[derive(Clone, serde::Serialize)]
struct Payload {
    args: Vec<String>,
    cwd: String,
}

fn main() {
    // NVIDIA/Linux + WebKitGTK render a blank window with the default DMABUF
    // renderer on many driver combos; fall back to the non-DMABUF path unless
    // the user already configured it. Must run before any webview is created.
    #[cfg(target_os = "linux")]
    if std::env::var_os("WEBKIT_DISABLE_DMABUF_RENDERER").is_none() {
        std::env::set_var("WEBKIT_DISABLE_DMABUF_RENDERER", "1");
    }

    let _ = fix_path_env::fix();

    // Check if app is started with --background flag
    let args: Vec<String> = std::env::args().collect();
    let is_background_mode = args.contains(&"--background".to_string());
    
    tauri::Builder::default()
        .plugin(tauri_plugin_single_instance::init(|app, argv, cwd| {
            app.emit("single-instance", Payload { args: argv, cwd })
                .unwrap();
        }))
        .plugin(
            tauri_plugin_log::Builder::new()
                .format(|out, message, record| {
                    // Ending with a triple ideographic space as separator so then we can group texts that belongs to the same log
                    out.finish(format_args!(
                        "[{}][{}][{}] {}　　　",
                        chrono::Local::now().format("%Y-%m-%d %H:%M:%S"),
                        record.level(),
                        record.target(),
                        message
                    ))
                })
                .build(),
        )
        .plugin(tauri_plugin_http::init())
        .plugin(tauri_plugin_os::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(
            tauri_plugin_global_shortcut::Builder::new()
                .with_shortcuts([
                    "super+shift+o",
                    "control+shift+o",
                    "super+shift+k",
                    "control+shift+k",
                ])
                .unwrap()
                .with_handler(
                    |app: &tauri::AppHandle,
                     shortcut: &tauri_plugin_global_shortcut::Shortcut,
                     event: tauri_plugin_global_shortcut::ShortcutEvent| {
                        global_shortcut_handler(app, *shortcut, event)
                    },
                )
                .build(),
        )
        .plugin(tauri_plugin_deep_link::init())
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_autostart::init(
            tauri_plugin_autostart::MacosLauncher::LaunchAgent,
            Some(vec!["--background"]),
        ))
        .plugin(tauri_remote_ui::init())
        .invoke_handler(tauri::generate_handler![
            hide_spotlight_window_app,
            show_spotlight_window_app,
            open_main_window_with_path_app,
            show_zoo_node_manager_window,
            zoo_node_is_running,
            zoo_node_status,
            zoo_node_get_options,
            zoo_node_set_options,
            zoo_node_spawn,
            zoo_node_kill,
            zoo_node_remove_storage,
            zoo_node_open_storage_location,
            zoo_node_open_storage_location_with_path,
            zoo_node_open_chat_folder,
            zoo_node_set_default_options,
            zoo_node_get_ollama_api_url,
            zoo_node_get_default_model,
            zoo_node_get_default_embedding_model,
            hardware_get_summary,
            galxe_generate_proof,
            get_request,
            post_request,
            zoo_node_get_ollama_version,
            retrieve_logs,
            download_logs,
            get_available_engine_versions,
            get_engine_info,
            download_engine_version,
            switch_engine_version,
            check_claude_installed,
            is_server_registered_in_claude,
            register_server_in_claude,
            get_claude_config_help,
            check_cursor_installed,
            is_server_registered_in_cursor,
            register_command_server_in_cursor,
            register_sse_server_in_cursor,
            get_cursor_command_config_help,
            get_cursor_sse_config_help,
            write_debug_log,
            is_dev_build,
            save_debug_logs,
            get_debug_logs_path,
            open_debug_logs_folder,
            commands::store::get_store_response,
            commands::store::store_api_proxy,
            mining_get_state,
            mining_start,
            mining_stop,
            mining_get_balance,
            mining_get_wallet_address,
            secure_storage_set,
            secure_storage_get,
            secure_storage_delete,
            evm_get_balance,
            evm_get_token_balance,
            evm_send_transaction,
            evm_estimate_gas,
            evm_get_transaction_status,
            evm_switch_network,
            wallet_create,
            wallet_import_mnemonic,
            wallet_import_private_key,
            wallet_get_address,
            wallet_get_mnemonic,
            wallet_exists,
            commands::service::install_system_service,
            commands::service::uninstall_system_service,
            commands::service::system_service_status,
            commands::service::restart_system_service,
            // check_hanzo_update,
            // check_ollama_update,
            // download_and_install_update,
        ])
        .setup(move |app| {
            log::info!("starting app version: {} (background: {})", env!("CARGO_PKG_VERSION"), is_background_mode);
            let app_resource_dir = app.path().resource_dir()?;
            let app_data_dir = app.path().app_data_dir()?;

            {
                let _ = ZOO_NODE_MANAGER_INSTANCE.set(Arc::new(RwLock::new(
                    ZooNodeManager::new(app.handle().clone(), app_resource_dir.clone(), app_data_dir.clone()),
                )));
            }

            // Initialize mining manager
            let mining_storage_path = app_data_dir.join("mining_state.json");
            let mining_manager = Arc::new(MiningManager::new(mining_storage_path));
            app.manage(mining_manager.clone());

            // Initialize secure storage
            let secure_store = init_secure_storage();
            app.manage(secure_store);

            // Initialize EVM wallet manager
            let evm_wallet = init_evm_wallet();
            app.manage(evm_wallet);

            create_tray(app.handle())?;
            // Keep the tray's Node status line live by subscribing to node
            // manager state-change events.
            wire_node_events(app.handle());
            setup_deep_links(app.handle())?;

            /*
                This is the initialization pipeline
                At some point we will need to add a UI because some tasks can be hard/slow to execute
            */
            tauri::async_runtime::spawn({
                let app_handle = app.handle().clone();
                async move {
                    // Check for external nodes before killing
                    let mut zoo_node_manager_guard =
                        ZOO_NODE_MANAGER_INSTANCE.get().unwrap().write().await;
                    
                    // Check if external nodes are running
                    let (zoo_external, ollama_external) = zoo_node_manager_guard.check_external_nodes().await;
                    
                    if !zoo_external && !ollama_external {
                        // Only kill if no external nodes are detected
                        zoo_node_manager_guard.kill().await;
                    } else {
                        log::info!("External nodes detected (zoo: {}, ollama: {}), skipping kill", zoo_external, ollama_external);
                    }
                    drop(zoo_node_manager_guard);

                    // Only create windows if not in background mode
                    if !is_background_mode {
                        let _ = recreate_window(app_handle.clone(), Window::Coordinator, false);
                        let _ = recreate_window(app_handle.clone(), Window::Spotlight, false);
                        let _ = recreate_window(app_handle.clone(), Window::Main, true);
                    } else {
                        log::info!("Running in background mode, skipping window creation");
                    }

                    // Auto-start mining after windows are created
                    let mining_manager = app_handle.state::<Arc<MiningManager>>();
                    // Small delay to ensure event system is ready
                    tokio::time::sleep(tokio::time::Duration::from_millis(500)).await;
                    // Note: Empty string means mining will use stored or generate new address
                    mining_manager.start_mining(app_handle.clone(), String::new()).await;
                }
            });

            tauri::async_runtime::spawn({
                let app_handle = app.handle().clone();
                async move {
                    let mut zoo_node_manager_guard =
                        ZOO_NODE_MANAGER_INSTANCE.get().unwrap().write().await;
                    let mut receiver = zoo_node_manager_guard.subscribe_to_events();
                    drop(zoo_node_manager_guard);
                    while let Ok(state_change) = receiver.recv().await {
                        app_handle
                            .emit("zoo-node-state-change", state_change)
                            .unwrap_or_else(|e| {
                                log::error!("failed to emit global event for state change: {}", e);
                            });
                    }
                }
            });
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("error while running tauri application")
        .run(move |app_handle, event| match event {
            RunEvent::ExitRequested { api, .. } => {
                api.prevent_exit();
            }
            RunEvent::Exit { .. } => {
                tauri::async_runtime::spawn(async {
                    log::debug!("killing ollama and zoo-node before exit");

                    // For some reason process::exit doesn't fire RunEvent::ExitRequested event in tauri
                    let mut zoo_node_manager_guard =
                        ZOO_NODE_MANAGER_INSTANCE.get().unwrap().write().await;
                    zoo_node_manager_guard.kill().await;
                    drop(zoo_node_manager_guard);
                    // Force exit the application
                    std::process::exit(0);
                });
            }
            #[cfg(target_os = "macos")]
            RunEvent::Reopen { .. } => {
                let main_window_label = "main";
                if let Some(window) = app_handle.get_webview_window(main_window_label) {
                    window.show().unwrap();
                    window.center().unwrap();
                    let _ = window.set_focus();
                } else {
                    let main_window_config = app_handle
                        .config()
                        .app
                        .windows
                        .iter()
                        .find(|w| w.label == main_window_label)
                        .unwrap()
                        .clone();
                    match tauri::WebviewWindowBuilder::from_config(app_handle, &main_window_config)
                    {
                        Ok(builder) => {
                            if let Err(e) = builder.build() {
                                log::error!("failed to build main window: {}", e);
                            }
                        }
                        Err(e) => {
                            log::error!("failed to create WebviewWindowBuilder from config: {}", e);
                        }
                    }
                }
            }
            RunEvent::Ready => {}
            RunEvent::Resumed => {}
            RunEvent::MainEventsCleared => {}
            RunEvent::WindowEvent {
                label,
                event: WindowEvent::Focused(focused),
                ..
            } => match label {
                label if label == Window::Spotlight.as_str() => {
                    if !focused {
                        if let Some(spotlight_window) =
                            app_handle.get_webview_window(Window::Spotlight.as_str())
                        {
                            if spotlight_window.is_visible().unwrap_or(false) {
                                let _ = spotlight_window.hide();
                            }
                        }
                    }
                }
                _ => {}
            },
            _ => {}
        });
}
