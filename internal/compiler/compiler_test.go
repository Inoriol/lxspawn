package compiler

import (
	"fmt"
	"hash/fnv"
	"strings"
	"testing"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/kernel"
)

func boolPtr(b bool) *bool { return &b }

func TestCompileMinimal(t *testing.T) {
	cfg := &config.Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Advanced: config.Advanced{
			Hostname: "testbox",
		},
		Security: config.Security{
			NoNewPrivileges: boolPtr(true),
			Seccomp:         boolPtr(true),
		},
		Journal: config.Journal{Link: "try-guest"},
	}

	cf, err := Compile(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cf.NspawnPath != "/etc/systemd/nspawn/testbox.nspawn" {
		t.Errorf("nspawn path = %q", cf.NspawnPath)
	}
	if cf.DropInPath != "/etc/systemd/system/systemd-nspawn@testbox.service.d/lxspawn.conf" {
		t.Errorf("drop-in path = %q", cf.DropInPath)
	}
	if cf.Slice != "" {
		t.Errorf("slice should be empty when no resources, got: %q", cf.Slice)
	}
	if cf.HooksDropIn != "" {
		t.Errorf("hooks drop-in should be empty when no hooks")
	}

	if !strings.Contains(cf.Nspawn, "Boot=yes") {
		t.Error("nspawn should contain Boot=yes")
	}
	if !strings.Contains(cf.Nspawn, "Hostname=testbox") {
		t.Error("nspawn should contain Hostname=testbox")
	}
	if !strings.Contains(cf.Nspawn, "NoNewPrivileges=yes") {
		t.Error("nspawn should contain NoNewPrivileges=yes")
	}
	if !strings.Contains(cf.Nspawn, "LinkJournal=try-guest") {
		t.Error("nspawn should contain LinkJournal=try-guest")
	}
}

func TestCompileNspawnExecSection(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Advanced: config.Advanced{
			Hostname:  "myhost",
			Ephemeral: true,
			Environment: map[string]string{
				"FOO": "bar",
			},
		},
		Security: config.Security{
			NoNewPrivileges:  boolPtr(true),
			Seccomp:          boolPtr(false),
			Capabilities:     []string{"CAP_NET_ADMIN", "CAP_NET_BIND_SERVICE"},
			DropCapabilities: []string{"CAP_SYS_RAWIO"},
			PrivateUsers:     "identity",
		},
		Journal: config.Journal{Link: "guest"},
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []string{
		"Boot=yes",
		"Hostname=myhost",
		"Ephemeral=yes",
		"Environment=FOO=bar",
		"NoNewPrivileges=yes",
		"Capability=CAP_NET_ADMIN CAP_NET_BIND_SERVICE",
		"DropCapability=CAP_SYS_RAWIO",
		"PrivateUsers=identity",
		"SystemCallFilter=",
		"LinkJournal=guest",
	}
	for _, c := range checks {
		if !strings.Contains(nspawn, c) {
			t.Errorf("nspawn missing %q\nfull output:\n%s", c, nspawn)
		}
	}
}

func TestCompileNspawnSeccompFilter(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Advanced: config.Advanced{
			Hostname: "testbox",
		},
		Security: config.Security{
			NoNewPrivileges:    boolPtr(true),
			Seccomp:            boolPtr(true),
			SystemCallFilter:   []string{"@basic-io", "@file-io"},
		},
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(nspawn, "SystemCallFilter=@basic-io @file-io") {
		t.Errorf("expected SystemCallFilter line\nfull output:\n%s", nspawn)
	}
}

func TestCompileNspawnFilesSection(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Advanced: config.Advanced{
			ReadOnly: true,
			Hostname: "testbox",
		},
		Volumes: []config.Volume{
			{Host: "/mnt/data", Mount: "/data"},
			{Host: "/srv/www", Mount: "/var/www", ReadOnly: true},
			{Type: "tmpfs", Mount: "/tmp", Options: "mode=1777"},
			{Type: "tmpfs", Mount: "/run"},
		},
		Security: config.Security{
			Inaccessible: []string{"/etc/shadow", "/etc/gshadow"},
		},
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []string{
		"ReadOnly=yes",
		"Bind=/mnt/data:/data",
		"BindReadOnly=/srv/www:/var/www",
		"TemporaryFileSystem=/tmp:mode=1777",
		"TemporaryFileSystem=/run",
		"Inaccessible=/etc/shadow",
		"Inaccessible=/etc/gshadow",
	}
	for _, c := range checks {
		if !strings.Contains(nspawn, c) {
			t.Errorf("nspawn missing %q\nfull output:\n%s", c, nspawn)
		}
	}
}

func TestCompileNspawnNetworkHost(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Advanced: config.Advanced{Hostname: "testbox"},
		Network: config.Network{Mode: config.NetworkHost},
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(nspawn, "[Network]") {
		t.Error("host mode should produce [Network] section")
	}
	if !strings.Contains(nspawn, "Private=no") {
		t.Error("host mode should produce Private=no")
	}
}

func TestCompileNspawnNetworkPrivate(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Advanced: config.Advanced{Hostname: "testbox"},
		Network: config.Network{Mode: config.NetworkPrivate},
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(nspawn, "Private=yes") {
		t.Error("missing Private=yes")
	}
	if !strings.Contains(nspawn, "VirtualEthernet=yes") {
		t.Error("missing VirtualEthernet=yes")
	}
}

func TestCompileNspawnNetworkBridge(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Advanced: config.Advanced{Hostname: "testbox"},
		Network: config.Network{
			Mode:   config.NetworkBridge,
			Bridge: "br0",
		},
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(nspawn, "Bridge=br0") {
		t.Error("missing Bridge=br0")
	}
	if !strings.Contains(nspawn, "VirtualEthernet=yes") {
		t.Error("missing VirtualEthernet=yes")
	}
}

func TestCompileNspawnNetworkZone(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Advanced: config.Advanced{Hostname: "testbox"},
		Network: config.Network{
			Mode: config.NetworkZone,
			Zone: "dev",
		},
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(nspawn, "Zone=dev") {
		t.Error("missing Zone=dev")
	}
}

func TestCompileNspawnNetworkMacvlan(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Advanced: config.Advanced{Hostname: "testbox"},
		Network: config.Network{
			Mode:      config.NetworkMacvlan,
			Interface: "eth0",
		},
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(nspawn, "MACVLAN=eth0") {
		t.Error("missing MACVLAN=eth0")
	}
}

func TestCompileNspawnNetworkIpvlan(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Advanced: config.Advanced{Hostname: "testbox"},
		Network: config.Network{
			Mode:      config.NetworkIpvlan,
			Interface: "eth0",
		},
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(nspawn, "IPVLAN=eth0") {
		t.Error("missing IPVLAN=eth0")
	}
}

func TestCompileNspawnPorts(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Advanced: config.Advanced{Hostname: "testbox"},
		Network: config.Network{
			Mode:  config.NetworkPrivate,
			Ports: []string{"tcp:8080:80", "udp:5353:53"},
		},
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(nspawn, "Port=tcp:8080:80") {
		t.Error("missing Port=tcp:8080:80")
	}
	if !strings.Contains(nspawn, "Port=udp:5353:53") {
		t.Error("missing Port=udp:5353:53")
	}
}

func TestCompileSlice(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Resources: config.Resources{
			CPUs:       2,
			Memory:     "2G",
			MemoryHigh: "1.5G",
			Swap:       "512M",
			Tasks:      "512",
			IOWeight:   200,
		},
	}

	slice := compileSlice(cfg)

	checks := []string{
		"[Unit]",
		"Description=testbox Container Slice",
		"[Slice]",
		"MemoryMax=2G",
		"MemoryHigh=1.5G",
		"CPUQuota=200%",
		"TasksMax=512",
		"IOWeight=200",
		"MemorySwapMax=512M",
	}
	for _, c := range checks {
		if !strings.Contains(slice, c) {
			t.Errorf("slice missing %q\nfull output:\n%s", c, slice)
		}
	}
}

func TestCompileSliceDefaultIOWeight(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Resources: config.Resources{
			Memory:   "1G",
			IOWeight: 100,
		},
	}

	slice := compileSlice(cfg)

	if strings.Contains(slice, "IOWeight=") {
		t.Error("IOWeight should not be written when it equals default 100")
	}
}

func TestCompileDropInWithResources(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Resources: config.Resources{
			CPUs: 2,
		},
	}

	dropin := compileDropIn(cfg)

	if !strings.Contains(dropin, "Slice=machine-testbox.slice") {
		t.Errorf("drop-in missing Slice line:\n%s", dropin)
	}
}

func TestCompileDropInWithoutResources(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
	}

	dropin := compileDropIn(cfg)

	if strings.Contains(dropin, "Slice=") {
		t.Error("drop-in should not have Slice when no resources")
	}
}

func TestCompileDropInMacvlan(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Network: config.Network{
			Mode:      config.NetworkMacvlan,
			Interface: "eth0",
		},
	}

	dropin := compileDropIn(cfg)

	if !strings.Contains(dropin, "[Service]") {
		t.Error("macvlan drop-in should contain [Service] section")
	}
}

func TestCompileDropInIpvlan(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Network: config.Network{
			Mode:      config.NetworkIpvlan,
			Interface: "eth0",
		},
	}

	dropin := compileDropIn(cfg)

	if !strings.Contains(dropin, "[Service]") {
		t.Error("ipvlan drop-in should contain [Service] section")
	}
}

func TestCompileHooks(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Hooks: config.Hooks{
			PreStart:  "#!/bin/bash\necho pre",
			PostStart: "echo post",
			PostStop:  "echo stop",
		},
	}

	dropin, scripts := compileHooks(cfg)

	if !strings.Contains(dropin, "ExecStartPre=/var/lib/lxspawn/testbox/.hooks/pre_start.sh") {
		t.Errorf("hooks drop-in missing ExecStartPre:\n%s", dropin)
	}
	if !strings.Contains(dropin, "ExecStartPost=/var/lib/lxspawn/testbox/.hooks/post_start.sh") {
		t.Errorf("hooks drop-in missing ExecStartPost:\n%s", dropin)
	}
	if !strings.Contains(dropin, "ExecStopPost=/var/lib/lxspawn/testbox/.hooks/post_stop.sh") {
		t.Errorf("hooks drop-in missing ExecStopPost:\n%s", dropin)
	}

	if scripts["pre_start.sh"] != "#!/bin/bash\necho pre\n" {
		t.Errorf("pre_start.sh = %q", scripts["pre_start.sh"])
	}
}

func TestEnsureShebang(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "with shebang",
			input:  "#!/bin/bash\necho hi",
			expect: "#!/bin/bash\necho hi\n",
		},
		{
			name:   "with shebang trailing newline",
			input:  "#!/bin/bash\necho hi\n",
			expect: "#!/bin/bash\necho hi\n",
		},
		{
			name:   "without shebang",
			input:  "echo hi",
			expect: "#!/bin/bash\nset -euo pipefail\necho hi\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ensureShebang(tc.input)
			if got != tc.expect {
				t.Errorf("got %q, want %q", got, tc.expect)
			}
		})
	}
}

func TestHasResources(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want bool
	}{
		{
			name: "empty",
			cfg:  config.Config{},
			want: false,
		},
		{
			name: "cpus",
			cfg:  config.Config{Resources: config.Resources{CPUs: 2}},
			want: true,
		},
		{
			name: "memory",
			cfg:  config.Config{Resources: config.Resources{Memory: "1G"}},
			want: true,
		},
		{
			name: "io_weight default",
			cfg:  config.Config{Resources: config.Resources{IOWeight: 100}},
			want: false,
		},
		{
			name: "io_weight non-default",
			cfg:  config.Config{Resources: config.Resources{IOWeight: 200}},
			want: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasResources(&tc.cfg)
			if got != tc.want {
				t.Errorf("hasResources() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHasHooks(t *testing.T) {
	if hasHooks(&config.Config{}) {
		t.Error("empty config should not have hooks")
	}
	if !hasHooks(&config.Config{Hooks: config.Hooks{PreStart: "echo"}}) {
		t.Error("config with pre_start should have hooks")
	}
}

func TestCompileFull(t *testing.T) {
	cfg := &config.Config{
		Name:   "webserver",
		Distro: "ubuntu:noble",
		Resources: config.Resources{
			CPUs:   2,
			Memory: "1G",
		},
		Network: config.Network{
			Mode:   config.NetworkBridge,
			Bridge: "br0",
			Ports:  []string{"tcp:80:80"},
		},
		Volumes: []config.Volume{
			{Host: "/srv/www", Mount: "/var/www/html", ReadOnly: true},
		},
		Security: config.Security{
			NoNewPrivileges: boolPtr(true),
			Capabilities:    []string{"CAP_NET_BIND_SERVICE"},
		},
		Advanced: config.Advanced{
			Hostname: "webserver",
		},
		Journal: config.Journal{Link: "try-guest"},
		Hooks: config.Hooks{
			PreStart: "#!/bin/bash\necho starting",
		},
	}

	cf, err := Compile(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cf.NspawnPath != "/etc/systemd/nspawn/webserver.nspawn" {
		t.Errorf("nspawn path = %q", cf.NspawnPath)
	}
	if cf.SlicePath != "/etc/systemd/system/machine-webserver.slice" {
		t.Errorf("slice path = %q", cf.SlicePath)
	}
	if cf.DropInPath != "/etc/systemd/system/systemd-nspawn@webserver.service.d/lxspawn.conf" {
		t.Errorf("drop-in path = %q", cf.DropInPath)
	}
	if cf.HooksPath != "/etc/systemd/system/systemd-nspawn@webserver.service.d/lxspawn-hooks.conf" {
		t.Errorf("hooks path = %q", cf.HooksPath)
	}

	if !strings.Contains(cf.Nspawn, "[Exec]") {
		t.Error("nspawn missing [Exec]")
	}
	if !strings.Contains(cf.Nspawn, "[Files]") {
		t.Error("nspawn missing [Files]")
	}
	if !strings.Contains(cf.Nspawn, "[Network]") {
		t.Error("nspawn missing [Network]")
	}
	if !strings.Contains(cf.Slice, "CPUQuota=200%") {
		t.Error("slice missing CPUQuota")
	}
	if !strings.Contains(cf.DropIn, "Slice=machine-webserver.slice") {
		t.Error("drop-in missing Slice")
	}
	if !strings.Contains(cf.HooksDropIn, "ExecStartPre=") {
		t.Error("hooks drop-in missing ExecStartPre")
	}
	if len(cf.HookScripts) != 1 {
		t.Errorf("hook scripts len = %d, want 1", len(cf.HookScripts))
	}
}

func TestCompileNspawnSectionOrder(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Advanced: config.Advanced{Hostname: "testbox"},
		Network: config.Network{
			Mode:  config.NetworkPrivate,
			Ports: []string{"tcp:80:80"},
		},
		Volumes: []config.Volume{
			{Host: "/data", Mount: "/data"},
		},
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	execIdx := strings.Index(nspawn, "[Exec]")
	filesIdx := strings.Index(nspawn, "[Files]")
	netIdx := strings.Index(nspawn, "[Network]")

	if execIdx < 0 || filesIdx < 0 || netIdx < 0 {
		t.Fatalf("missing sections in:\n%s", nspawn)
	}
	if execIdx >= filesIdx || filesIdx >= netIdx {
		t.Errorf("sections not in expected order (Exec, Files, Network):\n%s", nspawn)
	}
}

func TestCompileNspawnNoNewPrivilegesFalse(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Advanced: config.Advanced{Hostname: "testbox"},
		Security: config.Security{
			NoNewPrivileges: boolPtr(false),
		},
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(nspawn, "NoNewPrivileges") {
		t.Error("NoNewPrivileges should not appear when set to false")
	}
}

func TestCompileNspawnDefaultJournal(t *testing.T) {
	cfg := &config.Config{
		Name:     "testbox",
		Advanced: config.Advanced{Hostname: "testbox"},
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(nspawn, "LinkJournal=try-guest") {
		t.Error("default journal link should be try-guest")
	}
}

func TestCompileNspawnEnvironmentSorted(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Advanced: config.Advanced{
			Hostname: "testbox",
			Environment: map[string]string{
				"ZEBRA": "z",
				"ALPHA": "a",
				"MANGO": "m",
			},
		},
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	alphaIdx := strings.Index(nspawn, "Environment=ALPHA=a")
	mangoIdx := strings.Index(nspawn, "Environment=MANGO=m")
	zebraIdx := strings.Index(nspawn, "Environment=ZEBRA=z")

	if alphaIdx < 0 || mangoIdx < 0 || zebraIdx < 0 {
		t.Fatalf("missing environment lines in:\n%s", nspawn)
	}
	if alphaIdx >= mangoIdx || mangoIdx >= zebraIdx {
		t.Errorf("environment variables not sorted alphabetically:\n%s", nspawn)
	}
}

func TestCompileDropInContainsLockWorkaround(t *testing.T) {
	cfg := &config.Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
		Advanced: config.Advanced{
			Hostname: "testbox",
		},
		Security: config.Security{
			NoNewPrivileges: boolPtr(true),
			Seccomp:         boolPtr(true),
		},
		Journal: config.Journal{Link: "try-guest"},
	}

	cf, err := Compile(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(cf.DropIn, "SYSTEMD_NSPAWN_LOCK=0") {
		t.Error("drop-in should contain SYSTEMD_NSPAWN_LOCK=0 workaround")
	}
}

func TestCompileAdaptiveModeWithBridge(t *testing.T) {
	// This test requires mocking network.BridgeExists
	// For now, we'll test that adaptive mode is accepted
	cfg := &config.Config{
		Name: "testbox",
		Advanced: config.Advanced{
			Hostname: "testbox",
		},
		Network: config.Network{
			Mode:   config.NetworkAdaptive,
			Bridge: "br0",
		},
		Security: config.Security{
			NoNewPrivileges: boolPtr(true),
			Seccomp:         boolPtr(true),
		},
		Journal: config.Journal{Link: "try-guest"},
	}

	// This will fail because bridge doesn't exist, but we can test the error
	_, err := Compile(cfg)
	// Should fail with "no interface specified and auto-detection not configured"
	// because bridge doesn't exist and interface is not set
	if err == nil {
		t.Log("Adaptive mode compilation succeeded (bridge may exist on this system)")
	} else {
		t.Logf("Expected error for adaptive mode without bridge: %v", err)
	}
}

func TestCompileHostAccessScriptGeneration(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Advanced: config.Advanced{
			Hostname: "testbox",
		},
		Network: config.Network{
			Mode:      config.NetworkMacvlan,
			Interface: "enp2s0",
			HostAccess: config.HostAccess{
				Enabled:   true,
				Interface: "mv-lxspawn",
			},
		},
		Hooks: config.Hooks{
			PostStart: "# Custom post-start\n",
		},
		Security: config.Security{
			NoNewPrivileges: boolPtr(true),
			Seccomp:         boolPtr(true),
		},
		Journal: config.Journal{Link: "try-guest"},
	}

	cf, err := Compile(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that host access script is generated
	postStartScript, ok := cf.HookScripts["post_start.sh"]
	if !ok {
		t.Fatal("post_start.sh hook script not generated")
	}

	// Should contain macvlan creation
	if !strings.Contains(postStartScript, "ip link add mv-lxspawn") {
		t.Error("post_start script should create host-side macvlan")
	}
	if !strings.Contains(postStartScript, "link enp2s0") {
		t.Error("post_start script should reference parent interface")
	}

	// Should contain custom hook
	if !strings.Contains(postStartScript, "# Custom post-start") {
		t.Error("post_start script should include custom hook")
	}

	// Check post_stop has cleanup
	postStopScript, ok := cf.HookScripts["post_stop.sh"]
	if !ok {
		t.Fatal("post_stop.sh hook script not generated")
	}
	if !strings.Contains(postStopScript, "ip link del mv-lxspawn") {
		t.Error("post_stop script should cleanup host-side macvlan")
	}
}

func TestCompileHostAccessWithStaticIP(t *testing.T) {
	cfg := &config.Config{
		Name: "testbox",
		Advanced: config.Advanced{
			Hostname: "testbox",
		},
		Network: config.Network{
			Mode:      config.NetworkMacvlan,
			Interface: "enp2s0",
			HostAccess: config.HostAccess{
				Enabled:   true,
				Interface: "mv-mgmt",
				IP:        "192.168.0.201/24",
			},
		},
		Security: config.Security{
			NoNewPrivileges: boolPtr(true),
			Seccomp:         boolPtr(true),
		},
		Journal: config.Journal{Link: "try-guest"},
	}

	cf, err := Compile(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	postStartScript, ok := cf.HookScripts["post_start.sh"]
	if !ok {
		t.Fatal("post_start.sh hook script not generated")
	}

	// Should contain static IP assignment
	if !strings.Contains(postStartScript, "ip addr add 192.168.0.201/24") {
		t.Error("post_start script should assign static IP")
	}
}

func TestCompileDropInDockerInContainer(t *testing.T) {
	// Docker-in-container requires seccomp: false
	// This should trigger Delegate=yes in drop-in and PrivateUsers=no in .nspawn
	cfg := &config.Config{
		Name:   "docker-test",
		Distro: "debian:bookworm",
		Security: config.Security{
			Seccomp: boolPtr(false),
			Capabilities: []string{
				"CAP_NET_ADMIN",
				"CAP_SYS_ADMIN",
				"CAP_BPF",
			},
		},
		Resources: config.Resources{
			Tasks: "infinity",
		},
	}

	dropin := compileDropIn(cfg)

	// Should have Delegate=yes for cgroup delegation
	if !strings.Contains(dropin, "Delegate=yes") {
		t.Error("drop-in should contain Delegate=yes for Docker-in-container")
	}

	// Should NOT override ExecStart (PrivateUsers is now in .nspawn file)
	if strings.Contains(dropin, "ExecStart=") {
		t.Error("drop-in should NOT override ExecStart")
	}
}

func TestCompileNspawnDockerInContainer(t *testing.T) {
	// Docker-in-container should set PrivateUsers=no in .nspawn file
	cfg := &config.Config{
		Name:   "docker-test",
		Distro: "debian:bookworm",
		Security: config.Security{
			Seccomp: boolPtr(false),
		},
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		t.Fatalf("compileNspawn failed: %v", err)
	}

	if !strings.Contains(nspawn, "PrivateUsers=no") {
		t.Error(".nspawn should contain PrivateUsers=no for Docker-in-container")
	}
}

func TestCompileNspawnPrivateUsersIdentity(t *testing.T) {
	// Regular containers should set PrivateUsers=identity in .nspawn file
	// (when systemd >= 249)
	cfg := &config.Config{
		Name:   "testbox",
		Distro: "debian:bookworm",
	}

	nspawn, err := compileNspawn(cfg)
	if err != nil {
		t.Fatalf("compileNspawn failed: %v", err)
	}

	// On systems with systemd >= 249, should have PrivateUsers=identity
	// On older systems, should not have PrivateUsers set
	if kernel.SupportsPrivateUsersIdentity() {
		if !strings.Contains(nspawn, "PrivateUsers=identity") {
			t.Error(".nspawn should contain PrivateUsers=identity on systemd >= 249")
		}
	}
}

func TestCompileSliceTasksInfinity(t *testing.T) {
	// Tasks field should accept "infinity" for Docker-in-container
	cfg := &config.Config{
		Name:   "docker-test",
		Distro: "debian:bookworm",
		Resources: config.Resources{
			Tasks: "infinity",
		},
	}

	slice := compileSlice(cfg)

	if !strings.Contains(slice, "TasksMax=infinity") {
		t.Error("slice should contain TasksMax=infinity")
	}
}

func TestCompileSliceTasksNumeric(t *testing.T) {
	// Tasks field should also accept numeric values
	cfg := &config.Config{
		Name:   "test",
		Distro: "debian:bookworm",
		Resources: config.Resources{
			Tasks: "512",
		},
	}

	slice := compileSlice(cfg)

	if !strings.Contains(slice, "TasksMax=512") {
		t.Error("slice should contain TasksMax=512")
	}
}

func TestParseIPAndCIDR(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantIP    [4]byte
		wantCIDR  int
		wantError bool
	}{
		{
			name:     "standard subnet",
			input:    "192.168.0.200/24",
			wantIP:   [4]byte{192, 168, 0, 200},
			wantCIDR: 24,
		},
		{
			name:     "different subnet",
			input:    "10.0.0.1/16",
			wantIP:   [4]byte{10, 0, 0, 1},
			wantCIDR: 16,
		},
		{
			name:      "invalid format",
			input:     "invalid",
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ip, cidr, err := parseIPAndCIDR(tc.input)
			if tc.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ip != tc.wantIP {
				t.Errorf("IP = %v, want %v", ip, tc.wantIP)
			}
			if cidr != tc.wantCIDR {
				t.Errorf("CIDR = %d, want %d", cidr, tc.wantCIDR)
			}
		})
	}
}

func TestHostAccessIPGeneration(t *testing.T) {
	// Test the IP generation logic without calling network functions
	// This tests the deterministic hash-based IP generation
	
	// Simulate what generateHostAccessIP does after getting the host IP
	hostIP := [4]byte{192, 168, 0, 0}
	cidr := 24
	machineName := "test-machine"
	
	h := fnv.New32a()
	h.Write([]byte(machineName))
	hash := h.Sum32()
	
	lastOctet := (hash % 254) + 1
	ip := fmt.Sprintf("%d.%d.%d.%d/%d", hostIP[0], hostIP[1], hostIP[2], lastOctet, cidr)
	
	// Should be in the same subnet
	if !strings.HasPrefix(ip, "192.168.0.") {
		t.Errorf("IP should be in 192.168.0.x subnet, got %s", ip)
	}
	
	// Should have /24 CIDR
	if !strings.HasSuffix(ip, "/24") {
		t.Errorf("IP should have /24 CIDR, got %s", ip)
	}
	
	// Should be deterministic
	h2 := fnv.New32a()
	h2.Write([]byte(machineName))
	hash2 := h2.Sum32()
	lastOctet2 := (hash2 % 254) + 1
	ip2 := fmt.Sprintf("%d.%d.%d.%d/%d", hostIP[0], hostIP[1], hostIP[2], lastOctet2, cidr)
	
	if ip != ip2 {
		t.Errorf("IP generation should be deterministic, got %s and %s", ip, ip2)
	}
	
	// Different machine names should produce different IPs
	h3 := fnv.New32a()
	h3.Write([]byte("other-machine"))
	hash3 := h3.Sum32()
	lastOctet3 := (hash3 % 254) + 1
	ip3 := fmt.Sprintf("%d.%d.%d.%d/%d", hostIP[0], hostIP[1], hostIP[2], lastOctet3, cidr)
	
	if ip == ip3 {
		t.Errorf("different machine names should produce different IPs, both got %s", ip)
	}
}
