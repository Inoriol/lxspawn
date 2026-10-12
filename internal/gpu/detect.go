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
	gpus = append(gpus, nvidiaGPUs...)

	armGPUs, err := d.detectARMGPUs()
	if err != nil {
		return nil, fmt.Errorf("failed to detect ARM GPUs: %w", err)
	}
	gpus = append(gpus, armGPUs...)

	sort.Slice(gpus, func(i, j int) bool {
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

		renderNode := fmt.Sprintf("/dev/dri/renderD%d", 128+cardIndex)
		if _, err := os.Stat(renderNode); err == nil {
			devices = append(devices, renderNode)
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

		lines := strings.Split(string(resources), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				size, err := strconv.ParseInt(fields[2], 0, 64)
				if err == nil && size > 256*1024*1024 {
					return true
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

	switch gpu.Vendor {
	case VendorAMD:
		if _, err := os.Stat("/dev/kfd"); err == nil {
			rules = append(rules, "/dev/kfd rw")
		}
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
