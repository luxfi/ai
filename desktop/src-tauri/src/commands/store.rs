use serde_json::Value;
use std::fs;

/// Store backend that serves responses from local JSON file
/// This can be updated from upstream store API when available
#[tauri::command]
pub async fn get_store_response(endpoint: String) -> Result<String, String> {
    // Read the store responses JSON file
    let store_file_path = concat!(
        env!("CARGO_MANIFEST_DIR"),
        "/../src/lib/store-responses.json"
    );
    
    let store_data = fs::read_to_string(store_file_path)
        .map_err(|e| format!("Failed to read store data: {}", e))?;
    
    let store_json: Value = serde_json::from_str(&store_data)
        .map_err(|e| format!("Failed to parse store data: {}", e))?;
    
    // Route to appropriate store response based on endpoint
    let response = match endpoint.as_str() {
        "agents" => store_json.get("agents")
            .ok_or("Agents data not found")?,
        "tools" => store_json.get("tools")
            .ok_or("Tools data not found")?,
        "categories" => store_json.get("categories")
            .ok_or("Categories data not found")?,
        "featured" => store_json.get("featured_collections")
            .ok_or("Featured collections not found")?,
        "purchases" => store_json.get("user_purchases")
            .ok_or("User purchases not found")?,
        endpoint if endpoint.starts_with("tool/") => {
            let tool_key = endpoint.strip_prefix("tool/").unwrap();
            store_json.get("tool_details")
                .and_then(|details| details.get(tool_key))
                .ok_or(format!("Tool details not found for: {}", tool_key))?
        },
        _ => return Err(format!("Unknown endpoint: {}", endpoint))
    };
    
    Ok(response.to_string())
}

/// Intercept store API calls and return data from local JSON
/// This maintains the API structure while serving from local store
#[tauri::command]
pub async fn store_api_proxy(url: String) -> Result<Value, String> {
    // Parse the URL to determine what's being requested
    let store_response = if url.contains("/store/products") && url.contains("type=agent") {
        get_store_response("agents".to_string()).await?
    } else if url.contains("/store/products") && url.contains("type=tool") {
        get_store_response("tools".to_string()).await?
    } else if url.contains("/store/categories") {
        get_store_response("categories".to_string()).await?
    } else if url.contains("/store/featured") {
        get_store_response("featured".to_string()).await?
    } else if url.contains("/store/purchases") {
        get_store_response("purchases".to_string()).await?
    } else if url.contains("/v2/tool_store_proxy/") {
        // Extract tool router key from URL
        let parts: Vec<&str> = url.split('/').collect();
        if let Some(tool_key) = parts.last() {
            get_store_response(format!("tool/{}", tool_key)).await?
        } else {
            return Err("Invalid tool URL".to_string());
        }
    } else {
        return Err(format!("Unhandled store URL: {}", url));
    };
    
    // Parse and return as JSON Value
    serde_json::from_str(&store_response)
        .map_err(|e| format!("Failed to parse response: {}", e))
}