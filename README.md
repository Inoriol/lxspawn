# lxspawn

**LXC-style system containers powered by systemd-nspawn**

A daemonless, lightweight "Light VM" orchestrator for immutable desktop OSs and server homelabs. Fills the gap between Podman (app containers) and KVM/QEMU (full VMs).

## Philosophy

- **Zero persistent background daemons** - Everything runs through systemd
- **Zero complex state databases** - Filesystem-based state with JSON metadata
- **100% open-source** - No proprietary components
- **Declarative GitOps-first** - Define machines in YAML, track in git
- **"Lightweight VM" mental model** - Not "containers" - these are system containers with their own init, networking, and resources

## Quick Start

### Installation

```bash
# Build from source
git clone https://github.com/Inoriol/lxspawn
cd lxspawn
make build
sudo make install
```

### First-Time Setup

Configure group permissions and polkit rules (one-time, requires sudo):

```bash
sudo lxspawn setup
```

This creates:
- `lxspawn` group for passwordless management
- Directory permissions for `/var/lib/lxspawn`, `/etc/systemd/nspawn`, etc.
- Polkit rules for systemd unit management (scoped to lxspawn machines only)
- A single sudoers rule for `/usr/libexec/lxspawn/sudo-helper` — a privileged helper binary that validates all arguments before executing system commands (path validation, machine name validation, PID ownership checks)

Add your user to the group:

```bash
sudo usermod -aG lxspawn $USER
# Log out and back in for group changes to take effect
```

### Your First Machine

1. **Create a YAML config** (`myserver.yaml`):

```yaml
name: myserver
distro: debian:bookworm

resources:
  cpus: 2
  memory: 2G

network:
  mode: host

volumes:
  - host: /mnt/data
    mount: /data
```

2. **Launch the machine**:

```bash
lxspawn launch myserver.yaml
```

3. **Shell into it**:

```bash
lxspawn shell myserver
```

4. **Stop when done**:

```bash
lxspawn stop myserver
```

5. **Delete when no longer needed**:

```bash
lxspawn delete myserver
```

## CLI Commands

### Setup

```bash
lxspawn setup                    # First-run system configuration (requires sudo)
lxspawn setup network            # Detect and select network interface for containers
```

### Lifecycle

```bash
lxspawn create <yaml>            # Create machine (don't start)
lxspawn launch <yaml>            # Create + start machine
lxspawn start <name>             # Start existing machine
lxspawn stop <name>              # Stop machine (graceful)
lxspawn delete <name>            # Delete machine and all data
lxspawn provision <name>         # Re-run provisioning (full cycle)
lxspawn edit <name>              # Edit config and re-apply
```

### Access

```bash
lxspawn shell <name>             # Interactive shell
lxspawn exec <name> -- <cmd>     # Run command
lxspawn logs <name>              # Follow logs (journalctl -M)
```

### Snapshots

```bash
lxspawn snapshot <name> <snap>         # Create snapshot
lxspawn list-snapshots <name>          # List snapshots
lxspawn rollback <name> <snap>         # Rollback to snapshot
lxspawn delete-snapshot <name> <snap>  # Delete snapshot
```

Note: Snapshots require btrfs filesystem. 
- **btrfs**: Machine rootfs is automatically created as a btrfs subvolume when the host filesystem is btrfs. Snapshots use btrfs copy-on-write (instant, space-efficient).
- **Other filesystems**: Snapshots are not supported. Use btrfs for snapshot functionality.
- **LVM/ZFS**: Support planned for future release (currently disabled pending testing).

### Status

```bash
lxspawn list                     # List all machines
lxspawn list --json              # JSON output (for scripting/Cockpit)
lxspawn status <name>            # Detailed machine info
lxspawn status <name> --json     # JSON output
lxspawn show <name>              # Display generated config files
lxspawn stats <name>             # Runtime CPU/memory/disk stats
lxspawn stats <name> --json      # JSON output
lxspawn config <name>            # Show detailed configuration
lxspawn config <name> --json     # JSON output
lxspawn provision-status <name>  # Show provisioning status
lxspawn provision-status <name> --json  # JSON output
```

### Information

```bash
lxspawn list-distros             # List available distributions
lxspawn list-distros --json      # JSON output
lxspawn list-templates           # List available machine templates
lxspawn list-templates --json    # JSON output
lxspawn validate <yaml>          # Validate YAML configuration
lxspawn validate <yaml> --json   # JSON output
```

### Configuration Editing

```bash
lxspawn edit-resources <name> --cpus 4 --memory 4G        # Edit CPU/memory limits
lxspawn edit-network <name> --mode bridge --bridge br0    # Edit network config
lxspawn add-volume <name> --host /mnt/data --mount /data  # Add bind mount
lxspawn edit-volume <name> --mount /data --host /new/path # Edit bind mount
lxspawn remove-volume <name> --mount /data                # Remove bind mount
```

All edit commands support `--json` for JSON output.

### Global Flags

All commands support:
- `--json` - Output in JSON format
- `-v, --verbose` - Show detailed information
- `-q, --quiet` - Suppress non-error output

## Configuration Reference

See [SPEC.md](SPEC.md) for the complete template specification with a full example manifest and detailed field descriptions.

### Basic Structure

```yaml
name: myserver                   # Machine name (required)
distro: debian:bookworm          # Distro:release (required)

resources:                       # Resource limits (optional)
  cpus: 2                        # CPU cores (→ CPUQuota=200%)
  memory: 2G                     # Hard memory limit (→ MemoryMax=2G)
  memory_high: 1.5G              # Soft limit, throttle (→ MemoryHigh=1.5G)
  swap: 512M                     # Swap limit (→ MemorySwapMax=512M)
  tasks: 512                     # Max processes (→ TasksMax=512)
  io_weight: 100                 # IO priority (→ IOWeight=100)

storage:                         # Storage configuration (optional)
  size: 10G                      # Volume size for LVM thin (default: 10G)

network:                         # Network configuration (optional)
  mode: host                     # host | private | bridge | zone | macvlan | ipvlan
  bridge: br0                    # Host bridge (if mode: bridge)
  zone: dev                      # Zone name (if mode: zone)
  interface: eth0                # Host interface (if mode: macvlan or ipvlan)
  ports:                         # Port forwarding (requires private network)
    - "tcp:8080:80"              # host:container
    - "udp:5353:53"

volumes:                         # Bind mounts (optional)
  - host: /mnt/data              # Host path (absolute)
    mount: /data                 # Container path (absolute)
    readonly: false              # Read-only mount
  - type: tmpfs                  # Temporary filesystem
    mount: /tmp
    options: "mode=1777"

security:                        # Security settings (optional)
  no_new_privileges: true        # Prevent privilege escalation
  seccomp: false                 # Disable seccomp filtering (clears SystemCallFilter)
  capabilities:                  # Add capabilities
    - CAP_NET_ADMIN
    - CAP_NET_BIND_SERVICE
  drop_capabilities:             # Drop capabilities
    - CAP_SYS_RAWIO
    - CAP_MKNOD
  inaccessible:                  # Hide paths
    - /etc/shadow
    - /etc/gshadow

gpu:                             # GPU pass-through (optional)
  enabled: true                  # Enable GPU access
  type: auto                     # auto | intel | amd | nvidia | mali | adreno

advanced:                        # Advanced options (optional)
  ephemeral: false               # Ephemeral machine (changes lost on stop)
  read_only: false               # Read-only rootfs
  hostname: myserver             # Container hostname
  environment:                   # Environment variables
    FOO: bar
    BAZ: qux

journal:                         # Journal integration (optional)
  link: try-guest                # LinkJournal mode

hooks:                           # Host-side lifecycle hooks (optional)
  pre_start: |                   # Runs before container starts
    #!/bin/bash
    modprobe br_netfilter
  post_start: |                  # Runs after container starts
    #!/bin/bash
    ip link set dev ve-myserver master br0
  post_stop: |                   # Runs after container stops
    echo "Container stopped"

provision:                       # Provisioning (optional)
  - mode: boot                   # boot | data | yq | system | user
    script: |
      #!/bin/bash
      systemctl mask systemd-udevd
  - mode: data
    path: /etc/myapp/config.yaml
    content: |
      hostname: {{.Name}}
      port: {{.Param.port}}
  - mode: system
    script: |
      #!/bin/bash
      apt-get update && apt-get install -y nginx

param:                           # Custom parameters for templates
  port: "8080"
  env: "production"

probes:                          # Readiness probes (optional)
  - mode: readiness
    description: "nginx to be serving"
    script: |
      #!/bin/bash
      until curl -sf http://localhost/health; do sleep 2; done
```

### Provisioning

lxspawn supports Lima-inspired provisioning with five modes:

| Mode | Phase | Where it runs | Purpose |
|------|-------|---------------|---------|
| `boot` | Pre-boot | Host → rootfs | Early setup before systemd starts (mask services, write drop-ins) |
| `data` | Pre-boot | Host → rootfs | Write config files cleanly (no shell script needed) |
| `yq` | Pre-boot | Host → rootfs | Edit existing config files with expressions |
| `system` | Post-boot | `nsenter` (root) | Package installation, service management |
| `user` | Post-boot | `nsenter` (user) | User-level setup, dotfiles |

**Template variables** (available in all modes):

| Variable | Value | Example |
|----------|-------|---------|
| `{{.Name}}` | Machine name | `myserver` |
| `{{.Hostname}}` | Container hostname | `myserver` |
| `{{.Distro}}` | Distro:release | `debian:bookworm` |
| `{{.User}}` | Host username | `inoriol` |
| `{{.UID}}` | Host user ID | `1000` |
| `{{.Home}}` | Host home directory | `/home/inoriol` |
| `{{.Dir}}` | Directory containing YAML | `/home/inoriol/projects` |
| `{{.Param.Key}}` | Custom param value | (user-defined) |

**Provisioning flow:**

1. `lxspawn create` runs pre-boot provisioning (boot, data, yq)
2. `lxspawn launch` starts the machine
3. Post-boot provisioning (system, user) runs after container starts
4. Readiness probes verify services are ready

### Networking Modes

| Mode | Isolation | LAN Access | Root Required | Use Case |
|------|-----------|------------|---------------|----------|
| `host` | None | Yes (same IP) | No | Simple setups, shared network |
| `private` | Full | No | Yes | Isolated containers |
| `bridge` | Partial | Yes | Yes | LAN-accessible with isolation |
| `zone` | Partial | Zone only | Yes | Multiple isolated networks |
| `macvlan` | Partial | Yes | Yes | Container as separate LAN device |
| `ipvlan` | Partial | Yes | Yes | Container shares host MAC |
| `adaptive` | Varies | Varies | Varies | Automatic selection based on host config |

**Notes:**
- `host` mode works rootless (default)
- `bridge` requires pre-created bridge on host
- `macvlan` requires `interface:` field (e.g., `interface: eth0`)
- `ipvlan` requires `interface:` field (e.g., `interface: eth0`)
- `adaptive` automatically selects bridge (if available) or macvlan with host access
- Port forwarding requires private network mode

#### Adaptive Network Mode

The `adaptive` mode automatically selects the best networking option based on your host configuration:

```yaml
network:
  mode: adaptive
  bridge: br0        # Optional: preferred bridge if it exists
  interface: auto    # Optional: "auto" uses interface from `lxspawn setup network`
```

**Selection logic:**
1. If `bridge` is specified AND exists on host → uses **bridge** mode
2. Otherwise → uses **macvlan** mode with host-side interface for visibility

When falling back to macvlan, lxspawn automatically creates a host-side macvlan interface (`mv-lxspawn` by default) so the host can reach the container. This solves the kernel limitation where the host cannot ping macvlan containers.

**Setup:**
Before using `adaptive` mode with `interface: auto`, run:
```bash
sudo lxspawn setup network
```

This detects your primary network interface and saves it for automatic use.

#### Macvlan Host Access

When using macvlan mode (explicitly or via adaptive fallback), the host cannot reach the container due to a kernel limitation. lxspawn solves this by creating a host-side macvlan interface:

```yaml
network:
  mode: macvlan
  interface: enp2s0
  host_access:
    enabled: true           # Create host-side interface
    interface: mv-mgmt      # Name for host-side interface
    ip: 192.168.0.201/24    # Static IP (or omit for DHCP)
```

The host-side interface is created in the `post_start` hook and removed in `post_stop`. You can now ping and access the container from the host.

### Security Defaults

lxspawn applies secure defaults:
- `NoNewPrivileges=yes` - Prevents privilege escalation
- `LinkJournal=try-guest` - Host can see container logs
- `DropCapability=CAP_SYS_RAWIO CAP_MKNOD` - Drops dangerous capabilities

Override in YAML with `security:` section.

### GPU Pass-Through

lxspawn supports GPU pass-through for hardware acceleration in containers (media transcoding, compute workloads, 3D rendering).

**Supported vendors:**
- **Intel** (iGPU and discrete Arc) - VA-API, OpenCL
- **AMD** (iGPU and discrete) - VA-API, ROCm
- **NVIDIA** (discrete only) - CUDA, NVENC/NVDEC
- **ARM Mali** - 3D rendering (limited)
- **Qualcomm Adreno** - 3D rendering (limited)

**Auto-detection priority:**
1. Discrete GPUs (PCIe, dedicated VRAM)
2. Integrated GPUs (iGPU)
3. First detected GPU

**Example:**

```yaml
gpu:
  enabled: true
  type: auto  # auto-detect and prefer discrete GPU
```

**Container requirements:**
- GPU device nodes are bind-mounted automatically
- Cgroup `DeviceAllow` rules are added automatically
- Required capabilities (e.g., `CAP_SYS_ADMIN` for NVIDIA) are added automatically
- Userspace drivers must be installed in the container via provisioning

**NVIDIA-specific:** Driver version in container MUST match host exactly.

See [SPEC.md](SPEC.md) for full GPU configuration options.

## Examples

### Web Server with Resource Limits

```yaml
name: webserver
distro: ubuntu:noble

resources:
  cpus: 2
  memory: 1G
  tasks: 256

network:
  mode: bridge
  bridge: br0
  ports:
    - "tcp:80:80"
    - "tcp:443:443"

volumes:
  - host: /srv/www
    mount: /var/www/html
    readonly: true

security:
  capabilities:
    - CAP_NET_BIND_SERVICE
```

### Database Server

```yaml
name: database
distro: debian:bookworm

resources:
  cpus: 4
  memory: 8G
  memory_high: 6G
  swap: 2G

network:
  mode: host

volumes:
  - host: /mnt/db-data
    mount: /var/lib/postgresql

security:
  no_new_privileges: true
  drop_capabilities:
    - CAP_SYS_RAWIO
    - CAP_MKNOD
    - CAP_SYS_ADMIN
```

### Development Environment

```yaml
name: devbox
distro: debian:bookworm

resources:
  cpus: 4
  memory: 4G

network:
  mode: private

volumes:
  - host: /home/user/projects
    mount: /workspace
  - type: tmpfs
    mount: /tmp
    options: "mode=1777,size=2G"

advanced:
  hostname: devbox
  environment:
    EDITOR: vim
```

### DNS Server with Macvlan and Provisioning

```yaml
name: dns-server
distro: debian:bookworm

resources:
  cpus: 1
  memory: 512M

network:
  mode: macvlan
  interface: eth0

param:
  dns_port: "53"
  admin_email: "admin@example.com"

provision:
  - mode: data
    path: /etc/pihole/setupVars.conf
    content: |
      PIHOLE_INTERFACE=eth0
      IPV4_ADDRESS=192.168.1.100/24
      PIHOLE_DNS_1=8.8.8.8
      PIHOLE_DNS_2=8.8.4.4

  - mode: system
    script: |
      #!/bin/bash
      set -eux -o pipefail
      apt-get update
      apt-get install -y curl
      curl -sSL https://install.pi-hole.net | bash /dev/stdin --unattended

probes:
  - mode: readiness
    description: "Pi-hole to be serving DNS"
    script: |
      #!/bin/bash
      until dig @localhost example.com +short | grep -q .; do sleep 2; done
    hint: |
      Pi-hole may not have started. Check: journalctl -M {{.Name}} -u pihole-FTL
```

## How It Works

lxspawn generates systemd-native configuration files:

1. **YAML → .nspawn** - Container configuration (exec, files, network)
2. **YAML → slice** - Resource limits (CPU, memory, IO)
3. **YAML → drop-in** - Service overrides (slice assignment, hooks)
4. **Image fetch** - Downloads rootfs from `images.linuxcontainers.org`
5. **Pre-boot provisioning** - Writes boot scripts, config files, edits (boot, data, yq modes)
6. **systemd-nspawn@.service** - Manages lifecycle via systemd
7. **Post-boot provisioning** - Executes scripts inside container (system, user modes)
8. **Readiness probes** - Verifies services are ready

All state is filesystem-based:
- `/etc/systemd/nspawn/<name>.nspawn` - Container config
- `/etc/systemd/system/machine-<name>.slice` - Resource limits
- `/var/lib/machines/<name>/` - Container rootfs
- `/var/lib/lxspawn/<name>/` - Machine metadata and provision state
- `~/.local/share/lxspawn/machines.json` - Machine registry
- `~/.local/share/lxspawn/<name>/snapshots.json` - Snapshot metadata

## Privilege Model

After `lxspawn setup`, users in the `lxspawn` group can:
- Create/start/stop/delete machines without sudo
- Manage only `machine-*.slice` and `systemd-nspawn@*.service` units (validated by polkit regex)
- Reload the systemd daemon (required for `lxspawn start`)
- Cannot modify other systemd units
- Execute privileged operations only through the `sudo-helper` binary, which validates all arguments

### Sudo Helper

All privileged operations (storage management, provisioning, snapshots) go through a single privileged helper binary at `/usr/libexec/lxspawn/sudo-helper`. This binary validates:

- **Path restrictions**: All paths must be under `/var/lib/machines/`, `/var/lib/lxspawn/`, `/etc/systemd/nspawn/`, or `/etc/systemd/system/`
- **No path traversal**: Paths containing `..` are rejected
- **Machine name validation**: Names must match the strict regex `^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]$`
- **PID ownership**: `nsenter` calls verify the target PID belongs to an lxspawn machine (checks `/proc/<pid>/exe`, cmdline, and cgroup)
- **LVM name restrictions**: LV names must start with `lxspawn-`
- **Subcommand restrictions**: `btrfs` only allows `subvolume` and `filesystem usage`; `zfs` only allows `snapshot`, `rollback`, `destroy` on lxspawn datasets

Commands that require group membership:
- `create`, `launch`, `start`, `stop`, `delete`

Commands that work without group:
- `shell`, `exec`, `list`, `status`, `logs`

## Comparison

| Feature | lxspawn | Podman | KVM/QEMU |
|---------|---------|--------|----------|
| Boot time | ~2s | ~1s | ~30s |
| Memory overhead | ~50MB | ~30MB | ~512MB |
| Full init system | Yes | No | Yes |
| Systemd integration | Native | Via quadlet | Manual |
| Resource limits | Via slices | Via cgroups | Via libvirt |
| Networking | systemd-nspawn | CNI/Netavark | TAP/bridge |
| Use case | System containers | App containers | Full VMs |

## Requirements

- **Host OS:** Linux with systemd 245+
- **Tools:** `systemd-nspawn`, `machinectl`, `xz` (for image extraction)
- **Permissions:** Root for setup, `lxspawn` group for management
- **SELinux:** If enforcing, hook scripts are automatically labeled with `bin_t` context
- **Images:** systemd-based distributions (Debian, Ubuntu, Fedora, etc.). Note: Arch Linux is not supported on btrfs hosts due to filesystem compatibility issues.

## Storage Backends

lxspawn automatically detects and uses the best storage backend based on your filesystem:

### Btrfs (Recommended)
- **Detection:** Automatic when `/var/lib/machines/` is on btrfs
- **Features:** Subvolumes for machines, copy-on-write snapshots
- **Requirements:** Kernel 5.12+ for idmapped mounts support

### LVM Thin (Proxmox-style)
- **Detection:** Automatic when LVM thin pool exists in the volume group
- **Features:** Thin provisioning, over-provisioning, efficient snapshots
- **Setup:** Create a thin pool:
  ```bash
  sudo lvcreate -L 50G -n lxspawn-pool <vg-name>
  sudo lvconvert --type thin-pool <vg-name>/lxspawn-pool
  ```
- **Benefits:** Space-efficient, supports over-provisioning, fast snapshots

### Directory (Universal Fallback)
- **Detection:** Used when no advanced backend is available
- **Features:** Simple directory-based storage
- **Works on:** Any filesystem (ext4, xfs, etc.)
- **Limitations:** No snapshot support, less efficient than btrfs or LVM
- **Note:** Uses the sudo-helper for deletion (handles root-owned files from container namespace)

### Automatic Selection
lxspawn selects the backend automatically:
1. If btrfs → Btrfs backend
2. If LVM with thin pool → LVM thin backend
3. If LVM without thin pool → Directory backend (with warning)
4. Otherwise → Directory backend

The selected backend is logged at startup: `Using storage backend: <type>`

## Known Issues

### machinectl Shell PTY Errors

Intermittent "Failed to get shell PTY: Protocol error" or "There is no system bus in container" when using `lxspawn shell` or post-boot provisioning.

**Fixed:** Post-boot provisioning and readiness probes now use `nsenter` for non-interactive execution, which avoids PTY allocation issues entirely. Interactive shell (`lxspawn shell`) still uses `machinectl shell` with retry logic for reliability.

### Macvlan Host Visibility

Host cannot ping macvlan containers (expected limitation of macvlan). Container can communicate with other network devices on the LAN.

### Container Network Interface Naming

- macvlan creates `mv-<interface>` in container
- ipvlan creates `iv-<interface>` in container
- Network config must match these names

## Roadmap

**Phase 1 (V1) - Complete:**
- [x] Basic machine lifecycle (create, start, stop, delete)
- [x] Resource limits via systemd slices
- [x] All networking modes (host, private, bridge, zone, macvlan, ipvlan)
- [x] Bind mounts and volumes
- [x] JSON output for all commands
- [x] Polkit rules for passwordless management

**Phase 2 (V1.5) - Complete:**
- [x] Provisioning engine (boot, data, yq, system, user modes)
- [x] Template variables (`{{.Name}}`, `{{.Param.*}}`)
- [x] Readiness probes
- [x] Host-side lifecycle hooks (pre_start, post_start, post_stop)
- [x] `lxspawn provision` command for re-running provisioning (full cycle)
- [x] `lxspawn show` command to display generated config
- [x] `lxspawn edit` command to edit and re-apply config
- [x] Machine state queried via D-Bus (machine1 interface)
- [x] Lock file workaround (`SYSTEMD_NSPAWN_LOCK=0`)

**Phase 3 (V2) - Snapshots - Complete (btrfs only):**
- [x] Snapshot support (btrfs only; LVM/ZFS disabled pending testing)
- [x] Automatic btrfs subvolume creation
- [x] Snapshot create/list/rollback/delete commands
- [x] JSON output for snapshot commands

**Phase 4 (V2) - Templates - Complete:**
- [x] Docker-in-container template (fully working)
- [x] Base system templates (Debian, Ubuntu, Fedora)
- [x] Adaptive network mode with host-side macvlan
- [x] Docker-in-container support (Delegate, --private-users=no, CAP_BPF)
- [x] DNS server templates (pihole, adguard) - tested and working
- [x] Jellyfin template - GPU pass-through tested and working (Intel iGPU)
- [x] Plex template - GPU pass-through tested and working (Intel iGPU)
- [x] File server templates (samba, nextcloud) - tested and working
- [x] Arch Linux removed (btrfs/pacman incompatibility)
- [x] Jellyfin repository GPG key URL fixed

**Phase 4.5 (V2) - Code Quality & Security - Complete:**
- [x] Path traversal protection in data and yq provisioners
- [x] Command injection protection in hook scripts
- [x] Rollback protection for edit command
- [x] Strict machine and snapshot name validation
- [x] Volume path validation with sensitive path blacklist
- [x] Port range validation (1-65535)
- [x] GPU device path validation
- [x] Download timeout (30 minutes)
- [x] Secure cache directory permissions (0700)
- [x] Resource leak fixes (defer mgr.Close())

**Phase 5 (V2+) - Cockpit Web UI:**
- [ ] Cockpit web UI plugin

**Phase 6 (V2+) - ZFS/LVM Snapshots - DEFERRED:**
- [ ] Set up fast SSD-based test environment for LVM testing
- [ ] Test LVM thin pool snapshots (create, list, rollback, delete)
- [ ] Re-enable LVM backend
- [ ] Set up test environment with ZFS filesystem
- [ ] Test ZFS snapshots (create, list, rollback, delete)
- [ ] Re-enable ZFS backend

**Note:** Directory mode snapshots removed (unreliable, slow). Snapshots now require btrfs filesystem. LVM/ZFS support deferred to after Phase 5 (Cockpit Web UI).

## License

MIT

## Credits

Inspired by:
- [Lima](https://github.com/lima-vm/lima) - YAML format and provisioning model
- [Jailmaker](https://github.com/Jip-Hop/jailmaker) - Host-side hooks, seccomp control
- [cockpit-machines](https://github.com/cockpit-project/cockpit-machines) - UI inspiration
- [comquad](https://github.com/Inoriol/comquad) - Reusable code patterns
