# Architecture

This document describes the internal architecture of lxspawn, design decisions, and how the system works.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│           Cockpit Plugin (React/PatternFly)                 │
│           mirrors cockpit-machines UI/UX                    │
└──────────────────────┬──────────────────────────────────────┘
                        │ cockpit.spawn() --json
                        ▼
┌─────────────────────────────────────────────────────────────┐
│              lxspawn (Go CLI)                               │
│   YAML parser → Compiler → .nspawn + slice + drop-in + hooks│
│   D-Bus API (machine1) for status, machinectl for lifecycle │
│   JSON output on every command                              │
└──────────────────────┬──────────────────────────────────────┘
                        │
          ┌─────────────┼─────────────┬─────────────┐
          ▼             ▼             ▼             ▼
   ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
   │ .nspawn     │ │ Slice unit  │ │ Drop-in     │ │ Hooks       │
   │ [Exec]      │ │ [Slice]     │ │ [Service]   │ │ pre/post    │
   │ [Files]     │ │ MemoryMax   │ │ Slice=      │ │ scripts     │
   │ [Network]   │ │ CPUQuota    │ │             │ │             │
   └──────┬──────┘ └──────┬──────┘ └──────┬──────┘ └──────┬──────┘
          │               │               │               │
          ▼               ▼               ▼               ▼
   ┌─────────────────────────────────────────────────────────────┐
   │     systemd-nspawn@<name>.service                           │
   │     /var/lib/machines/<name>/                               │
   └──────────────────────┬───────────────────────────────────────┘
                          │
           ┌──────────────┼──────────────┐
           ▼              ▼              ▼
    ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
    │ machinectl  │ │ D-Bus API   │ │ journalctl  │
    │ start/stop  │ │ machine1    │ │ -M <name>   │
    │ shell       │ │ ListMachines│ │ -f          │
    └─────────────┘ └─────────────┘ └─────────────┘
```

## Package Structure

```
lxspawn/
├── cmd/lxspawn/              # CLI entry point and commands
│   ├── main.go               # Root command, global flags
│   ├── setup.go              # First-run system configuration
│   ├── create.go             # Create machine (don't start)
│   ├── launch.go             # Create + start
│   ├── start.go              # Start existing machine
│   ├── stop.go               # Stop machine (with --force flag)
│   ├── restart.go            # Restart machine (stop then start)
│   ├── delete.go             # Delete machine and cleanup
│   ├── list.go               # List all machines
│   ├── shell.go              # Interactive shell
│   ├── exec.go               # Run command
│   ├── status.go             # Detailed status
│   ├── logs.go               # Follow logs
│   ├── provision.go          # Re-run provisioning
│   ├── snapshot.go           # Create snapshot
│   ├── list_snapshots.go     # List snapshots
│   ├── rollback.go           # Rollback to snapshot
│   ├── delete_snapshot.go    # Delete snapshot
│   ├── stats.go              # Runtime CPU/memory/disk stats
│   ├── config.go             # Show detailed configuration
│   ├── provision_status.go   # Show provisioning status
│   ├── list_distros.go       # List available distributions
│   ├── list_templates.go     # List available templates
│   ├── validate.go           # Validate YAML configuration
│   ├── edit_resources.go     # Edit CPU/memory limits
│   ├── edit_network.go       # Edit network configuration
│   ├── add_volume.go         # Add bind mount
│   ├── edit_volume.go        # Edit bind mount
│   ├── remove_volume.go      # Remove bind mount
│   ├── autostart.go          # Enable/disable machine autostart
│   ├── version.go            # Show version and system info
│   ├── doctor.go             # Check system configuration
│   └── apply.go              # Declarative config apply
│
├── internal/
│   ├── config/               # YAML parsing and validation
│   │   ├── schema.go         # Config struct definitions
│   │   ├── parser.go         # Parse, validate, defaults
│   │   └── parser_test.go    # Tests for parsing and validation
│   │
│   ├── compiler/             # Generate systemd files
│   │   ├── compiler.go       # Main compiler orchestration
│   │   ├── nspawn.go         # Generate .nspawn files
│   │   ├── slice.go          # Generate slice units, drop-ins, hooks
│   │   └── compiler_test.go  # Tests for compiler output
│   │
│   ├── provision/            # Provisioning engine
│   │   ├── engine.go         # Main orchestrator
│   │   ├── template.go       # Template rendering
│   │   ├── boot.go           # boot mode (write scripts to rootfs)
│   │   ├── data.go           # data mode (write files to rootfs)
│   │   ├── yq.go             # yq mode (edit config files)
│   │   ├── exec.go           # system/user modes (systemd-run)
│   │   ├── file.go           # File reference handling
│   │   └── probe.go          # Readiness probes
│   │
│   ├── image/                # Rootfs image management
│   │   ├── fetcher.go        # Download and extract images
│   │   ├── xz.go             # XZ decompression wrapper
│   │   ├── validate.go       # Validate init system presence
│   │   ├── cleanup.go        # Remove machine-id, resolv.conf, mask services
│   │   └── network.go        # Inject systemd-networkd configs
│   │
│   ├── machine/              # Machine lifecycle management
│   │   └── manager.go        # Create, start, stop, force stop, restart, delete, shell, exec, status, apply
│   │
│   ├── storage/              # Storage backend abstraction
│   │   ├── backend.go        # StorageBackend interface
│   │   ├── btrfs.go          # Btrfs subvolume backend
│   │   ├── lvm_thin.go       # LVM thin provisioning backend
│   │   ├── directory.go      # Directory-based fallback backend
│   │   ├── detector.go       # Automatic backend detection
│   │   └── storage_test.go   # Tests for storage backends
│   │
│   ├── snapshot/             # Snapshot management
│   │   ├── snapshot.go       # Core types and interfaces
│   │   ├── detector.go       # Filesystem type detection
│   │   ├── btrfs.go          # btrfs subvolume operations
│   │   ├── zfs.go            # zfs snapshot operations
│   │   ├── lvm.go            # LVM snapshot operations
│   │   ├── directory.go      # Directory-based snapshot operations (rsync)
│   │   └── manager.go        # Snapshot orchestration and state
│   │
│   ├── network/              # Network detection and configuration
│   │   ├── detect.go         # Interface detection, primary interface selection
│   │   ├── bridge.go         # Bridge existence checking
│   │   ├── config.go         # Network config storage (~/.local/share/lxspawn/config.json)
│   │   └── *_test.go         # Tests for network detection
│   │
│   ├── gpu/                  # GPU detection and pass-through
│   │   ├── detect.go         # GPU detection, vendor identification, device selection
│   │   └── detect_test.go    # Tests for GPU detection
│   │
│   ├── deploy/               # Systemd integration (adapted from comquad)
│   │   ├── dbus.go           # systemd D-Bus connection (always system bus)
│   │   ├── state.go          # State management (machines.json)
│   │   ├── filesystem.go     # File operations, paths, atomic writes, SELinux
│   │   ├── interfaces.go     # SystemdClient, StateStore interfaces
│   │   └── is_rootless.go    # Root detection
│   │
│   ├── logger/               # Logging (adapted from comquad)
│   │   └── logger.go         # Colored output, verbose/quiet modes
│   │
│   └── output/               # JSON formatting (adapted from comquad)
│       ├── formatter.go      # JSON envelope, print helpers
│       └── schemas.go        # Machine-specific JSON schemas
│
└── examples/                 # Example YAML configurations
    └── myserver.yaml
```

## Data Flow

### Machine Creation Flow

```
1. User runs: lxspawn launch myserver.yaml
   ↓
2. config.Parse() reads and parses YAML
   ↓
3. config.Validate() checks required fields, network modes, volumes
   ↓
4. compiler.Compile() generates:
   - Resolves adaptive mode (bridge if exists, else macvlan with host access)
   - .nspawn file content
   - slice unit content (if resources defined)
   - drop-in content
   - hook scripts (if hooks defined, including host-side macvlan setup)
   ↓
5. image.Fetcher downloads rootfs from images.linuxcontainers.org
   ↓
6. image.CleanupRootfs() removes machine-id, writes resolv.conf with public DNS, writes securetty
   ↓
7. image.InjectNetworkConfig() adds systemd-networkd configs (if private/macvlan/ipvlan mode)
   ↓
8. deploy.WriteFileAtomic() writes:
   - /etc/systemd/nspawn/<name>.nspawn
   - /etc/systemd/system/machine-<name>.slice
   - /etc/systemd/system/systemd-nspawn@<name>.service.d/lxspawn.conf
   - /var/lib/lxspawn/<name>/.hooks/*.sh (if hooks)
   ↓
9. state.RegisterMachine() saves metadata to /var/lib/lxspawn/machines.json
   ↓
10. provision.Engine.Provision() runs pre-boot provisioning:
    - boot mode: writes scripts to rootfs
    - data mode: writes config files to rootfs
    - yq mode: edits existing config files
    ↓
11. systemd.ReloadDaemon() triggers systemd to pick up new units
    ↓
12. systemd.StartUnit("systemd-nspawn@<name>.service")
    ↓
13. provision.Engine.ProvisionPostBoot() runs post-boot provisioning:
    - system mode: executes scripts as root via machinectl
    - user mode: executes scripts as user via machinectl
    ↓
14. provision.Engine.RunProbes() verifies services are ready
    ↓
15. Machine is running and fully provisioned
```

### State Management

State is filesystem-based with a system-wide JSON metadata file:

```
/var/lib/lxspawn/machines.json
```

Structure:
```json
{
  "myserver": {
    "name": "myserver",
    "distro": "debian:stable",
    "source_path": "/home/user/myserver.yaml",
    "files": [
      "/etc/systemd/nspawn/myserver.nspawn",
      "/etc/systemd/system/machine-myserver.slice",
      "/etc/systemd/system/systemd-nspawn@myserver.service.d/lxspawn.conf"
    ],
    "network": "host",
    "cpus": 2,
    "memory": "2G",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

**Why system-wide state?**
- All users see the same machine list (no per-user registry confusion)
- Works correctly with Cockpit web UI (no user-switching issues)
- State file has 0600 permissions (owner read/write only)
- Atomic writes via temp file + rename with file locking

**Why not a database?**
- Simpler to debug (just `cat` the file)
- Git-trackable (if desired)
- No additional dependencies
- Sufficient for the use case (hundreds of machines, not millions)

## Provisioning Engine

The provisioning engine implements Lima's provisioning model adapted for systemd-nspawn:

### Execution Model

```
┌─────────────────────────────────────────────────────────────┐
│                    PRE-BOOT PHASE                           │
│   Container is NOT running. Write directly to rootfs.       │
│                                                             │
│   1. boot   → Write files/scripts into rootfs               │
│                   (written to /etc/lxspawn/boot/, executed  │
│                    by oneshot systemd service on every boot)│
│   2. data   → Write config files into rootfs                │
│                   (clean file creation, no shell needed)    │
│   3. yq     → Edit existing config files in rootfs          │
│                   (structured editing of JSON/YAML/TOML)    │
└──────────────────────────┬──────────────────────────────────┘
                            │ machinectl start <name>
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    POST-BOOT PHASE                          │
│   Container IS running. Exec via systemd-run -M.            │
│                                                             │
│   4. system → sudo systemd-run -M <name> --pipe --wait \    │
│              -- /bin/bash -c "<script>"                     │
│                   (run as root inside container)            │
│   5. user   → sudo systemd-run -M <name> --pipe --wait \    │
│              -- /bin/bash -c "su <user> -c '<script>'"      │
│                   (run as specified user inside container,  │
│                    user must exist in container)            │
│                                                             │
│   Note: Post-boot commands use systemd-run to execute       │
│   commands inside the container via machined, avoiding      │
│   PTY allocation issues and providing machine name          │
│   validation (more secure than PID-based nsenter).          │
└──────────────────────────┬──────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    READINESS PHASE                          │
│   Check if container is actually ready.                     │
│                                                             │
│   6. probes → Exec readiness scripts via systemd-run -M     │
│                   (wait for services to be available)       │
│                   (10s timeout per attempt, 30 attempts)    │
└─────────────────────────────────────────────────────────────┘
```

### Provisioning State

State is tracked in `/var/lib/lxspawn/<name>/provision-state.json`:

```json
{
  "last_run": "2024-01-15T10:30:00Z",
  "phases_completed": ["pre-boot", "post-boot", "probes"],
  "probes_passed": ["nginx", "postgresql"],
  "script_hashes": {
    "system:0": "sha256:abc...",
    "user:0": "sha256:def..."
  }
}
```

### Template Context

All provision scripts have access to template variables:

```go
type TemplateContext struct {
    Name     string            // Machine name
    Hostname string            // Container hostname
    Distro   string            // Distro:release
    User     string            // Host username
    UID      string            // Host user ID
    Home     string            // Host home directory
    Dir      string            // Directory containing YAML
    Param    map[string]string // Custom parameters
}
```

### Idempotency

- All scripts MUST be idempotent (may run multiple times on container restart)
- `data` mode: `overwrite: false` skips if file exists
- `boot` mode: scripts re-run on every container boot by design
- `system`/`user` modes: script hashes are tracked to skip unchanged scripts on re-run
- `user` mode: requires the specified user to exist in the container (create in a prior `system` step if needed)

### Provisioning Environment

All system and user provision scripts run with the following environment variables automatically set:

- `DEBIAN_FRONTEND=noninteractive` - Prevents apt prompts during package installation
- `DBUS_SYSTEM_BUS_ADDRESS=unix:path=/run/dbus/system_bus_socket` - Required for systemctl commands
- `XDG_RUNTIME_DIR=/run/user/0` - Required for D-Bus connections
- `LXSPAWN_PARAM_<key>=<value>` - All `param:` values exported as environment variables

## Generated Files

### .nspawn File

Location: `/etc/systemd/nspawn/<name>.nspawn`

Purpose: Container configuration (exec, files, network)

Example:
```ini
[Exec]
Boot=yes
Hostname=myserver
NoNewPrivileges=yes
Capability=CAP_NET_BIND_SERVICE
DropCapability=CAP_SYS_RAWIO CAP_MKNOD
LinkJournal=try-guest

[Files]
Bind=/mnt/data:/data
BindReadOnly=/etc/ssl/certs:/etc/ssl/certs
TemporaryFileSystem=/tmp:mode=1777

[Network]
Private=yes
VirtualEthernet=yes
Bridge=br0
Port=tcp:8080:80
```

### Slice Unit

Location: `/etc/systemd/system/machine-<name>.slice`

Purpose: Resource limits (CPU, memory, IO)

Example:
```ini
[Unit]
Description=myserver Container Slice

[Slice]
MemoryMax=2G
MemoryHigh=1.5G
CPUQuota=200%
TasksMax=512
MemorySwapMax=512M
IOWeight=100
```

**Why slices instead of .nspawn?**
- `.nspawn` files don't support resource limits
- Slices are the systemd-native way to control resources
- Can be managed independently of the container
- Work with `systemd-cgtop`, `systemd-cgls` for monitoring

### Drop-in

Location: `/etc/systemd/system/systemd-nspawn@<name>.service.d/lxspawn.conf`

Purpose: Link the machine to its slice

Example:
```ini
[Service]
Slice=machine-myserver.slice
```

### Hook Scripts

Location: `/var/lib/lxspawn/<name>/.hooks/*.sh`

Purpose: Host-side lifecycle hooks (pre_start, post_start, post_stop)

Referenced via drop-in:
```ini
[Service]
ExecStartPre=/var/lib/lxspawn/myserver/.hooks/pre_start.sh
ExecStartPost=/var/lib/lxspawn/myserver/.hooks/post_start.sh
ExecStopPost=/var/lib/lxspawn/myserver/.hooks/post_stop.sh
```

## Design Decisions

### 1. Why systemd-nspawn over LXC/Docker?

**systemd-nspawn advantages:**
- Native systemd integration (no separate daemon)
- First-class support for systemd init inside containers
- Simpler architecture (no containerd, no runc)
- Better security model (leverages systemd's security features)
- Works with existing systemd tools (machinectl, journalctl, systemd-cgtop)

**Trade-offs:**
- Less ecosystem than Docker (no Docker Hub, but that's fine for system containers)
- Fewer features than LXC (no live migration, but we don't need it)
- Requires systemd on host (acceptable for our target audience)

### 2. Why YAML over JSON/TOML?

**YAML advantages:**
- More readable for complex configs (multi-line scripts, nested structures)
- Supports comments (unlike JSON)
- Familiar to DevOps community (Kubernetes, Ansible, Docker Compose)
- Lima uses YAML (our inspiration)

**Trade-offs:**
- Slower parsing (not a concern for our use case)
- Whitespace-sensitive (but so is Python, and that's fine)

### 3. Why Filesystem State over Database?

**Filesystem advantages:**
- Simpler to debug (`cat machines.json`)
- No additional dependencies (no SQLite, no BoltDB)
- Git-trackable (if users want to version their state)
- Atomic writes via temp file + rename
- Sufficient for our scale (hundreds of machines, not millions)

**Trade-offs:**
- No transactions (but we don't need them)
- No complex queries (but we don't need them)
- Manual cleanup if file gets corrupted (rare, and we have atomic writes)

### 4. Why Group + Polkit over Sudo?

**Group + Polkit advantages:**
- No password prompts after setup
- Fine-grained control (only manage lxspawn units, not all systemd)
- Standard Linux security model
- Works with Cockpit (no sudo configuration needed)

**Trade-offs:**
- More complex setup (one-time `lxspawn setup`)
- Requires polkit (not always available on minimal installs)
- Fallback to sudoers if polkit unavailable

### 5. Why images.linuxcontainers.org?

**Advantages:**
- Large catalog of distros (Debian, Ubuntu, Fedora, Arch, Alpine, etc.)
- Regularly updated
- Pre-built rootfs tarballs (no need to run debootstrap, pacstrap, etc.)
- Used by LXC (battle-tested)

**Trade-offs:**
- External dependency (but we can add mirrors/custom sources later)
- Images may not have systemd by default (we validate and reject if missing)

### 6. Why Separate Create and Launch?

**Separate create/launch advantages:**
- Allows dry-run / inspection before starting
- Supports GitOps workflows (create in CI, launch in prod)
- Matches Docker/Podman UX (`docker create` vs `docker run`)
- Enables advanced workflows (create multiple, start in order)

**Trade-offs:**
- Two commands instead of one (but `launch` combines them)
- Slightly more complex state management (created but not started)

### 7. Why systemd-run for Exec and machinectl for Shell?

**systemd-run advantages for exec:**
- No PTY allocation issues (machinectl shell has intermittent PTY errors)
- Works reliably in non-interactive contexts
- Goes through systemd-machined (machine name validation, no arbitrary PID access)
- Clean exit status reporting
- D-Bus works inside container (systemctl commands work)
- More secure than nsenter (machine names instead of PIDs, can be restricted with sudoers regex)

**machinectl advantages for shell:**
- Interactive PTY allocation handled correctly
- Native systemd tool (no additional dependencies)
- Works rootless (after setup)
- Integrates with journal (logs visible on host)

**Implementation:**
- `internal/machine/manager.go:Exec()` uses `systemd-run -M <name> --pipe --wait` to execute commands
- `internal/machine/manager.go:Shell()` uses `machinectl shell` for interactive sessions
- Exec sets `DBUS_SYSTEM_BUS_ADDRESS` and `XDG_RUNTIME_DIR` for systemctl commands

**Trade-offs:**
- Exec requires sudo (systemd-run needs root to create transient services)
- Shell requires machine to be running (can't shell into stopped container)
- systemd-run output includes service status reporting (cosmetic, not a functional issue)

**Security:**
- Sudoers rule uses regex to restrict machine names: `^-M [a-zA-Z0-9][a-zA-Z0-9_-]* --pipe --wait -- /bin/bash -c .*$`
- Prevents arbitrary PID access (unlike nsenter which allowed `nsenter -t 1` for host root)
- Machine names validated by both sudoers regex and machined

### 8. Why D-Bus for Status?

**D-Bus advantages:**
- No command output parsing (fragile, locale-dependent)
- Real-time updates via signals (future enhancement)
- Type safety (structured data, not strings)
- Detailed error messages

**Trade-offs:**
- More complex code (but we abstract it behind interfaces)
- Requires D-Bus (always available on systemd systems)

### 9. Why Always Use System D-Bus Connection?

**System connection advantages:**
- Can manage system units (systemd-nspawn@.service)
- Works with polkit for authentication
- Consistent behavior regardless of user privileges

**Trade-offs:**
- Requires polkit rules for non-root users (handled by `lxspawn setup`)
- Slightly more complex privilege model

### 10. SELinux Context for Hook Scripts

Hook scripts are written to `/var/lib/lxspawn/<name>/.hooks/` and executed by systemd.
On SELinux-enforcing systems, these scripts must have the `bin_t` context to be executable.
The deploy package automatically applies this context when writing hook scripts.

### 11. D-Bus Machine Name Escaping

D-Bus object paths require special characters to be escaped as `_XX` where XX is the hex code.
For example, machine names with hyphens (e.g., `test-machine`) must be escaped as `test_2dmachine`
in D-Bus object paths. The `machineName()` function in `internal/deploy/dbus.go` handles this
escaping automatically for all machine1 D-Bus queries.

### 12. Adaptive Network Mode

The `adaptive` network mode automatically selects between bridge and macvlan based on host configuration:

**Selection logic:**
1. If `bridge` is specified AND exists on host → use bridge mode
2. Otherwise → use macvlan mode with `host_access` enabled

**Why adaptive mode?**
- **Simplifies templates** - Same YAML works across different environments
- **Better UX** - Users don't need to understand bridge vs macvlan tradeoffs
- **Solves host visibility** - Automatically creates host-side macvlan interface

**Implementation:**
- `internal/compiler/compiler.go:resolveNetworkMode()` resolves adaptive mode at compile time
- `internal/network/detect.go` provides interface detection
- `internal/network/bridge.go` checks bridge existence
- `internal/network/config.go` stores detected interface in `~/.local/share/lxspawn/config.json`
- `internal/compiler/slice.go:generateHostAccessScript()` creates host-side macvlan in post_start hook

**Host access for macvlan:**
When falling back to macvlan, lxspawn creates a host-side macvlan interface (`mv-<machinename>`)
so the host can reach the container. This solves the kernel limitation where the host cannot ping
macvlan containers. The interface name is per-machine (truncated to 15 characters with hash suffix
for uniqueness) to prevent conflicts when multiple adaptive machines exist. The interface is created
in `post_start` and removed in `post_stop`.

### 13. DNS Resolution in Containers

Containers need working DNS to install packages and access the network. The image cleanup process
(`internal/image/cleanup.go`) writes a `/etc/resolv.conf` with public DNS servers (8.8.8.8, 1.1.1.1, 8.8.4.4)
instead of the systemd-resolved stub resolver (127.0.0.53) which doesn't work in containers.

This ensures containers can resolve DNS names immediately after boot, even before systemd-resolved
starts inside the container.

### 14. Host Networking Mode

Host networking mode requires explicit `Private=no` in the `[Network]` section of the `.nspawn` file.
Without this, systemd-nspawn defaults to private networking even when no network configuration is
specified. The compiler (`internal/compiler/nspawn.go`) now generates `Private=no` for host mode.

### 15. GPU Pass-Through (Shared Mode)

GPU pass-through in containers is device sharing, not PCI passthrough (which is for VMs). Containers
share the host kernel, so we bind-mount device nodes and configure cgroup device access.

**Implementation:**
- `internal/gpu/detect.go` - Detects GPUs by parsing `/sys/class/drm/card*` and `/dev/nvidia*`
- Identifies vendor (Intel/AMD/NVIDIA/ARM) by reading PCI vendor IDs
- Render nodes read from sysfs (`/sys/class/drm/cardN/device/drm/renderD*`) instead of calculated from card index (handles unrecognized GPUs)
- Discrete GPU detection uses PCI resource file (computes size as `end - start + 1`)
- NVIDIA GPUs deduplicated (DRM scan and `/dev/nvidia*` scan)
- For AMD GPUs, `/dev/kfd` is bind-mounted for ROCm compute support
- Prioritizes discrete GPUs over integrated GPUs (stable sort)
- Generates bind mounts for device nodes in `.nspawn` file
- Generates `DeviceAllow` rules in slice/drop-in for cgroup access
- Adds required capabilities (e.g., `CAP_SYS_ADMIN` for NVIDIA)

**Why shared mode only:**
- Full GPU isolation (VFIO-PCI) is out of scope for homelab use cases
- Shared mode is simpler and sufficient for media servers, light compute
- No IOMMU group management needed
- Works with all GPU vendors

**NVIDIA considerations:**
- Driver version in container MUST match host exactly
- No driver virtualization (unlike AMD/Intel)
- Requires proprietary userspace libraries

**AMD ROCm support:**
- `/dev/kfd` (Kernel Fusion Driver) bind-mounted for compute workloads
- Enables ROCm runtime in containers for GPU compute

**ARM GPU support:**
- Mali (`/dev/mali*`) and Adreno (`/dev/kgsl-3d0`) detected
- Limited 3D support in containers
- Proprietary blobs may have licensing restrictions

### 16. Snapshot Support

Snapshots provide point-in-time copies of machine rootfs for backup, testing, and rollback.

**Implementation:**
- `internal/snapshot/detector.go` - Detects filesystem type and selects appropriate backend
- `internal/snapshot/btrfs.go` - btrfs subvolume snapshot operations
- `internal/snapshot/zfs.go` - ZFS snapshot operations (disabled, untested)
- `internal/snapshot/lvm.go` - LVM snapshot operations (disabled, untested)
- `internal/snapshot/manager.go` - Orchestrates snapshot operations and state tracking
- `internal/machine/lvm.go` - LVM volume creation, formatting, and mounting

**Current status:** Only btrfs backend is supported. LVM and ZFS backends are disabled pending proper testing infrastructure.

**Why only btrfs?**
- btrfs provides instant, space-efficient copy-on-write snapshots
- Other backends (directory mode with rsync) were unreliable and slow
- Proper testing requires SSD-backed VMs (current test environment uses HDD RAID)
- LVM and ZFS support deferred to after Phase 5 (Cockpit Web UI)

**Backend selection:**
- btrfs: Used when machine rootfs is on btrfs filesystem
- Other filesystems: Snapshots not supported (error returned)

**Btrfs implementation:**
- Machine rootfs automatically created as btrfs subvolume when host is btrfs
- Snapshots created with `btrfs subvolume snapshot -r` (read-only)
- Snapshots stored in `<machine>.snapshots/<name>` (sibling directory, not child of rootfs)
- Rollback: uses safe rename-aside pattern - renames old rootfs to `.old`, moves snapshot copy in, deletes `.old` only on success. Snapshots preserved since they're in a separate directory.
- Delete: recursively deletes child subvolumes (deepest first) before deleting parent
- List: parses `btrfs subvolume list -o` output

**State tracking:**
- Snapshot metadata stored in `/var/lib/lxspawn/<name>/snapshots.json`
- Tracks snapshot name, backend, creation time, and location

### 17. Docker-in-Container Support

Running Docker inside lxspawn containers requires special configuration to work around systemd-nspawn limitations.

**Implementation:**
- `internal/compiler/slice.go` - Generates special drop-in when `seccomp: false`
- `internal/provision/exec.go` - Adds D-Bus environment variables to systemd-run commands
- `examples/templates/docker-host.yaml` - Reference template for Docker-in-container

**Key requirements:**

1. **Disable seccomp filtering** (`security.seccomp: false`)
   - Allows Docker's syscalls (mount, unshare, etc.)
   - Triggers automatic `Delegate=yes` in drop-in for cgroup delegation
   - Triggers `--private-users=no` in ExecStart to allow nested namespaces

2. **Add required capabilities**
   - `CAP_BPF` - Required for Docker's cgroup device control via BPF
   - `CAP_SYS_ADMIN` - Required for namespace operations
   - `CAP_NET_ADMIN`, `CAP_NET_RAW` - Required for Docker networking

3. **Set unlimited tasks** (`resources.tasks: infinity`)
   - Docker needs unlimited processes/threads

4. **Configure Docker to use cgroupfs driver**
   - systemd-nspawn doesn't expose the host's systemd to the container
   - Docker must use `cgroupfs` instead of `systemd` cgroup driver
   - Configured via `/etc/docker/daemon.json`

5. **Enable IP forwarding in pre_start hook**
   - Required for Docker's bridge networking
   - `echo 1 > /proc/sys/net/ipv4/ip_forward`
   - `modprobe br_netfilter`

**D-Bus environment for systemd-run:**

Post-boot provisioning uses `systemd-run -M` to execute commands inside the container. For `systemctl` commands to work, the following environment variables must be set:
- `DBUS_SYSTEM_BUS_ADDRESS=unix:path=/run/dbus/system_bus_socket`
- `XDG_RUNTIME_DIR=/run/user/0`

However, even with these variables, `systemctl start/stop/restart` doesn't work reliably in systemd-run context. The workaround is to:
- Use `systemctl enable` (creates symlinks, doesn't need D-Bus)
- Start services manually in readiness probes if needed
- Or use boot scripts for services that must start automatically

**GPG key installation:**

When installing GPG keys in systemd-run context (no TTY), use:
```bash
gpg --batch --no-tty --dearmor -o /path/to/keyring.gpg /path/to/key.asc
```

The `--batch --no-tty` flags prevent GPG from trying to access `/dev/tty` which doesn't exist in systemd-run context.

### 18. User Namespaces and Idmapped Mounts

User namespaces and idmapped mounts solve the EOVERFLOW error that occurs when installing packages on btrfs filesystems.

**The Problem:**

On btrfs, inode numbers can be very large (64-bit). When systemd-nspawn uses mapped UIDs (the default `-U` flag), files extracted on the host have host UIDs, and chown operations inside the container fail with "Value too large for defined data type" (EOVERFLOW) because the UID translation breaks with large inode numbers.

**The Solution:**

Use `PrivateUsers=identity` (systemd 249+) which creates a user namespace with 1:1 UID mapping. This allows the kernel to use idmapped mounts to handle UID translation correctly, avoiding EOVERFLOW errors.

**Implementation:**

- `internal/kernel/systemd.go` - Detects systemd version to check if `PrivateUsers=identity` is supported (requires systemd 249+)
- `internal/kernel/version.go` - Detects kernel version to check if idmapped mounts are supported (requires kernel 5.12+)
- `internal/compiler/nspawn.go` - Automatically sets `PrivateUsers=` in the .nspawn file based on:
  1. Docker-in-container (`seccomp: false`): Uses `PrivateUsers=no` (no user namespace, required for nested containers)
  2. Macvlan/IPvlan networking: Uses `PrivateUsers=no` (avoids compatibility issues with macvlan/ipvlan)
  3. Other network modes: Uses `PrivateUsers=identity` on systemd 249+, falls back to no setting on older versions
- `internal/machine/manager.go` - Checks kernel version before creating machines on btrfs (requires 5.12+)

**Why not use `-U` flag in ExecStart override?**

The previous approach used an ExecStart override in the drop-in to add `-U` or `--private-users=no` flags. This caused crashes with macvlan networking because the ExecStart override replaced the entire systemd-nspawn command line, conflicting with systemd's Type=notify handling. Moving `PrivateUsers=` to the .nspawn file avoids this issue.

**Fallback behavior:**

- On systemd < 249: `PrivateUsers=identity` is not supported, so no `PrivateUsers` setting is written. Containers use the default mapped UIDs, which may cause EOVERFLOW on btrfs.
- On kernel < 5.12: Idmapped mounts are not supported. A warning is logged, but machine creation continues. Package installation may fail with EOVERFLOW on btrfs.

### 19. Hybrid Storage Backend System

lxspawn uses a hybrid storage backend system that automatically selects the best storage method based on the host filesystem, following Proxmox's approach for LVM.

**The Problem:**

The original LVM implementation used thick logical volumes (10GB per machine), which wasted space and required the volume group to have free space before creating machines. This didn't scale well and was inefficient.

**The Solution:**

Implement a storage backend abstraction layer that automatically selects the best backend:

1. **Btrfs Backend** (when `/var/lib/machines/` is on btrfs)
   - Uses btrfs subvolumes for machine storage
   - Uses btrfs snapshots for efficient copy-on-write snapshots
   - Best performance on btrfs filesystems

2. **LVM Thin Backend** (when LVM thin pool exists)
   - Uses LVM thin provisioning (like Proxmox)
   - Creates thin volumes from a thin pool
   - Supports over-provisioning (volumes larger than available space)
   - Efficient thin snapshots with copy-on-write
   - Space is allocated on-demand, not upfront

3. **Directory Backend** (fallback)
   - Simple directory-based storage in `/var/lib/machines/<name>/`
   - Works on any filesystem (ext4, xfs, etc.)
   - Uses file copying for snapshots (less efficient but universal)
   - Used when no advanced backend is available

**Implementation:**

- `internal/storage/backend.go` - Defines `StorageBackend` interface
- `internal/storage/btrfs.go` - Btrfs subvolume backend
- `internal/storage/lvm_thin.go` - LVM thin provisioning backend
- `internal/storage/directory.go` - Directory-based fallback backend
- `internal/storage/detector.go` - Automatic backend detection and selection
- `internal/machine/manager.go` - Integrates storage backend

**Backend Selection Logic:**

```
1. Detect filesystem type of /var/lib/machines/
   ↓
2. If btrfs → Use BtrfsBackend
   ↓
3. If LVM:
   a. Check for thin pool (lvs --noheadings -o lv_attr)
   b. If thin pool exists → Use LVMThinBackend
   c. If no thin pool → Use DirectoryBackend (with warning)
   ↓
4. Otherwise → Use DirectoryBackend
```

**LVM Thin Pool Setup:**

If no thin pool is found on LVM, lxspawn warns the user and suggests creating one:

```bash
# Create a 50GB thin pool
sudo lvcreate -L 50G -n lxspawn-pool <vg-name>
sudo lvconvert --type thin-pool <vg-name>/lxspawn-pool
```

**Why This Approach:**

- **Efficient space usage:** Thin volumes allocated on-demand
- **Universal compatibility:** Works on any filesystem with graceful fallback
- **Best performance:** Uses native filesystem features when available
- **Proxmox-like:** Follows industry best practices for virtualization storage
- **Clean separation:** Storage backend is independent of machine lifecycle

### 20. Runtime Stats Implementation

The `lxspawn stats` command reads runtime CPU, memory, and disk statistics from cgroup v2 files.

**Cgroup Path:**

The cgroup path is dynamically determined by reading the leader process's cgroup from `/proc/<pid>/cgroup`. This is necessary because systemd creates hierarchical slice structures for machine names with hyphens.

For example:
- Machine `testbox` → `/sys/fs/cgroup/machine.slice/machine-testbox.slice/`
- Machine `test-stats` → `/sys/fs/cgroup/machine.slice/machine-test.slice/machine-test-stats.slice/`
- Machine `docker-host` → `/sys/fs/cgroup/machine.slice/machine-docker.slice/machine-docker-host.slice/`

The `getMachineCgroupPath()` function:
1. Gets the leader PID via D-Bus (`org.freedesktop.machine1.Machine.Leader`)
2. Reads `/proc/<pid>/cgroup` to get the actual cgroup path
3. Extracts the slice path (removing the service and init.scope suffixes)
4. Returns the full path: `/sys/fs/cgroup` + cgroup path

Note: The path uses `.slice` not `.scope`. The `.slice` directory contains resource stats for the entire machine.

**CPU Usage:**

CPU usage is calculated by sampling `cpu.stat` twice with a 100ms interval:
1. Read `usage_usec` at time T1
2. Wait 100ms
3. Read `usage_usec` at time T2
4. Calculate: `(T2 - T1) / 100000 * 100` to get percentage

The cgroup files are world-readable (mode `r--r--r--`), so no sudo is required.

**Memory Usage:**

Memory stats are read from:
- `memory.current` - Current memory usage in bytes
- `memory.max` - Memory limit in bytes (or "max" for unlimited)

**Disk Usage:**

Disk stats are read using `syscall.Statfs()` which returns filesystem usage in bytes. This is more efficient and reliable than shelling out to `df -B1`.

**Permissions:**

All cgroup files are world-readable, so the stats command works without sudo or special permissions.

## Security Model

### Privilege Model

**Important:** Membership in the `lxspawn` group is equivalent to root access. This is by design and matches the security model of `docker`, `libvirt`, and `lxc` groups.

After `lxspawn setup`, users in the `lxspawn` group can:
- Create/start/stop/delete machines without sudo
- Manage only `machine-*.slice` and `systemd-nspawn@*.service` units (validated by polkit regex)
- Reload the systemd daemon (required for `lxspawn start`)
- Execute privileged operations via specific sudoers rules with regex argument restrictions

**Why root-equivalent?**

The setup process creates group-writable directories (`/etc/systemd/nspawn`, `/etc/systemd/system/systemd-nspawn@.service.d`, `/var/lib/lxspawn/<name>/.hooks`) to allow machine management without sudo. A group member can write malicious systemd unit files or hook scripts that execute as root. This is the same trade-off made by Docker, libvirt, and LXC for usability.

**Mitigations:**
- Sudoers rules use regex (sudo 1.9.10+) to restrict arguments to valid machine names
- Polkit rules restrict unit management to lxspawn machine patterns
- Hook scripts are validated and SELinux-labeled
- Only trusted users should be added to the `lxspawn` group

### Security Features

**Container Isolation:**
- `NoNewPrivileges=yes` (default) - Prevents privilege escalation
- `SystemCallFilter` - Seccomp filtering (enabled by default)
- Capability dropping - Dangerous capabilities dropped by default
- Path validation - Volumes cannot expose sensitive system paths
- Symlink protection - Tar extraction validates symlink targets

**File Security:**
- State files (`machines.json`, `snapshots.json`) stored with `0600` permissions (owner-only read/write)
- Atomic file writes prevent corruption
- SELinux context automatically applied to hook scripts

**Network Security:**
- HTTPS-only downloads - Image downloads and file references require HTTPS
- Download size limits - Prevents disk exhaustion attacks (2GB max for images)
- Digest verification - File references support SHA256 digest validation
- Port validation - Port forwarding restricted to valid ranges (1-65535)

**Sudoers Hardening:**
- Regex argument matching (sudo 1.9.10+) - Machine names validated by pattern
- Path restrictions - All operations scoped to `/var/lib/machines/*`
- Command-specific rules - Each operation has explicit sudoers rule
- visudo validation - Sudoers file syntax-checked before installation

### Sudo Implementation

lxspawn uses direct `sudo` calls with specific sudoers rules instead of a privileged helper binary. This approach provides better security through explicit rule definitions.

**Sudoers Rules:**

The `lxspawn setup` command generates specific sudoers rules in `/etc/sudoers.d/lxspawn` for each required operation:

- **Core operations**: `rm`, `chown`, `chmod`, `mv`, `mkdir`, `systemd-run`, `journalctl`
- **btrfs operations**: `btrfs subvolume create/delete/show/list/snapshot`, `btrfs filesystem usage`
- **LVM storage operations**: `lvs`, `lvcreate`, `lvremove`, `mkfs.ext4`, `mount`, `umount` (snapshots disabled)

Note: ZFS and LVM snapshot rules are not generated since these backends are disabled pending testing infrastructure.

Each rule is scoped to specific paths and arguments. For example:
```
%lxspawn ALL=(root) NOPASSWD: /usr/sbin/btrfs subvolume delete /var/lib/machines/*
%lxspawn ALL=(root) NOPASSWD: /usr/sbin/btrfs subvolume delete /var/lib/machines/*/.snapshots/*
```

**Implementation:**

The `deploy.Sudo()` function wraps `exec.Command("sudo", args...)` to execute privileged operations. All sudo calls go through this function, making it easy to audit and test.

**Security Benefits:**

1. **Explicit rules**: Each operation has a specific sudoers rule with path and argument restrictions
2. **No helper binary**: Reduces attack surface by eliminating an additional binary
3. **Standard approach**: Uses well-understood sudo mechanisms instead of custom validation logic
4. **Easy to audit**: Sudoers rules are human-readable and can be reviewed independently

**Path Handling:**

Sudoers wildcards (`*`) don't match `/`, so we need multiple rules for different path depths. For example, to delete both machine rootfs and snapshots:
```
%lxspawn ALL=(root) NOPASSWD: /usr/sbin/btrfs subvolume delete /var/lib/machines/*
%lxspawn ALL=(root) NOPASSWD: /usr/sbin/btrfs subvolume delete /var/lib/machines/*/.snapshots/*
```

## Performance Characteristics

### Boot Time

- **Typical:** 1-3 seconds
- **Bottleneck:** systemd init inside container
- **Optimization:** Mask unnecessary services (systemd-udevd, systemd-modules-load)

### Memory Overhead

- **Base:** ~50MB (systemd + minimal services)
- **With services:** Depends on workload
- **Optimization:** Use `memory_high` for soft limits (throttle before OOM)

### Disk Usage

- **Rootfs:** ~200MB (Debian), ~500MB (Ubuntu), ~1GB (Fedora)
- **Metadata:** ~1KB per machine (machines.json)
- **Generated files:** ~5KB per machine (.nspawn, slice, drop-in)

### CPU Overhead

- **Idle:** <1% (systemd waiting for events)
- **Under load:** Native performance (no virtualization overhead)
- **Optimization:** Use `cpus` for hard limits, `CPUWeight` for soft priorities

## Testing Strategy

### Unit Tests

**320 tests across 22 test files, all passing (1 skipped, requires root):**

- `internal/config/parser_test.go` - Config parsing and validation (64 tests)
- `internal/compiler/compiler_test.go` - Compiler output for .nspawn, slice, drop-in, hooks (27 tests)
- `internal/deploy/state_test.go` - State management: register, unregister, persistence, list sorting (18 tests)
- `internal/deploy/filesystem_test.go` - Path helpers, WriteFileAtomic, RemoveAll (21 tests)
- `internal/provision/template_test.go` - Template rendering with all variables, params, edge cases (20 tests)
- `internal/provision/yq_test.go` - YQ provisioner: expression parsing, format detection, value types (20 tests)
- `internal/provision/boot_test.go` - Boot provisioner: script writing, service install, numbering (8 tests)
- `internal/provision/data_test.go` - Data provisioner: file writing, overwrite control, permissions, path validation (18 tests)
- `internal/provision/file_test.go` - File handler: digest validation, local file resolution (12 tests)
- `internal/provision/exec_test.go` - Exec provisioner: env export, provisioner modes, param handling (13 tests)
- `internal/image/fetcher_test.go` - Image fetcher: URL building, tar extraction, path traversal (8 tests)
- `internal/image/cleanup_test.go` - Rootfs cleanup: machine-id, resolv.conf, securetty (7 tests)
- `internal/image/validate_test.go` - Init system detection at multiple paths (7 tests)
- `internal/image/network_test.go` - Network config injection for all modes (10 tests)
- `internal/gpu/detect_test.go` - GPU detection, vendor identification, device selection (10 tests)
- `internal/machine/manager_test.go` - Shell quoting, command building, rootfs population check, ownership fix, stats functions (25 tests)
- `internal/machine/lvm_test.go` - LVM helper functions: isMounted, IsLVM, getVGName (9 tests)
- `internal/snapshot/snapshot_test.go` - Snapshot validation, formatting, types (10 tests)
- `internal/snapshot/detector_test.go` - Backend creation, filesystem detection (10 tests)
- `internal/snapshot/manager_test.go` - Snapshot state serialization, JSON fields (5 tests)
- `internal/kernel/version_test.go` - Kernel version detection, btrfs detection (5 tests)
- `internal/kernel/systemd_test.go` - Systemd version detection, PrivateUsers=identity support (2 tests)
- `internal/provision/engine_test.go` - Provision engine: GetState, state serialization (2 tests)

### Integration Tests

- Full lifecycle (create → start → shell → stop → delete)
- Networking modes (host, private, bridge, zone)
- Resource limits (verify cgroup values)
- Volume mounts (read-only, read-write, tmpfs)
- Host-side hooks (pre_start, post_start, post_stop)
- Snapshot lifecycle (create → list → rollback → delete)
- Snapshot rollback preserves pre-snapshot state
- Machine delete with snapshots (recursive subvolume cleanup)

### Manual Testing

- Test on multiple distros (Fedora, Ubuntu, Arch)
- Test with different images (Debian, Ubuntu, Fedora, Arch)
- Test edge cases (large rootfs, many machines, concurrent operations)
- GPU pass-through tested on Intel iGPU (Gemini Lake UHD Graphics 600)

## Troubleshooting

### Common Issues

**"machine already exists"**
- Machine with this name already in state
- Run `lxspawn delete <name>` to clean up

**"failed to start unit"**
- Check `journalctl -u systemd-nspawn@<name>.service`
- Verify .nspawn file is valid: `systemd-analyze verify /etc/systemd/nspawn/<name>.nspawn`

**"no init system found"**
- Image doesn't contain systemd
- Use systemd-based distro (Debian, Ubuntu, Fedora, Arch)

**"permission denied"**
- User not in `lxspawn` group
- Run `sudo usermod -aG lxspawn $USER` and log out/in

**"bridge not found"**
- Bridge doesn't exist on host
- Create bridge first: `ip link add name br0 type bridge`

### Debug Mode

```bash
lxspawn --verbose launch myserver.yaml
```

Shows:
- YAML parsing details
- Generated file contents
- systemd D-Bus calls
- Image download progress

## References

- [systemd.nspawn(5)](https://man7.org/linux/man-pages/man5/systemd.nspawn.5.html)
- [systemd-nspawn(1)](https://man7.org/linux/man-pages/man1/systemd-nspawn.1.html)
- [machinectl(1)](https://www.freedesktop.org/software/systemd/man/machinectl.html)
- [systemd.resource-control(5)](https://www.freedesktop.org/software/systemd/man/systemd.resource-control.html)
- [org.freedesktop.machine1](https://www.freedesktop.org/software/systemd/man/org.freedesktop.machine1.html)
