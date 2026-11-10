package gpu

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectGPUs(t *testing.T) {
	detector := NewDetector()
	gpus, err := detector.DetectGPUs()
	
	if err != nil {
		t.Fatalf("DetectGPUs failed: %v", err)
	}
	
	t.Logf("Detected %d GPUs", len(gpus))
	for i, gpu := range gpus {
		t.Logf("GPU %d: Vendor=%s Type=%s Driver=%s Devices=%v Primary=%v",
			i, gpu.Vendor, gpu.Type, gpu.Driver, gpu.Devices, gpu.IsPrimary)
	}
}

func TestSelectGPU(t *testing.T) {
	tests := []struct {
		name      string
		preferred string
		gpus      []GPUInfo
		wantErr   bool
		wantIdx   int
	}{
		{
			name:      "auto selects first (discrete)",
			preferred: "auto",
			gpus: []GPUInfo{
				{Vendor: VendorNVIDIA, Type: TypeDiscrete, Index: 0},
				{Vendor: VendorIntel, Type: TypeIGPU, Index: 1},
			},
			wantErr: false,
			wantIdx: 0,
		},
		{
			name:      "auto selects iGPU if only iGPU",
			preferred: "auto",
			gpus: []GPUInfo{
				{Vendor: VendorIntel, Type: TypeIGPU, Index: 0},
			},
			wantErr: false,
			wantIdx: 0,
		},
		{
			name:      "explicit vendor selection",
			preferred: "intel",
			gpus: []GPUInfo{
				{Vendor: VendorNVIDIA, Type: TypeDiscrete, Index: 0},
				{Vendor: VendorIntel, Type: TypeIGPU, Index: 1},
			},
			wantErr: false,
			wantIdx: 1,
		},
		{
			name:      "vendor not found",
			preferred: "amd",
			gpus: []GPUInfo{
				{Vendor: VendorIntel, Type: TypeIGPU, Index: 0},
			},
			wantErr: true,
		},
		{
			name:      "no GPUs",
			preferred: "auto",
			gpus:      []GPUInfo{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SelectGPU(tt.preferred, tt.gpus)
			if (err != nil) != tt.wantErr {
				t.Errorf("SelectGPU() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Index != tt.wantIdx {
				t.Errorf("SelectGPU() got index = %d, want %d", got.Index, tt.wantIdx)
			}
		})
	}
}

func TestGetDeviceAllowRules(t *testing.T) {
	gpu := &GPUInfo{
		Vendor:  VendorIntel,
		Type:    TypeIGPU,
		Devices: []string{"/dev/dri/card0", "/dev/dri/renderD128"},
	}

	rules := GetDeviceAllowRules(gpu)
	
	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}

	expected := []string{"/dev/dri/card0 rw", "/dev/dri/renderD128 rw"}
	for i, rule := range rules {
		if rule != expected[i] {
			t.Errorf("Rule %d: got %q, want %q", i, rule, expected[i])
		}
	}
}

func TestGetRequiredCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		vendor   Vendor
		wantCaps int
	}{
		{
			name:     "NVIDIA requires CAP_SYS_ADMIN",
			vendor:   VendorNVIDIA,
			wantCaps: 1,
		},
		{
			name:     "Intel requires no extra caps",
			vendor:   VendorIntel,
			wantCaps: 0,
		},
		{
			name:     "AMD requires no extra caps",
			vendor:   VendorAMD,
			wantCaps: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gpu := &GPUInfo{Vendor: tt.vendor}
			caps := GetRequiredCapabilities(gpu)
			if len(caps) != tt.wantCaps {
				t.Errorf("GetRequiredCapabilities() got %d caps, want %d", len(caps), tt.wantCaps)
			}
		})
	}
}

func TestDetectorIdentifyDRMDevice(t *testing.T) {
	tmpDir := t.TempDir()
	devicePath := filepath.Join(tmpDir, "device")
	os.MkdirAll(devicePath, 0755)

	tests := []struct {
		name       string
		vendorID   string
		wantVendor Vendor
		wantDriver string
		wantErr    bool
	}{
		{
			name:       "Intel i915",
			vendorID:   "0x8086",
			wantVendor: VendorIntel,
			wantDriver: "i915",
			wantErr:    false,
		},
		{
			name:       "AMD",
			vendorID:   "0x1002",
			wantVendor: VendorAMD,
			wantDriver: "amdgpu",
			wantErr:    false,
		},
		{
			name:       "NVIDIA",
			vendorID:   "0x10de",
			wantVendor: VendorNVIDIA,
			wantDriver: "nvidia",
			wantErr:    false,
		},
		{
			name:     "Unknown vendor",
			vendorID: "0x9999",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.WriteFile(filepath.Join(devicePath, "vendor"), []byte(tt.vendorID+"\n"), 0644)
			
			d := &Detector{drmPath: tmpDir, pciPath: tmpDir}
			vendor, driver, err := d.identifyDRMDevice(devicePath)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("identifyDRMDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if vendor != tt.wantVendor {
					t.Errorf("identifyDRMDevice() vendor = %v, want %v", vendor, tt.wantVendor)
				}
				if driver != tt.wantDriver {
					t.Errorf("identifyDRMDevice() driver = %v, want %v", driver, tt.wantDriver)
				}
			}
		})
	}
}

func TestIsDiscreteGPU(t *testing.T) {
	tmpDir := t.TempDir()
	devicePath := filepath.Join(tmpDir, "device")
	os.MkdirAll(devicePath, 0755)

	// Write a VGA class code (0x0300)
	os.WriteFile(filepath.Join(devicePath, "class"), []byte("0x030000\n"), 0644)

	tests := []struct {
		name         string
		resourceData string
		wantDiscrete bool
	}{
		{
			name: "large VRAM (discrete)",
			// start end flags - 1GB region (0xe0000000 to 0x11fffffff)
			resourceData:   "0x00000000e0000000 0x000000011fffffff 0x000000000004020c\n",
			wantDiscrete:   true,
		},
		{
			name: "small VRAM (integrated)",
			// start end flags - 16MB region
			resourceData:   "0x00000000e0000000 0x00000000e0ffffff 0x000000000004020c\n",
			wantDiscrete:   false,
		},
		{
			name: "invalid resource format",
			resourceData:   "invalid\n",
			wantDiscrete:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.WriteFile(filepath.Join(devicePath, "resource"), []byte(tt.resourceData), 0644)
			
			d := &Detector{drmPath: tmpDir, pciPath: tmpDir}
			got := d.isDiscreteGPU(devicePath)
			
			if got != tt.wantDiscrete {
				t.Errorf("isDiscreteGPU() = %v, want %v", got, tt.wantDiscrete)
			}
		})
	}
}

func TestFindRenderNode(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := filepath.Join(tmpDir, "card0")
	drmPath := filepath.Join(cardPath, "device", "drm")
	os.MkdirAll(drmPath, 0755)

	tests := []struct {
		name        string
		setupFiles  []string
		wantRender  string
	}{
		{
			name:       "renderD128 exists",
			setupFiles: []string{"renderD128"},
			wantRender: "/dev/dri/renderD128",
		},
		{
			name:       "renderD129 exists",
			setupFiles: []string{"renderD129"},
			wantRender: "/dev/dri/renderD129",
		},
		{
			name:       "no render node",
			setupFiles: []string{"card0"},
			wantRender: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up drm directory
			entries, _ := os.ReadDir(drmPath)
			for _, e := range entries {
				os.Remove(filepath.Join(drmPath, e.Name()))
			}
			
			// Create test files
			for _, f := range tt.setupFiles {
				os.WriteFile(filepath.Join(drmPath, f), []byte{}, 0644)
			}
			
			d := &Detector{drmPath: tmpDir, pciPath: tmpDir}
			got := d.findRenderNode(cardPath)
			
			if got != tt.wantRender {
				t.Errorf("findRenderNode() = %q, want %q", got, tt.wantRender)
			}
		})
	}
}

func TestIsNVIDIAGPUDetected(t *testing.T) {
	d := &Detector{}
	
	tests := []struct {
		name     string
		existing []GPUInfo
		nvGPU    GPUInfo
		want     bool
	}{
		{
			name: "NVIDIA already detected via DRM",
			existing: []GPUInfo{
				{Vendor: VendorNVIDIA, Devices: []string{"/dev/dri/card0", "/dev/dri/renderD128"}},
			},
			nvGPU: GPUInfo{Vendor: VendorNVIDIA, Devices: []string{"/dev/nvidia0", "/dev/dri/card0"}},
			want:  true,
		},
		{
			name: "NVIDIA not detected",
			existing: []GPUInfo{
				{Vendor: VendorIntel, Devices: []string{"/dev/dri/card0"}},
			},
			nvGPU: GPUInfo{Vendor: VendorNVIDIA, Devices: []string{"/dev/nvidia0"}},
			want:  false,
		},
		{
			name:     "empty existing list",
			existing: []GPUInfo{},
			nvGPU:    GPUInfo{Vendor: VendorNVIDIA, Devices: []string{"/dev/nvidia0"}},
			want:     false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.isNVIDIAGPUDetected(tt.existing, tt.nvGPU)
			if got != tt.want {
				t.Errorf("isNVIDIAGPUDetected() = %v, want %v", got, tt.want)
			}
		})
	}
}
