
use std::sync::Arc;

use crate::local_zoo_node::zoo_node_manager::ZooNodeManager;
use once_cell::sync::OnceCell;

pub static ZOO_NODE_MANAGER_INSTANCE: OnceCell<Arc<tokio::sync::RwLock<ZooNodeManager>>> =
    OnceCell::new();
