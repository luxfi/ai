use std::fs::{self, OpenOptions};
use std::io::Write;
use tauri::{AppHandle, Manager};

/// Whether this is a debug/development build (`cargo build` without --release).
/// The frontend uses this to skip the auto-updater check on local builds, which
/// have no published release to compare against and would otherwise log a 404.
#[tauri::command]
pub fn is_dev_build() -> bool {
    cfg!(debug_assertions)
}

#[tauri::command]
pub async fn write_debug_log(app: AppHandle, log: String) -> Result<(), String> {
    let app_data_dir = app
        .path()
        .app_data_dir()
        .map_err(|e| format!("Failed to get app data dir: {}", e))?;
    
    let logs_dir = app_data_dir.join("logs");
    
    // Create logs directory if it doesn't exist
    if !logs_dir.exists() {
        fs::create_dir_all(&logs_dir)
            .map_err(|e| format!("Failed to create logs directory: {}", e))?;
    }
    
    // Create log file name with current date
    let today = chrono::Local::now().format("%Y-%m-%d").to_string();
    let log_file = logs_dir.join(format!("debug-{}.log", today));
    
    // Append to log file
    let mut file = OpenOptions::new()
        .create(true)
        .append(true)
        .open(&log_file)
        .map_err(|e| format!("Failed to open log file: {}", e))?;
    
    writeln!(file, "{}", log)
        .map_err(|e| format!("Failed to write to log file: {}", e))?;
    
    Ok(())
}

#[tauri::command]
pub async fn save_debug_logs(app: AppHandle, logs: String) -> Result<String, String> {
    let app_data_dir = app
        .path()
        .app_data_dir()
        .map_err(|e| format!("Failed to get app data dir: {}", e))?;
    
    let logs_dir = app_data_dir.join("logs");
    
    // Create logs directory if it doesn't exist
    if !logs_dir.exists() {
        fs::create_dir_all(&logs_dir)
            .map_err(|e| format!("Failed to create logs directory: {}", e))?;
    }
    
    // Create export file name with timestamp
    let timestamp = chrono::Local::now().format("%Y%m%d_%H%M%S").to_string();
    let export_file = logs_dir.join(format!("export-{}.json", timestamp));
    
    // Write logs to file
    fs::write(&export_file, logs)
        .map_err(|e| format!("Failed to write logs to file: {}", e))?;
    
    Ok(export_file.to_string_lossy().to_string())
}

#[tauri::command]
pub async fn get_debug_logs_path(app: AppHandle) -> Result<String, String> {
    let app_data_dir = app
        .path()
        .app_data_dir()
        .map_err(|e| format!("Failed to get app data dir: {}", e))?;
    
    let logs_dir = app_data_dir.join("logs");
    Ok(logs_dir.to_string_lossy().to_string())
}

#[tauri::command]
pub async fn open_debug_logs_folder(app: AppHandle) -> Result<(), String> {
    let app_data_dir = app
        .path()
        .app_data_dir()
        .map_err(|e| format!("Failed to get app data dir: {}", e))?;
    
    let logs_dir = app_data_dir.join("logs");
    
    // Create logs directory if it doesn't exist
    if !logs_dir.exists() {
        fs::create_dir_all(&logs_dir)
            .map_err(|e| format!("Failed to create logs directory: {}", e))?;
    }
    
    // Open the folder
    opener::open(&logs_dir)
        .map_err(|e| format!("Failed to open logs folder: {}", e))?;
    
    Ok(())
}