package gpu

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Vendor string

const (
	VendorIntel   Vendor = "intel"
	VendorAMD     Vendor = "amd"
	VendorNVIDIA  Vendor = "nvidia"
	VendorMali    Vendor = "mali"
	VendorAdreno  Vendor = "adreno"
	VendorUnknown Vendor = "unknown"
)

type GPUType string

const (
	TypeIGPU     GPUType = "igpu"
	TypeDiscrete GPUType = "discrete"
)

type GPUInfo struct {
	Vendor    Vendor
	Type      GPUType
	Devices   []string
	Driver    string
	IsPrimary bool
	Index     int
}

type Detector struct {
	drmPath string
	pciPath string
}

func NewDetector() *Detector {
	return &Detector{
		drmPath: "/sys/class/drm",
		pciPath: "/sys/bus/pci/devices",
	}
}

func (d *Detector) DetectGPUs() ([]GPUInfo, error) {
	var gpus []GPUInfo

	drmGPUs, err := d.detectDRMGPUs()
	if err != nil {
		return nil, fmt.Errorf("failed to detect DRM GPUs: %w", err)
	}
	gpus = append(gpus, drmGPUs...)

	nvidiaGPUs, err := d.detectNVIDIAGPUs()
	if err != nil {
		return nil, fmt.Errorf("failed to detect NVIDIA GPUs: %w", err)
	}
	
	// Deduplicate: only add NVIDIA GPUs not already detected via DRM
	for _, nvGPU := range nvidiaGPUs {
		if !d.isNVIDIAGPUDetected(gpus, nvGPU) {
			gpus = append(gpus, nvGPU)
		}
	}

	armGPUs, err := d.detectARMGPUs()
	if err != nil {
		return nil, fmt.Errorf("failed to detect ARM GPUs: %w", err)
	}
	gpus = append(gpus, armGPUs...)

	// Use stable sort to preserve order for equal elements
	sort.SliceStable(gpus, func(i, j int) bool {
		if gpus[i].Type == TypeDiscrete && gpus[j].Type != TypeDiscrete {
			return true
		}
		if gpus[i].Type != TypeDiscrete && gpus[j].Type == TypeDiscrete {
			return false
		}
		return gpus[i].Index < gpus[j].Index
	})

	if len(gpus) > 0 {
		gpus[0].IsPrimary = true
	}

	return gpus, nil
}

// isNVIDIAGPUDetected checks if an NVIDIA GPU was already detected via DRM
func (d *Detector) isNVIDIAGPUDetected(existing []GPUInfo, nvGPU GPUInfo) bool {
	for _, gpu := range existing {
		if gpu.Vendor == VendorNVIDIA {
			// Check if any device nodes overlap
			for _, existingDev := range gpu.Devices {
				for _, nvDev := range nvGPU.Devices {
					if existingDev == nvDev {
						return true
					}
				}
			}
		}
	}
	return false
}

func (d *Detector) detectDRMGPUs() ([]GPUInfo, error) {
	var gpus []GPUInfo

	entries, err := os.ReadDir(d.drmPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	cardIndex := 0
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "card") || strings.Contains(name, "-") {
			continue
		}

		cardPath := filepath.Join(d.drmPath, name)
		devicePath := filepath.Join(cardPath, "device")

		vendor, driver, err := d.identifyDRMDevice(devicePath)
		if err != nil {
			continue
		}

		devices := []string{
			fmt.Sprintf("/dev/dri/%s", name),
		}

		// Read render node from sysfs instead of calculating from cardIndex
		// This handles cases where some cardN entries are not recognized GPUs
		renderNode := d.findRenderNode(cardPath)
		if renderNode != "" {
			devices = append(devices, renderNode)
		}

		// For AMD GPUs, also include /dev/kfd for ROCm support
		if vendor == VendorAMD {
			if _, err := os.Stat("/dev/kfd"); err == nil {
				devices = append(devices, "/dev/kfd")
			}
		}

		gpuType := TypeIGPU
		if d.isDiscreteGPU(devicePath) {
			gpuType = TypeDiscrete
		}

		gpus = append(gpus, GPUInfo{
			Vendor:  vendor,
			Type:    gpuType,
			Devices: devices,
			Driver:  driver,
			Index:   cardIndex,
		})

		cardIndex++
	}

	return gpus, nil
}

// findRenderNode reads the render node for a card from sysfs
func (d *Detector) findRenderNode(cardPath string) string {
	// Look for renderD* in the card's device/drm directory
	drmPath := filepath.Join(cardPath, "device", "drm")
	entries, err := os.ReadDir(drmPath)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "renderD") {
			return fmt.Sprintf("/dev/dri/%s", entry.Name())
		}
	}
	return ""
}

func (d *Detector) identifyDRMDevice(devicePath string) (Vendor, string, error) {
	vendorData, err := os.ReadFile(filepath.Join(devicePath, "vendor"))
	if err != nil {
		return VendorUnknown, "", err
	}

	vendorID := strings.TrimSpace(string(vendorData))
	vendorID = strings.TrimPrefix(vendorID, "0x")

	switch vendorID {
	case "8086":
		driver := d.detectIntelDriver(devicePath)
		return VendorIntel, driver, nil
	case "1002":
		return VendorAMD, "amdgpu", nil
	case "10de":
		return VendorNVIDIA, "nvidia", nil
	default:
		return VendorUnknown, "", fmt.Errorf("unknown vendor: %s", vendorID)
	}
}

func (d *Detector) detectIntelDriver(devicePath string) string {
	driverLink := filepath.Join(devicePath, "driver")
	driver, err := os.Readlink(driverLink)
	if err != nil {
		return "i915"
	}

	driverName := filepath.Base(driver)
	if driverName == "xe" {
		return "xe"
	}
	return "i915"
}

func (d *Detector) isDiscreteGPU(devicePath string) bool {
	pciClass, err := os.ReadFile(filepath.Join(devicePath, "class"))
	if err != nil {
		return false
	}

	classCode := strings.TrimSpace(string(pciClass))
	classCode = strings.TrimPrefix(classCode, "0x")

	if len(classCode) >= 4 && classCode[:4] == "0300" {
		resourcePath := filepath.Join(devicePath, "resource")
		resources, err := os.ReadFile(resourcePath)
		if err != nil {
			return false
		}

		// PCI resource file format: start end flags (one per line)
		// Size = end - start + 1
		lines := strings.Split(string(resources), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				start, errStart := strconv.ParseInt(fields[0], 0, 64)
				end, errEnd := strconv.ParseInt(fields[1], 0, 64)
				if errStart == nil && errEnd == nil && end > start {
					size := end - start + 1
					if size > 256*1024*1024 {
						return true
					}
				}
			}
		}
	}

	return false
}

func (d *Detector) detectNVIDIAGPUs() ([]GPUInfo, error) {
	var gpus []GPUInfo

	nvidiaDevices, err := filepath.Glob("/dev/nvidia[0-9]*")
	if err != nil {
		return nil, err
	}

	for i, device := range nvidiaDevices {
		devices := []string{device}

		if _, err := os.Stat("/dev/nvidiactl"); err == nil {
			devices = append(devices, "/dev/nvidiactl")
		}
		if _, err := os.Stat("/dev/nvidia-uvm"); err == nil {
			devices = append(devices, "/dev/nvidia-uvm")
		}
		if _, err := os.Stat("/dev/nvidia-modeset"); err == nil {
			devices = append(devices, "/dev/nvidia-modeset")
		}

		gpus = append(gpus, GPUInfo{
			Vendor:  VendorNVIDIA,
			Type:    TypeDiscrete,
			Devices: devices,
			Driver:  "nvidia",
			Index:   i,
		})
	}

	return gpus, nil
}

func (d *Detector) detectARMGPUs() ([]GPUInfo, error) {
	var gpus []GPUInfo

	if _, err := os.Stat("/dev/mali0"); err == nil {
		gpus = append(gpus, GPUInfo{
			Vendor:  VendorMali,
			Type:    TypeIGPU,
			Devices: []string{"/dev/mali0"},
			Driver:  "panfrost",
			Index:   0,
		})
	}

	if _, err := os.Stat("/dev/kgsl-3d0"); err == nil {
		gpus = append(gpus, GPUInfo{
			Vendor:  VendorAdreno,
			Type:    TypeIGPU,
			Devices: []string{"/dev/kgsl-3d0"},
			Driver:  "msm",
			Index:   0,
		})
	}

	return gpus, nil
}

func SelectGPU(preferred string, gpus []GPUInfo) (*GPUInfo, error) {
	if len(gpus) == 0 {
		return nil, fmt.Errorf("no GPUs detected")
	}

	if preferred == "auto" || preferred == "" {
		return &gpus[0], nil
	}

	preferredVendor := Vendor(preferred)
	for i := range gpus {
		if gpus[i].Vendor == preferredVendor {
			return &gpus[i], nil
		}
	}

	return nil, fmt.Errorf("GPU vendor %q not found", preferred)
}

func GetDeviceNodes(gpu *GPUInfo) []string {
	return gpu.Devices
}

func GetDeviceAllowRules(gpu *GPUInfo) []string {
	var rules []string

	for _, device := range gpu.Devices {
		rules = append(rules, fmt.Sprintf("%s rw", device))
	}

	return rules
}

func GetRequiredCapabilities(gpu *GPUInfo) []string {
	var caps []string

	switch gpu.Vendor {
	case VendorNVIDIA:
		caps = append(caps, "CAP_SYS_ADMIN")
	case VendorIntel, VendorAMD:
		// Usually don't need extra capabilities
	}

	return caps
}
