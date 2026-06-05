use std::process::Command;
use std::time::Duration;
use std::thread;

#[test]
fn test_zood_binary_exists() {
    // Check that the zood binary exists and is executable
    let zood_path = if cfg!(target_os = "macos") {
        "../external-binaries/zood/zood"
    } else if cfg!(target_os = "windows") {
        "../external-binaries/zood/zood.exe"
    } else {
        "../external-binaries/zood/zood"
    };
    
    assert!(
        std::path::Path::new(zood_path).exists(),
        "zood binary not found at: {}",
        zood_path
    );
    
    // Check that it's not a placeholder
    let contents = std::fs::read_to_string(zood_path).unwrap_or_default();
    assert!(
        !contents.contains("placeholder"),
        "zood binary is a placeholder script, not the actual binary"
    );
}

#[test]
fn test_zood_can_start() {
    // Test that zood can actually start
    let zood_path = if cfg!(target_os = "macos") {
        "../external-binaries/zood/zood"
    } else if cfg!(target_os = "windows") {
        "../external-binaries/zood/zood.exe"
    } else {
        "../external-binaries/zood/zood"
    };
    
    // Start zood with test configuration
    let mut child = Command::new(zood_path)
        .env("NODE_API_PORT", "12000") // Use different port for testing
        .env("NODE_WS_PORT", "12001")
        .env("NODE_PORT", "12002")
        .env("NODE_HTTPS_PORT", "12003")
        .env("NODE_STORAGE_PATH", "./test_storage")
        .env("INITIAL_AGENT_NAMES", "Test Agent")
        .env("INITIAL_AGENT_URLS", "http://127.0.0.1:11435")
        .env("INITIAL_AGENT_MODELS", "ollama:test")
        .env("INITIAL_AGENT_API_KEYS", ",")
        .spawn()
        .expect("Failed to start zood");
    
    // Give it a moment to start
    thread::sleep(Duration::from_secs(2));
    
    // Check if process is still running
    match child.try_wait() {
        Ok(None) => {
            // Process is still running - good!
            child.kill().ok(); // Clean up
        }
        Ok(Some(status)) => {
            panic!("zood exited immediately with status: {:?}", status);
        }
        Err(e) => {
            panic!("Error checking zood status: {}", e);
        }
    }
    
    // Clean up test storage
    std::fs::remove_dir_all("./test_storage").ok();
}

#[test]
fn test_ollama_binary_exists() {
    // Check that ollama binary exists for local AI
    let ollama_path = if cfg!(target_os = "macos") {
        "../external-binaries/ollama"
    } else if cfg!(target_os = "windows") {
        "../external-binaries/ollama.exe"
    } else {
        "../external-binaries/ollama"
    };
    
    // This is optional, so we just warn if it doesn't exist
    if !std::path::Path::new(ollama_path).exists() {
        println!("Warning: Ollama binary not found at: {}", ollama_path);
    }
}

#[test]
fn test_agent_configuration() {
    use crate::local_zoo_node::zoo_node_options::ZooNodeOptions;
    
    let options = ZooNodeOptions::default();
    
    // Check that we have matching agent configurations
    let names = options.initial_agent_names.unwrap_or_default();
    let urls = options.initial_agent_urls.unwrap_or_default();
    let models = options.initial_agent_models.unwrap_or_default();
    let api_keys = options.initial_agent_api_keys.unwrap_or_default();
    
    let name_count = names.split(',').count();
    let url_count = urls.split(',').count();
    let model_count = models.split(',').count();
    let api_key_count = api_keys.split(',').count();
    
    assert_eq!(
        name_count, url_count,
        "Agent names ({}) and URLs ({}) count mismatch",
        name_count, url_count
    );
    
    assert_eq!(
        name_count, model_count,
        "Agent names ({}) and models ({}) count mismatch",
        name_count, model_count
    );
    
    assert_eq!(
        name_count, api_key_count,
        "Agent names ({}) and API keys ({}) count mismatch",
        name_count, api_key_count
    );
    
    // Check that local agents have empty API keys
    if urls.contains("127.0.0.1:11435") {
        for (i, url) in urls.split(',').enumerate() {
            if url.contains("127.0.0.1:11435") {
                let api_key = api_keys.split(',').nth(i).unwrap_or("");
                assert!(
                    api_key.is_empty(),
                    "Local agent at index {} should have empty API key",
                    i
                );
            }
        }
    }
}