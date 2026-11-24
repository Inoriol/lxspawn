# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Configurable filesystem paths (`PathConfig`)** - All filesystem paths (nspawn dir, systemd dir, machines dir, lxspawn data dir) are now configurable via `deploy.SetPaths()`. Enables support for non-standard layouts such as NixOS. Default paths remain unchanged (`/etc/systemd/nspawn`, `/etc/systemd/system`, `/var/lib/machines`, `/var/lib/lxspawn`).
- **Dependency injection for `machine.Manager`** - Added `NewManagerWithDeps(state, systemd, storage)` constructor accepting `deploy.StateStore`, `deploy.SystemdClient`, and `storage.StorageBackend` interfaces. Enables unit and integration testing without real systemd, state files, or storage backends.
- **Fake implementations for testing** - New `internal/testutil` package with `FakeSystemdClient` and `FakeStorageBackend`. Both record method calls for assertion and support custom function hooks for error injection.
- **Integration tests for machine lifecycle** - 20+ tests in `manager_integration_test.go` covering Start, Stop, Restart, Delete, GetStatus, ListMachines, ListMachinesWithState, Close, and error paths. All tests use fakes, no root or systemd required.
- **Template validation tests (T3)** - New `templates/templates_test.go` with tests that parse, validate, and compile all embedded templates. Catches regressions in template configuration and defaults.
- **`config.ToYAML()` function** - Serialize a `*config.Config` back to YAML. Used to persist the effective configuration (after `--name` and `--param` overrides) to the canonical config path.
- **`machine.LoadMachineConfig()` helper** - Load machine config preferring `ConfigPath` over `SourcePath`. Used by all commands that need to read machine configuration.

### Changed

- **`machine/manager.go` split into focused files** - Extracted `stats.go` (MachineStats, GetStats, GetAllStats, cgroup readers) and `shell.go` (Shell, Exec, Logs, LogsWithOptions, shell quoting). `manager.go` now contains only core lifecycle and state methods.
- **`Manager` fields use interface types** - `state` changed from `*deploy.StateManager` to `deploy.StateStore`, `systemd` changed from `*deploy.SystemdManager` to `deploy.SystemdClient`. `storage` was already an interface.
- **`SystemdClient` interface expanded** - Added `ResetFailedUnit` and `GetMachineLeader` methods to match what `Manager` actually calls.
- **`parse()` refactored to call `parseString()`** - Eliminated duplicated default-setting logic. Both `Parse()` and `ParseString()` now share the same code path.
- **Edit commands use canonical config** - All edit commands (`edit-resources`, `edit-network`, `add-volume`, `edit-volume`, `remove-volume`, `edit`) now read from and write to the canonical config path (`/var/lib/lxspawn/<name>/config.yaml`) instead of the original source path. This ensures template and stdin-created machines can be edited.
- **Canonical config written with 0600 permissions** - Canonical config files are now written with 0600 permissions instead of 0644, protecting sensitive data like `param:` secrets.

### Fixed

- **`ParseString()` missing defaults** - `ParseString()` (used by templates and stdin) was missing defaults for `security.seccomp` (`true`), `resources.io_weight` (`100`), and `storage.size` (`"10G"`). Templates loaded via `--template` or stdin now get the same defaults as file-based configs.
- **Template and stdin machines have no usable config (C1)** - Machines created with `--template` or `-` (stdin) now have a proper canonical config. The effective `*config.Config` (after `--name` and `--param` overrides) is serialized and written to `CanonicalConfigPath`. Previously, `copyFile(sourcePath, canonical)` would fail with "template:nextcloud: no such file" and only log a warning.
- **`--param` and `--name` overrides are lost (C2)** - Parameter and name overrides are now preserved in the canonical config. Previously, the canonical copy was the raw YAML, so a later `provision` would re-render with template defaults.
- **Rollback sudoers rules mismatch (D1)** - Added sudoers rules for rollback paths (`.rollback-temp`, `.rollback-old`) and snapshot show commands. Previously, non-root rollback failed at the first sudo call because the rules didn't match the actual paths.
- **Rollback failure recovery never runs (D2)** - Recovery commands are now actually executed and their errors checked. If the second `mv` fails, the old rootfs is recovered to its original location. Previously, recovery commands were built but never called, leading to data loss.
- **Rollback auto-deletes leftovers (D2)** - Rollback now refuses to proceed if `.rollback-temp` or `.rollback-old` exist, asking the user to inspect and remove them manually. Previously, it would auto-delete these directories, potentially destroying data from a failed previous rollback.
- **Rollback flattens all ownership (D3)** - Removed the `chown -R 0:0` after rollback. With `PrivateUsers=identity`, container uids are real host uids, so chown would break services like MariaDB/Apache that expect specific ownership.
- **GPU device path injection (V1)** - `gpu.devices` now validated with strict regex `^/dev/[A-Za-z0-9._/-]+$` and rejects colons. Previously, newlines in device paths could inject INI lines into `.nspawn` files.
- **tmpfs options injection (V2)** - tmpfs `options` now validated against `^[A-Za-z0-9=,._-]*$`. Colons are rejected in volume host and mount paths. Previously, crafted YAML could inject INI lines.
- **Sensitive path check bypassed (V3)** - Expanded sensitive paths list to include `/bin`, `/boot`, `/etc`, `/lib`, `/lib64`, `/opt`, `/root`, `/sbin`, `/srv`, `/usr`, `/var/lib/lxspawn`, `/var/lib/machines`. Previously, paths like `/etc`, `/root`, `/var/lib/machines` were not blocked.

## [0.1.3] - 2026-11-21

### Added

- **Snapshot description (C12)** - `lxspawn snapshot` now accepts `--description` flag to store a description with the snapshot. Description is included in JSON output from `list-snapshots`.
- **`edit-resources` new flags** - Added `--memory-high` (soft memory limit) and `--io-weight` (IO scheduling priority) flags to `edit-resources` command.
- **Complete `config --json` output (C3)** - `config --json` now returns full configuration including `hooks`, `provision`, `param`, `probes`, `journal`, `resources.memory_high`, and `resources.io_weight` fields. Previously only returned basic fields.
- **Embedded templates (C6)** - Templates are now embedded in the binary using `go:embed`. Templates moved from `examples/templates/` to `templates/` directory at project root. `list-templates` shows embedded templates without requiring filesystem paths.
- **Canonical config copy (C4)** - On `create`/`launch`, YAML is copied to `/var/lib/lxspawn/<name>/config.yaml`. Added `ConfigPath` field to `MachineInfo`. `config` and `status` commands read from canonical path first, fallback to `SourcePath`. Config survives source file deletion/moves.
- **Enhanced logs command (C13)** - Unified JSON/text modes to both use container journal (`-M`). Added flags: `--since` (filter by date), `--priority` (filter by log level), `--follow`/`-f` (streaming mode). `--lines` now works in both modes.
- **Batched stats (C14)** - Added `stats --all` to get stats for all machines in one call. Returns `cpu_usage_usec` cumulative counter with timestamp, allowing UI to compute CPU rates without the 100ms sleep. No more N processes per polling cycle.
- **Create flow enhancements (C5)** - `create -` / `launch -` reads YAML from stdin. `create --template NAME --name N` uses embedded template. `--param key=value` (repeatable) overrides template params. Shared config-loading logic in `config_loader.go`.
- **Structured error codes (C9)** - New `internal/errors` package with typed errors: `MACHINE_NOT_FOUND`, `MACHINE_EXISTS`, `MACHINE_RUNNING`, `PERMISSION_DENIED`, `VALIDATION_FAILED`, `STORAGE_ERROR`, `PROVISION_FAILED`, `NETWORK_ERROR`, `CONFIG_INVALID`, `SNAPSHOT_ERROR`, `SYSTEMD_ERROR`, `OPERATION_CANCELLED`, `INTERNAL_ERROR`. Field-level validation errors. Distinct exit codes for each error type (e.g., 65 for validation errors, 2 for machine not found).
- **Progress stream (C10)** - Added `--progress` flag to `create` and `launch` commands. Newline-delimited JSON events via `internal/progress` package. Event types: `downloading`, `extracting`, `provisioning`, `starting`, `done`, `error`.

### Changed

- **Template deduplication** - Templates moved from `examples/templates/` to `templates/` at project root. Single source of truth for templates. `go:embed` in `templates/embed.go` embeds all `.yaml` files. Removed duplicate copies and `examples/templates` fallback from `list-templates`.

### Fixed

- **Cockpit plugin `editVolume` readonly bug** - Fixed plugin to use `--readonly=true` format instead of `--readonly true` which caused cobra boolean flag parsing to fail.
- **Cockpit plugin `editResources` memory_high bug** - Plugin was sending `--memory-high` flag that didn't exist in CLI. Added the flag to the CLI.

## [0.1.2] - 2026-11-20

### Added

- **Provision `user` field (S1)** - New `user:` field in provision steps for `user` mode. Specifies which container user to run the script as. The user must exist inside the container (create in a prior `system` step if needed). Defaults to `root` with a warning if not specified.
- **Lifecycle and Semantics section in SPEC.md (S2)** - New section documenting which commands run which modes, execution order, idempotency requirements, and differences from Lima.
- **Probe per-attempt timeout (S7)** - Each probe attempt now has a 10-second timeout using `context.WithTimeout`. Prevents hanging probe scripts from blocking forever.
- **New CLI commands for lifecycle management (C8):**
  - `lxspawn restart <name>` - Stop then start a machine
  - `lxspawn stop --force <name>` - Force stop using poweroff, terminate, then SIGKILL
  - `lxspawn autostart <name> on|off` - Enable/disable machine autostart at boot
- **`lxspawn version` command (C11)** - Show version, Go version, platform, systemd version, kernel version, and storage backend. Supports `--json` for structured output.
- **`lxspawn doctor` command (C11)** - Check system configuration and dependencies. Verifies lxspawn group membership, systemd-nspawn, machinectl, polkit rules, sudoers rules, storage backend, nftables, and machines directory. Supports `--json` for structured output.
- **`lxspawn apply` command (C1)** - Declarative configuration apply. Reads YAML from file or stdin, validates, and regenerates config files in place without touching rootfs. Supports `--dry-run` for preview. Returns `restart_required` in JSON output.
- **`ParseString()` function** - Parse YAML config from string instead of file, used by `apply` command.
- **Service masking in image cleanup (X2)** - `systemd-udevd`, `systemd-udevd-control.socket`, `systemd-udevd-kernel.socket`, and `systemd-modules-load.service` are now automatically masked during image extraction via symlinks to `/dev/null`. Eliminates need for duplicated boot scripts in templates.
- **Sudoers rules for systemctl enable/disable** - Added rules for `systemctl enable/disable systemd-nspawn@<name>.service` to support `autostart` command.
- **Nextcloud `trusted_domain` param (T2)** - New parameter for LAN access configuration via `occ config:system:set trusted_domains`.
- **AdGuard Home `admin_pass` param (T5)** - New required parameter for web admin password.
- **Fedora-base DNS params (T21)** - New `dns1`, `dns2`, `dns3` parameters for configurable DNS servers.

### Changed

- **User mode now uses explicit `user:` field (S1)** - `user` provision mode now uses the `user:` field from the provision step instead of the host username. The user must exist in the container. Previously used host username which didn't exist in container.
- **Boot mode description clarified (S4)** - SPEC.md and README.md now correctly describe boot mode as running "during boot via oneshot systemd service" instead of "before systemd starts". Boot scripts run on every container boot.
- **Boot mode example updated (S4)** - SPEC.md example now shows sysctl tuning instead of service masking (services are already masked by image cleanup).
- **`DEBIAN_FRONTEND=noninteractive` always exported (X6)** - All system/user provision scripts now have `DEBIAN_FRONTEND=noninteractive` set, preventing apt prompts during package installation.
- **Template version pinning (X3):**
  - AdGuard Home pinned to `v0.107.55` (was `master` branch)
  - Pi-hole pinned to `v5.18.3` (was unpinned)
  - Nextcloud pinned to `29.0.9` (was `latest.tar.bz2`)
- **Template security improvements (X4):**
  - Nextcloud: placeholder passwords (`CHANGE_ME_DB`, `CHANGE_ME_ADMIN`), guard clause rejects defaults, `set +x` around password-bearing commands
  - Pi-hole: placeholder password (`CHANGE_ME`), guard clause rejects default
  - Samba: placeholder password (`CHANGE_ME`), guard clause rejects default, `set +x` around smbpasswd
- **Template fixes:**
  - **T1 (Nextcloud):** Volume mount path corrected to `/var/www/html/nextcloud/data`, added `--data-dir` flag, SQL uses `IF NOT EXISTS` for idempotency, `occ status` check prevents re-install
  - **T8 (Docker):** Replaced `dockerd &` background process with `systemctl enable docker.service`, removed `chmod 666` on socket, added `docker` group creation, probe uses `systemctl is-active`
  - **T11 (Samba):** Secure defaults (`guest ok = no`, `read only = yes`, `map to guest = Never`), added `valid users` and `force user` params, probe uses port 445 check instead of smbclient
- **Boot script deduplication (X2)** - Removed identical `boot:` blocks from all 10 templates (debian-base, ubuntu-base, fedora-base, docker-host, nextcloud, samba, adguard, pihole, jellyfin, plex). Fedora-base retains only `systemd-resolved` masking and resolv.conf workaround.
- **Template leftovers (Phase 4.19):**
  - **T5 (AdGuard):** Pre-seeded `AdGuardHome.yaml` config to skip setup wizard, DNS works immediately after launch
  - **T6 (Pi-hole):** Moved `setupVars.conf` to system mode, detect actual interface at provision time
  - **T14 (Samba):** Removed redundant `CAP_DAC_OVERRIDE` (already in nspawn default set)
  - **T15 (Jellyfin/Plex):** Added `vainfo` check after driver install, logs whether GPU acceleration is available
  - **T16 (Jellyfin):** Replaced hard-coded `arch=amd64` with `arch=$(dpkg --print-architecture)` for ARM support
  - **T17 (Jellyfin/Plex):** Changed host paths from `/var/lib/jellyfin|plex` to `/var/lib/lxspawn-data/jellyfin|plex` to avoid collision with host installs
  - **T18 (Jellyfin/Plex):** Added `chown` in pre_start hook for correct UID (121 for Jellyfin, 123 for Plex)
  - **T19 (Jellyfin/Plex):** Removed `software-properties-common`, added comments about rocm not in bookworm
  - **T20 (Base templates):** Added `# Host networking: no isolation, container shares host ports` comment
  - **T21 (Fedora):** Made DNS servers configurable via params, removed non-working boot mode block
  - **T22 (Base templates):** Ubuntu merged apt-get into single command, Fedora added upgrade guard (`/etc/lxspawn-upgraded`)

### Fixed

- **User mode broken (S1)** - `user` provision mode now uses the `user:` field instead of host username. Previously tried to `su - <host_username>` inside container where that user didn't exist.
- **DNS workaround section removed (S5)** - Removed "DNS Configuration with systemd-resolved" section from SPEC.md. The workaround documented a bug that doesn't exist (boot mode works correctly via oneshot service).
- **Probe timeout missing (S7)** - Individual probe attempts now have 10-second timeout. Previously a hanging probe script (e.g., `curl` to unreachable host) could block forever.
- **BuildEnvExport now always returns non-empty string** - Previously returned empty string when no params, now always includes `DEBIAN_FRONTEND=noninteractive`. Updated tests accordingly.
- **Template validation** - All templates now pass `lxspawn validate` after boot block removal and version pinning.
- **T2 (Nextcloud):** LAN access now works via `trusted_domain` param (was rejected with "Access through untrusted domain")
- **T5 (AdGuard):** Probe no longer hangs waiting for wizard completion (pre-seeded config)
- **T6 (Pi-hole):** Interface detection now works with adaptive/macvlan networking (was hard-coded to eth0)
- **T16 (Jellyfin):** ARM architecture now supported (was hard-coded to amd64)
- **T17 (Jellyfin/Plex):** Host paths no longer collide with host installations
- **T21 (Fedora):** DNS servers now configurable, boot mode block removed (didn't work due to systemd-resolved timing)
- **T22 (Ubuntu):** Removed duplicate `apt-get update` call
- **T22 (Fedora):** System upgrade only runs on first provision (idempotent)

### Security

- **Template secret handling** - Passwords in templates use placeholder values with guard clauses that refuse to run if defaults are not changed. Secret-bearing commands wrapped in `set +x` to prevent logging.

## [0.1.1] - 2026-11-10

### Security

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
- **R21: D-Bus machine name escaping for leading digits** - Machine names starting with digits (e.g., `3dprint`) are now correctly escaped in D-Bus object paths (`_33dprint`). Previously, sd-bus would reject these paths. Also, `GetMachineState` now returns real errors instead of silently returning "stopped" for D-Bus errors, allowing proper error handling. Added `TestMachineName` and `TestIsMachineNotFound` unit tests.
- **R22: GPU detection fixes** - Multiple fixes to GPU detection: (1) Render node now read from sysfs (`/sys/class/drm/cardN/device/drm/renderD*`) instead of calculated from card index, fixing cases where some cardN entries are not recognized GPUs; (2) PCI resource file parsing fixed to compute size as `end - start + 1` instead of reading the flags field; (3) NVIDIA GPU deduplication added to prevent the same GPU being reported twice (once via DRM scan, once via `/dev/nvidia*`); (4) `/dev/kfd` now bind-mounted for AMD GPUs, enabling ROCm compute support. Added `TestIsDiscreteGPU`, `TestFindRenderNode`, `TestIsNVIDIAGPUDetected` unit tests.
- **R23: `logs --json` consistency** - `GetLogs()` now reads from the container journal using `-M` flag (same as text mode `Logs()`), instead of reading the host unit log with `-u systemd-nspawn@<name>.service`. Previously, text mode showed container logs while JSON mode showed host logs, causing confusion.
- **R24: Remote file handling hardened** - Multiple security improvements: (1) HTTP URLs now rejected (HTTPS-only as documented); (2) Digest is now required for remote files (prevents cache poisoning); (3) Path traversal prevention for relative paths; (4) HTTP redirects blocked; (5) Truncation detection for files exceeding size limit. Added `TestResolveFileRefRejectsHTTP`, `TestResolveFileRefRequiresDigestForHTTPS`, `TestResolveFileRefRejectsPathTraversal` unit tests.
- **R25: yq mode parser fixes** - Multiple fixes: (1) `parseValue` now correctly handles IP addresses (e.g., `10.0.0.1`) and version strings (e.g., `1.2.3`) as strings instead of parsing them as numbers; (2) Operator parsing uses proper parsing instead of `Contains`, preventing false matches when values contain operators (e.g., `.a = "x|=y"`); (3) Unsupported operators now return errors instead of silently doing nothing. Added `TestParseValueIPAddress`, `TestParseValueVersionString`, `TestParseOperator` unit tests.
- **R26: Template handling hardened** - Template rendering now uses `missingkey=error` option, causing errors when referencing undefined parameters (e.g., `{{.Param.dirr}}`) instead of silently rendering empty strings. This prevents dangerous situations like `rm -rf /opt/{{.Param.dirr}}` becoming `rm -rf /opt/`. Environment variable keys are now validated to be valid shell variable names. Added `TestIsValidEnvVarName` unit test, updated `TestRenderTemplateMissingParam`.
- **R27: Per-machine macvlan interface naming** - Host-side macvlan interface now uses per-machine naming (`mv-<machinename>`) instead of shared `mv-lxspawn`. This prevents conflicts when multiple adaptive machines exist, as the second machine's `ip link add` would fail with "File exists". Interface names are truncated to 15 characters (Linux limit) with a hash suffix for uniqueness. Added `TestGenerateHostAccessInterfaceName` unit test.

**Deferred:**
- **R20: LVM thin volumes not remounted after reboot** - Deferred because LVM backend is disabled pending testing infrastructure (see Phase 6).

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

[Unreleased]: https://github.com/Inoriol/lxspawn/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/Inoriol/lxspawn/releases/tag/v0.1.2
[0.1.1]: https://github.com/Inoriol/lxspawn/releases/tag/v0.1.1
[0.1.0]: https://github.com/Inoriol/lxspawn/releases/tag/v0.1.0
