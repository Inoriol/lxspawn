# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Security

#### P1 Security Hardening (SUGGESTIONS.md R1-R8)

All P1 (data loss and host compromise) issues from code review have been fixed:

- **R1: `lxspawn edit` no longer destroys container contents** - Added `Manager.Apply()` method that regenerates config files in place, preserving rootfs. Previously, edit would delete and recreate the machine, losing all installed packages and data.
- **R2: Btrfs snapshots moved to sibling directory** - Snapshots now stored in `<machine>.snapshots/` instead of `<rootfs>/.snapshots/`. Rollback no longer deletes snapshots since they're not child subvolumes of the rootfs.
- **R3: Safe rollback pattern** - Rollback now renames old rootfs to `.old`, moves snapshot copy in, deletes `.old` only on success. Prevents data loss if rollback fails midway.
- **R4: Path traversal prevention** - Added `ValidateMachineName()` exported function. Snapshot manager validates names at all entry points (create, rollback, delete). Prevents attacks like `lxspawn delete-snapshot m ../../victim`.
- **R5: Symlink-safe tar extraction** - Tar extraction now tracks symlinks created during extraction. Prevents writes through symlinks using `O_NOFOLLOW` and path resolution. Blocks attacks where a symlink like `a -> /` followed by `a/home/u/.bashrc` would write to host.
- **R6: Secure path resolution in provisioners** - Added `secureJoin()` function that resolves symlinks and validates paths don't escape rootfs. Prevents provisioners from writing through symlinks in the rootfs to host paths.
- **R7: Shell injection prevention in hooks** - Hook generation now returns errors for invalid interface names instead of generating echo commands. All values shell-quoted with `shellQuote()`. Prevents attacks like interface name `eth0; rm -rf /`.
- **R8: Comprehensive input validation** - Added validation for:
  - Hostname (no newlines, valid characters)
  - Environment variables (no newlines in keys/values, no `=` in keys)
  - Capabilities (must start with `CAP_`, no spaces)
  - GPU devices (no path traversal with `..`)
  - Network interfaces (alphanumeric, hyphens, underscores only)
  - Volume paths (no newlines)
  - Resource sizes (valid format)

### Changed

- **Snapshot storage location** - Btrfs snapshots now stored in `/var/lib/machines/<machine>.snapshots/` instead of `/var/lib/machines/<machine>/.snapshots/`. This is a breaking change for existing snapshots.
- **Sudoers rules updated** - Updated btrfs sudoers rules to match new snapshot path structure.

### Added

- **`Manager.Apply()` method** - New method that regenerates config files in place without touching rootfs. Used by `lxspawn edit` to preserve container contents.
- **`ValidateMachineName()` function** - Exported validation function for machine names, used by snapshot manager.
- **`secureJoin()` function** - Secure path resolution that follows symlinks and validates they don't escape the rootfs.
- **`shellQuote()` function** - Shell quoting for hook script values to prevent injection.
- **`ParseStrict()` function** - YAML parser with `KnownFields(true)` that rejects unknown fields and typos. Used by `lxspawn validate` to catch configuration errors.

### Fixed

- **R1: `lxspawn edit` data loss** - Edit now preserves rootfs and installed packages
- **R2: Snapshot deletion on rollback** - Snapshots preserved during rollback
- **R3: Failed rollback data loss** - Atomic rollback with recovery path
- **R4: Path traversal in snapshot commands** - Name validation at all entry points
- **R5: Symlink escape in tar extraction** - Tracks and validates symlinks during extraction
- **R6: Symlink escape in provisioners** - Secure path resolution with symlink following
- **R7: Shell injection in hooks** - Error on invalid names, shell-quote all values
- **R8: Injection in generated files** - Comprehensive input validation

#### P2 Bug Fixes (SUGGESTIONS.md R9-R27, partial)

11 of 19 P2 (broken features and wrong behaviour) issues have been fixed:

- **R9: Edit commands now regenerate unit files** - `edit-resources`, `edit-network`, `add-volume`, `edit-volume`, and `remove-volume` now call `config.Validate()` and `mgr.Apply()` after editing YAML. Previously, these commands only rewrote the source YAML without regenerating the `.nspawn`, slice, and drop-in files, so changes had no effect until the machine was recreated.
- **R10: `lxspawn validate` now actually validates** - The validate command now calls `config.Validate()` and uses `ParseStrict()` with `yaml.Decoder.KnownFields(true)`. Previously, it only called `config.Parse()` which silently ignored typos like `memroy:` or `no_new_privilege:`. Now reports "field not found" errors for unknown fields.
- **R11: Boot service symlink fixed** - The boot service enable symlink now uses a relative path (`../lxspawn-boot.service`) instead of an absolute host path that dangled inside the container. Previously, the symlink pointed to `/var/lib/machines/<name>/etc/systemd/system/lxspawn-boot.service` which doesn't exist inside the container's filesystem namespace.
- **R12: GPU capabilities moved to .nspawn file** - `Capability=` is now written to the `.nspawn` file's `[Exec]` section instead of the `[Service]` drop-in where it was ignored. GPU capabilities (like `CAP_SYS_ADMIN` for NVIDIA) now take effect correctly.
- **R13: Machine registry moved to system-wide location** - Registry moved from per-user `~/.local/share/lxspawn/machines.json` to system-wide `/var/lib/lxspawn/machines.json`. `Create` now refuses to adopt pre-existing rootfs directories, preventing accidental deletion of directories not managed by lxspawn. This fixes the issue where multiple users or a user switching to superuser would see different machine lists.
- **R14: Registry resurrection and lock leak fixed** - Registry `load()` now decodes into a fresh map instead of merging into the existing map, preventing deleted machines from resurrecting if a concurrent operation was in progress. Lock leak fixed - `unlock()` is now called on load failure. Added `TestLoadUsesFreshMap` unit test.
- **R15: Delete safety improvements** - `Delete` now checks machine state after stop and aborts if still running, preventing deletion of running containers. Provisioning failure now cleans up registered machine and storage, preventing half-created machines.
- **R16: Setuid/setgid/sticky bit preservation** - Tar extraction now preserves special permission bits using proper Unix-to-Go mode conversion. Applies uid/gid from tar header when running as root. Previously, `os.FileMode(header.Mode)` dropped setuid/setgid/sticky bits, breaking `su`, `sudo`, and `ping` in containers. Added `TestUnixModeToFileMode` unit test.
- **R17: Btrfs usage parsing fixed** - `GetUsage()` now trims whitespace from lines and uses proper prefix matching. Previously, `fmt.Sscanf(line, "Device size: %d", ...)` failed on indented lines like `    Device size:    50.00GiB` because the format string didn't skip leading whitespace, always returning 0/0.
- **R18: Snapshot/rollback check machine state** - `snapshot` and `rollback` commands now verify the machine is stopped before proceeding. Previously, the commands would proceed on running machines, potentially causing data loss when rollback deleted the rootfs of a running container.
- **R19: Btrfs subvolume listing uses inode detection** - Btrfs subvolume listing now uses inode-based detection (inode 256) instead of parsing `btrfs subvolume list` output. This fixes snapshot listing on Fedora/Ubuntu layouts where `/` is not the top-level subvolume (e.g., Fedora's `root` or Ubuntu's `@`). Added `TestIsBtrfsSubvolume` unit tests in both snapshot and storage packages.

#### P2 Bug Fixes (SUGGESTIONS.md R21-R27, continued)

7 additional P2 issues have been fixed:

- **R21: D-Bus machine name escaping for leading digits** - Machine names starting with digits (e.g., `3dprint`) are now correctly escaped in D-Bus object paths (`_33dprint`). Previously, sd-bus would reject these paths. Also, `GetMachineState` now returns real errors instead of silently returning "stopped" for D-Bus errors, allowing proper error handling. Added `TestMachineName` and `TestIsMachineNotFound` unit tests.
- **R22: GPU detection fixes** - Multiple fixes to GPU detection: (1) Render node now read from sysfs (`/sys/class/drm/cardN/device/drm/renderD*`) instead of calculated from card index, fixing cases where some cardN entries are not recognized GPUs; (2) PCI resource file parsing fixed to compute size as `end - start + 1` instead of reading the flags field; (3) NVIDIA GPU deduplication added to prevent the same GPU being reported twice (once via DRM scan, once via `/dev/nvidia*`); (4) `/dev/kfd` now bind-mounted for AMD GPUs, enabling ROCm compute support. Added `TestIsDiscreteGPU`, `TestFindRenderNode`, `TestIsNVIDIAGPUDetected` unit tests.
- **R23: `logs --json` consistency** - `GetLogs()` now reads from the container journal using `-M` flag (same as text mode `Logs()`), instead of reading the host unit log with `-u systemd-nspawn@<name>.service`. Previously, text mode showed container logs while JSON mode showed host logs, causing confusion.
- **R24: Remote file handling hardened** - Multiple security improvements: (1) HTTP URLs now rejected (HTTPS-only as documented); (2) Digest is now required for remote files (prevents cache poisoning); (3) Path traversal prevention for relative paths; (4) HTTP redirects blocked; (5) Truncation detection for files exceeding size limit. Added `TestResolveFileRefRejectsHTTP`, `TestResolveFileRefRequiresDigestForHTTPS`, `TestResolveFileRefRejectsPathTraversal` unit tests.
- **R25: yq mode parser fixes** - Multiple fixes: (1) `parseValue` now correctly handles IP addresses (e.g., `10.0.0.1`) and version strings (e.g., `1.2.3`) as strings instead of parsing them as numbers; (2) Operator parsing uses proper parsing instead of `Contains`, preventing false matches when values contain operators (e.g., `.a = "x|=y"`); (3) Unsupported operators now return errors instead of silently doing nothing. Added `TestParseValueIPAddress`, `TestParseValueVersionString`, `TestParseOperator` unit tests.
- **R26: Template handling hardened** - Template rendering now uses `missingkey=error` option, causing errors when referencing undefined parameters (e.g., `{{.Param.dirr}}`) instead of silently rendering empty strings. This prevents dangerous situations like `rm -rf /opt/{{.Param.dirr}}` becoming `rm -rf /opt/`. Environment variable keys are now validated to be valid shell variable names. Added `TestIsValidEnvVarName` unit test, updated `TestRenderTemplateMissingParam`.
- **R27: Per-machine macvlan interface naming** - Host-side macvlan interface now uses per-machine naming (`mv-<machinename>`) instead of shared `mv-lxspawn`. This prevents conflicts when multiple adaptive machines exist, as the second machine's `ip link add` would fail with "File exists". Interface names are truncated to 15 characters (Linux limit) with a hash suffix for uniqueness. Added `TestGenerateHostAccessInterfaceName` unit test.

**Deferred:**
- **R20: LVM thin volumes not remounted after reboot** - Deferred because LVM backend is disabled pending testing infrastructure (see Phase 6).

#### P3 Hardening & Cleanup (SUGGESTIONS.md)

All P3 (hardening and cleanup) issues have been fixed:

- **File modes/ownership in provision/data.go** - `os.Chmod()` now called after write to ensure permissions are applied correctly. `setOwnership()` errors now returned instead of logged as warnings, preventing silent failures when ownership cannot be set.
- **Network config permissions** - `/var/lib/lxspawn/config.json` now written with 0644 permissions instead of 0600, allowing group users to read it for `interface: auto`. Previously, non-root users couldn't read the config, breaking adaptive network mode.
- **Interface filtering in network/detect.go** - `isPhysicalInterface()` now only checks for `/sys/class/net/<name>/device` existence, removing fallback that allowed virtual interfaces (br0, bond0, ve-*, mv-lxspawn) to pass. Interface name parsing now strips `@...` suffix from `ip -o link` output.
- **Relative source path stored** - `create.go` and `launch.go` now use `filepath.Abs()` to store absolute paths in machine registry. Previously, relative paths like `./config.yaml` would fail when running commands from different directories.
- **CLI behaviour improvements** - Added `SilenceUsage: true` and `SilenceErrors: true` to root command to prevent duplicate error messages. `exec` command now properly propagates child exit codes. `status.Leader` now populated for running machines.
- **Download/extract hardening** - `xzReader.Close()` error now checked. Added 10GB cap on decompressed size to prevent disk exhaustion. Downloads now use `/var/lib/lxspawn/cache/` instead of `/tmp` to avoid tmpfs issues on Fedora.
- **Correctness fixes** - `ListSnapshots` now reads actual btrfs subvolume creation time via `btrfs subvolume show` instead of directory mtime. Post-rollback chown now uses `root:root` instead of invoking user. `setup network` now validates non-numeric input. `CPUQuota` now uses `math.Round()` to prevent truncation (e.g., 0.29 CPUs → 29% instead of 28%). `ParseSizeToGB("0.5G")` now returns 1 GB instead of 0.
- **Dead code privileges removed** - Removed ZFS and LVM snapshot sudoers rules from setup.go since these backends are disabled. Kept only LVM storage rules needed for thin provisioning. Reduces attack surface by removing unused privileges.
- **GNU-only commands replaced** - Replaced all uses of GNU-specific `stat -f -c %T` and `df --output` with portable `syscall.Statfs()` calls across 6 files. More efficient (no process spawning), more reliable (no text parsing), and cleaner code.

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
