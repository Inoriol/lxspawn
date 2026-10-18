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
│   ├── stop.go               # Stop machine
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
│   └── delete_snapshot.go    # Delete snapshot
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
│   │   ├── exec.go           # system/user modes (nsenter)
│   │   ├── file.go           # File reference handling
│   │   └── probe.go          # Readiness probes
│   │
│   ├── image/                # Rootfs image management
│   │   ├── fetcher.go        # Download and extract images
│   │   ├── xz.go             # XZ decompression wrapper
│   │   ├── validate.go       # Validate init system presence
│   │   ├── cleanup.go        # Remove machine-id, resolv.conf, write securetty
│   │   └── network.go        # Inject systemd-networkd configs
│   │
│   ├── machine/              # Machine lifecycle management
│   │   └── manager.go        # Create, start, stop, delete, shell, exec, status
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
9. state.RegisterMachine() saves metadata to ~/.local/share/lxspawn/machines.json
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

State is filesystem-based with a JSON metadata file:

```
~/.local/share/lxspawn/machines.json
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

**Why not a database?**
- Simpler to debug (just `cat` the file)
- Git-trackable (if desired)
- No additional dependencies
- Atomic writes via temp file + rename
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
│                   (mask services, write drop-ins, etc.)     │
│   2. data   → Write config files into rootfs                │
│                   (clean file creation, no shell needed)    │
│   3. yq     → Edit existing config files in rootfs          │
│                   (structured editing of JSON/YAML/TOML)    │
└──────────────────────────┬──────────────────────────────────┘
                           │ machinectl start <name>
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    POST-BOOT PHASE                          │
│   Container IS running. Exec via nsenter.                   │
│                                                             │
│   4. system → sudo nsenter -t <leader-pid> -m -u -i -n -p \ │
│              -- /bin/bash -c "<script>"                     │
│                   (run as root inside container)            │
│   5. user   → sudo nsenter -t <leader-pid> -m -u -i -n -p \ │
│              -- /bin/bash -c "su <user> -c '<script>'"      │
│                   (run as unprivileged user inside)         │
│                                                             │
│   Note: Post-boot commands use nsenter to enter the         │
│   container's namespaces directly, avoiding PTY allocation  │
│   issues that occurred with machinectl shell.               │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    READINESS PHASE                          │
│   Check if container is actually ready.                     │
│                                                             │
│   6. probes → Exec readiness scripts via nsenter            │
│                   (wait for services to be available)       │
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
- `boot` mode: scripts re-run on each boot by design
- Script hashes are tracked to detect changes

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

### 7. Why machinectl for Shell/Exec?

**machinectl advantages:**
- Native systemd tool (no additional dependencies)
- Works rootless (after setup)
- PTY allocation handled correctly
- Integrates with journal (logs visible on host)

**Trade-offs:**
- Slower than `nsenter` (but more secure)
- Requires machine to be running (can't exec into stopped container)

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
When falling back to macvlan, lxspawn creates a host-side macvlan interface (`mv-lxspawn` by default)
so the host can reach the container. This solves the kernel limitation where the host cannot ping
macvlan containers. The interface is created in `post_start` and removed in `post_stop`.

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
- Prioritizes discrete GPUs over integrated GPUs
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
- `internal/snapshot/lvm.go` - LVM snapshot operations
- `internal/snapshot/directory.go` - Directory-based snapshot operations (rsync)
- `internal/snapshot/manager.go` - Orchestrates snapshot operations and state tracking
- `internal/machine/lvm.go` - LVM volume creation, formatting, and mounting

**Current status:** btrfs, LVM, and directory backends are supported. ZFS backend is disabled pending testing.

**Backend selection:**
- btrfs: Used when machine rootfs is on btrfs filesystem
- LVM: Used when machine rootfs is on a dedicated LVM logical volume (thin pool)
- Directory: Used when machine rootfs is a regular directory (fallback for ext4, xfs, etc.)

**Btrfs implementation:**
- Machine rootfs automatically created as btrfs subvolume when host is btrfs
- Snapshots created with `btrfs subvolume snapshot -r` (read-only)
- Snapshots stored in `<rootfs>/.snapshots/<name>`
- Rollback: creates writable snapshot to sibling location, recursively deletes old rootfs tree (children first), renames snapshot to rootfs path
- Delete: recursively deletes child subvolumes (deepest first) before deleting parent
- List: parses `btrfs subvolume list -o` output

**LVM implementation:**
- Machine rootfs automatically created as a dedicated logical volume when host uses LVM thin pool
- LV naming: `<vg-name>/lxspawn-<machine-name>` (e.g., `debian-vg/lxspawn-myserver`)
- LV size: 4GB (hardcoded in `internal/machine/manager.go`)
- Formatted with ext4 and mounted at `/var/lib/machines/<name>/`
- Snapshots created with `lvcreate --snapshot --size 512M` (copy-on-write)
- Snapshot naming: `<lv-name>-snap-<snap-name>` (e.g., `lxspawn-myserver-snap-snap1`)
- Rollback: uses `lvconvert --merge` to merge snapshot back to origin (requires next activation)
- Delete: uses `lvremove -f` to remove snapshot LV
- List: parses `lvs --select vg_name=<vg>` output
- Machine deletion: unmounts LV, removes LV with `lvremove -f`

**LVM limitations:**
- Cannot create new snapshots while a merge is pending (LVM restriction)
- Merge completes on next activation (mount) of the origin LV
- Snapshot size is fixed at 512MB (may need adjustment for heavy write workloads)
- Machine LV size is fixed at 4GB (may need adjustment for large rootfs)

**Directory implementation:**
- Used when machine rootfs is a regular directory (no btrfs or LVM thin pool)
- Snapshots created with `rsync -a --delete --exclude=.snapshots` (preserves permissions, handles large files)
- Snapshots stored in `<rootfs>/.snapshots/<name>`
- Rollback: uses `rsync -a --delete --checksum --exclude=.snapshots` to restore from snapshot
- Delete: uses `sudo rm -rf` to remove snapshot directory
- List: reads `<rootfs>/.snapshots/` directory entries
- Uses `--checksum` flag to ensure files are compared by content, not just size/mtime

**Directory limitations:**
- Slower than btrfs or LVM snapshots (file-level copy instead of block-level)
- Requires more disk space (full copy instead of copy-on-write)
- No atomic snapshot creation (rsync is not atomic)
- Suitable for testing and small machines, not recommended for production with large rootfs

**State tracking:**
- Snapshot metadata stored in `~/.local/share/lxspawn/<name>/snapshots.json`
- Tracks snapshot name, backend, creation time, and location

**Sudoers requirements:**
- btrfs: `btrfs`, `chown`, `mv` commands
- LVM: `lvs`, `lvcreate`, `lvconvert`, `lvremove`, `mkfs.ext4`, `mount`, `umount`, `mountpoint`, `chown`, `chmod`
- Directory: `rsync`, `rm`, `findmnt` commands
- Added to `/etc/sudoers.d/lxspawn` by `lxspawn setup`

### 17. Docker-in-Container Support

Running Docker inside lxspawn containers requires special configuration to work around systemd-nspawn limitations.

**Implementation:**
- `internal/compiler/slice.go` - Generates special drop-in when `seccomp: false`
- `internal/provision/exec.go` - Adds D-Bus environment variables to nsenter commands
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

**D-Bus environment for nsenter:**

Post-boot provisioning uses `nsenter` to execute commands inside the container. For `systemctl` commands to work, the following environment variables must be set:
- `DBUS_SYSTEM_BUS_ADDRESS=unix:path=/run/dbus/system_bus_socket`
- `XDG_RUNTIME_DIR=/run/user/0`

However, even with these variables, `systemctl start/stop/restart` doesn't work reliably in nsenter context. The workaround is to:
- Use `systemctl enable` (creates symlinks, doesn't need D-Bus)
- Start services manually in readiness probes if needed
- Or use boot scripts for services that must start automatically

**GPG key installation:**

When installing GPG keys in nsenter context (no TTY), use:
```bash
gpg --batch --no-tty --dearmor -o /path/to/keyring.gpg /path/to/key.asc
```

The `--batch --no-tty` flags prevent GPG from trying to access `/dev/tty` which doesn't exist in nsenter context.

### 18. Adaptive Network Mode

The `adaptive` network mode automatically selects the best network mode based on host configuration.

**Implementation:**
- `internal/compiler/compiler.go:resolveNetworkMode()` - Resolves adaptive mode at compile time
- `internal/network/detect.go` - Provides interface detection and IP retrieval
- `internal/network/bridge.go` - Checks bridge existence
- `internal/network/config.go` - Stores detected interface in `~/.local/share/lxspawn/config.json`
- `internal/compiler/slice.go:generateHostAccessScript()` - Creates host-side macvlan in post_start hook

**Selection logic:**
1. If `bridge` is specified AND exists on host → use bridge mode
2. Otherwise → use macvlan mode with `host_access` enabled

**Host access for macvlan:**

When falling back to macvlan, lxspawn creates a host-side macvlan interface so the host can reach the container. This solves the kernel limitation where the host cannot ping macvlan containers.

**IP address generation:**

When `host_access.ip` is not specified, lxspawn automatically generates an IP address in the same subnet as the host interface:
1. Detects the host's primary interface IP and subnet (e.g., `192.168.0.200/24`)
2. Generates a deterministic IP using FNV hash of the machine name
3. Assigns the generated IP to the host-side macvlan interface (e.g., `192.168.0.151/24`)

This ensures the host-side interface is in the same subnet as the container, enabling bidirectional communication.

**Setup:**

Before using `adaptive` mode with `interface: auto`, run:
```bash
sudo lxspawn setup network
```

This detects the primary network interface and saves it for automatic use.

### 19. User Namespaces and Idmapped Mounts

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

**Sudoers requirements:**

The `chown` command is still required in sudoers for fixing ownership of btrfs subvolumes after creation (during image extraction), but the recursive `fixRootfsOwnership` function (which chowned the entire rootfs) has been removed since idmapped mounts handle UID translation at runtime.

### 20. Hybrid Storage Backend System

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

**Trade-offs:**

- More complex than single-backend approach (but hidden from users via auto-detection)
- Directory backend snapshots are less efficient (but it's a fallback)
- Requires LVM thin pool setup for best LVM experience (one-time)

### 21. Directory Backend Root-Owned File Handling

The directory backend uses `sudo rm -rf` instead of `os.RemoveAll` for machine and snapshot deletion.

**The Problem:**

When containers run with `PrivateUsers=identity` (the default for btrfs and host networking), files created inside the container are owned by root on the host. When the container is stopped and the user attempts to delete the machine, `os.RemoveAll` fails with "permission denied" because the user doesn't have permission to delete root-owned files.

**The Solution:**

Use `sudo rm -rf` for all deletion operations in the directory backend:
- `DeleteMachine` - Removes the machine directory
- `DeleteSnapshot` - Removes snapshot directories
- `RollbackSnapshot` - Removes the backup directory after rollback

**Implementation:**

- `internal/storage/directory.go:DeleteMachine` - Uses `sudo rm -rf <path>` instead of `os.RemoveAll`
- `internal/storage/directory.go:DeleteSnapshot` - Same approach for snapshot deletion
- `internal/storage/directory.go:RollbackSnapshot` - Uses `sudo rm -rf` for backup cleanup

**Sudoers requirements:**

The `rm` command is already allowed in sudoers for `/var/lib/machines/*`:
```
%lxspawn ALL=(root) NOPASSWD: /usr/bin/rm -rf /var/lib/machines/*
```

### 22. Rootfs Population Check

The machine creation flow checks whether the rootfs directory is populated before skipping image download.

**The Problem:**

When the storage backend creates an empty directory for the machine, `deploy.FileExists(rootfsPath)` returns `true` (the directory exists), causing the image fetch step to be skipped. This results in an empty rootfs that systemd-nspawn refuses to start ("os-release file is missing").

**The Solution:**

Add an `isPopulatedRootfs` check that verifies the presence of `/etc/os-release` in the rootfs directory:

```go
if !deploy.FileExists(rootfsPath) || !isPopulatedRootfs(rootfsPath) {
    // Download and extract image
}
```

**Implementation:**

- `internal/machine/manager.go:isPopulatedRootfs` - Checks for `/etc/os-release` in the rootfs
- This ensures that even if the directory exists but is empty, the image will be downloaded

**Why `/etc/os-release`:**

- Present in all modern Linux distributions
- Required by systemd-nspawn to validate the rootfs
- Reliable indicator that the image was successfully extracted

### 23. Rootfs Ownership Fix

The machine creation flow fixes rootfs ownership after pre-boot provisioning to ensure files have correct uid/gid.

**The Problem:**

When the storage backend creates a btrfs subvolume or LVM volume, it sets ownership to `root:lxspawn` with group-writable permissions. This allows the current user (in the `lxspawn` group) to extract the rootfs tarball and run pre-boot provisioning. However, files extracted from the tarball inherit the current user's UID (e.g., 1000) instead of root (0). This breaks operations inside the container that expect files to be owned by root, such as `sudo` commands.

**The Solution:**

After pre-boot provisioning completes, run `sudo chown -R 0:0` on the rootfs to fix ownership:

```go
func fixRootfsOwnership(rootfsPath string) error {
    cmd := exec.Command("sudo", "chown", "-R", "0:0", rootfsPath)
    if output, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("chown failed: %w (%s)", err, string(output))
    }
    return nil
}
```

**Implementation:**

- `internal/machine/manager.go:fixRootfsOwnership` - Recursively chowns rootfs to root:root
- Called in `Create()` after pre-boot provisioning completes
- Uses `sudo` because the user can't modify root-owned files

**Why After Pre-Boot Provisioning:**

- Pre-boot provisioning needs to write files to the rootfs (boot scripts, config files)
- The directory is group-writable, so the current user can write during provisioning
- After provisioning, all files (including those created by provisioning) need root ownership
- Running chown after provisioning ensures both extracted files and provisioned files have correct ownership

**Sudoers Requirements:**

The `chown` command is already allowed in sudoers for `/var/lib/machines/*`:
```
%lxspawn ALL=(root) NOPASSWD: /usr/bin/chown * /var/lib/machines/*
```

### 24. systemd-resolved DNS Handling

Fedora containers with systemd-resolved require special handling to configure DNS.

**The Problem:**

Fedora images include systemd-resolved, which manages `/etc/resolv.conf`. Even if you mask the service in boot scripts, systemd-resolved starts before boot scripts run and overwrites `/etc/resolv.conf` with its own configuration (pointing to 127.0.0.53). This breaks DNS resolution in containers.

**The Solution:**

Overwrite `/etc/resolv.conf` in the system provision script (post-boot), after systemd-resolved has started:

```yaml
provision:
  - mode: boot
    script: |
      #!/bin/bash
      systemctl mask systemd-resolved.service
      rm -f /etc/resolv.conf
      
  - mode: system
    script: |
      #!/bin/bash
      # Overwrite resolv.conf AFTER systemd-resolved has started
      cat > /etc/resolv.conf << 'EOF'
      nameserver 8.8.8.8
      nameserver 1.1.1.1
      nameserver 8.8.4.4
      EOF
```

**Why This Works:**

- Boot scripts run before systemd starts, so masking doesn't prevent systemd-resolved from starting
- systemd-resolved starts during boot and overwrites `/etc/resolv.conf`
- System provision runs after the container is fully booted, so systemd-resolved has already started
- Overwriting `/etc/resolv.conf` in system provision happens after systemd-resolved has written its config

**Alternative Approaches:**

1. **Configure systemd-resolved directly** (doesn't work in containers):
   - `resolvectl dns` requires D-Bus, which isn't available in nsenter context
   - `/etc/systemd/resolved.conf.d/` config is ignored if systemd-resolved isn't properly initialized

2. **Mask systemd-resolved in boot scripts** (doesn't work):
   - Boot scripts run before systemd, so masking doesn't prevent the service from starting
   - systemd-resolved starts during boot and overwrites the config

3. **Use data mode to write resolv.conf** (doesn't work):
   - Data mode runs during pre-boot, before systemd-resolved starts
   - systemd-resolved overwrites the file during boot

## Security Model

### Privilege Escalation Prevention

1. **NoNewPrivileges=yes** (default)
   - Prevents processes from gaining additional privileges
   - Cannot be disabled by container processes

2. **Capability Dropping**
   - Drops dangerous capabilities by default (CAP_SYS_RAWIO, CAP_MKNOD)
   - Users can add capabilities explicitly

3. **Seccomp Filtering** (planned)
   - Restrict syscalls available to container
   - Docker-in-container requires explicit opt-out

4. **User Namespaces** (planned)
   - Map container UIDs to unprivileged host UIDs
   - Prevents container root from being host root

### Sudoers Requirements

The `lxspawn setup` command creates sudoers rules for the `lxspawn` group:
- `/usr/bin/rm -rf /var/lib/machines/*` - For deleting machine rootfs
- `/usr/bin/nsenter` - For post-boot provisioning (executing scripts inside running containers)
- `/usr/bin/btrfs` - For btrfs subvolume snapshot operations
- `/usr/bin/zfs` - For ZFS snapshot operations
- `/usr/sbin/lvcreate`, `/usr/sbin/lvconvert`, `/usr/sbin/lvremove` - For LVM snapshot operations
- `/usr/bin/chown` - For fixing ownership of btrfs subvolume after creation
- `/usr/bin/mv` - For renaming subvolumes during rollback

These rules allow non-root users in the `lxspawn` group to perform these operations without a password prompt.

### Filesystem Isolation

1. **Read-only mounts** for sensitive paths
   - `/etc/ssl/certs` (read-only by default)
   - User can override with `readonly: false`

2. **Inaccessible paths**
   - `/etc/shadow`, `/etc/gshadow` hidden by default
   - User can add more with `security.inaccessible`

3. **Temporary filesystems**
   - `/tmp` can be tmpfs (not shared with host)
   - Prevents container from filling host `/tmp`

### Network Isolation

1. **Host mode** (default)
   - Shared network namespace
   - No isolation, but works rootless
   - Suitable for trusted containers

2. **Private mode**
   - Isolated network namespace
   - No LAN access
   - Requires root

3. **Bridge mode**
   - Attached to host bridge
   - LAN accessible
   - Requires root + pre-created bridge

4. **Zone mode**
   - Automatic bridge management
   - Zone-scoped access
   - Requires root

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

**411 tests across 21 test files, all passing:**

- `internal/config/parser_test.go` - Config parsing and validation (64 tests) - NEW: storage size parsing and validation
- `internal/compiler/compiler_test.go` - Compiler output for .nspawn, slice, drop-in, hooks (27 tests)
- `internal/deploy/state_test.go` - State management: register, unregister, persistence, list sorting (18 tests)
- `internal/deploy/filesystem_test.go` - Path helpers, WriteFileAtomic, RemoveAll (21 tests)
- `internal/provision/template_test.go` - Template rendering with all variables, params, edge cases (20 tests)
- `internal/provision/yq_test.go` - YQ provisioner: expression parsing, format detection, value types (20 tests)
- `internal/provision/boot_test.go` - Boot provisioner: script writing, service install, numbering (8 tests)
- `internal/provision/data_test.go` - Data provisioner: file writing, overwrite control, permissions, path validation (18 tests) - NEW: validatePath tests
- `internal/provision/file_test.go` - File handler: digest validation, local file resolution (12 tests)
- `internal/provision/exec_test.go` - Exec provisioner: env export, provisioner modes, param handling (13 tests)
- `internal/image/fetcher_test.go` - Image fetcher: URL building, tar extraction, path traversal (8 tests)
- `internal/image/cleanup_test.go` - Rootfs cleanup: machine-id, resolv.conf, securetty (7 tests)
- `internal/image/validate_test.go` - Init system detection at multiple paths (7 tests)
- `internal/image/network_test.go` - Network config injection for all modes (10 tests)
- `internal/gpu/detect_test.go` - GPU detection, vendor identification, device selection (10 tests)
- `internal/machine/manager_test.go` - Shell quoting, command building, rootfs population check, ownership fix (20 tests) - NEW: fixRootfsOwnership test
- `internal/machine/lvm_test.go` - LVM helper functions: isMounted, IsLVM, getVGName (9 tests)
- `internal/snapshot/snapshot_test.go` - Snapshot validation, formatting, types (10 tests)
- `internal/snapshot/detector_test.go` - Backend creation, filesystem detection (10 tests)
- `internal/snapshot/directory_test.go` - Directory backend operations (6 tests) - NEW: create, list, delete, rollback tests
- `internal/snapshot/manager_test.go` - Snapshot state serialization, JSON fields (5 tests)
- `internal/kernel/version_test.go` - Kernel version detection, btrfs detection (5 tests)
- `internal/kernel/systemd_test.go` - Systemd version detection, PrivateUsers=identity support (2 tests)

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

## Input Validation and Security Hardening

### 20. Path Traversal Protection

The `validatePath()` function in `internal/provision/data.go` prevents path traversal attacks in data and yq provisioners:

**Implementation:**
- Validates that paths are absolute
- Cleans the path to resolve `..` components
- Verifies the resolved path stays under the rootfs directory
- Returns an error if the path escapes the rootfs

**Why this matters:**
- Prevents malicious YAML configs from writing files outside the container rootfs
- Protects against attacks like `path: /etc/../../../etc/passwd`
- Applied to both `data` and `yq` provision modes

### 21. Command Injection Protection

Interface names in hook scripts are validated with regex `^[a-zA-Z0-9_-]+$`:

**Implementation:**
- `internal/compiler/slice.go` validates interface names before using them in shell commands
- Invalid interface names produce safe error messages instead of executing shell commands
- Prevents attacks like `interface: eth0; rm -rf /`

**Why this matters:**
- Hook scripts run as root on the host
- Unsanitized input could execute arbitrary commands
- Network interface names have a well-defined character set

### 22. Name Validation

Machine and snapshot names are validated with strict regex patterns:

**Machine names:** `^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]$`
- Must start and end with alphanumeric character
- Can contain hyphens and underscores in the middle
- Length: 1-64 characters
- Prevents dangerous characters: `;`, `|`, `&`, `$`, backticks

**Snapshot names:** `^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,62}[a-zA-Z0-9]$`
- Same as machine names, but also allows dots
- Dots useful for versioned snapshots (e.g., "snap.v1")

**Why this matters:**
- Names are used in file paths, systemd unit names, and D-Bus object paths
- Special characters could break systemd or cause security issues
- Consistent naming makes debugging easier

### 23. Volume Path Validation

Volume host paths are validated against a sensitive path blacklist:

**Blacklisted paths:**
- `/etc/shadow`, `/etc/gshadow`, `/etc/master.passwd`
- `/proc`, `/sys`, `/dev`, `/run`, `/var/run`

**Implementation:**
- `internal/config/parser.go:validateVolumes()` checks if host path matches or is under a blacklisted path
- Returns an error if a sensitive path is detected

**Why this matters:**
- Prevents accidental exposure of sensitive system files to containers
- Protects against misconfiguration that could compromise host security
- Containers should use bind mounts for specific data directories, not system directories

### 24. Port Range Validation

Port forwarding rules are validated for correct port ranges:

**Implementation:**
- `internal/config/parser.go:validatePort()` parses and validates port numbers
- Both host and container ports must be in range 1-65535
- Invalid ports like `tcp:99999:80` are rejected

**Why this matters:**
- Prevents invalid configurations that would fail at runtime
- Catches typos early (e.g., `tcp:8080:88888`)
- Provides clear error messages

### 25. GPU Device Path Validation

GPU device paths must start with `/dev/`:

**Implementation:**
- `internal/config/parser.go:validateGPU()` checks device path prefix
- Rejects paths like `/etc/passwd` or `/root/.ssh`

**Why this matters:**
- GPU devices are bind-mounted into the container
- Arbitrary host files could be exposed if validation is missing
- Device paths have a well-defined location (`/dev/`)

### 26. Edit Command Rollback Protection

The `lxspawn edit` command creates a backup before making changes:

**Implementation:**
- Rootfs is renamed to `<name>.backup-<timestamp>` before deletion
- If launch fails, the backup is automatically restored
- Backup is cleaned up on successful launch

**Why this matters:**
- Edit is a destructive operation (delete + launch)
- If launch fails, the machine would be permanently lost
- Rollback protection ensures recovery from failures

### 27. Resource Management

Snapshot managers are properly closed to prevent resource leaks:

**Implementation:**
- All snapshot commands use `defer mgr.Close()` after creating a manager
- `snapshot.Manager.Close()` method added for API consistency
- Matches the pattern used by `machine.Manager`

**Why this matters:**
- Prevents file descriptor leaks
- Ensures consistent resource management across the codebase
- Future-proofs for when managers hold actual resources

### 28. Machine Cleanup and Failed Service Handling

When machines are deleted or stopped, especially after crashes, proper cleanup is essential to prevent lingering network interfaces and systemd-machined registrations.

**The Problem:**
When a container crashes (e.g., SIGABRT, timeout), the systemd service enters a "failed" state. Without proper cleanup:
- The service remains in "failed" state indefinitely
- systemd-machined keeps the machine registered in memory (visible in `machinectl list`)
- Network interfaces (macvlan, host-side macvlan) may not be cleaned up
- The `post_stop` hook may not execute during crashes

**The Solution:**

The `Delete` and `Stop` methods now perform comprehensive cleanup:

1. **Stop the service** - Attempt graceful stop via D-Bus
2. **Reset failed state** - Call `ResetFailedUnit` to clear "failed" state
3. **Terminate via machinectl** - Call `machinectl terminate` to clean up systemd-machined registrations
4. **Remove files** - Delete .nspawn, slice, drop-in, hooks, and rootfs
5. **Reload systemd** - Ensure systemd picks up the changes
6. **Unregister from state** - Remove from machines.json

**Implementation:**

- `internal/deploy/dbus.go:ResetFailedUnit()` - Wraps D-Bus `ResetFailedUnit` call
- `internal/machine/manager.go:Delete()` - Comprehensive cleanup with error handling
- `internal/machine/manager.go:Stop()` - Falls back to `ResetFailedUnit` + `machinectl terminate` if stop fails

**Error handling:**
- All cleanup steps log warnings on failure but continue with remaining steps
- This ensures maximum cleanup even if some steps fail
- Errors are logged with context for debugging

**Why this matters:**
- Prevents lingering network interfaces on the host
- Prevents stale entries in `machinectl list`
- Ensures clean state for future machine creation
- Handles crash scenarios gracefully

## Future Enhancements

### Template Creation & Testing (Phase 4)

Turnkey templates for common use cases:
- Docker-in-container (reference: jailmaker)
- DNS servers (Pi-hole, AdGuard with macvlan/bridge)
- Media servers (Plex, Jellyfin with GPU pass-through)
- File servers (Nextcloud, Samba with permission handling)
- Base systems (Debian, Ubuntu, Fedora, etc.)

### Cockpit Integration (Phase 5)

Framework-agnostic plugin:
- Thin wrapper over `lxspawn --json` commands
- Leverage Cockpit's built-in terminal and journal
- Mirror cockpit-machines UX patterns

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
