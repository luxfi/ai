use serde::Serialize;
use sysinfo::System;
use wgpu::DeviceType;

#[derive(Serialize, Debug)]
pub struct CpuInfo {
    pub name: String,
    pub cores: usize,
    pub threads: usize,
    pub frequency: u64,  // MHz
    pub architecture: String,
}

#[derive(Serialize, Debug)]
pub struct GpuInfo {
    pub name: String,
    pub vram: u64,  // GB
    pub acceleration: String,  // Metal, CUDA, Vulkan, etc.
    pub discrete: bool,
}

#[derive(Serialize, Debug)]
pub struct MemoryInfo {
    pub total: u64,  // GB
    pub available: u64,  // GB
}

#[derive(Serialize, Debug)]
pub struct SystemResources {
    pub cpu: CpuInfo,
    pub gpu: GpuInfo,
    pub memory: MemoryInfo,
}

#[derive(Serialize, Debug)]
pub struct Hardware {
    pub cpus: usize,
    pub memory: u64,
    pub discrete_gpu: bool,
}

#[derive(Serialize, Debug)]
pub struct Requirement {
    cpus: usize,
    memory: u64,
    discrete_gpu: bool,
}

#[derive(Serialize, Debug)]
pub struct Requirements {
    still_usable: Requirement,
    minimum: Requirement,
    recommended: Requirement,
    optimal: Requirement,
}

#[derive(Serialize, Debug)]
pub enum RequirementsStatus {
    Unmeet,
    StillUsable,
    Minimum,
    Recommended,
    Optimal,
}

#[derive(Serialize, Debug)]
pub struct HardwareSummary {
    pub hardware: Hardware,
    pub requirements: Requirements,
    pub requirements_status: RequirementsStatus,
    pub system_resources: SystemResources,
}

pub const STILL_USABLE_CPUS: usize = 4;
pub const STILL_USABLE_MEMORY: u64 = 8;
pub const MIN_CPUS: usize = 4;
pub const MIN_MEMORY: u64 = 16;
pub const RECOMMENDED_CPUS: usize = 10;
pub const RECOMMENDED_MEMORY: u64 = 32;
pub const REQUIREMENTS: Requirements = Requirements {
    still_usable: Requirement {
        cpus: STILL_USABLE_CPUS,
        memory: STILL_USABLE_MEMORY,
        discrete_gpu: false,
    },
    minimum: Requirement {
        cpus: MIN_CPUS,
        memory: MIN_MEMORY,
        discrete_gpu: false,
    },
    recommended: Requirement {
        cpus: RECOMMENDED_CPUS,
        memory: RECOMMENDED_MEMORY,
        discrete_gpu: false,
    },
    optimal: Requirement {
        cpus: RECOMMENDED_CPUS,
        memory: RECOMMENDED_MEMORY,
        discrete_gpu: true,
    },
};

fn get_total_vram(adapter: &wgpu::Adapter) -> u64 {
    if adapter.get_info().device_type != DeviceType::DiscreteGpu {
        let mut sys = System::new_all();
        sys.refresh_all();
        return sys.total_memory() / 1024 / 1024 / 1024;
    }

    let features = adapter.features();
    if features.contains(wgpu::Features::TEXTURE_ADAPTER_SPECIFIC_FORMAT_FEATURES) {
        let limits = adapter.limits();
        limits.max_texture_dimension_1d as u64 * limits.max_texture_dimension_2d as u64 * 4
    } else {
        0
    }
}

fn get_gpu_acceleration(adapter: &wgpu::Adapter) -> String {
    let info = adapter.get_info();

    // Determine acceleration based on backend and OS
    #[cfg(target_os = "macos")]
    {
        if info.backend == wgpu::Backend::Metal {
            return "Metal".to_string();
        }
    }

    #[cfg(target_os = "windows")]
    {
        if info.backend == wgpu::Backend::Dx12 {
            return "DirectX 12".to_string();
        } else if info.backend == wgpu::Backend::Vulkan {
            return "Vulkan".to_string();
        }
    }

    #[cfg(target_os = "linux")]
    {
        if info.backend == wgpu::Backend::Vulkan {
            return "Vulkan".to_string();
        }
    }

    // Fallback for WebGPU or other backends
    match info.backend {
        wgpu::Backend::Vulkan => "Vulkan".to_string(),
        wgpu::Backend::Metal => "Metal".to_string(),
        wgpu::Backend::Dx12 => "DirectX 12".to_string(),
        wgpu::Backend::Gl => "OpenGL".to_string(),
        wgpu::Backend::BrowserWebGpu => "WebGPU".to_string(),
        wgpu::Backend::Empty => "Software".to_string(),
    }
}

fn get_cpu_architecture() -> String {
    #[cfg(target_arch = "x86_64")]
    return "x86_64".to_string();

    #[cfg(target_arch = "aarch64")]
    return "ARM64".to_string();

    #[cfg(target_arch = "x86")]
    return "x86".to_string();

    #[cfg(target_arch = "arm")]
    return "ARM".to_string();

    #[cfg(not(any(target_arch = "x86_64", target_arch = "aarch64", target_arch = "x86", target_arch = "arm")))]
    return "Unknown".to_string();
}

pub fn hardware_get_summary() -> HardwareSummary {
    let instance = wgpu::Instance::default();
    let adapters = instance.enumerate_adapters(wgpu::Backends::all());
    let adapter = adapters
        .iter()
        .find(|adapter| adapter.get_info().device_type == wgpu::DeviceType::DiscreteGpu)
        .unwrap_or_else(|| adapters.first().expect("no gpu adapter found"));

    let discrete_gpu = adapter.get_info().device_type == DeviceType::DiscreteGpu;
    let is_macos = cfg!(target_os = "macos");

    let mut sys = System::new_all();
    sys.refresh_all();
    sys.refresh_cpu();
    sys.refresh_memory();

    // Get GPU info
    let gpu_info = adapter.get_info();
    let gpu_name = gpu_info.name.clone();
    let gpu_acceleration = get_gpu_acceleration(adapter);

    // Memory in GB
    let vram = get_total_vram(adapter);
    let memory = vram;  // For legacy compatibility
    let cpus = sys.cpus().len();

    // Get CPU info
    let cpu_name = if !sys.cpus().is_empty() {
        sys.cpus()[0].brand().to_string()
    } else {
        "Unknown CPU".to_string()
    };

    // Calculate average frequency in MHz
    let cpu_frequency = if !sys.cpus().is_empty() {
        sys.cpus()[0].frequency()
    } else {
        0
    };

    // Get physical core count (logical cores / threads per core)
    let physical_cores = sys.physical_core_count().unwrap_or(cpus);
    let threads = cpus;

    // Get memory info
    let total_memory = sys.total_memory() / 1024 / 1024 / 1024; // Convert to GB
    let available_memory = sys.available_memory() / 1024 / 1024 / 1024; // Convert to GB

    let requirement_status;
    if is_macos || (cpus >= RECOMMENDED_CPUS && memory >= RECOMMENDED_MEMORY && discrete_gpu) {
        requirement_status = RequirementsStatus::Optimal;
    } else if cpus >= RECOMMENDED_CPUS && memory >= RECOMMENDED_MEMORY {
        requirement_status = RequirementsStatus::Recommended;
    } else if cpus >= MIN_CPUS && memory >= MIN_MEMORY {
        requirement_status = RequirementsStatus::Minimum;
    } else if cpus >= STILL_USABLE_CPUS && memory >= STILL_USABLE_MEMORY {
        requirement_status = RequirementsStatus::StillUsable;
    } else {
        requirement_status = RequirementsStatus::Unmeet;
    }

    HardwareSummary {
        hardware: Hardware {
            cpus,
            memory,
            discrete_gpu,
        },
        requirements: REQUIREMENTS,
        requirements_status: requirement_status,
        system_resources: SystemResources {
            cpu: CpuInfo {
                name: cpu_name,
                cores: physical_cores,
                threads,
                frequency: cpu_frequency,
                architecture: get_cpu_architecture(),
            },
            gpu: GpuInfo {
                name: gpu_name,
                vram,
                acceleration: gpu_acceleration,
                discrete: discrete_gpu,
            },
            memory: MemoryInfo {
                total: total_memory,
                available: available_memory,
            },
        },
    }
}
