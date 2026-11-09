# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-11-09

Initial public release of lxspawn - a daemonless, lightweight "Light VM" orchestrator for immutable desktop OSs and server homelabs.

### Added

#### Core Machine Lifecycle
- Machine creation, start, stop, delete commands
- Resource limits via systemd slices (CPU, memory, swap, tasks, IO weight)
- All networking modes: host, private, bridge, zone, macvlan, ipvlan, adaptive
- Bind mounts and tmpfs volumes
- JSON output for all commands (for scripting and automation)
- Polkit rules for passwordless management by `lxspawn` group

#### Provisioning Engine
- Lima-inspired provisioning with five modes:
  - `boot` - Pre-boot host-to-rootfs setup (mask services, write drop-ins)
  - `data` - Pre-boot config file creation
  - `yq` - Pre-boot config file editing with yq expressions
  - `system` - Post-boot root-level execution via nsenter
  - `user` - Post-boot user-level execution via nsenter
- Template variables (`{{.Name}}`, `{{.Param.*}}`, etc.)
- Readiness probes for service verification
- Host-side lifecycle hooks (pre_start, post_start, post_stop)

#### Storage Backends
- **Btrfs backend** - Subvolumes for machines, copy-on-write snapshots
- **LVM thin backend** - Thin provisioning with over-provisioning support
- **Directory backend** - Universal fallback for any filesystem
- Automatic backend detection and selection

#### Snapshots (btrfs only)
- Create, list, rollback, delete snapshots
- Automatic btrfs subvolume creation for machines
- JSON output for snapshot commands

#### Templates
- Base system templates: Debian, Ubuntu, Fedora
- Docker-in-container template (fully working)
- DNS server templates: Pi-hole, AdGuard Home
- Media server templates: Jellyfin, Plex (with GPU pass-through)
- File server templates: Samba, Nextcloud
- Adaptive network mode with automatic host-side macvlan interface

#### Security Features
- `NoNewPrivileges=yes` by default
- Capability management (add/drop)
- Seccomp filtering support
- Path validation for volumes (sensitive path blacklist)
- HTTPS-only downloads for file references
- Download size limits (2GB max)
- Digest verification for remote files
- State files with secure permissions (0600)
- SELinux context for hook scripts

#### Privilege Model
- Group-based access control (`lxspawn` group)
- Polkit rules scoped to lxspawn machines only
- Specific sudoers rules for each privileged operation
- Path and argument validation for all sudo commands

#### GPU Pass-Through
- Shared GPU access for hardware acceleration
- Auto-detection with discrete GPU preference
- Support for Intel, AMD, NVIDIA, ARM Mali, Qualcomm Adreno
- Automatic device node binding and cgroup rules

#### CLI Commands
- `lxspawn setup` - First-run system configuration
- `lxspawn setup network` - Network interface detection
- `lxspawn create/launch/start/stop/delete` - Machine lifecycle
- `lxspawn shell/exec` - Container access
- `lxspawn snapshot/list-snapshots/rollback/delete-snapshot` - Snapshot management
- `lxspawn edit/edit-resources/edit-network` - Configuration editing
- `lxspawn add-volume/edit-volume/remove-volume` - Volume management
- `lxspawn stats/status/config/provision-status` - Status and monitoring
- `lxspawn list-distros/list-templates/validate` - Information commands

#### Documentation
- Comprehensive README with quick start guide
- Complete YAML specification (SPEC.md)
- Architecture documentation (ARCHITECTURE.md)
- Example templates with detailed comments

### Security

- Path traversal protection in data and yq provisioners
- Command injection protection in hook scripts
- Rollback protection for edit command
- Strict machine and snapshot name validation
- Volume path validation with sensitive path blacklist
- Port range validation (1-65535)
- GPU device path validation
- Download timeout (30 minutes)
- Secure cache directory permissions (0700)
- Resource leak fixes (defer mgr.Close())
- Symlink path traversal protection in tar extraction
- HTTPS-only enforcement for file references
- Sudoers rules hardened with specific path and argument patterns

### Fixed

- machinectl shell PTY errors - exec now uses nsenter for reliability
- Memory stats for machines with hyphens in names
- Stop command failures - added systemctl stop fallback
- Delete command warnings - changed to debug level
- Fedora DNS resolution - systemd-resolved handling
- Rootfs ownership after pre-boot provisioning
- Duplicate btrfs subvolume creation
- chown/chmod permission issues with sudo
- ParseSizeToGB calculation for megabyte values
- State file permissions (world-readable → 0600)
- Snapshot state file permissions (world-readable → 0600)

### Known Issues

- **Snapshots require btrfs** - LVM/ZFS snapshot support deferred to future release
- **machinectl shell PTY errors** - Intermittent issues with interactive shell (exec uses nsenter as workaround)
- **Macvlan host visibility** - Host cannot ping macvlan containers (kernel limitation, solved with host_access feature)

[Unreleased]: https://github.com/Inoriol/lxspawn/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Inoriol/lxspawn/releases/tag/v0.1.0
