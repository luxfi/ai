use log::{error, info};
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;
use tauri::Manager;
use tauri::Emitter;

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct UpdateInfo {
    pub engine: String,
    pub current_version: String,
    pub latest_version: String,
    pub download_url: String,
    pub release_notes: String,
    pub update_available: bool,
    pub size: Option<u64>,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct UpdateProgress {
    pub engine: String,
    pub status: UpdateStatus,
    pub progress: f64,
    pub message: String,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
#[serde(rename_all = "lowercase")]
pub enum UpdateStatus {
    Checking,
    Downloading,
    Installing,
    Complete,
    Error,
    Idle,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct OllamaRelease {
    pub tag_name: String,
    pub name: String,
    pub body: String,
    pub assets: Vec<OllamaAsset>,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct OllamaAsset {
    pub name: String,
    pub browser_download_url: String,
    pub size: u64,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct HanzoUpdateResponse {
    pub version: String,
    pub download_url: String,
    pub release_notes: String,
    pub size: u64,
}

/// Check for Ollama updates
#[tauri::command]
pub async fn check_ollama_update(current_version: String) -> Result<UpdateInfo, String> {
    info!("Checking for Ollama updates. Current version: {}", current_version);

    // Fetch latest release from GitHub
    let client = reqwest::Client::new();
    let response = client
        .get("https://api.github.com/repos/ollama/ollama/releases/latest")
        .header("User-Agent", "Zoo-Desktop")
        .send()
        .await
        .map_err(|e| format!("Failed to check for updates: {}", e))?;

    if !response.status().is_success() {
        return Err(format!("GitHub API returned status: {}", response.status()));
    }

    let release: OllamaRelease = response
        .json()
        .await
        .map_err(|e| format!("Failed to parse release data: {}", e))?;

    let latest_version = release.tag_name.trim_start_matches('v').to_string();
    let update_available = version_compare(&current_version, &latest_version);

    // Find appropriate asset for current platform
    let platform_asset = find_platform_asset(&release.assets)?;

    Ok(UpdateInfo {
        engine: "Ollama".to_string(),
        current_version: current_version.clone(),
        latest_version: latest_version.clone(),
        download_url: platform_asset.browser_download_url.clone(),
        release_notes: release.body.clone(),
        update_available,
        size: Some(platform_asset.size),
    })
}

/// Check for Hanzo Engine updates
#[tauri::command]
pub async fn check_hanzo_update(current_version: String) -> Result<UpdateInfo, String> {
    info!("Checking for Hanzo Engine updates. Current version: {}", current_version);

    // Check against Hanzo update server
    let client = reqwest::Client::new();
    let platform = get_platform_string();
    let arch = get_arch_string();

    let url = format!(
        "https://updates.hanzo.ai/api/check?version={}&platform={}&arch={}&product=hanzo-engine",
        current_version, platform, arch
    );

    let response = client
        .get(&url)
        .header("User-Agent", "Zoo-Desktop")
        .send()
        .await
        .map_err(|e| format!("Failed to check for updates: {}", e))?;

    if !response.status().is_success() {
        return Err(format!("Update server returned status: {}", response.status()));
    }

    let update_response: HanzoUpdateResponse = response
        .json()
        .await
        .map_err(|e| format!("Failed to parse update data: {}", e))?;

    let update_available = version_compare(&current_version, &update_response.version);

    Ok(UpdateInfo {
        engine: "Hanzo Engine".to_string(),
        current_version: current_version.clone(),
        latest_version: update_response.version,
        download_url: update_response.download_url,
        release_notes: update_response.release_notes,
        update_available,
        size: Some(update_response.size),
    })
}

/// Download and install runtime update
#[tauri::command]
pub async fn download_and_install_update(
    app_handle: tauri::AppHandle,
    update_info: UpdateInfo,
) -> Result<(), String> {
    info!("Starting download for {} update", update_info.engine);

    // Emit progress event
    let _ = app_handle.emit("update-progress", UpdateProgress {
        engine: update_info.engine.clone(),
        status: UpdateStatus::Downloading,
        progress: 0.0,
        message: format!("Starting download of {} {}", update_info.engine, update_info.latest_version),
    });

    // Create temp directory for download
    let temp_dir = std::env::temp_dir().join("hanzo-updates");
    fs::create_dir_all(&temp_dir)
        .map_err(|e| format!("Failed to create temp directory: {}", e))?;

    let filename = extract_filename_from_url(&update_info.download_url)?;
    let download_path = temp_dir.join(&filename);

    // Download the update
    let client = reqwest::Client::new();
    let mut response = client
        .get(&update_info.download_url)
        .header("User-Agent", "Zoo-Desktop")
        .send()
        .await
        .map_err(|e| format!("Failed to download update: {}", e))?;

    if !response.status().is_success() {
        return Err(format!("Download failed with status: {}", response.status()));
    }

    let total_size = response
        .content_length()
        .unwrap_or(update_info.size.unwrap_or(0));

    let mut file = fs::File::create(&download_path)
        .map_err(|e| format!("Failed to create download file: {}", e))?;

    let mut downloaded: u64 = 0;
    let mut last_progress = 0.0;

    // Stream download with progress
    while let Some(chunk) = response
        .chunk()
        .await
        .map_err(|e| format!("Download interrupted: {}", e))?
    {
        use std::io::Write;
        file.write_all(&chunk)
            .map_err(|e| format!("Failed to write chunk: {}", e))?;

        downloaded += chunk.len() as u64;
        let progress = (downloaded as f64 / total_size as f64) * 100.0;

        // Only emit progress if it changed significantly (every 1%)
        if progress - last_progress >= 1.0 {
            last_progress = progress;
            let _ = app_handle.emit("update-progress", UpdateProgress {
                engine: update_info.engine.clone(),
                status: UpdateStatus::Downloading,
                progress,
                message: format!(
                    "Downloading {} {} - {:.0}%",
                    update_info.engine, update_info.latest_version, progress
                ),
            });
        }
    }

    // Emit installation progress
    let _ = app_handle.emit("update-progress", UpdateProgress {
        engine: update_info.engine.clone(),
        status: UpdateStatus::Installing,
        progress: 100.0,
        message: format!("Installing {} {}", update_info.engine, update_info.latest_version),
    });

    // Install the update based on engine type
    match update_info.engine.as_str() {
        "Ollama" => install_ollama_update(download_path)?,
        "Hanzo Engine" => install_hanzo_update(app_handle.clone(), download_path)?,
        _ => return Err(format!("Unknown engine: {}", update_info.engine)),
    }

    // Emit completion
    let _ = app_handle.emit("update-progress", UpdateProgress {
        engine: update_info.engine.clone(),
        status: UpdateStatus::Complete,
        progress: 100.0,
        message: format!(
            "{} {} installed successfully. Restart required.",
            update_info.engine, update_info.latest_version
        ),
    });

    Ok(())
}

/// Install Ollama update
fn install_ollama_update(download_path: PathBuf) -> Result<(), String> {
    #[cfg(target_os = "macos")]
    {
        // On macOS, Ollama is typically installed as an app bundle
        // We need to extract and replace the binary
        std::process::Command::new("tar")
            .args(&["-xzf", download_path.to_str().unwrap(), "-C", "/usr/local/bin/"])
            .output()
            .map_err(|e| format!("Failed to extract Ollama: {}", e))?;
    }

    #[cfg(target_os = "windows")]
    {
        // On Windows, copy the exe to the installation directory
        let install_path = PathBuf::from("C:\\Program Files\\Ollama\\ollama.exe");
        fs::copy(&download_path, &install_path)
            .map_err(|e| format!("Failed to install Ollama: {}", e))?;
    }

    #[cfg(target_os = "linux")]
    {
        // On Linux, extract to /usr/local/bin
        std::process::Command::new("tar")
            .args(&["-xzf", download_path.to_str().unwrap(), "-C", "/usr/local/bin/"])
            .output()
            .map_err(|e| format!("Failed to extract Ollama: {}", e))?;

        // Make executable
        std::process::Command::new("chmod")
            .args(&["+x", "/usr/local/bin/ollama"])
            .output()
            .map_err(|e| format!("Failed to make Ollama executable: {}", e))?;
    }

    Ok(())
}

/// Install Hanzo Engine update
fn install_hanzo_update(app_handle: tauri::AppHandle, download_path: PathBuf) -> Result<(), String> {
    // Get the resource directory where binaries are stored
    let resource_dir = app_handle
        .path()
        .resource_dir()
        .map_err(|e| format!("Failed to get resource directory: {}", e))?;

    let binary_name = if cfg!(target_os = "windows") {
        "hanzo-node.exe"
    } else {
        "hanzo-node"
    };

    let install_path = resource_dir.join("binaries").join(binary_name);

    // Backup current binary
    let backup_path = install_path.with_extension("backup");
    if install_path.exists() {
        fs::copy(&install_path, &backup_path)
            .map_err(|e| format!("Failed to backup current binary: {}", e))?;
    }

    // Install new binary
    fs::copy(&download_path, &install_path)
        .map_err(|e| format!("Failed to install new binary: {}", e))?;

    // Make executable on Unix systems
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        let mut perms = fs::metadata(&install_path)
            .map_err(|e| format!("Failed to get file metadata: {}", e))?
            .permissions();
        perms.set_mode(0o755);
        fs::set_permissions(&install_path, perms)
            .map_err(|e| format!("Failed to set permissions: {}", e))?;
    }

    Ok(())
}

/// Helper to compare versions
fn version_compare(current: &str, latest: &str) -> bool {
    // Simple version comparison - could be enhanced with semver crate
    let current_parts: Vec<u32> = current
        .split('.')
        .filter_map(|s| s.parse().ok())
        .collect();
    let latest_parts: Vec<u32> = latest
        .split('.')
        .filter_map(|s| s.parse().ok())
        .collect();

    for i in 0..std::cmp::max(current_parts.len(), latest_parts.len()) {
        let current_part = current_parts.get(i).unwrap_or(&0);
        let latest_part = latest_parts.get(i).unwrap_or(&0);

        if latest_part > current_part {
            return true;
        } else if latest_part < current_part {
            return false;
        }
    }

    false
}

/// Get platform string for update checks
fn get_platform_string() -> &'static str {
    #[cfg(target_os = "macos")]
    return "darwin";
    #[cfg(target_os = "windows")]
    return "windows";
    #[cfg(target_os = "linux")]
    return "linux";
}

/// Get architecture string for update checks
fn get_arch_string() -> &'static str {
    #[cfg(target_arch = "x86_64")]
    return "x64";
    #[cfg(target_arch = "aarch64")]
    return "arm64";
    #[cfg(not(any(target_arch = "x86_64", target_arch = "aarch64")))]
    return "unknown";
}

/// Find appropriate asset for current platform
fn find_platform_asset(assets: &[OllamaAsset]) -> Result<&OllamaAsset, String> {
    let platform = get_platform_string();
    let arch = get_arch_string();

    // Look for matching asset
    for asset in assets {
        let name = asset.name.to_lowercase();
        if name.contains(platform) && name.contains(arch) {
            return Ok(asset);
        }
    }

    // Fallback patterns
    for asset in assets {
        let name = asset.name.to_lowercase();
        if name.contains(platform) {
            return Ok(asset);
        }
    }

    Err("No compatible binary found for this platform".to_string())
}

/// Extract filename from URL
fn extract_filename_from_url(url: &str) -> Result<String, String> {
    url.split('/')
        .last()
        .map(|s| s.to_string())
        .ok_or_else(|| "Invalid URL format".to_string())
}