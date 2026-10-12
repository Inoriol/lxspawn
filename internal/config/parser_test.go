package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseMinimal(t *testing.T) {
	path := writeTempYAML(t, `
name: testbox
distro: debian:bookworm
`)
	cfg, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "testbox" {
		t.Errorf("name = %q, want %q", cfg.Name, "testbox")
	}
	if cfg.Distro != "debian:bookworm" {
		t.Errorf("distro = %q, want %q", cfg.Distro, "debian:bookworm")
	}
}

func TestParseDefaults(t *testing.T) {
	path := writeTempYAML(t, `
name: testbox
distro: debian:bookworm
`)
	cfg, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Network.Mode != NetworkHost {
		t.Errorf("network.mode = %q, want %q", cfg.Network.Mode, NetworkHost)
	}
	if cfg.Advanced.Hostname != "testbox" {
		t.Errorf("hostname = %q, want %q", cfg.Advanced.Hostname, "testbox")
	}
	if cfg.Security.NoNewPrivileges == nil || !*cfg.Security.NoNewPrivileges {
		t.Error("no_new_privileges default should be true")
	}
	if cfg.Security.Seccomp == nil || !*cfg.Security.Seccomp {
		t.Error("seccomp default should be true")
	}
	if cfg.Resources.IOWeight != 100 {
		t.Errorf("io_weight = %d, want 100", cfg.Resources.IOWeight)
	}
}

func TestParseFull(t *testing.T) {
	path := writeTempYAML(t, `
name: full
distro: ubuntu:noble

resources:
  cpus: 4
  memory: 8G
  memory_high: 6G
  swap: 2G
  tasks: 1024
  io_weight: 200

network:
  mode: bridge
  bridge: br0
  ports:
    - "tcp:8080:80"
    - "udp:5353:53"

volumes:
  - host: /mnt/data
    mount: /data
    readonly: true
  - type: tmpfs
    mount: /tmp
    options: "mode=1777"

security:
  no_new_privileges: false
  private_users: "identity"
  seccomp: false
  capabilities:
    - CAP_NET_ADMIN
  drop_capabilities:
    - CAP_SYS_RAWIO
  inaccessible:
    - /etc/shadow
  system_call_filter:
    - "@basic-io"

advanced:
  ephemeral: true
  read_only: true
  hostname: custom-host
  environment:
    FOO: bar

journal:
  link: guest

hooks:
  pre_start: "echo pre"
  post_start: "echo post_start"
  post_stop: "echo post_stop"

param:
  port: "8080"
  env: production

provision:
  - mode: boot
    script: |
      #!/bin/bash
      echo boot
  - mode: data
    path: /etc/config
    content: "hello"
  - mode: yq
    path: /etc/config.json
    expression: '.key = "value"'
    format: json
  - mode: system
    script: |
      #!/bin/bash
      echo system
  - mode: user
    script: |
      #!/bin/bash
      echo user

probes:
  - mode: readiness
    description: "service ready"
    script: |
      #!/bin/bash
      true
    hint: "check logs"
`)
	cfg, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Resources.CPUs != 4 {
		t.Errorf("cpus = %v, want 4", cfg.Resources.CPUs)
	}
	if cfg.Resources.Memory != "8G" {
		t.Errorf("memory = %q, want %q", cfg.Resources.Memory, "8G")
	}
	if cfg.Resources.MemoryHigh != "6G" {
		t.Errorf("memory_high = %q, want %q", cfg.Resources.MemoryHigh, "6G")
	}
	if cfg.Resources.Swap != "2G" {
		t.Errorf("swap = %q, want %q", cfg.Resources.Swap, "2G")
	}
	if cfg.Resources.Tasks != "1024" {
		t.Errorf("tasks = %s, want 1024", cfg.Resources.Tasks)
	}
	if cfg.Resources.IOWeight != 200 {
		t.Errorf("io_weight = %d, want 200", cfg.Resources.IOWeight)
	}

	if cfg.Network.Mode != NetworkBridge {
		t.Errorf("mode = %q, want %q", cfg.Network.Mode, NetworkBridge)
	}
	if cfg.Network.Bridge != "br0" {
		t.Errorf("bridge = %q, want %q", cfg.Network.Bridge, "br0")
	}
	if len(cfg.Network.Ports) != 2 {
		t.Errorf("ports len = %d, want 2", len(cfg.Network.Ports))
	}

	if len(cfg.Volumes) != 2 {
		t.Fatalf("volumes len = %d, want 2", len(cfg.Volumes))
	}
	if cfg.Volumes[0].Host != "/mnt/data" || cfg.Volumes[0].ReadOnly != true {
		t.Errorf("volume[0] unexpected: %+v", cfg.Volumes[0])
	}
	if cfg.Volumes[1].Type != "tmpfs" {
		t.Errorf("volume[1].type = %q, want %q", cfg.Volumes[1].Type, "tmpfs")
	}

	if cfg.Security.NoNewPrivileges == nil || *cfg.Security.NoNewPrivileges {
		t.Error("no_new_privileges should be false")
	}
	if cfg.Security.PrivateUsers != "identity" {
		t.Errorf("private_users = %q, want %q", cfg.Security.PrivateUsers, "identity")
	}
	if cfg.Security.Seccomp == nil || *cfg.Security.Seccomp {
		t.Error("seccomp should be false")
	}
	if len(cfg.Security.Capabilities) != 1 || cfg.Security.Capabilities[0] != "CAP_NET_ADMIN" {
		t.Errorf("capabilities = %v", cfg.Security.Capabilities)
	}
	if len(cfg.Security.DropCapabilities) != 1 {
		t.Errorf("drop_capabilities len = %d", len(cfg.Security.DropCapabilities))
	}
	if len(cfg.Security.Inaccessible) != 1 {
		t.Errorf("inaccessible len = %d", len(cfg.Security.Inaccessible))
	}

	if !cfg.Advanced.Ephemeral {
		t.Error("ephemeral should be true")
	}
	if !cfg.Advanced.ReadOnly {
		t.Error("read_only should be true")
	}
	if cfg.Advanced.Hostname != "custom-host" {
		t.Errorf("hostname = %q", cfg.Advanced.Hostname)
	}
	if cfg.Advanced.Environment["FOO"] != "bar" {
		t.Errorf("environment FOO = %q", cfg.Advanced.Environment["FOO"])
	}

	if cfg.Journal.Link != "guest" {
		t.Errorf("journal.link = %q", cfg.Journal.Link)
	}

	if cfg.Hooks.PreStart != "echo pre" {
		t.Errorf("pre_start = %q", cfg.Hooks.PreStart)
	}

	if cfg.Param["port"] != "8080" {
		t.Errorf("param port = %q", cfg.Param["port"])
	}

	if len(cfg.Provision) != 5 {
		t.Errorf("provision len = %d, want 5", len(cfg.Provision))
	}
	if len(cfg.Probes) != 1 {
		t.Errorf("probes len = %d, want 1", len(cfg.Probes))
	}
}

func TestParseFileNotFound(t *testing.T) {
	_, err := Parse("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseInvalidYAML(t *testing.T) {
	path := writeTempYAML(t, `{{{invalid`)
	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestValidateOK(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMissingName(t *testing.T) {
	cfg := &Config{Distro: "debian:bookworm"}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Errorf("expected name required error, got: %v", err)
	}
}

func TestValidateMissingDistro(t *testing.T) {
	cfg := &Config{Name: "testbox"}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "distro is required") {
		t.Errorf("expected distro required error, got: %v", err)
	}
}

func TestValidateNameWithSpaces(t *testing.T) {
	cfg := &Config{Name: "my box", Distro: "debian:bookworm"}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "alphanumeric") {
		t.Errorf("expected name error, got: %v", err)
	}
}

func TestValidateNameWithSlash(t *testing.T) {
	cfg := &Config{Name: "my/box", Distro: "debian:bookworm"}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "alphanumeric") {
		t.Errorf("expected name error, got: %v", err)
	}
}

func TestValidateNameWithBackslash(t *testing.T) {
	cfg := &Config{Name: `my\box`, Distro: "debian:bookworm"}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "alphanumeric") {
		t.Errorf("expected name error, got: %v", err)
	}
}

func TestValidateInvalidNetworkMode(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Network: Network{Mode: "invalid"},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "invalid network mode") {
		t.Errorf("expected network mode error, got: %v", err)
	}
}

func TestValidateBridgeWithoutName(t *testing.T) {
	cfg := &Config{
		Name:    "testbox",
		Distro:  "debian:bookworm",
		Network: Network{Mode: NetworkBridge},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "network.bridge is required") {
		t.Errorf("expected bridge required error, got: %v", err)
	}
}

func TestValidateZoneWithoutName(t *testing.T) {
	cfg := &Config{
		Name:    "testbox",
		Distro:  "debian:bookworm",
		Network: Network{Mode: NetworkZone},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "network.zone is required") {
		t.Errorf("expected zone required error, got: %v", err)
	}
}

func TestValidateMacvlanWithoutInterface(t *testing.T) {
	cfg := &Config{
		Name:    "testbox",
		Distro:  "debian:bookworm",
		Network: Network{Mode: NetworkMacvlan},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "network.interface is required") {
		t.Errorf("expected interface required error, got: %v", err)
	}
}

func TestValidateIpvlanWithoutInterface(t *testing.T) {
	cfg := &Config{
		Name:    "testbox",
		Distro:  "debian:bookworm",
		Network: Network{Mode: NetworkIpvlan},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "network.interface is required") {
		t.Errorf("expected interface required error, got: %v", err)
	}
}

func TestValidatePortsWithHostMode(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Network: Network{
			Mode:  NetworkHost,
			Ports: []string{"tcp:8080:80"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "port forwarding requires") {
		t.Errorf("expected port forwarding error, got: %v", err)
	}
}

func TestValidatePortsPrivateOK(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Network: Network{
			Mode:  NetworkPrivate,
			Ports: []string{"tcp:8080:80"},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateVolumeTmpfsOK(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Volumes: []Volume{
			{Type: "tmpfs", Mount: "/tmp"},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateVolumeTmpfsMissingMount(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Volumes: []Volume{
			{Type: "tmpfs"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "tmpfs volume requires mount point") {
		t.Errorf("expected tmpfs mount error, got: %v", err)
	}
}

func TestValidateVolumeMissingHost(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Volumes: []Volume{
			{Mount: "/data"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "host path is required") {
		t.Errorf("expected host required error, got: %v", err)
	}
}

func TestValidateVolumeMissingMount(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Volumes: []Volume{
			{Host: "/mnt/data"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "mount point is required") {
		t.Errorf("expected mount required error, got: %v", err)
	}
}

func TestValidateVolumeRelativeHost(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Volumes: []Volume{
			{Host: "relative/path", Mount: "/data"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "host path must be absolute") {
		t.Errorf("expected absolute path error, got: %v", err)
	}
}

func TestValidateVolumeRelativeMount(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Volumes: []Volume{
			{Host: "/mnt/data", Mount: "relative"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "mount point must be absolute") {
		t.Errorf("expected absolute path error, got: %v", err)
	}
}

func TestValidatePortRangeInvalid(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Network: Network{
			Mode:  NetworkPrivate,
			Ports: []string{"tcp:99999:80"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "out of valid range") {
		t.Errorf("expected port range error, got: %v", err)
	}
}

func TestValidatePortRangeContainerInvalid(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Network: Network{
			Mode:  NetworkPrivate,
			Ports: []string{"tcp:8080:70000"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "out of valid range") {
		t.Errorf("expected port range error, got: %v", err)
	}
}

func TestValidatePortRangeZero(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Network: Network{
			Mode:  NetworkPrivate,
			Ports: []string{"tcp:0:80"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "out of valid range") {
		t.Errorf("expected port range error, got: %v", err)
	}
}

func TestValidatePortRangeValid(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Network: Network{
			Mode:  NetworkPrivate,
			Ports: []string{"tcp:8080:80", "udp:443:443"},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateVolumeSensitivePathShadow(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Volumes: []Volume{
			{Host: "/etc/shadow", Mount: "/data/shadow"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "sensitive system directory") {
		t.Errorf("expected sensitive path error, got: %v", err)
	}
}

func TestValidateVolumeSensitivePathProc(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Volumes: []Volume{
			{Host: "/proc", Mount: "/data/proc"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "sensitive system directory") {
		t.Errorf("expected sensitive path error, got: %v", err)
	}
}

func TestValidateVolumeSensitivePathSys(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Volumes: []Volume{
			{Host: "/sys/kernel", Mount: "/data/sys"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "sensitive system directory") {
		t.Errorf("expected sensitive path error, got: %v", err)
	}
}

func TestValidateVolumeSensitivePathDev(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Volumes: []Volume{
			{Host: "/dev/sda", Mount: "/data/dev"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "sensitive system directory") {
		t.Errorf("expected sensitive path error, got: %v", err)
	}
}

func TestValidateVolumeNonSensitivePath(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Volumes: []Volume{
			{Host: "/mnt/data", Mount: "/data"},
			{Host: "/home/user/documents", Mount: "/docs"},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateGPUDevicePathInvalid(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		GPU: GPU{
			Enabled: true,
			Devices: []string{"/etc/passwd"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "must start with /dev/") {
		t.Errorf("expected GPU device path error, got: %v", err)
	}
}

func TestValidateGPUDevicePathValid(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		GPU: GPU{
			Enabled: true,
			Devices: []string{"/dev/dri/card0", "/dev/dri/renderD128"},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateProvisionBootOK(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Provision: []Provision{
			{Mode: ProvisionBoot, Script: "#!/bin/bash\necho hi"},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateProvisionBootMissingScript(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Provision: []Provision{
			{Mode: ProvisionBoot},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "requires script or file") {
		t.Errorf("expected script required error, got: %v", err)
	}
}

func TestValidateProvisionBootWithFile(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Provision: []Provision{
			{Mode: ProvisionBoot, File: &FileRef{URL: "setup.sh"}},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateProvisionDataMissingPath(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Provision: []Provision{
			{Mode: ProvisionData, Content: "hello"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "data mode requires path") {
		t.Errorf("expected path required error, got: %v", err)
	}
}

func TestValidateProvisionDataMissingContent(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Provision: []Provision{
			{Mode: ProvisionData, Path: "/etc/config"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "data mode requires content or file") {
		t.Errorf("expected content required error, got: %v", err)
	}
}

func TestValidateProvisionDataWithFile(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Provision: []Provision{
			{Mode: ProvisionData, Path: "/etc/config", File: &FileRef{URL: "config.txt"}},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateProvisionYQMissingPath(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Provision: []Provision{
			{Mode: ProvisionYQ, Expression: ".key = val"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "yq mode requires path") {
		t.Errorf("expected path required error, got: %v", err)
	}
}

func TestValidateProvisionYQMissingExpression(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Provision: []Provision{
			{Mode: ProvisionYQ, Path: "/etc/config.json"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "yq mode requires expression") {
		t.Errorf("expected expression required error, got: %v", err)
	}
}

func TestValidateProvisionInvalidMode(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Provision: []Provision{
			{Mode: "invalid"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "invalid mode") {
		t.Errorf("expected invalid mode error, got: %v", err)
	}
}

func TestValidateProvisionSystemWithFile(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Provision: []Provision{
			{Mode: ProvisionSystem, File: &FileRef{URL: "setup.sh"}},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateProvisionUserMissingScript(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Provision: []Provision{
			{Mode: ProvisionUser},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "requires script or file") {
		t.Errorf("expected script required error, got: %v", err)
	}
}

func TestValidateAllNetworkModes(t *testing.T) {
	modes := []struct {
		mode NetworkMode
		ok   bool
	}{
		{NetworkHost, true},
		{NetworkPrivate, true},
		{NetworkBridge, false},
		{NetworkZone, false},
		{NetworkMacvlan, false},
		{NetworkIpvlan, false},
		{"bogus", false},
	}
	for _, tc := range modes {
		cfg := &Config{
			Name:    "testbox",
			Distro:  "debian:bookworm",
			Network: Network{Mode: tc.mode},
		}
		err := Validate(cfg)
		if tc.ok && err != nil {
			t.Errorf("mode %q: unexpected error: %v", tc.mode, err)
		}
	}
}

func TestValidateProbesOK(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Probes: []Probe{
			{Mode: "readiness", Description: "service ready", Script: "#!/bin/bash\ntrue"},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateProbeInvalidMode(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Probes: []Probe{
			{Mode: "liveness", Description: "check", Script: "#!/bin/bash\ntrue"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "invalid mode") {
		t.Errorf("expected invalid mode error, got: %v", err)
	}
}

func TestValidateProbeMissingDescription(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Probes: []Probe{
			{Mode: "readiness", Script: "#!/bin/bash\ntrue"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "description is required") {
		t.Errorf("expected description required error, got: %v", err)
	}
}

func TestValidateProbeMissingScript(t *testing.T) {
	cfg := &Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Probes: []Probe{
			{Mode: "readiness", Description: "check"},
		},
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "script is required") {
		t.Errorf("expected script required error, got: %v", err)
	}
}
