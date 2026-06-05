use once_cell::sync::OnceCell;
use tauri::{
    menu::{
        CheckMenuItem, CheckMenuItemBuilder, MenuBuilder, MenuItem, MenuItemBuilder, SubmenuBuilder,
    },
    tray::{MouseButton, TrayIconBuilder, TrayIconEvent},
    AppHandle, Emitter, Manager, Runtime,
};

use crate::{
    commands::service::{install_system_service, system_service_status, uninstall_system_service},
    globals::ZOO_NODE_MANAGER_INSTANCE,
    windows::{recreate_window, Window},
};

/// Event emitted to the frontend when the user picks "Check for Updates" from
/// the tray. The main window listens for this and kicks off the existing
/// updater query (which owns the download/install UX). We use an event rather
/// than driving the updater from Rust so there's a single source of truth for
/// the update flow (the `updater-client` in the frontend).
pub const TRAY_CHECK_UPDATES_EVENT: &str = "tray://check-for-updates";

/// Handles to the tray menu items whose label / checked / enabled state changes
/// at runtime, kept so we can update them in place instead of rebuilding the
/// whole menu (which can flicker and drop submenu state on some platforms).
struct TrayMenuItems<R: Runtime> {
    /// Non-clickable status line: "Node: running" / "Node: stopped".
    node_status: MenuItem<R>,
    /// "Start Node" — enabled only when the node is stopped.
    node_start: MenuItem<R>,
    /// "Stop Node" — enabled only when the node is running.
    node_stop: MenuItem<R>,
    /// "Launch on login" — reflects systemd user-service enabled state.
    launch_on_login: CheckMenuItem<R>,
}

static TRAY_MENU_ITEMS: OnceCell<TrayMenuItems<tauri::Wry>> = OnceCell::new();

pub fn create_tray(app: &AppHandle) -> tauri::Result<()> {
    // --- Static items -------------------------------------------------------
    let open_dashboard = MenuItemBuilder::with_id("open_dashboard", "Open Dashboard").build(app)?;
    let zoo_spotlight = MenuItemBuilder::with_id("zoo_spotlight", "Spotlight")
        .accelerator("CmdOrCtrl+Shift+J")
        .build(app)?;
    let open_node_manager =
        MenuItemBuilder::with_id("open_node_manager", "Open Node Manager").build(app)?;
    let check_for_updates =
        MenuItemBuilder::with_id("check_for_updates", "Check for Updates…").build(app)?;
    let quit = MenuItemBuilder::with_id("quit", "Quit").build(app)?;

    // --- Dynamic items (handles retained) -----------------------------------
    // The status line is disabled (acts as a label). We can't read async node
    // state synchronously here, so seed with "checking…" and let an async
    // refresh fix it up right after the tray is built.
    let node_status = MenuItemBuilder::with_id("node_status", "Node: checking…")
        .enabled(false)
        .build(app)?;
    let node_start = MenuItemBuilder::with_id("node_start", "Start Node").build(app)?;
    let node_stop = MenuItemBuilder::with_id("node_stop", "Stop Node").build(app)?;
    let launch_on_login = CheckMenuItemBuilder::with_id("launch_on_login", "Launch on login")
        .checked(false)
        .build(app)?;

    let node_submenu = SubmenuBuilder::new(app, "Node")
        .item(&node_status)
        .separator()
        .item(&node_start)
        .item(&node_stop)
        .separator()
        .item(&open_node_manager)
        .build()?;

    let menu = MenuBuilder::new(app)
        .item(&open_dashboard)
        .item(&zoo_spotlight)
        .separator()
        .item(&node_submenu)
        .separator()
        .item(&launch_on_login)
        .item(&check_for_updates)
        .separator()
        .item(&quit)
        .build()?;

    // Stash dynamic handles for later updates. `set` only succeeds once; on a
    // (re)create we ignore the error and reuse the existing handles.
    let _ = TRAY_MENU_ITEMS.set(TrayMenuItems {
        node_status,
        node_start,
        node_stop,
        launch_on_login,
    });

    let is_template = cfg!(target_os = "macos");
    let icon = if cfg!(target_os = "macos") {
        tauri::image::Image::from_bytes(include_bytes!("../icons/tray-icon-macos.png"))?
    } else {
        app.default_window_icon().unwrap().clone()
    };

    let _ = TrayIconBuilder::with_id("tray")
        .icon(icon)
        .icon_as_template(is_template)
        .tooltip("Zoo")
        .on_tray_icon_event(|tray, event| {
            // On Windows the menu is NOT shown on left click (see
            // show_menu_on_left_click(false) below), so a left click opens the
            // dashboard directly — matching the prior behavior.
            if cfg!(target_os = "windows") {
                if let TrayIconEvent::Click { button, .. } = event {
                    if button == MouseButton::Left {
                        recreate_window(tray.app_handle().clone(), Window::Main, true);
                    }
                }
            }
        })
        // Was `menu_on_left_click`, deprecated since tauri 2.2.0 in favor of
        // `show_menu_on_left_click` (identical behavior). Left click shows the
        // menu everywhere except Windows, where it opens the dashboard.
        .show_menu_on_left_click(!cfg!(target_os = "windows"))
        .menu(&menu)
        .on_menu_event(move |tray, event| {
            let app = tray.app_handle().clone();
            match event.id().as_ref() {
                "open_dashboard" => {
                    recreate_window(app, Window::Main, true);
                }
                "open_node_manager" => {
                    recreate_window(app, Window::ZooNodeManager, true);
                }
                "zoo_spotlight" => {
                    recreate_window(app, Window::Spotlight, true);
                }
                "node_start" => {
                    tauri::async_runtime::spawn(async move {
                        let mut guard = ZOO_NODE_MANAGER_INSTANCE.get().unwrap().write().await;
                        if let Err(e) = guard.spawn().await {
                            log::error!("tray: failed to start node: {e}");
                        }
                        drop(guard);
                        refresh_node_status().await;
                    });
                }
                "node_stop" => {
                    tauri::async_runtime::spawn(async move {
                        let mut guard = ZOO_NODE_MANAGER_INSTANCE.get().unwrap().write().await;
                        guard.kill().await;
                        drop(guard);
                        refresh_node_status().await;
                    });
                }
                "launch_on_login" => {
                    tauri::async_runtime::spawn(async move {
                        toggle_launch_on_login().await;
                    });
                }
                "check_for_updates" => {
                    // Ensure a window exists to receive + render the result,
                    // then ask the frontend updater to run a check.
                    let _ = recreate_window(app.clone(), Window::Main, true);
                    if let Err(e) = app.emit(TRAY_CHECK_UPDATES_EVENT, ()) {
                        log::error!("tray: failed to emit check-for-updates: {e}");
                    }
                }
                "quit" => {
                    tauri::async_runtime::spawn(async move {
                        // process::exit doesn't fire RunEvent::ExitRequested in
                        // tauri, so tear the node down explicitly here before
                        // exiting.
                        let mut guard = ZOO_NODE_MANAGER_INSTANCE.get().unwrap().write().await;
                        if guard.is_running().await {
                            guard.kill().await;
                        }
                        drop(guard);
                        std::process::exit(0);
                    });
                }
                _ => (),
            }
        })
        .build(app)?;

    // Seed dynamic state asynchronously (node running? service enabled?).
    tauri::async_runtime::spawn(async move {
        refresh_node_status().await;
        refresh_launch_on_login().await;
    });

    Ok(())
}

/// Re-read node running state and update the status line + start/stop
/// enablement. Cheap enough to call on every node state-change event.
pub async fn refresh_node_status() {
    let Some(items) = TRAY_MENU_ITEMS.get() else {
        return;
    };
    let running = match ZOO_NODE_MANAGER_INSTANCE.get() {
        Some(mgr) => mgr.read().await.is_running().await,
        None => false,
    };
    let _ = items
        .node_status
        .set_text(if running { "Node: running" } else { "Node: stopped" });
    let _ = items.node_start.set_enabled(!running);
    let _ = items.node_stop.set_enabled(running);
}

/// Re-read systemd service state and reflect it on the "Launch on login" check.
pub async fn refresh_launch_on_login() {
    let Some(items) = TRAY_MENU_ITEMS.get() else {
        return;
    };
    let (enabled, supported) = match system_service_status().await {
        Ok(s) => (s.enabled, s.supported),
        Err(_) => (false, false),
    };
    let _ = items.launch_on_login.set_checked(enabled);
    // On platforms without the systemd flow, grey the toggle out.
    let _ = items.launch_on_login.set_enabled(supported);
}

/// Toggle "Launch on login" by installing/uninstalling the systemd user
/// service, then sync the checkbox to the resulting (authoritative) state.
async fn toggle_launch_on_login() {
    let currently_enabled = system_service_status()
        .await
        .map(|s| s.enabled)
        .unwrap_or(false);

    let result = if currently_enabled {
        uninstall_system_service().await
    } else {
        install_system_service().await
    };

    match result {
        Ok(status) => {
            if let Some(items) = TRAY_MENU_ITEMS.get() {
                let _ = items.launch_on_login.set_checked(status.enabled);
            }
        }
        Err(e) => {
            log::error!("tray: launch-on-login toggle failed: {e}");
            // Revert the visual to the real state on failure.
            refresh_launch_on_login().await;
        }
    }
}

/// Subscribe to the node manager's broadcast events and refresh the tray status
/// line on every state change, keeping it live without polling. Called from
/// `main.rs` setup after the manager + tray exist.
pub fn wire_node_events(app: &AppHandle) {
    let _ = app;
    tauri::async_runtime::spawn(async move {
        let mut receiver = {
            let guard = match ZOO_NODE_MANAGER_INSTANCE.get() {
                Some(m) => m,
                None => return,
            };
            let mut g = guard.write().await;
            g.subscribe_to_events()
        };
        // Initial sync, then update on every state change.
        refresh_node_status().await;
        while receiver.recv().await.is_ok() {
            refresh_node_status().await;
        }
    });
}
