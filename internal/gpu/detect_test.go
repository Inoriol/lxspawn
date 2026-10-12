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
