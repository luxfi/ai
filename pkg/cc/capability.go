// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cc

import (
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// CommandRunner abstracts command execution for testability
type CommandRunner interface {
	Run(cmd string, args ...string) ([]byte, error)
}

// FileReader abstracts file system access for testability
type FileReader interface {
	ReadFile(path string) ([]byte, error)
	Stat(path string) (os.FileInfo, error)
}

// DefaultCommandRunner uses exec.Command
type DefaultCommandRunner struct{}

func (r *DefaultCommandRunner) Run(cmd string, args ...string) ([]byte, error) {
	return exec.Command(cmd, args...).Output()
}

// DefaultFileReader uses os package functions
type DefaultFileReader struct{}

func (r *DefaultFileReader) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (r *DefaultFileReader) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// Package-level defaults for production use
var (
	defaultCommandRunner CommandRunner = &DefaultCommandRunner{}
	defaultFileReader    FileReader    = &DefaultFileReader{}
)

// GPUVendor represents a GPU hardware vendor
type GPUVendor string

const (
	VendorNVIDIA   GPUVendor = "NVIDIA"
	VendorAMD      GPUVendor = "AMD"
	VendorIntel    GPUVendor = "Intel"
	VendorApple    GPUVendor = "Apple"
	VendorQualcomm GPUVendor = "Qualcomm"
	VendorUnknown  GPUVendor = "Unknown"
)

// CPUTEEType represents CPU TEE technology
type CPUTEEType string

const (
	TEESEVSNP    CPUTEEType = "SEV-SNP"
	TEETDX       CPUTEEType = "TDX"
	TEESGX       CPUTEEType = "SGX"
	TEECCA       CPUTEEType = "CCA"
	TEETrustZone CPUTEEType = "TrustZone"
	TEESecureEnclave CPUTEEType = "SecureEnclave"
	TEENone      CPUTEEType = "None"
)

// HardwareCapability represents detected hardware CC capabilities
type HardwareCapability struct {
	// GPU capabilities
	GPUVendor      GPUVendor `json:"gpu_vendor"`
	GPUModel       string    `json:"gpu_model"`
	GPUSerial      string    `json:"gpu_serial"`
	GPUMemoryMB    uint64    `json:"gpu_memory_mb"`
	GPUDriverVer   string    `json:"gpu_driver_version"`
	ComputeCap     string    `json:"compute_capability"` // e.g., "9.0" for Blackwell

	// GPU CC capabilities
	GPUCCSupported bool `json:"gpu_cc_supported"` // Hardware supports CC
	GPUCCEnabled   bool `json:"gpu_cc_enabled"`   // CC currently enabled
	NVTrustAvail   bool `json:"nvtrust_available"` // nvtrust local verifier available
	TEEIOSupported bool `json:"tee_io_supported"` // TEE-IO for Blackwell
	MIGSupported   bool `json:"mig_supported"`    // Multi-Instance GPU

	// CPU TEE capabilities
	CPUVendor    string     `json:"cpu_vendor"`
	CPUModel     string     `json:"cpu_model"`
	CPUTEEType   CPUTEEType `json:"cpu_tee_type"`
	CPUTEEActive bool       `json:"cpu_tee_active"` // Currently running in TEE

	// Device TEE capabilities (mobile/edge)
	DeviceTEEType    string `json:"device_tee_type,omitempty"`
	DeviceTEEEnabled bool   `json:"device_tee_enabled,omitempty"`
	NPUModel         string `json:"npu_model,omitempty"` // Neural Processing Unit

	// Maximum achievable tier based on capabilities
	MaxTier CCTier `json:"max_tier"`
}

// DetectCapabilities detects hardware CC capabilities on the current system
func DetectCapabilities() (*HardwareCapability, error) {
	cap := &HardwareCapability{
		GPUVendor:  VendorUnknown,
		CPUTEEType: TEENone,
		MaxTier:    Tier4Standard,
	}

	// Detect GPU capabilities
	detectGPUCapabilities(cap)

	// Detect CPU TEE capabilities
	detectCPUTEECapabilities(cap)

	// Detect device TEE capabilities (mobile/edge)
	detectDeviceTEECapabilities(cap)

	// Calculate maximum achievable tier
	cap.MaxTier = calculateMaxTier(cap)

	return cap, nil
}

// detectGPUCapabilities detects GPU vendor and CC capabilities
func detectGPUCapabilities(cap *HardwareCapability) {
	// Try NVIDIA first (most common for AI)
	if detectNVIDIACapabilities(cap) {
		return
	}

	// Try AMD
	if detectAMDCapabilities(cap) {
		return
	}

	// Try Intel
	if detectIntelCapabilities(cap) {
		return
	}

	// On macOS, detect Apple Silicon
	if runtime.GOOS == "darwin" {
		detectAppleSiliconCapabilities(cap)
	}
}

// detectNVIDIACapabilities detects NVIDIA GPU capabilities
func detectNVIDIACapabilities(cap *HardwareCapability) bool {
	return detectNVIDIACapabilitiesWithDeps(cap, defaultCommandRunner, defaultFileReader)
}

// detectNVIDIACapabilitiesWithDeps is the testable version with injected dependencies
func detectNVIDIACapabilitiesWithDeps(cap *HardwareCapability, cmdRunner CommandRunner, fileReader FileReader) bool {
	// Try nvidia-smi
	output, err := cmdRunner.Run("nvidia-smi", "--query-gpu=name,memory.total,driver_version,serial", "--format=csv,noheader,nounits")
	if err != nil {
		return false
	}

	cap.GPUVendor = VendorNVIDIA

	// Parse output: "Model, Memory, Driver, Serial"
	parts := strings.Split(strings.TrimSpace(string(output)), ", ")
	if len(parts) >= 4 {
		cap.GPUModel = strings.TrimSpace(parts[0])
		if mem, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64); err == nil {
			cap.GPUMemoryMB = mem
		}
		cap.GPUDriverVer = strings.TrimSpace(parts[2])
		cap.GPUSerial = strings.TrimSpace(parts[3])
	}

	// Detect CC capabilities based on GPU model
	detectNVIDIACCCapabilitiesByModel(cap)

	// Check if nvtrust is available for local verification
	if cap.GPUCCSupported {
		cap.NVTrustAvail = checkNVTrustAvailableWithDeps(fileReader)
	}

	// Check if CC mode is currently enabled (requires nvidia-smi query)
	if cap.GPUCCSupported {
		cap.GPUCCEnabled = checkNVIDIACCEnabledWithDeps(cmdRunner)
	}

	return true
}

// detectNVIDIACCCapabilitiesByModel sets CC capabilities based on GPU model string
func detectNVIDIACCCapabilitiesByModel(cap *HardwareCapability) {
	model := cap.GPUModel
	switch {
	// Blackwell datacenter - highest CC tier (9.0)
	case strings.Contains(model, "B100") || strings.Contains(model, "B200") || strings.Contains(model, "GB200"):
		cap.ComputeCap = "9.0"
		cap.GPUCCSupported = true
		cap.TEEIOSupported = true
		cap.MIGSupported = true

	// Hopper datacenter - full CC support (9.0)
	case strings.Contains(model, "H100") || strings.Contains(model, "H200"):
		cap.ComputeCap = "9.0"
		cap.GPUCCSupported = true
		cap.TEEIOSupported = false // TEE-IO is Blackwell only
		cap.MIGSupported = true

	// Ada professional - CC support (8.9)
	case strings.Contains(model, "RTX 6000") && strings.Contains(model, "Ada"):
		cap.ComputeCap = "8.9"
		cap.GPUCCSupported = true
		cap.TEEIOSupported = false
		cap.MIGSupported = false

	// RTX PRO 6000 Blackwell - CC support (9.0)
	case strings.Contains(model, "RTX PRO 6000"):
		cap.ComputeCap = "9.0"
		cap.GPUCCSupported = true
		cap.TEEIOSupported = true
		cap.MIGSupported = false

	// Grace Hopper Superchip - full CC (9.0)
	case strings.Contains(model, "Grace"):
		cap.ComputeCap = "9.0"
		cap.GPUCCSupported = true
		cap.TEEIOSupported = false
		cap.MIGSupported = true

	// Consumer Blackwell - NO CC support (confirmed by NVIDIA forums)
	case strings.Contains(model, "5090") || strings.Contains(model, "5080"):
		cap.ComputeCap = "9.0"
		cap.GPUCCSupported = false // Explicitly disabled

	// DGX Spark (GB10) - NO CC support (confirmed by NVIDIA forums)
	case strings.Contains(model, "GB10"):
		cap.ComputeCap = "9.0"
		cap.GPUCCSupported = false // Explicitly disabled

	// Consumer Ada - no CC support (8.9)
	case strings.Contains(model, "4090") || strings.Contains(model, "4080"):
		cap.ComputeCap = "8.9"
		cap.GPUCCSupported = false
	}
}

// checkNVTrustAvailable checks if nvtrust local verifier tools are available
func checkNVTrustAvailable() bool {
	return checkNVTrustAvailableWithDeps(defaultFileReader)
}

// checkNVTrustAvailableWithDeps is the testable version
func checkNVTrustAvailableWithDeps(fileReader FileReader) bool {
	// Check for nvtrust guest tools
	// See: https://github.com/NVIDIA/nvtrust
	paths := []string{
		"/usr/local/bin/nv-attestation-tool",
		"/opt/nvidia/nvtrust/bin/nv-attestation-tool",
		"/usr/bin/nv-attestation-tool",
	}
	for _, path := range paths {
		if _, err := fileReader.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// checkNVIDIACCEnabled checks if NVIDIA CC mode is currently enabled
func checkNVIDIACCEnabled() bool {
	return checkNVIDIACCEnabledWithDeps(defaultCommandRunner)
}

// checkNVIDIACCEnabledWithDeps is the testable version
func checkNVIDIACCEnabledWithDeps(cmdRunner CommandRunner) bool {
	// Query nvidia-smi for CC mode status
	output, err := cmdRunner.Run("nvidia-smi", "--query-gpu=conf-compute.mode", "--format=csv,noheader")
	if err != nil {
		return false
	}
	mode := strings.ToLower(strings.TrimSpace(string(output)))
	return mode == "on" || mode == "enabled" || mode == "1"
}

// detectAMDCapabilities detects AMD GPU capabilities
func detectAMDCapabilities(cap *HardwareCapability) bool {
	return detectAMDCapabilitiesWithDeps(cap, defaultCommandRunner)
}

// detectAMDCapabilitiesWithDeps is the testable version
func detectAMDCapabilitiesWithDeps(cap *HardwareCapability, cmdRunner CommandRunner) bool {
	// Try rocm-smi for AMD GPUs
	output, err := cmdRunner.Run("rocm-smi", "--showproductname", "--csv")
	if err != nil {
		return false
	}

	cap.GPUVendor = VendorAMD
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "MI300") || strings.Contains(line, "MI250") {
			cap.GPUModel = strings.TrimSpace(line)
			// AMD Instinct MI300X supports CC when paired with SEV-SNP
			// GPU itself doesn't have hardware CC, but VM+GPU combo works
			cap.GPUCCSupported = false // AMD GPUs don't have native GPU CC
			break
		}
	}

	return cap.GPUModel != ""
}

// detectIntelCapabilities detects Intel GPU capabilities
func detectIntelCapabilities(cap *HardwareCapability) bool {
	// Intel discrete GPUs (Arc, Data Center GPU Max)
	// Intel GPUs don't currently have hardware CC support
	// but can run in TDX confidential VMs
	return false
}

// detectAppleSiliconCapabilities detects Apple Silicon capabilities
func detectAppleSiliconCapabilities(cap *HardwareCapability) {
	detectAppleSiliconCapabilitiesWithDeps(cap, defaultCommandRunner)
}

// detectAppleSiliconCapabilitiesWithDeps is the testable version
func detectAppleSiliconCapabilitiesWithDeps(cap *HardwareCapability, cmdRunner CommandRunner) {
	// On macOS, check for Apple Silicon
	output, err := cmdRunner.Run("sysctl", "-n", "machdep.cpu.brand_string")
	if err != nil {
		return
	}

	brand := strings.TrimSpace(string(output))
	if strings.Contains(brand, "Apple") {
		cap.GPUVendor = VendorApple
		cap.GPUModel = brand

		// Detect specific chip for Neural Engine capabilities
		switch {
		case strings.Contains(brand, "M4"):
			cap.NPUModel = "Neural Engine 18-core"
			cap.ComputeCap = "apple-m4"
			cap.DeviceTEEType = "SecureEnclave"
			cap.DeviceTEEEnabled = true
		case strings.Contains(brand, "M3"):
			cap.NPUModel = "Neural Engine 16-core"
			cap.ComputeCap = "apple-m3"
			cap.DeviceTEEType = "SecureEnclave"
			cap.DeviceTEEEnabled = true
		case strings.Contains(brand, "M2"):
			cap.NPUModel = "Neural Engine 16-core"
			cap.ComputeCap = "apple-m2"
			cap.DeviceTEEType = "SecureEnclave"
			cap.DeviceTEEEnabled = true
		case strings.Contains(brand, "M1"):
			cap.NPUModel = "Neural Engine 16-core"
			cap.ComputeCap = "apple-m1"
			cap.DeviceTEEType = "SecureEnclave"
			cap.DeviceTEEEnabled = true
		}
	}
}

// detectCPUTEECapabilities detects CPU TEE capabilities
func detectCPUTEECapabilities(cap *HardwareCapability) {
	// Get CPU info
	switch runtime.GOOS {
	case "linux":
		detectLinuxCPUTEE(cap)
	case "darwin":
		// macOS - Secure Enclave is handled in Apple Silicon detection
		if cap.DeviceTEEType == "SecureEnclave" {
			cap.CPUTEEType = TEESecureEnclave
			cap.CPUTEEActive = true
		}
	}
}

// detectLinuxCPUTEE detects CPU TEE on Linux
func detectLinuxCPUTEE(cap *HardwareCapability) {
	detectLinuxCPUTEEWithDeps(cap, defaultFileReader)
}

// detectLinuxCPUTEEWithDeps is the testable version
func detectLinuxCPUTEEWithDeps(cap *HardwareCapability, fileReader FileReader) {
	// Read CPU info
	data, err := fileReader.ReadFile("/proc/cpuinfo")
	if err != nil {
		return
	}
	cpuinfo := string(data)

	// Parse vendor and model
	vendorRe := regexp.MustCompile(`vendor_id\s*:\s*(.+)`)
	modelRe := regexp.MustCompile(`model name\s*:\s*(.+)`)

	if match := vendorRe.FindStringSubmatch(cpuinfo); len(match) > 1 {
		cap.CPUVendor = strings.TrimSpace(match[1])
	}
	if match := modelRe.FindStringSubmatch(cpuinfo); len(match) > 1 {
		cap.CPUModel = strings.TrimSpace(match[1])
	}

	// Detect SEV-SNP (AMD)
	if strings.Contains(cap.CPUVendor, "AMD") {
		if _, err := fileReader.Stat("/dev/sev-guest"); err == nil {
			cap.CPUTEEType = TEESEVSNP
			// Check if we're running inside a SEV-SNP VM
			cap.CPUTEEActive = checkSEVSNPActiveWithDeps(fileReader)
		}
	}

	// Detect TDX (Intel)
	if strings.Contains(cap.CPUVendor, "Intel") {
		if _, err := fileReader.Stat("/dev/tdx-guest"); err == nil {
			cap.CPUTEEType = TEETDX
			cap.CPUTEEActive = checkTDXActiveWithDeps(fileReader)
		} else if _, err := fileReader.Stat("/dev/sgx_enclave"); err == nil {
			cap.CPUTEEType = TEESGX
			cap.CPUTEEActive = true
		}
	}

	// Detect ARM CCA
	if strings.Contains(strings.ToLower(cpuinfo), "aarch64") || strings.Contains(strings.ToLower(cpuinfo), "arm") {
		if _, err := fileReader.Stat("/sys/devices/platform/arm-cca"); err == nil {
			cap.CPUTEEType = TEECCA
			cap.CPUTEEActive = true
		}
	}
}

// checkSEVSNPActive checks if running inside a SEV-SNP VM
func checkSEVSNPActive() bool {
	return checkSEVSNPActiveWithDeps(defaultFileReader)
}

// checkSEVSNPActiveWithDeps is the testable version
func checkSEVSNPActiveWithDeps(fileReader FileReader) bool {
	// Read SEV-SNP status from sysfs
	data, err := fileReader.ReadFile("/sys/kernel/security/coco/sev-snp/")
	if err == nil && len(data) > 0 {
		return true
	}
	// Alternative: try to generate attestation report
	if _, err := fileReader.Stat("/dev/sev-guest"); err == nil {
		return true
	}
	return false
}

// checkTDXActive checks if running inside a TDX VM
func checkTDXActive() bool {
	return checkTDXActiveWithDeps(defaultFileReader)
}

// checkTDXActiveWithDeps is the testable version
func checkTDXActiveWithDeps(fileReader FileReader) bool {
	// Check for TDX guest device
	if _, err := fileReader.Stat("/dev/tdx-guest"); err == nil {
		return true
	}
	// Check sysfs for TDX status
	data, err := fileReader.ReadFile("/sys/kernel/security/coco/tdx/")
	return err == nil && len(data) > 0
}

// detectDeviceTEECapabilities detects mobile/edge device TEE capabilities
func detectDeviceTEECapabilities(cap *HardwareCapability) {
	// Apple Secure Enclave is already handled
	if cap.DeviceTEEType == "SecureEnclave" {
		return
	}

	// On Android/Qualcomm devices, check for TrustZone
	// This would need Android-specific detection
	// For now, we only support detection on Linux/macOS
}

// calculateMaxTier determines the maximum achievable CC tier
func calculateMaxTier(cap *HardwareCapability) CCTier {
	// Tier 1: GPU-native CC (NVIDIA with NVTrust)
	if cap.GPUCCSupported && cap.GPUCCEnabled && cap.NVTrustAvail {
		return Tier1GPUNativeCC
	}

	// Tier 2: Confidential VM + GPU
	if cap.CPUTEEActive && (cap.CPUTEEType == TEESEVSNP || cap.CPUTEEType == TEETDX || cap.CPUTEEType == TEECCA) {
		return Tier2ConfidentialVM
	}

	// Tier 3: Device TEE + AI engine
	if cap.DeviceTEEEnabled && (cap.DeviceTEEType == "SecureEnclave" || cap.DeviceTEEType == "TrustZone") {
		return Tier3DeviceTEE
	}

	// Tier 4: Standard (no CC)
	return Tier4Standard
}

// CanAchieveTier checks if the hardware can achieve a specific tier
func (c *HardwareCapability) CanAchieveTier(tier CCTier) bool {
	return c.MaxTier <= tier // Lower tier number = higher capability
}

// GetSupportedTiers returns all tiers this hardware can support
func (c *HardwareCapability) GetSupportedTiers() []CCTier {
	tiers := []CCTier{}
	// Add all tiers at or below the max achievable tier
	for t := c.MaxTier; t <= Tier4Standard; t++ {
		tiers = append(tiers, t)
	}
	return tiers
}

// IsGPUCCCapable returns true if the GPU supports hardware CC
func (c *HardwareCapability) IsGPUCCCapable() bool {
	return c.GPUCCSupported
}

// IsCPUTEECapable returns true if the CPU supports TEE
func (c *HardwareCapability) IsCPUTEECapable() bool {
	return c.CPUTEEType != TEENone
}

// IsDeviceTEECapable returns true if the device has TEE support
func (c *HardwareCapability) IsDeviceTEECapable() bool {
	return c.DeviceTEEEnabled
}

// RequiresSetup returns true if additional setup is needed to enable CC
func (c *HardwareCapability) RequiresSetup() (bool, string) {
	if c.GPUCCSupported && !c.GPUCCEnabled {
		return true, "GPU CC mode needs to be enabled. Run: nvidia-smi -i 0 -cc 1"
	}
	if c.GPUCCSupported && !c.NVTrustAvail {
		return true, "nvtrust tools not found. Install from: https://github.com/NVIDIA/nvtrust"
	}
	return false, ""
}
