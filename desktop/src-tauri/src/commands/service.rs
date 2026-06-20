//! Install-as-service commands (Linux / systemd USER units).
//!
//! Phase 0 goal: let the desktop run as a headless run-once daemon, supervised
//! by the user's own systemd instance — NO root, NO system units, NO sudo.
//!
//! We write a `--user` unit to `~/.config/systemd/user/lux-desktop.service`
//! that launches THIS binary (resolved via `std::env::current_exe()`, or
//! `$APPIMAGE` when bundled as an AppImage) with `--headless` and an
//! auto-restart policy (see `render_unit` for why `on-failure` over `always`),
//! then:
//!   - `systemctl --user daemon-reload`
//!   - `systemctl --user enable --now lux-desktop`
//!   - `loginctl enable-linger $USER`   (so it survives logout / starts at boot)
//!
//! On non-Linux platforms these commands compile but return a friendly error so
//! the UI can hide / disable the control. (macOS would use a LaunchAgent and
//! Windows a Run key / Task Scheduler entry — out of scope for Phase 0.)

#[cfg(target_os = "linux")]
use std::process::Command;

/// Status of the systemd user service, surfaced to the Settings UI.
#[derive(Debug, Clone, serde::Serialize)]
pub struct ServiceStatus {
    /// Whether the platform supports this install flow at all (Linux + systemd).
    pub supported: bool,
    /// The unit file exists on disk.
    pub installed: bool,
    /// `systemctl --user is-enabled` reports enabled (starts on login/boot).
    pub enabled: bool,
    /// `systemctl --user is-active` reports active (currently running).
    pub active: bool,
    /// Absolute path of the unit file (for display / debugging).
    pub unit_path: String,
    /// Human-readable note when unsupported or partially configured.
    pub detail: Option<String>,
}

impl ServiceStatus {
    fn unsupported(detail: &str) -> Self {
        ServiceStatus {
            supported: false,
            installed: false,
            enabled: false,
            active: false,
            unit_path: String::new(),
            detail: Some(detail.to_string()),
        }
    }
}

#[cfg(target_os = "linux")]
const SERVICE_NAME: &str = "lux-desktop";

#[cfg(target_os = "linux")]
fn unit_file_path() -> Result<std::path::PathBuf, String> {
    // Prefer $XDG_CONFIG_HOME, falling back to ~/.config (systemd's own default).
    let base = if let Ok(xdg) = std::env::var("XDG_CONFIG_HOME") {
        if !xdg.trim().is_empty() {
            std::path::PathBuf::from(xdg)
        } else {
            home_config_dir()?
        }
    } else {
        home_config_dir()?
    };
    Ok(base
        .join("systemd")
        .join("user")
        .join(format!("{SERVICE_NAME}.service")))
}

#[cfg(target_os = "linux")]
fn home_config_dir() -> Result<std::path::PathBuf, String> {
    let home = std::env::var("HOME").map_err(|_| "HOME is not set".to_string())?;
    Ok(std::path::PathBuf::from(home).join(".config"))
}

/// Resolve the binary to launch from the unit. We use the real, canonical path
/// of the running executable so the unit keeps working across shells/cwd.
///
/// NOTE: when the app runs as an AppImage, `current_exe()` points at the
/// extracted binary inside the mounted squashfs (`/tmp/.mount_*`), which is NOT
/// stable across launches. In that case we prefer `$APPIMAGE` (the path of the
/// .AppImage file itself), which systemd can exec directly.
#[cfg(target_os = "linux")]
fn exec_path() -> Result<String, String> {
    if let Ok(appimage) = std::env::var("APPIMAGE") {
        if !appimage.trim().is_empty() && std::path::Path::new(&appimage).exists() {
            return Ok(appimage);
        }
    }
    let exe = std::env::current_exe().map_err(|e| format!("current_exe() failed: {e}"))?;
    let exe = std::fs::canonicalize(&exe).unwrap_or(exe);
    Ok(exe.to_string_lossy().into_owned())
}

#[cfg(target_os = "linux")]
fn run_systemctl(args: &[&str]) -> Result<std::process::Output, String> {
    Command::new("systemctl")
        .args(args)
        .output()
        .map_err(|e| format!("failed to run `systemctl {}`: {e}", args.join(" ")))
}

/// Whether `systemctl --user` is usable in this session at all (i.e. there is a
/// user systemd manager to talk to). Containers / minimal envs may lack it.
#[cfg(target_os = "linux")]
fn user_systemd_available() -> bool {
    Command::new("systemctl")
        .args(["--user", "is-system-running"])
        .output()
        .map(|o| {
            // `is-system-running` exits non-zero for "degraded"/"starting" but
            // still proves a manager is present and answering. We only treat a
            // hard spawn failure (no systemctl / no bus) as unavailable, which
            // surfaces as Err above. Any captured output means it answered.
            !o.stdout.is_empty() || !o.stderr.is_empty() || o.status.success()
        })
        .unwrap_or(false)
}

/// Build the unit file contents.
#[cfg(target_os = "linux")]
fn render_unit(exec: &str) -> String {
    // %h is expanded by systemd to the user's home dir. We pass --headless so
    // the daemon starts with no visible window (tray + node only).
    //
    // Restart policy: we deliberately use `on-failure` (NOT `always`). The app
    // uses tauri-plugin-single-instance, so a SECOND instance — and the tray
    // "Quit" — both call `process::exit(0)` (a CLEAN exit). With `always`,
    // systemd would respawn on those clean exits, causing a restart loop if a
    // foreground GUI instance is already running, and making the tray "Quit"
    // un-quittable. `on-failure` still auto-restarts on crashes (non-zero /
    // signal), which is the actual resilience goal, while letting a clean exit
    // stay stopped. Updates are applied via an explicit `systemctl restart`
    // (see `restart_system_service`), which restarts regardless of this policy.
    //
    // `default.target` is the user-session analogue of `multi-user.target`.
    format!(
        "[Unit]\n\
         Description=Zoo Desktop (headless daemon)\n\
         Documentation=https://github.com/zooai/app\n\
         After=network-online.target\n\
         Wants=network-online.target\n\
         \n\
         [Service]\n\
         Type=simple\n\
         ExecStart=\"{exec}\" --headless\n\
         Restart=on-failure\n\
         RestartSec=3\n\
         Environment=ZOO_HEADLESS=1\n\
         \n\
         [Install]\n\
         WantedBy=default.target\n"
    )
}

/// Install + enable + start the systemd USER service, and enable linger so it
/// survives logout and starts at boot. Idempotent: re-running rewrites the unit
/// and re-enables.
#[tauri::command]
pub async fn install_system_service() -> Result<ServiceStatus, String> {
    #[cfg(not(target_os = "linux"))]
    {
        return Err("Installing as a system service is only supported on Linux (systemd) in this build.".to_string());
    }

    #[cfg(target_os = "linux")]
    {
        if !user_systemd_available() {
            return Err("No user systemd manager is available in this session (`systemctl --user` failed).".to_string());
        }

        let exec = exec_path()?;
        let unit_path = unit_file_path()?;
        if let Some(parent) = unit_path.parent() {
            std::fs::create_dir_all(parent)
                .map_err(|e| format!("failed to create {}: {e}", parent.display()))?;
        }
        std::fs::write(&unit_path, render_unit(&exec))
            .map_err(|e| format!("failed to write unit {}: {e}", unit_path.display()))?;
        log::info!("wrote systemd user unit to {}", unit_path.display());

        let reload = run_systemctl(&["--user", "daemon-reload"])?;
        if !reload.status.success() {
            return Err(format!(
                "`systemctl --user daemon-reload` failed: {}",
                String::from_utf8_lossy(&reload.stderr)
            ));
        }

        let enable = run_systemctl(&["--user", "enable", "--now", SERVICE_NAME])?;
        if !enable.status.success() {
            return Err(format!(
                "`systemctl --user enable --now {SERVICE_NAME}` failed: {}",
                String::from_utf8_lossy(&enable.stderr)
            ));
        }

        // Best-effort: enable-linger so the unit runs without an active login
        // session (i.e. on boot). Failure here is non-fatal — the service still
        // works while the user is logged in — so we only log it.
        if let Ok(user) = std::env::var("USER") {
            match Command::new("loginctl")
                .args(["enable-linger", &user])
                .output()
            {
                Ok(out) if out.status.success() => {
                    log::info!("enabled linger for user {user}");
                }
                Ok(out) => {
                    log::warn!(
                        "loginctl enable-linger failed (non-fatal): {}",
                        String::from_utf8_lossy(&out.stderr)
                    );
                }
                Err(e) => log::warn!("could not run loginctl enable-linger (non-fatal): {e}"),
            }
        }

        return system_service_status().await;
    }

    // Unreachable on every supported target (each cfg block above returns);
    // present so the function type-checks regardless of `target_os`.
    #[allow(unreachable_code)]
    return Err("unsupported platform".to_string());
}

/// Disable + stop + remove the unit, and disable linger. Idempotent.
#[tauri::command]
pub async fn uninstall_system_service() -> Result<ServiceStatus, String> {
    #[cfg(not(target_os = "linux"))]
    {
        return Err("System service management is only supported on Linux (systemd) in this build.".to_string());
    }

    #[cfg(target_os = "linux")]
    {
        if !user_systemd_available() {
            return Err("No user systemd manager is available in this session.".to_string());
        }

        // disable --now both disables (removes the WantedBy symlink) and stops.
        // Ignore failure: the unit may already be gone; we still rm + reload.
        let _ = run_systemctl(&["--user", "disable", "--now", SERVICE_NAME]);

        let unit_path = unit_file_path()?;
        if unit_path.exists() {
            std::fs::remove_file(&unit_path)
                .map_err(|e| format!("failed to remove unit {}: {e}", unit_path.display()))?;
            log::info!("removed systemd user unit {}", unit_path.display());
        }

        let _ = run_systemctl(&["--user", "daemon-reload"]);
        // Reset any lingering failed state from the now-removed unit.
        let _ = run_systemctl(&["--user", "reset-failed", SERVICE_NAME]);

        // Best-effort disable-linger (don't fail the uninstall on it).
        if let Ok(user) = std::env::var("USER") {
            if let Err(e) = Command::new("loginctl")
                .args(["disable-linger", &user])
                .output()
            {
                log::warn!("could not run loginctl disable-linger (non-fatal): {e}");
            }
        }

        return system_service_status().await;
    }

    #[allow(unreachable_code)]
    return Err("unsupported platform".to_string());
}

/// Report whether the service is installed / enabled / active.
#[tauri::command]
pub async fn system_service_status() -> Result<ServiceStatus, String> {
    #[cfg(not(target_os = "linux"))]
    {
        return Ok(ServiceStatus::unsupported(
            "Running as a system service is supported on Linux (systemd) only.",
        ));
    }

    #[cfg(target_os = "linux")]
    {
        if !user_systemd_available() {
            return Ok(ServiceStatus::unsupported(
                "No user systemd manager is available in this session.",
            ));
        }

        let unit_path = unit_file_path()?;
        let installed = unit_path.exists();

        // `is-enabled` / `is-active` exit non-zero when not enabled/active; we
        // read stdout rather than the exit code so we don't conflate "disabled"
        // with an error.
        let enabled = run_systemctl(&["--user", "is-enabled", SERVICE_NAME])
            .map(|o| String::from_utf8_lossy(&o.stdout).trim() == "enabled")
            .unwrap_or(false);
        let active = run_systemctl(&["--user", "is-active", SERVICE_NAME])
            .map(|o| String::from_utf8_lossy(&o.stdout).trim() == "active")
            .unwrap_or(false);

        return Ok(ServiceStatus {
            supported: true,
            installed,
            enabled,
            active,
            unit_path: unit_path.to_string_lossy().into_owned(),
            detail: None,
        });
    }

    #[allow(unreachable_code)]
    return Ok(ServiceStatus::unsupported("unsupported platform"));
}

/// Restart the systemd USER unit cleanly. Used by the updater after applying an
/// update while running under the service: relaunching the *process* would
/// detach it from systemd (and Restart=always would then fight us). Instead we
/// ask systemd to restart the unit, which re-execs the freshly-updated binary.
///
/// Returns `Ok(true)` if a restart was issued, `Ok(false)` if we are NOT running
/// under the service (caller should fall back to a normal relaunch).
#[tauri::command]
pub async fn restart_system_service() -> Result<bool, String> {
    #[cfg(not(target_os = "linux"))]
    {
        return Ok(false);
    }

    #[cfg(target_os = "linux")]
    {
        // Heuristic: we're running under our own unit when systemd set
        // INVOCATION_ID for us AND the unit is installed+active. We avoid
        // restarting if the user merely has the unit installed but is currently
        // running a *separate* foreground instance.
        let under_systemd = std::env::var("INVOCATION_ID").is_ok();
        if !under_systemd {
            return Ok(false);
        }
        let status = system_service_status().await?;
        if !(status.installed && status.active) {
            return Ok(false);
        }
        // `--no-block` so we don't deadlock waiting on our own teardown.
        let out = run_systemctl(&["--user", "restart", "--no-block", SERVICE_NAME])?;
        if !out.status.success() {
            return Err(format!(
                "`systemctl --user restart {SERVICE_NAME}` failed: {}",
                String::from_utf8_lossy(&out.stderr)
            ));
        }
        return Ok(true);
    }

    #[allow(unreachable_code)]
    return Ok(false);
}
