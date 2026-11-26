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
lxspawn create [yaml]                    # Create machine (don't start)
lxspawn create --template NAME --name N  # Create from embedded template
lxspawn create -                         # Create from stdin
lxspawn launch [yaml]                    # Create + start machine
lxspawn launch --template NAME --name N  # Launch from embedded template
lxspawn start <name>                     # Start existing machine
lxspawn stop <name>                      # Stop machine (graceful)
lxspawn stop --force <name>              # Force stop (poweroff, terminate, SIGKILL)
lxspawn restart <name>                   # Restart machine (stop then start)
lxspawn delete <name>                    # Delete machine and all data
lxspawn provision <name>                 # Re-run provisioning (full cycle)
lxspawn edit <name>                      # Edit config and re-apply
lxspawn apply <name> <yaml>              # Apply config from file (preserves rootfs)
lxspawn autostart <name> on              # Enable machine autostart at boot
lxspawn autostart <name> off             # Disable machine autostart at boot
```

Template flags: `--template NAME` uses an embedded template, `--name N` overrides the machine name, `--param key=value` (repeatable) overrides template parameters. Add `--progress` to emit newline-delimited JSON progress events.

### Access

```bash
lxspawn shell <name>                     # Interactive shell
lxspawn exec <name> -- <cmd>             # Run command
lxspawn logs <name>                      # Follow logs (journalctl -M)
lxspawn logs <name> --since "1h ago"     # Logs since a date/time
lxspawn logs <name> --priority err       # Filter by priority level
lxspawn logs <name> --lines 50           # Last 50 lines
```

### Snapshots

```bash
lxspawn snapshot <name> <snap> [--description <desc>]  # Create snapshot
lxspawn list-snapshots <name>          # List snapshots
lxspawn rollback <name> <snap>         # Rollback to snapshot
lxspawn delete-snapshot <name> <snap>  # Delete snapshot
```

**Note:** Snapshots require btrfs filesystem. Other filesystems do not support snapshots.

### Status

```bash
lxspawn list                     # List all machines
lxspawn status <name>            # Detailed machine info
lxspawn show <name>              # Display generated config files
lxspawn stats [name]             # Runtime CPU/memory/disk stats
lxspawn stats --all              # Stats for all machines (efficient for UI polling)
lxspawn config <name>            # Show detailed configuration
lxspawn provision-status <name>  # Show provisioning status
```

### Information

```bash
lxspawn list-distros             # List available distributions
lxspawn list-templates           # List available machine templates
lxspawn show-template <name>     # Display embedded template content
lxspawn validate <yaml>          # Validate YAML configuration
lxspawn version                  # Show version and system info
lxspawn doctor                   # Check system configuration and dependencies
```

### Configuration Editing

```bash
lxspawn edit-resources <name> --cpus 4 --memory 4G --memory-high 3G --io-weight 200  # Edit resource limits
lxspawn edit-network <name> --mode bridge --bridge br0    # Edit network config
lxspawn add-volume <name> --host /mnt/data --mount /data  # Add bind mount
lxspawn edit-volume <name> --mount /data --host /new/path # Edit bind mount
lxspawn remove-volume <name> --mount /data                # Remove bind mount
```

All commands support `--json` for JSON output, `-v/--verbose` for detailed information, and `-q/--quiet` to suppress non-error output.

## Configuration Reference

See [SPEC.md](SPEC.md) for the complete template specification.

### Basic Structure

```yaml
name: myserver                   # Machine name (required)
distro: debian:bookworm          # Distro:release (required)

resources:                       # Resource limits (optional)
  cpus: 2                        # CPU cores
  memory: 2G                     # Hard memory limit
  memory_high: 1.5G              # Soft limit, throttle
  swap: 512M                     # Swap limit
  tasks: 512                     # Max processes
  io_weight: 100                 # IO priority

storage:                         # Storage configuration (optional)
  size: 10G                      # Volume size for LVM thin

network:                         # Network configuration (optional)
  mode: host                     # host | private | bridge | zone | macvlan | ipvlan | adaptive
  bridge: br0                    # Host bridge (if mode: bridge)
  zone: dev                      # Zone name (if mode: zone)
  interface: eth0                # Host interface (if mode: macvlan or ipvlan)
  ports:                         # Port forwarding (requires private network)
    - "tcp:8080:80"              # host:container

volumes:                         # Bind mounts (optional)
  - host: /mnt/data              # Host path (absolute)
    mount: /data                 # Container path (absolute)
    readonly: false              # Read-only mount
  - type: tmpfs                  # Temporary filesystem
    mount: /tmp
    options: "mode=1777"

security:                        # Security settings (optional)
  no_new_privileges: true        # Prevent privilege escalation
  seccomp: false                 # Disable seccomp filtering
  capabilities:                  # Add capabilities
    - CAP_NET_ADMIN
  drop_capabilities:             # Drop capabilities
    - CAP_SYS_RAWIO
  inaccessible:                  # Hide paths
    - /etc/shadow

gpu:                             # GPU pass-through (optional)
  enabled: true                  # Enable GPU access
  type: auto                     # auto | intel | amd | nvidia | mali | adreno

advanced:                        # Advanced options (optional)
  ephemeral: false               # Ephemeral machine (changes lost on stop)
  read_only: false               # Read-only rootfs
  hostname: myserver             # Container hostname
  environment:                   # Environment variables
    FOO: bar

journal:                         # Journal integration (optional)
  link: try-guest                # LinkJournal mode

hooks:                           # Host-side lifecycle hooks (optional)
  pre_start: |                   # Runs before container starts
    #!/bin/bash
    modprobe br_netfilter
  post_start: |                  # Runs after container starts
    ip link set dev ve-myserver master br0
  post_stop: |                   # Runs after container stops
    echo "Container stopped"

provision:                       # Provisioning (optional)
  - mode: boot                   # boot | data | yq | system | user
    script: |
      #!/bin/bash
      sysctl -w net.core.somaxconn=1024
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
- `adaptive` automatically selects bridge (if available) or macvlan with host access

### Provisioning

lxspawn supports Lima-inspired provisioning with five modes:

| Mode | Phase | Where it runs | Purpose |
|------|-------|---------------|---------|
| `boot` | Pre-boot | Host → rootfs | Early setup during boot (via oneshot systemd service on every boot) |
| `data` | Pre-boot | Host → rootfs | Write config files cleanly (no shell script needed) |
| `yq` | Pre-boot | Host → rootfs | Edit existing config files with expressions |
| `system` | Post-boot | `systemd-run -M` (root) | Package installation, service management |
| `user` | Post-boot | `systemd-run -M` (user) | User-level setup, dotfiles. Specify container user with `user:` field |

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

### Directory (Universal Fallback)
- **Detection:** Used when no advanced backend is available
- **Features:** Simple directory-based storage
- **Works on:** Any filesystem (ext4, xfs, etc.)
- **Limitations:** No snapshot support

## Security Model

**Important:** Membership in the `lxspawn` group is equivalent to root access. This is by design and matches the security model of `docker`, `libvirt`, and `lxc` groups.

lxspawn applies secure defaults:
- `NoNewPrivileges=yes` - Prevents privilege escalation
- `LinkJournal=try-guest` - Host can see container logs
- `DropCapability=CAP_SYS_RAWIO CAP_MKNOD` - Drops dangerous capabilities
- HTTPS-only downloads - Image downloads and file references require HTTPS
- Path validation - Volumes cannot expose sensitive system paths

Override in YAML with `security:` section.

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed security model and privilege information.

## Requirements

- **Host OS:** Linux with systemd 245+
- **Tools:** `systemd-nspawn`, `machinectl`, `xz` (for image extraction)
- **Permissions:** Root for setup, `lxspawn` group for management
- **SELinux:** If enforcing, hook scripts are automatically labeled with `bin_t` context
- **Images:** systemd-based distributions (Debian, Ubuntu, Fedora, etc.). Note: Arch Linux is not supported on btrfs hosts due to filesystem compatibility issues.

## Documentation

- [SPEC.md](SPEC.md) - Complete YAML template specification
- [ARCHITECTURE.md](ARCHITECTURE.md) - Internal architecture and design decisions
- [DEVELOPMENT.md](DEVELOPMENT.md) - Developer guide and contribution guidelines
- [TODO.md](TODO.md) - Roadmap and known issues

## License

MIT

## Credits

Inspired by:
- [Lima](https://github.com/lima-vm/lima) - YAML format and provisioning model
- [Jailmaker](https://github.com/Jip-Hop/jailmaker) - Host-side hooks, seccomp control
- [cockpit-machines](https://github.com/cockpit-project/cockpit-machines) - UI inspiration
- [comquad](https://github.com/Inoriol/comquad) - Reusable code patterns
