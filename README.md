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
git clone https://github.com/Inoriol/lxspawn
cd lxspawn
make build
sudo make install
```

### First-Time Setup

```bash
sudo lxspawn setup
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

See [SPEC.md](SPEC.md) for the complete YAML schema, all networking modes, provisioning, GPU pass-through, and more examples.

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

Snapshots require btrfs filesystem.

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
lxspawn edit-resources <name> --cpus 4 --memory 4G --memory-high 3G --io-weight 200
lxspawn edit-network <name> --mode bridge --bridge br0
lxspawn add-volume <name> --host /mnt/data --mount /data
lxspawn edit-volume <name> --mount /data --host /new/path
lxspawn remove-volume <name> --mount /data
```

All commands support `--json` for JSON output, `-v/--verbose` for detailed information, and `-q/--quiet` to suppress non-error output.

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
- [CHANGELOG.md](CHANGELOG.md) - Release history

## License

MIT

## Credits

Inspired by:
- [Lima](https://github.com/lima-vm/lima) - YAML format and provisioning model
- [Jailmaker](https://github.com/Jip-Hop/jailmaker) - Host-side hooks, seccomp control
- [cockpit-machines](https://github.com/cockpit-project/cockpit-machines) - UI inspiration
- [comquad](https://github.com/Inoriol/comquad) - Reusable code patterns
