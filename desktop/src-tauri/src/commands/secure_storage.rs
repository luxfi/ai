use keyring::Entry;
use serde_json;
use std::collections::HashMap;
use std::sync::Mutex;
use tauri::State;

// In-memory fallback for development
pub type SecureStore = Mutex<HashMap<String, String>>;

const SERVICE_NAME: &str = "io.zoo.desktop";

#[tauri::command]
pub async fn secure_storage_set(
    key: String,
    value: String,
    _store: State<'_, SecureStore>,
) -> Result<(), String> {
    // Try to use OS keyring first
    match Entry::new(SERVICE_NAME, &key) {
        Ok(entry) => {
            entry
                .set_password(&value)
                .map_err(|e| format!("Failed to store in keyring: {}", e))?;
            Ok(())
        }
        Err(e) => {
            // Fallback to in-memory storage (development only)
            log::warn!("Keyring unavailable, using in-memory storage: {}", e);
            let mut store = _store.lock().unwrap();
            store.insert(key, value);
            Ok(())
        }
    }
}

#[tauri::command]
pub async fn secure_storage_get(
    key: String,
    _store: State<'_, SecureStore>,
) -> Result<String, String> {
    // Try keyring first
    match Entry::new(SERVICE_NAME, &key) {
        Ok(entry) => match entry.get_password() {
            Ok(value) => Ok(value),
            Err(keyring::Error::NoEntry) => Err("Key not found".to_string()),
            Err(e) => Err(format!("Failed to retrieve from keyring: {}", e)),
        },
        Err(_) => {
            // Fallback to in-memory storage
            let store = _store.lock().unwrap();
            store
                .get(&key)
                .cloned()
                .ok_or_else(|| "Key not found".to_string())
        }
    }
}

#[tauri::command]
pub async fn secure_storage_delete(
    key: String,
    _store: State<'_, SecureStore>,
) -> Result<(), String> {
    // Try keyring first
    match Entry::new(SERVICE_NAME, &key) {
        Ok(entry) => {
            entry
                .delete_credential()
                .map_err(|e| format!("Failed to delete from keyring: {}", e))?;
            Ok(())
        }
        Err(_) => {
            // Fallback to in-memory storage
            let mut store = _store.lock().unwrap();
            store.remove(&key);
            Ok(())
        }
    }
}

pub fn init_secure_storage() -> SecureStore {
    Mutex::new(HashMap::new())
}
