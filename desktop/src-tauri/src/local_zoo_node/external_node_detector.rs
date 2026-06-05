use std::time::Duration;
use anyhow::Result;

/// Checks if an external Zoo node is running on the specified ports
pub async fn is_external_node_running(
    api_port: u16,
    ollama_port: u16,
) -> (bool, bool) {
    let hanzo_node_running = check_port_service(api_port, "/v2/health_check").await;
    let ollama_running = check_port_service(ollama_port, "/").await;
    
    (hanzo_node_running, ollama_running)
}

/// Check if a service is running on a specific port with a health endpoint
async fn check_port_service(port: u16, endpoint: &str) -> bool {
    let url = format!("http://127.0.0.1:{}{}", port, endpoint);
    
    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(2))
        .build();
    
    if let Ok(client) = client {
        if let Ok(response) = client.get(&url).send().await {
            return response.status().is_success();
        }
    }
    
    false
}

/// Detect if ports are in use before starting
pub async fn are_ports_available(ports: &[u16]) -> Vec<(u16, bool)> {
    let mut results = Vec::new();
    
    for &port in ports {
        let available = !is_port_in_use(port).await;
        results.push((port, available));
    }
    
    results
}

async fn is_port_in_use(port: u16) -> bool {
    use tokio::net::TcpListener;
    
    match TcpListener::bind(format!("127.0.0.1:{}", port)).await {
        Ok(_) => false, // Port is available
        Err(_) => true, // Port is in use
    }
}