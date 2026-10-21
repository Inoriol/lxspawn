# TODO

Remaining tasks for lxspawn, organized by phase.

---

## Current Status (2026-10-21)

**Phase 4.12 (V2) - Privilege Model Hardening - COMPLETE ✅**

**Goal:** Eliminate critical privilege escalation vulnerabilities in sudoers rules.

**Status:** All critical and high priority security issues addressed. Replaced broad sudo access with a single privileged helper binary that validates all arguments.

### Changes Made

**1. Sudo Helper Binary (COMPLETE ✅)**
- **Created:** `cmd/lxspawn-sudo-helper/main.go` - Privileged helper binary
- **Validates:** All paths, machine names, PIDs, LVM names, subcommands
- **Prevents:** Path traversal, privilege escalation, arbitrary command execution
- **Installed to:** `/usr/libexec/lxspawn/sudo-helper`

**2. Sudoers Rules (COMPLETE ✅)**
- **Before:** 18 separate rules granting unrestricted access to `rm`, `nsenter`, `btrfs`, `zfs`, `lv*`, `mount`, `umount`, `chown`, `chmod`, `mv`, `rsync`, `findmnt`, `journalctl`
- **After:** Single rule for `/usr/libexec/lxspawn/sudo-helper`
- **Files:** `cmd/lxspawn/setup.go`

**3. Polkit Rules (COMPLETE ✅)**
- **Before:** Broad access to all machine1 actions
- **After:** Scoped to lxspawn machines only (validated by regex)
- **Files:** `cmd/lxspawn/setup.go`

**4. Code Migration (COMPLETE ✅)**
- **Updated:** All `exec.Command("sudo", ...)` calls to use `deploy.SudoHelper(...)`
- **Files:** `internal/machine/manager.go`, `internal/machine/lvm.go`, `internal/image/fetcher.go`, `internal/provision/exec.go`, `internal/provision/probe.go`, `internal/storage/btrfs.go`, `internal/storage/directory.go`, `internal/storage/lvm_thin.go`, `internal/storage/detector.go`, `internal/snapshot/btrfs.go`, `internal/snapshot/lvm.go`, `internal/snapshot/zfs.go`

### Security Vulnerabilities Fixed

**Critical:**
- ✅ `sudo nsenter` → Full root shell on host (now validates PID ownership)
- ✅ `sudo mount` → Mount anything anywhere (now validates paths and devices)
- ✅ `sudo chown` → Own any file (now validates paths)
- ✅ `sudo rsync` → Read/write any file (now validates paths)
- ✅ `sudo btrfs` → Full btrfs control (now restricted to subcommands)
- ✅ `sudo rm -rf /var/lib/machines/*` → Path traversal (now validates paths)

**High:**
- ✅ Polkit rules too broad (now scoped to lxspawn machines)
- ✅ No validation on sudo arguments (now all validated by helper)

### Testing Results

| Test | Status | Notes |
|------|--------|-------|
| `go build ./...` | ✅ PASS | Both binaries build successfully |
| `go vet ./...` | ✅ PASS | No issues found |
| `go test ./...` | ✅ PASS | All tests pass |
| Path validation | ✅ PASS | Rejects paths outside allowed directories |
| Path traversal | ✅ PASS | Rejects paths with `..` |
| Machine name validation | ✅ PASS | Rejects invalid names |
| PID ownership | ✅ PASS | Verifies PID belongs to lxspawn machine |
| LVM name validation | ✅ PASS | Rejects non-lxspawn LV names |

### Key Findings

1. **Attack surface minimized** - Single entry point for all privileged operations
2. **Defense in depth** - Multiple layers of validation (paths, names, PIDs, subcommands)
3. **Audit trail** - All privileged operations go through one binary that can be audited
4. **Backward compatible** - Existing functionality preserved, just more secure

### Documentation Updates

- ✅ README.md - Updated setup section and privilege model
- ✅ ARCHITECTURE.md - Added sudo-helper to package structure, security model, and design decisions
- ✅ SPEC.md - Updated security section (if needed)

### Action Items for Next Phase

**Start Phase 5: Cockpit Web UI (HIGH PRIORITY)**
- [ ] Design Cockpit plugin architecture
- [ ] Implement machine list dashboard
- [ ] Implement machine details page
- [ ] Implement create wizard
- [ ] Use new JSON API commands for all operations

---

## Previous Status (2026-10-19)

**Phase 4.11 (V2) - JSON API for Cockpit UI - COMPLETE ✅**

**Goal:** Implement missing JSON API commands to support Cockpit UI development.

**Status:** All Phase 1-3 commands implemented and tested on Bazzite machine. JSON API coverage increased from ~40% to ~95%.

### Changes Made

**Phase 1: Core JSON API (MVP) - COMPLETE ✅**
- **Added:** `lxspawn stats <name> --json` - Runtime CPU/memory/disk stats from cgroups
- **Added:** `lxspawn config <name> --json` - Detailed config (volumes, network, security, GPU, advanced)
- **Added:** `lxspawn provision-status <name> --json` - Provisioning state (last run, phases, probes)
- **Added:** `lxspawn list-distros --json` - Available distros for create wizard
- **Enhanced:** Added `description` field to `SnapshotJSON`
- **Enhanced:** Added `ephemeral`, `read_only` to `MachineStatusData`
- **Files:** `cmd/lxspawn/stats.go`, `cmd/lxspawn/config.go`, `cmd/lxspawn/provision_status.go`, `cmd/lxspawn/list_distros.go`, `internal/output/schemas.go`, `internal/machine/manager.go`, `internal/provision/engine.go`

**Phase 2: Advanced JSON API - COMPLETE ✅**
- **Added:** `lxspawn edit-resources <name>` - Edit CPU/memory/swap/tasks
- **Added:** `lxspawn edit-network <name>` - Edit network mode/bridge/interface
- **Added:** `lxspawn add-volume <name>` - Add bind mount
- **Added:** `lxspawn edit-volume <name>` - Edit bind mount
- **Added:** `lxspawn remove-volume <name>` - Remove bind mount
- **Files:** `cmd/lxspawn/edit_resources.go`, `cmd/lxspawn/edit_network.go`, `cmd/lxspawn/add_volume.go`, `cmd/lxspawn/edit_volume.go`, `cmd/lxspawn/remove_volume.go`

**Phase 3: Polish JSON API - COMPLETE ✅**
- **Added:** `lxspawn logs <name> --json` - Structured log output (with `--lines` flag)
- **Added:** `lxspawn validate <yaml> --json` - YAML validation without creating
- **Added:** `lxspawn list-templates --json` - Available templates
- **Files:** `cmd/lxspawn/logs.go` (updated), `cmd/lxspawn/validate.go`, `cmd/lxspawn/list_templates.go`

### Testing Results (Bazzite - 192.168.0.200)

| Command | Status | Notes |
|---------|--------|-------|
| `list-distros --json` | ✅ PASS | Returns 11 supported distros |
| `validate --json` | ✅ PASS | Validates YAML correctly |
| `config --json` | ✅ PASS | Shows full parsed config |
| `provision-status --json` | ✅ PASS | Shows provisioning state |
| `stats --json` | ✅ PASS | Shows runtime stats (disk stats working) |
| `edit-resources --json` | ✅ PASS | Updates CPU/memory correctly |
| `edit-network --json` | ✅ PASS | Updates network config correctly |
| `add-volume --json` | ✅ PASS | Adds volumes correctly |
| `edit-volume --json` | ✅ PASS | Edits volumes correctly |
| `remove-volume --json` | ✅ PASS | Removes volumes correctly |
| `logs --json` | ⚠️ PARTIAL | Implemented but requires sudo for journalctl -M |

### Key Findings

1. **JSON API coverage increased from ~40% to ~95%** - All commands working correctly
2. **All edit commands work correctly** - YAML manipulation via yaml.Node preserves formatting
3. **Stats command fixed** - Cgroup path corrected (`.slice` not `.scope`), CPU percentage calculated with 100ms sampling
4. **Logs command fixed** - Changed from `journalctl -M` to `journalctl -u systemd-nspawn@<name>.service`, added sudoers rule

### Known Issues

**All issues resolved!** ✅

### Action Items for Next Phase

**Start Phase 5: Cockpit Web UI (HIGH PRIORITY)**
- [ ] Design Cockpit plugin architecture
- [ ] Implement machine list dashboard
- [ ] Implement machine details page
- [ ] Implement create wizard
- [ ] Use new JSON API commands for all operations

---

## Previous Status (2026-10-19)

**Phase 4.10 (V2) - Snapshot Backend Cleanup & Directory Backend Fix - COMPLETE ✅**

**Goal:** Clean up snapshot backends, fix directory backend warnings, and verify templates.

**Status:** Directory mode snapshots removed, LVM/ZFS backends disabled pending testing. Chmod warning fixed. All templates verified with latest distro versions.

### Changes Made

**1. Snapshot Backend Cleanup (COMPLETE ✅)**
- **Removed:** Directory mode snapshot support (unreliable, slow, not true snapshots)
- **Disabled:** LVM and ZFS snapshot backends (deferred to after Phase 5)
- **Reason:** Directory mode used rsync (file-by-file copy), not atomic, snapshots of running machines were inconsistent
- **Result:** Snapshots now require btrfs filesystem only
- **Files:** `internal/snapshot/detector.go`, `internal/snapshot/directory.go` (deleted), `internal/snapshot/directory_test.go` (deleted)

**2. Directory Backend Chmod Warning Fix (COMPLETE ✅)**
- **Problem:** After `setOwnership()` changed directory to root:lxspawn, `os.Chmod()` failed because user can't chmod root-owned files
- **Fix:** Changed to use `sudo chmod` instead of `os.Chmod()` in directory backend
- **Files:** `internal/storage/directory.go:53-59`
- **Result:** No more "failed to set permissions" warnings during machine creation

**3. Template Version Updates (COMPLETE ✅)**
- **Updated:** `fedora:40` → `fedora:44` in test file and documentation
- **Verified:** All templates use latest distro versions:
  - `debian:bookworm` (Debian 12, latest stable)
  - `ubuntu:noble` (Ubuntu 24.04 LTS, latest LTS)
  - `fedora:44` (Fedora 44, latest)
- **Files:** `internal/deploy/state_test.go`, `JSON-API-COVERAGE.md`

### Testing Results (Debian VM - Directory Backend)

| Test | Status | Notes |
|------|--------|-------|
| debian-base | ✅ PASS | Full lifecycle working, no chmod warnings |
| ubuntu-base | ✅ PASS | Full lifecycle working |
| fedora-base | ✅ PASS | Full lifecycle working (fedora:44) |
| Snapshot rejection | ✅ PASS | Clear error: "snapshots require btrfs filesystem" |
| Provisioning | ✅ PASS | Pre-boot and post-boot work correctly |

### Key Validations

1. **Containers work without snapshots** - Core functionality intact
2. **Clear error messages** - Snapshot attempts on non-btrfs fail with helpful error
3. **No silent fallbacks** - System doesn't hide backend limitations
4. **No chmod warnings** - Directory backend works cleanly
5. **All templates use latest versions** - debian:bookworm, ubuntu:noble, fedora:44
6. **All tests pass** - 303 unit tests passing (1 skipped, requires root)

---

## Previous Status (2026-10-18)

**Phase 4.9 (V2) - E2E Testing on Bazzite - COMPLETE ✅**

**Goal:** Full E2E testing of all templates on Bazzite/Fedora 44 with btrfs backend.

**Test Machine:** Bazzite mini-PC at 192.168.0.200 (btrfs backend)

**Status:** All tested templates working correctly. Critical bugs fixed during testing.

### Bugs Fixed During E2E Testing

**1. Rootfs Ownership Regression (FIXED ✅)**
- **Problem:** After introducing storage backend abstraction, files extracted from tarballs had uid 1000 instead of uid 0, breaking sudo and root operations inside containers
- **Root Cause:** Storage backend created btrfs subvolume with root:lxspawn ownership, but tar extraction ran as current user (1000), creating files with wrong ownership
- **Fix:** Added `fixRootfsOwnership()` function in `internal/machine/manager.go` that runs `sudo chown -R 0:0` on rootfs AFTER pre-boot provisioning completes
- **Files:** `internal/machine/manager.go:620-626`, `internal/image/fetcher.go:88-101`

**2. Duplicate btrfs Subvolume Creation (FIXED ✅)**
- **Problem:** Storage backend created btrfs subvolume, then fetcher tried to create it again, failed with "File exists", fell back to regular directory with wrong permissions
- **Fix:** Added check in `prepareTargetDir()` to skip if directory already exists, and chown to current user for extraction
- **Files:** `internal/image/fetcher.go:88-101`

**3. chown/chmod Without Sudo (FIXED ✅)**
- **Problem:** After creating btrfs subvolume with sudo, chown/chmod failed because user can't modify root-owned files
- **Fix:** Changed to use `sudo chown` and `sudo chmod` in storage backends
- **Files:** `internal/storage/btrfs.go:60-68`, `internal/storage/directory.go:281-287`

**4. Fedora DNS Resolution Issue (FIXED ✅)**
- **Problem:** systemd-resolved overwrites /etc/resolv.conf even after masking the service, breaking DNS in Fedora containers
- **Root Cause:** systemd-resolved starts before boot scripts run, so masking doesn't prevent it from managing DNS
- **Fix:** Overwrite /etc/resolv.conf in system provision script AFTER systemd-resolved has started
- **Files:** `examples/templates/fedora-base.yaml`

### E2E Test Results (Bazzite - btrfs Backend)

| Template | Status | Notes |
|----------|--------|-------|
| debian-base | ✅ PASS | Full lifecycle: launch, provision, exec, snapshot, rollback, delete |
| ubuntu-base | ✅ PASS | Full lifecycle working |
| fedora-base | ✅ PASS | DNS fixed with resolv.conf overwrite in system provision |
| docker-host | ✅ PASS | Docker-in-container fully working (hello-world test passed) |
| pihole | ✅ PASS | Now working after ownership fix |
| adguard | ⏭️ SKIPPED | Similar to pihole, should work with ownership fix |
| samba | ⏭️ SKIPPED | Not tested in this session |
| nextcloud | ⏭️ SKIPPED | Not tested in this session |
| jellyfin | ⏭️ SKIPPED | Requires GPU testing |
| plex | ⏭️ SKIPPED | Requires GPU testing |

### Key Findings

1. **Core functionality is solid** - btrfs backend, snapshots, rollback all working correctly
2. **Ownership fix was critical** - Without it, containers had broken sudo and root operations
3. **systemd-resolved requires special handling** - Can't be masked in boot scripts, must overwrite resolv.conf after it starts
4. **All tested templates are production-ready** - debian-base, ubuntu-base, fedora-base, docker-host, pihole

### Unit Tests Added

- `TestFixRootfsOwnership` - Tests for rootfs ownership fix (requires root)
- Existing tests updated to reflect new ownership handling
- All 411+ unit tests passing

### Action Items for Next Phase

**1. Test Remaining Templates (MEDIUM PRIORITY)**
- [ ] Test adguard template (similar to pihole, should work)
- [ ] Test samba template (file server)
- [ ] Test nextcloud template (LAMP stack)
- [ ] Test jellyfin template (GPU pass-through)
- [ ] Test plex template (GPU pass-through)

**2. Document systemd-resolved Handling (LOW PRIORITY)**
- [ ] Add note to SPEC.md about systemd-resolved in Fedora containers
- [ ] Document the pattern for handling DNS in containers with systemd-resolved

---

## Previous Status (2026-10-17)

**Phase 4.7 (V2) - Storage Backend Overhaul - COMPLETE ✅**

**Problem:** Current LVM implementation uses thick LVs (10GB per machine), which wastes space and requires VG to have free space before creating machines. Proxmox uses LVM-thin for efficient space usage.

**Solution:** Implemented hybrid storage backend that automatically selects the best storage method based on filesystem type:
- btrfs → subvolumes (current behavior)
- LVM with thin pool → thin volumes (new, like Proxmox)
- LVM without thin pool → suggest creating thin pool OR use directory mode
- Other filesystems (ext4, etc.) → directory mode

**Status:** Core implementation complete. All three backends implemented and integrated.

### High Priority (Core Implementation) - COMPLETE ✅

**1. Storage Backend Abstraction**
- [x] Create `internal/storage/backend.go` with StorageBackend interface
  - `CreateMachine(name string, sizeGB int) error`
  - `DeleteMachine(name string) error`
  - `CreateSnapshot(machine, snapName string) error`
  - `DeleteSnapshot(machine, snapName string) error`
  - `ListSnapshots(machine string) ([]SnapshotInfo, error)`
  - `RollbackSnapshot(machine, snapName string) error`
  - `Type() string` (btrfs, lvm-thin, directory)

**2. Directory Backend**
- [x] Create `internal/storage/directory.go`
  - Simple directory-based storage in `/var/lib/machines/<name>/`
  - No block-level snapshots (but can use file-based backups)
  - Works on any filesystem (ext4, xfs, etc.)
  - Fallback when LVM-thin not available

**3. LVM-Thin Backend**
- [x] Create `internal/storage/lvm_thin.go`
  - Detect existing thin pool in VG
  - Create thin volumes from pool (allocated on-demand)
  - Support thin snapshots (efficient copy-on-write)
  - Handle over-provisioning (volumes can be larger than available space)

**4. Btrfs Backend**
- [x] Create `internal/storage/btrfs.go`
  - Implement StorageBackend interface for btrfs
  - Use btrfs subvolumes for machine storage
  - Use btrfs snapshots for efficient snapshots

**5. Backend Detection and Selection**
- [x] Create `internal/storage/detector.go`
  - Detect filesystem type of `/var/lib/machines/`
  - Check for LVM thin pool existence
  - Select appropriate backend:
    - btrfs → BtrfsBackend
    - LVM with thin pool → LVMThinBackend
    - LVM without thin pool → DirectoryBackend (with warning)
    - Other → DirectoryBackend

**6. Integration with Machine Manager**
- [x] Update `internal/machine/manager.go`
  - Use storage backend instead of direct LVM calls
  - Remove hardcoded LVM logic
  - Support all three backend types

**7. Snapshot Manager Updates**
- [x] Keep existing snapshot package (already has its own backends)
  - Snapshot operations remain in snapshot package
  - Storage package handles machine creation/deletion
  - Clean separation of concerns

### Medium Priority (UX and Safety)

**10. Storage Status Command**
- [ ] Add `lxspawn storage-status` command
  - Show current storage backend in use
  - Display space usage and availability
  - Show thin pool utilization (if applicable)

**11. Configuration Options**
- [ ] Add storage configuration to YAML
  - `storage.backend: auto|btrfs|lvm-thin|directory`
  - `storage.lvm_thin_pool: <pool-name>` (optional, auto-detect if not specified)
  - `storage.directory_path: <path>` (for directory mode)

**12. Documentation**
- [ ] Update README.md with storage backend information
- [ ] Create STORAGE.md with detailed backend comparison
- [ ] Add migration guide for existing users

### Low Priority (Polish)

**13. Testing**
- [ ] Test btrfs backend (existing tests should still pass)
- [ ] Test LVM-thin backend on Debian VM
- [ ] Test directory backend on various filesystems
- [ ] Test migration between backends
- [ ] Add integration tests for all three backends

**14. Performance Optimization**
- [ ] Benchmark directory mode vs LVM-thin
- [ ] Optimize snapshot creation for directory mode
- [ ] Add caching for storage backend detection

**15. Error Handling**
- [ ] Improve error messages for storage backend failures
- [ ] Add fallback logic (e.g., if thin pool full, suggest expanding)
- [ ] Validate storage backend compatibility before machine creation

### Implementation Order

1. **Storage Backend Abstraction** (foundation)
2. **Directory Backend** (simplest, good for testing)
3. **Btrfs Backend** (refactor existing code)
4. **LVM-Thin Backend** (most complex, but most important)
5. **Backend Detection** (tie everything together)
6. **Integration** (update machine manager and snapshot manager)
7. **Testing** (verify all backends work)
8. **UX improvements** (setup commands, migration tools)

### Backward Compatibility

- Existing machines on thick LVs should continue to work
- Add deprecation warning for thick LVs
- Provide migration path to thin volumes
- Don't break existing btrfs setups

### Success Criteria

- [ ] All three backends (btrfs, lvm-thin, directory) work correctly
- [ ] Automatic backend selection based on filesystem
- [ ] Existing machines continue to work
- [ ] No data loss during migration
- [ ] Clear documentation for users
- [ ] All unit tests pass
- [ ] Integration tests for all backends

**Phase 6 (V2+) - LVM Snapshot Support - COMPLETE ✅**

- ✅ LVM backend enabled and tested on Debian 13 (trixie)
- ✅ Per-machine LV creation (each machine gets its own logical volume)
- ✅ Automatic LV formatting (ext4) and mounting
- ✅ LVM snapshot create/list/delete all working
- ✅ LVM snapshot rollback working (merge on next activation)
- ✅ Machine delete properly cleans up LVM volumes
- ✅ Sudoers rules updated for LVM commands (lvs, lvcreate, lvconvert, lvremove, mkfs.ext4, mount, umount, mountpoint, chmod)
- ⏸️ ZFS backend still disabled (untested)

**Phase 4.6 (V2) - Cleanup Improvements - COMPLETE ✅**

- ✅ Fixed lingering network interfaces after container deletion
- ✅ Fixed stale entries in machinectl after crashes
- ✅ Added ResetFailedUnit to clear failed service state
- ✅ Added machinectl terminate to clean up systemd-machined registrations
- ✅ Improved Delete method to handle crashed services gracefully
- ✅ Improved Stop method to fall back to comprehensive cleanup
- ✅ All cleanup steps log warnings but continue on failure

**Phase 4 (V2) - Templates - COMPLETE ✅**

- ✅ Docker-in-container template fully working (fixed crash with PrivateUsers=no in .nspawn)
- ✅ Base templates tested (Debian, Ubuntu, Fedora)
- ✅ Adaptive network mode implemented and working
- ✅ Host-side macvlan connectivity working
- ✅ EOVERFLOW fix applied (using PrivateUsers=identity for correct UID mapping)
- ✅ Pi-hole template installs and runs successfully (was blocked by EOVERFLOW)
- ✅ AdGuard template installs and runs successfully (was blocked by EOVERFLOW)
- ✅ Samba template installs and runs successfully (was blocked by EOVERFLOW)
- ✅ Nextcloud template installs successfully (was blocked by EOVERFLOW)
- ✅ Jellyfin template GPU pass-through tested and working (Intel iGPU)
- ✅ Plex template GPU pass-through tested and working (Intel iGPU, vainfo shows full codec support)
- ✅ Jellyfin repository GPG key URL fixed (changed to https://repo.jellyfin.org/jellyfin.gpg)
- ✅ Macvlan/ipvlan crash fixed (using PrivateUsers=no for these modes)
- ❌ Arch Linux not supported (btrfs/pacman incompatibility)
- ✅ Added unit tests for provision/exec.go, image/fetcher.go, kernel/systemd.go (332 tests total)

**Phase 4.5 (V2) - Code Quality & Security - COMPLETE ✅**

Critical Issues Fixed:
- ✅ Path traversal in data.go - Added validatePath() function to ensure paths stay under rootfs
- ✅ Path traversal in yq.go - Uses same validatePath() function
- ✅ Command injection in hook scripts - Added regex validation for interface names
- ✅ Destructive edit without rollback - Added rootfs backup before deletion with restore on failure
- ✅ Tar extraction TOCTOU race condition - Path traversal checks already prevent symlink attacks

High Issues Fixed:
- ✅ Resource leaks in snapshot commands - Added defer mgr.Close() to all snapshot commands
- ✅ Machine name validation - Strict regex: ^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]$
- ✅ Snapshot name validation - Strict regex with dots allowed: ^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,62}[a-zA-Z0-9]$
- ✅ Volume path validation - Added sensitive path blacklist (/etc/shadow, /proc, /sys, etc.)
- ✅ Port range validation - Validates ports are in range 1-65535
- ✅ GPU device path validation - Requires paths to start with /dev/
- ✅ Nil pointer dereference in provision exec - Added default case returning error
- ✅ Download timeout - Set to 30 minutes instead of infinite
- ✅ Cache directory permissions - Changed from 0755 to 0700

**Phase 3 (V2) - Snapshots - Complete (btrfs + LVM)**
- ✅ Snapshot thin wrapper works (btrfs and LVM)
- ✅ Tested on Bazzite/Fedora 44 (btrfs)
- ✅ Tested on Debian 13 trixie (LVM with ext4)
- ✅ Automatic btrfs subvolume creation
- ✅ Automatic LVM volume creation (per-machine LV)
- ✅ All snapshot commands working (create, list, delete)
- ✅ Rollback tested with nested subvolumes (btrfs)
- ✅ Rollback tested with LVM merge (LVM)
- ✅ Machine delete handles child subvolumes correctly (btrfs)
- ✅ Machine delete cleans up LVM volumes correctly (LVM)
- ⏸️ ZFS backend disabled (untested, moved to Phase 6)

**Phase 2 (V1.5) - Complete ✅**
- Provisioning engine (boot, data, yq, system, user modes)
- Template variables and rendering
- Readiness probes with result tracking
- Host-side lifecycle hooks
- GPU pass-through (all vendors: Intel, AMD, NVIDIA, ARM Mali, Adreno)
- Script hash tracking for idempotent execution
- Enhanced YQ expression engine (supports `|=`, `+=`)
- Download client with timeout and size limits
- Improved data mode ownership (reads container rootfs)
- All CLI commands (show, edit, provision)
- Adaptive network mode with host access
- All networking modes tested and working

**Phase 1 (V1) - Complete ✅**
- Basic machine lifecycle
- Resource limits via slices
- All networking modes
- Bind mounts and volumes
- JSON output
- Polkit rules

---

## Phase 3: Snapshot Support (V2) - Complete ✅ (btrfs + LVM)

### Snapshot Support

**Design philosophy:** Thin wrapper around existing tools (btrfs, LVM).

**Status:** Complete for btrfs and LVM. ZFS moved to Phase 6.

**Implementation:**
1. Detect filesystem type of `/var/lib/machines/<name>/`
2. If btrfs: use `btrfs subvolume snapshot`
3. If LVM (ext4 on LV): use `lvcreate --snapshot`
4. Otherwise: error with helpful message

**CLI commands:**
- ✅ `lxspawn snapshot <name> <snapname>`
- ✅ `lxspawn list-snapshots <name>`
- ✅ `lxspawn rollback <name> <snapname>`
- ✅ `lxspawn delete-snapshot <name> <snapname>`

**Features:**
- ✅ Automatic btrfs subvolume creation for machine rootfs
- ✅ Read-only snapshots for btrfs
- ✅ JSON output for all snapshot commands
- ✅ State tracking in `~/.local/share/lxspawn/<name>/snapshots.json`
- ✅ Sudoers rules for btrfs/zfs/lvm commands
- ✅ Recursive deletion of child subvolumes (snapshots)
- ✅ Rollback with proper nested subvolume handling

---

## Phase 4: Template Creation & Testing (V2) - COMPLETE ✅

### Goal

Create and test turnkey templates for common LXC use cases.

### Completion Criteria

Phase 4 is **COMPLETE**:
1. [x] Plex template is tested with GPU pass-through
2. [x] Jellyfin repository GPG key URL is fixed
3. [x] All template testing checklists are verified (see below)

### Templates Created (10 total)

- ✅ docker-host.yaml + docker-host.md
- ✅ pihole.yaml + pihole.md
- ✅ adguard.yaml + adguard.md
- ✅ debian-base.yaml + debian-base.md
- ✅ ubuntu-base.yaml + ubuntu-base.md
- ✅ fedora-base.yaml + fedora-base.md
- ✅ jellyfin.yaml + jellyfin.md (with GPU support)
- ✅ plex.yaml + plex.md (with GPU support)
- ✅ samba.yaml + samba.md
- ✅ nextcloud.yaml + nextcloud.md
- ❌ arch-base.yaml + arch-base.md (removed: btrfs/pacman incompatibility)

### Testing Status

**Tested and Working:**
- ✅ debian-base: Created, started, executed commands, provisioning worked
- ✅ ubuntu-base: Created, started, executed commands, provisioning worked
- ✅ fedora-base: Fixed distro version (40 → 44), extraction works
- ✅ docker-host: Docker-in-container fully working (pull, run, hello-world test passed)
- ✅ pihole: DNS server installs and runs (EOVERFLOW fix applied, added dnsutils for probe)
- ✅ adguard: DNS server installs and runs (EOVERFLOW fix applied, added dnsutils for probe)
- ✅ samba: File server installs and runs (EOVERFLOW fix applied, bind mount path must exist)
- ✅ nextcloud: LAMP stack installs successfully (EOVERFLOW fix applied, added bzip2 for extraction)
- ✅ jellyfin: GPU pass-through tested and working (Intel iGPU on test machine, vainfo shows full codec support)
- ✅ plex: GPU pass-through tested and working (Intel iGPU on test machine, vainfo shows full codec support)

**Template Fixes Applied:**
- ✅ pihole.yaml: Added `dnsutils` to apt-get install (needed for `dig` in readiness probe)
- ✅ adguard.yaml: Added `dnsutils` to apt-get install (needed for `dig` in readiness probe)
- ✅ nextcloud.yaml: Added `bzip2` to apt-get install (needed for `tar -xjf` extraction)
- ✅ jellyfin.yaml: Added Intel GPU driver installation (intel-media-va-driver, vainfo), fixed GPG key URL
- ✅ plex.yaml: Tested and working with GPU pass-through

**Notes:**
- Bind mount host paths must exist before launching (samba: /mnt/data, nextcloud: /var/lib/nextcloud)
- Adaptive network mode works correctly with macvlan fallback and host-side interface creation
- All templates using adaptive mode tested successfully on test machine (192.168.0.200)
- GPU pass-through verified on test machine (Intel UHD Graphics 600, Gemini Lake)
- Jellyfin repository GPG key URL fixed (changed to https://repo.jellyfin.org/jellyfin.gpg)

**Bugs Fixed:**
- ✅ Fixed tar extraction permission denied for read-only directories (added 0700 to directory mode)
- ✅ Fixed host-side macvlan IP assignment (now generates IP in same subnet as host)
- ✅ Fixed tasks field to support "infinity" value (changed from int to string)
- ✅ Fixed Docker-in-container support:
  - Added `Delegate=yes` for cgroup delegation
  - Added `PrivateUsers=no` in .nspawn file to allow nested namespaces (moved from ExecStart override)
  - Added `CAP_BPF` capability for BPF operations
  - Configured Docker to use cgroupfs driver instead of systemd
  - Fixed GPG key installation with `--batch --no-tty` flags
  - Added D-Bus environment variables to nsenter commands
- ✅ Fixed EOVERFLOW error during package installation on btrfs:
  - Root cause: `-U` flag was using mapped UIDs instead of identity mapping
  - Fix: Changed to `PrivateUsers=identity` in .nspawn file for correct UID translation
  - Removed `fixRootfsOwnership` (no longer needed with identity mapping)
  - Added kernel version check (5.12+) for btrfs idmapped mounts support
  - Created `internal/kernel/version.go` and `internal/kernel/systemd.go` for version detection
- ✅ Fixed Docker-in-container crash:
  - Root cause: ExecStart override in drop-in replaced entire systemd-nspawn command, conflicting with Type=notify
  - Fix: Moved `PrivateUsers=no` from ExecStart override to .nspawn file
  - Removed all ExecStart overrides from drop-in (no longer needed)
- ✅ Fixed macvlan/ipvlan crash:
  - Root cause: `PrivateUsers=identity` incompatible with macvlan/ipvlan networking
  - Fix: Use `PrivateUsers=no` for macvlan/ipvlan modes (no user namespace)
  - Added automatic detection in compiler to select appropriate PrivateUsers value

**Remaining:**
- [x] Deploy updated binary to test machine (192.168.0.200)
- [x] Run `sudo lxspawn setup network`
- [x] Update templates to use adaptive mode
- [x] Fix host-side macvlan IP assignment (now generates IP in same subnet as host)
- [x] Fix tar extraction for read-only directories
- [x] Fix Docker-in-container support (all issues resolved)
- [x] Fix rootfs ownership (chown root:lxspawn + chmod g+rwX)
- [x] Fix EOVERFLOW error (PrivateUsers=identity)
- [x] Fix Docker-in-container crash (PrivateUsers=no in .nspawn)
- [x] Fix macvlan/ipvlan crash (PrivateUsers=no for these modes)
- [x] Test most templates with EOVERFLOW fix (jellyfin, samba, nextcloud)
- [x] Test GPU pass-through on jellyfin template
- [ ] **Test plex template with GPU pass-through (BLOCKING)**
- [ ] **Fix Jellyfin repository GPG key URL (BLOCKING)**

### Template Categories

#### 1. Docker-in-Container (High Priority) - ✅ COMPLETE

**Requirements:**
- Bridge networking (or adaptive mode with macvlan fallback)
- `pre_start` hook: `ip_forward`, `br_netfilter`, `bridge-nf-call-iptables`
- Seccomp disabled OR specific syscalls: `add_key`, `keyctl`, `bpf`
- `TasksMax=infinity`
- `Delegate=yes`
- `--private-users=no` for nested namespaces
- `CAP_BPF` for cgroup device control

**Testing checklist:**
- [x] Can pull and run containers
- [x] Docker networks work (bridge, host)
- [ ] Docker volumes persist
- [ ] Can build images (docker build)
- [ ] Nested containers work (docker-in-docker)

#### 2. DNS Servers - Pi-hole / AdGuard (High Priority)

**Requirements:**
- macvlan OR bridge networking
- Port 53 TCP/UDP
- Static IP or DHCP reservation

**Testing checklist:**
- [ ] DNS queries work from LAN clients
- [ ] Web admin interface accessible
- [ ] Blocklists update correctly
- [ ] DHCP works (if enabled)

#### 3. Media Servers - Plex / Jellyfin (Medium Priority)

**Requirements:**
- GPU device bind-mount (now automated via `gpu:` section)
- Large storage for media library
- Adequate resources (2+ CPUs, 4G+ RAM)

**Testing checklist:**
- [ ] GPU detected inside container
- [ ] Hardware transcoding works
- [ ] Media library accessible
- [ ] Web interface accessible from LAN

#### 4. File Servers - Nextcloud / Samba (Medium Priority)

**Requirements:**
- Large bind mounts for data
- Proper UID/GID mapping
- Samba needs specific capabilities

**Testing checklist:**
- [ ] Files accessible from LAN clients
- [ ] Permissions preserved correctly
- [ ] Write performance acceptable
- [ ] User authentication works

#### 5. Base Systems (High Priority)

**Distros to test:**
- [ ] Debian bookworm (stable)
- [ ] Debian trixie (testing)
- [ ] Ubuntu noble (24.04)
- [ ] Ubuntu jammy (22.04)
- [ ] Fedora 40
- [ ] Alpine Linux (if systemd available)

**Note:** Arch Linux is not supported on btrfs hosts due to filesystem compatibility issues (EOVERFLOW error during package operations).

**Testing checklist:**
- [ ] Image downloads successfully
- [ ] systemd starts correctly
- [ ] Network connectivity works
- [ ] Package manager works
- [ ] Journal integration works
- [ ] Shell access works

### Recommendations for Phase 4 Completion

In priority order:

1. **Add tests for `provision/exec.go` and `image/fetcher.go`**
   - These are the most bug-prone paths (nsenter command building, tar extraction, btrfs subvolume creation)
   - The tar extraction bug (read-only directories) would have been caught by a unit test

2. **Test remaining templates** (pihole, adguard, jellyfin, plex, samba, nextcloud) on the test machine
   - Verify adaptive mode works for all templates
   - Test GPU pass-through on jellyfin/plex

3. **Refactor `compileDropIn()`** to make the ExecStart logic clearer and more testable
   - Currently has multiple conditional paths (macvlan, ipvlan, seccomp-disabled)
   - Consider extracting ExecStart logic into its own function with clear test cases

4. **Consider a boot-script approach for Docker service startup**
   - Currently requires manually starting `dockerd` in the readiness probe
   - The boot provisioner writes scripts to `<rootfs>/etc/lxspawn/boot/` -- this is more reliable for service management
   - Would make Docker-in-container less fragile

5. **Create a simple integration test script**
   - Automate the create/start/probe/stop/delete cycle on the test machine
   - Could be a shell script that runs `lxspawn` commands and verifies expected state
   - Would catch regressions before they reach users

6. **Address EOVERFLOW warnings**
   - "Value too large for defined data type" warnings appear in apt-get (not just pacman)
   - Indicates deeper btrfs/container interaction issue
   - Cosmetic for now but could become a real problem with future package manager versions

---

## Phase 4.5: Code Quality & Security Issues - COMPLETE ✅

**Status:** Complete. All critical and high priority issues addressed (2026-10-12).

**Priority:** High - Security vulnerabilities addressed before Phase 5 (Cockpit UI).

### Critical Issues (5) - ALL FIXED ✅

1. **Path Traversal in Data Provisioning** (`internal/provision/data.go:41`) - FIXED
   - Added `validatePath()` function to ensure paths stay under rootfs
   - Validates absolute paths and prevents directory traversal

2. **Path Traversal in YQ Provisioning** (`internal/provision/yq.go:24`) - FIXED
   - Uses same `validatePath()` function from data.go

3. **Command Injection in Hook Scripts** (`internal/compiler/slice.go:169-193`) - FIXED
   - Added regex validation for interface names: `^[a-zA-Z0-9_-]+$`
   - Invalid interface names now produce safe error messages instead of shell commands

4. **Destructive Edit Without Rollback** (`cmd/lxspawn/edit.go:63-69`) - FIXED
   - Added rootfs backup before deletion (renamed to .backup-<timestamp>)
   - Automatic restore on launch failure
   - Cleanup of backup on success

5. **Unsafe Tar Extraction** (`internal/image/fetcher.go:216-273`) - FIXED
   - Path traversal checks already prevent symlink attacks
   - Symlink targets validated to stay within destination directory

### High Issues (12) - ALL FIXED ✅

6. **Resource Leaks in Snapshot Commands** - FIXED
   - Added `defer mgr.Close()` to all snapshot commands (snapshot.go, list_snapshots.go, rollback.go, delete_snapshot.go)
   - Added Close() method to snapshot.Manager for API consistency

7. **Insufficient Machine Name Validation** - FIXED
   - Added strict regex: `^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]$`
   - Prevents dangerous characters like `;`, `|`, `&`, `$`, backticks

8. **Insufficient Snapshot Name Validation** - FIXED
   - Added strict regex with dots: `^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,62}[a-zA-Z0-9]$`
   - Dots allowed for versioned snapshot names (e.g., "snap.v1")

9. **Unvalidated Volume Paths** - FIXED
   - Added sensitive path blacklist: /etc/shadow, /etc/gshadow, /proc, /sys, /dev, /run, /var/run
   - Prevents exposing critical system directories to containers

10. **Insecure Default Network Mode** - NOT CHANGED
    - Host mode is the default for rootless operation
    - Changing to private would break existing workflows
    - Documented trade-off in README

11. **Missing Port Range Validation** - FIXED
    - Added port range validation (1-65535) for both host and container ports
    - Invalid ports like 99999 now rejected

12. **Missing Input Validation for Resource Values** - NOT CHANGED
    - Resource values passed directly to systemd
    - systemd validates these at runtime
    - Adding regex validation would be fragile and duplicate systemd's validation

13. **Nil Pointer Dereference in Provision Exec** - FIXED
    - Added default case returning error for unsupported provision modes

14. **Hardcoded DNS Servers** - NOT CHANGED
    - Public DNS (8.8.8.8, 1.1.1.1, 8.8.4.4) used for initial container boot
    - Containers can override via systemd-networkd or resolv.conf
    - Making configurable would add complexity for minimal benefit

15. **File Cache Directory Permissions** - FIXED
    - Changed cache directory permissions from 0755 to 0700
    - Prevents other users from reading cached scripts

16. **Race Condition in State Manager** - NOT CHANGED
    - Nested locking is intentional for different synchronization scopes
    - Mutex protects in-memory state, file lock protects persistence
    - Documented in ARCHITECTURE.md

17. **No Timeout on Downloads** - FIXED
    - Set download timeout to 30 minutes
    - Prevents indefinite hangs on slow/unreachable servers

### Medium Issues (15)

18. **Ignored Errors from User Input** (`cmd/lxspawn/delete.go:25`, `delete_snapshot.go:28`, `rollback.go:30`, `setup.go:231`)
    - `reader.ReadString('\n')` errors ignored
    - Fix: Return error if stdin read fails

19. **Silently Ignored File Read Errors** (`cmd/lxspawn/show.go:45-48, 55-58`)
    - `os.ReadFile()` errors ignored with `continue`
    - Fix: Return error if file exists but is unreadable

20. **Ignored Input Parsing Error** (`cmd/lxspawn/setup.go:236`)
    - `fmt.Sscanf()` error ignored
    - Invalid input silently uses default
    - Fix: Return error for non-numeric input

21. **Missing Error Context** (`cmd/lxspawn/setup.go:158`)
    - chmod error returned without context
    - Fix: Add error wrapping with context

22. **Incomplete Setup Error Handling** (`cmd/lxspawn/setup.go:67-71, 92-104`)
    - Critical setup steps only log warnings on failure
    - Reports success even when essential components fail
    - Fix: Track failures and return error if any critical step fails

23. **Code Duplication** (Multiple command files)
    - Common patterns repeated: manager creation, JSON output, confirmation prompts
    - Fix: Extract into helper functions

24. **Improper Error Wrapping** (Multiple files)
    - Several places use `%v` instead of `%w` for error wrapping
    - Loses error chain context
    - Fix: Use `%w` verb consistently

25. **TOCTOU in File Cache** (`internal/provision/file.go:61-70`)
    - Check-then-use pattern for cached files
    - Fix: Open file first, validate digest on file descriptor

26. **Insecure Temporary File Location** (`internal/image/fetcher.go:176`)
    - Uses default temp directory (shared filesystem)
    - Vulnerable to symlink attacks
    - Fix: Use dedicated temp directory with restricted permissions

27. **Missing Validation on File Permissions** (`internal/provision/data.go:60-67`)
    - User-specified permissions not validated for dangerous bits
    - Could set setuid/setgid or world-writable
    - Fix: Reject setuid/setgid/sticky bits and world-writable

28. **Missing Validation on GPU Device Paths** (`internal/config/parser.go:264-268`)
    - Only checks for non-empty
    - Could bind-mount arbitrary host files
    - Fix: Validate paths start with `/dev/`

29. **Missing Validation on Security Capabilities** (`internal/compiler/nspawn.go:63-69`)
    - Capabilities passed directly to systemd without validation
    - Dangerous capabilities like CAP_SYS_ADMIN could allow escape
    - Fix: Maintain whitelist, warn for dangerous capabilities

30. **Magic Numbers** (Multiple files)
    - Hardcoded numbers without named constants
    - Examples: `maxAttempts := 30`, `lastOctet := (hash % 254) + 1`
    - Fix: Define named constants

31. **Missing Comments on Complex Logic** (Multiple files)
    - Complex logic lacks documentation
    - Examples: YQ expression parsing, D-Bus name encoding
    - Fix: Add comments explaining non-obvious operations

32. **Inconsistent Error Handling** (Multiple files)
    - Some errors logged and ignored, others returned
    - Makes code unpredictable
    - Fix: Establish clear policy and document

### Low Issues (13)

33. **Inconsistent Terminal Detection** (`cmd/lxspawn/delete.go:59-63`)
    - `isStdinTerminal()` returns false on error
    - Could skip confirmation unintentionally
    - Fix: Log error or be more explicit

34. **Editor Command Validation** (`cmd/lxspawn/edit.go:36-41`)
    - EDITOR env var used without validation
    - Standard Unix behavior but could be safer
    - Fix: Validate editor exists in PATH

35. **Inefficient IP Parsing** (`internal/compiler/compiler.go:165-175`)
    - Uses `fmt.Sscanf` instead of `net.ParseCIDR`
    - Slower and less robust
    - Fix: Use standard library

36. **GPU Detection Errors Silently Ignored** (`internal/compiler/nspawn.go:103-110`)
    - Errors completely ignored
    - Container starts without GPU, no indication of failure
    - Fix: Log warning at minimum

37. **Code Duplication in GPU Detection** (`internal/compiler/nspawn.go:137-154`, `slice.go:78-99`)
    - Nearly identical GPU detection logic
    - Fix: Extract common logic or cache result

38. **Missing Error Context in State Save** (`internal/provision/engine.go:92-94, 178-180`)
    - State save failures only logged as warnings
    - Fix: Return error or log at higher level

39. **XZ Reader Close Issues** (`internal/image/xz.go:38-41`)
    - Close() doesn't handle multiple calls
    - Fix: Add closed flag

40. **Btrfs Subvolume Detection Fragile** (`internal/machine/manager.go:295-301`)
    - Relies on string matching of command output
    - Fix: Use exit codes or more robust detection

41. **Unused Interface Methods** (`internal/deploy/interfaces.go:7-18`)
    - SystemdClient interface has potentially unused methods
    - Fix: Review and remove unused methods

42. **Missing Rate Limiting on Downloads** (`internal/provision/file.go:57-116`)
    - No rate limiting on downloads
    - Fix: Implement rate limiting

43. **Missing Content-Type Validation** (`internal/provision/file.go:82-90`)
    - HTTP response content-type not validated
    - Fix: Validate Content-Type header

44. **Sensitive Information in Logs** (`internal/provision/exec.go:50`)
    - Provision script execution logged
    - Fix: Make verbose logging configurable

45. **Error Messages May Leak Paths** (`internal/provision/file.go:50, 84, 110`)
    - Full file paths in error messages
    - Fix: Sanitize error messages for user display

### Implementation Priority

**Immediate (Critical - Security):**
1. Fix path traversal in data.go and yq.go
2. Add input validation for machine/snapshot names
3. Sanitize hook script inputs
4. Add rollback protection to edit command
5. Fix tar extraction race condition

**High Priority:**
6. Add `defer mgr.Close()` to snapshot commands
7. Add port range validation
8. Add resource value validation
9. Fix nil pointer dereference in provision exec
10. Make DNS configurable
11. Fix cache directory permissions
12. Add HTTP timeout

**Medium Priority:**
13. Fix ignored errors in user input
14. Improve error wrapping consistency
15. Extract common code patterns
16. Add path validation for volumes and GPU devices
17. Add capability whitelist

**Low Priority:**
18. Replace magic numbers with constants
19. Add missing comments
20. Improve error messages

---

## Phase 5: Cockpit Web UI (V2+)

### Cockpit Plugin

**Status:** Ready to start. JSON API prerequisites complete (Phase 4.11).

**Design:** Thin wrapper over `lxspawn --json` commands, mirroring cockpit-machines UX.

**Key principles:**
1. Framework agnostic (don't lock into React)
2. Thin wrapper (all data via `cockpit.spawn(["lxspawn", "command", "--json"])`)
3. Leverage Cockpit built-ins (terminal, journal, file APIs)
4. Mirror cockpit-machines UX patterns

**JSON API Coverage:** ~95% complete (Phase 4.11)
- ✅ Machine list with state, network, resources
- ✅ Machine status with leader PID, files, ephemeral/read-only flags
- ✅ Runtime stats (CPU, memory, disk) - all working
- ✅ Detailed config (volumes, network, security, GPU, advanced)
- ✅ Provisioning status (last run, phases, probes)
- ✅ Lifecycle operations (start, stop, delete, provision)
- ✅ Snapshot operations (create, list, rollback, delete)
- ✅ Generated config files (show command)
- ✅ Configuration editing (edit-resources, edit-network, add/edit/remove-volume)
- ✅ Distro list for create wizard
- ✅ Template list for create wizard
- ✅ YAML validation
- ✅ Structured logs (using journalctl -u with sudoers rule)

**All JSON API prerequisites complete! Ready to start Cockpit plugin development.**

---

## Phase 6: ZFS/LVM Snapshot Support (V2+) - DEFERRED

### Goal

Test and enable ZFS and LVM snapshot backends that are currently disabled.

### Status

**Deferred to after Phase 5 (Cockpit Web UI).**

**Rationale:**
- Directory mode snapshots removed (unreliable, slow, not true snapshots)
- LVM and ZFS backends disabled pending proper testing infrastructure
- Current testing environment (HDD RAID VMs) too slow for reliable testing
- Focus on Cockpit UI first, then return to storage backend testing

### Tasks (Deferred)

- [ ] Set up fast SSD-based test environment for LVM testing
- [ ] Test LVM thin pool snapshots (create, list, rollback, delete)
- [ ] Document LVM backend-specific issues found
- [ ] Re-enable LVM backend in `internal/snapshot/detector.go`
- [ ] Set up test environment with ZFS filesystem
- [ ] Test ZFS snapshots (create, list, rollback, delete)
- [ ] Document ZFS backend-specific issues found
- [ ] Re-enable ZFS backend in `internal/snapshot/detector.go`
- [ ] Update tests in `internal/snapshot/detector_test.go`
- [ ] Update documentation to reflect full snapshot support

### Current State

**Snapshots require btrfs filesystem.**

**Removed:**
- Directory mode snapshot support (unreliable, slow, not true snapshots)
- LVM backend (disabled, untested in current environment)
- ZFS backend (disabled, untested)

**Why directory mode was removed:**
- Snapshots used rsync (file-by-file copy), not block-level
- Not atomic - snapshots of running machines were inconsistent
- Slow for large rootfs
- Used full disk space (no copy-on-write)
- Not a true "snapshot" - more like a "backup of a moving target"

**Future requirements:**
- LVM: Requires thin pool setup by user (lxspawn will not modify user filesystem)
- ZFS: Requires ZFS dataset for machine rootfs
- Both: Require proper testing infrastructure (SSD-backed VMs)

### Implementation Notes (For Future Reference)

LVM backend (`internal/snapshot/lvm.go`):
- Uses `lvcreate --snapshot`, `lvs`, `lvconvert --merge`, `lvremove`
- Each machine gets its own logical volume (e.g., `debian-vg/lxspawn-<machinename>`)
- LV is formatted with ext4 and mounted at `/var/lib/machines/<name>/`
- Note: LVM snapshot merge requires next activation (not immediate)
- Note: Cannot create new snapshots while a merge is pending
- Snapshot size: 512MB (configurable in code)
- Machine LV size: 4GB (configurable in code)

ZFS backend (`internal/snapshot/zfs.go`):
- Uses `zfs snapshot`, `zfs list`, `zfs rollback`, `zfs destroy`
- Requires ZFS dataset for machine rootfs

---

## Future Enhancements (Not Yet Scheduled)

### Host-Side Macvlan: Static IP vs DHCP

**Current state:** `host_access.ip` field exists but not fully implemented.

**Options:**
1. **Static IP** (simple, default) - User provides IP, assigned in post_start hook
2. **DHCP** (complex) - Requires DHCP client, IP changes on reboot
3. **DHCP reservation** (recommended for production) - Configure on router

**Implementation tasks:**
- [ ] Implement static IP assignment in post_start hook
- [ ] Add validation for IP format (CIDR notation)
- [ ] Document DHCP reservation setup
- [ ] Future: Implement DHCP client support

### Rebuild Workflow

- [ ] `lxspawn rebuild <name>` - delete + launch (with confirmation)
- [ ] Preserve volumes if configured

### Alternative Image Sources

**Proxmox Community Images** (`http://download.proxmox.com/images/system/`)
- Well-maintained: Debian, Ubuntu, Fedora, Arch, Alpine, CentOS/Rocky/Alma
- Format: `tar.zst` (newer), `tar.xz`/`tar.gz` (older)
- Implementation needs:
  - [ ] zstd decompression support
  - [ ] Different URL builder (flat directory)
  - [ ] Filename parser for version discovery

**TurnKey Linux Images** (`https://mirror.turnkeylinux.org/turnkeylinux/images/proxmox/`)
- Pre-built appliance images (LAMP, WordPress, Nextcloud, etc.)
- Same format as Proxmox images

### Export/Backup

- [ ] `lxspawn export <name>` - `machinectl export-tar`
- [ ] `lxspawn import <archive>` - `machinectl import-tar`
- [ ] Compressed archives (`.tar.zst`)

### User Namespaces (Advanced)

- [ ] `PrivateUsers=yes` for UID mapping
- [ ] `BindUser=` for host user access
- [ ] Rootless operation with limitations

---

## Testing

### Unit Tests (404 tests, all passing)

- Config parsing and validation (64 tests) - NEW: storage size parsing and validation
- Compiler output (27 tests) - Added tests for Docker-in-container, tasks infinity, IP generation
- State management (18 tests)
- Filesystem helpers (21 tests)
- Template rendering (20 tests)
- YQ provisioner (20 tests)
- Boot provisioner (8 tests)
- Data provisioner (11 tests)
- File handler (12 tests)
- Image cleanup (7 tests)
- Image validation (7 tests)
- Image network injection (10 tests)
- Image fetcher (8 tests) - NEW: URL building, tar extraction, path traversal protection
- Shell quoting (15 tests)
- Rootfs population check (4 tests) - NEW: isPopulatedRootfs tests
- LVM helpers (9 tests) - NEW: isMounted, IsLVM, getVGName
- GPU detection (10 tests)
- Snapshot validation and types (10 tests)
- Snapshot backend detection (10 tests)
- Snapshot state serialization (5 tests)
- Provision exec (13 tests) - NEW: env export, provisioner modes, param handling
- Kernel version (5 tests)
- Kernel systemd (2 tests) - NEW: SystemdVersion, SupportsPrivateUsersIdentity

### Integration Tests

- [x] Full lifecycle (create → start → shell → stop → delete)
- [x] List command shows actual state via D-Bus
- [x] Show command displays generated config
- [x] Exec command properly quotes arguments
- [x] Provision command full cycle
- [x] Lock workaround (SYSTEMD_NSPAWN_LOCK=0)
- [x] Volume mounts (read-only, read-write, tmpfs)
- [x] Host-side hooks
- [x] Post-boot provisioning (system/user modes)
- [x] Snapshot lifecycle (create → list → rollback → delete)
- [x] Snapshot rollback preserves pre-snapshot state
- [x] Machine delete handles child subvolumes correctly
- [x] Cleanup of crashed machines (ResetFailedUnit + machinectl terminate)
- [x] No lingering entries in machinectl after delete
- [x] Network interfaces cleaned up after container stop/delete
- [x] Snapshot rollback preserves pre-snapshot state
- [x] Machine delete with snapshots (recursive subvolume cleanup)
- [ ] Networking modes (host, private, bridge, zone)
- [ ] Resource limits (verify cgroup values)

### Manual Testing

- [x] Test on Bazzite/Fedora 44
- [ ] Test with different images (Debian, Ubuntu, Fedora)
- [ ] Test edge cases (large rootfs, many machines, concurrent operations)

---

## Known Issues

### Fixed Issues

- ✅ **D-Bus start authentication** - Fixed with system D-Bus connection and polkit rules
- ✅ **systemd-nspawn lock files** - Fixed with `SYSTEMD_NSPAWN_LOCK=0` environment variable
- ✅ **machinectl shell PTY errors** - Fixed by using `nsenter` for non-interactive execution
- ✅ **DNS resolution in containers** - Fixed by writing working `/etc/resolv.conf` with public DNS
- ✅ **Host networking mode** - Fixed by generating `Private=no` for host mode
- ✅ **Permission issues on /var/lib/machines** - Fixed with correct group ownership
- ✅ **EOVERFLOW during package installation on btrfs** - Fixed by using `PrivateUsers=identity` instead of `-U` for correct UID mapping
- ✅ **Docker-in-container crash** - Fixed by moving `PrivateUsers=no` from ExecStart override to .nspawn file
- ✅ **Macvlan/ipvlan crash** - Fixed by using `PrivateUsers=no` for these network modes
- ✅ **Lingering network interfaces after delete** - Fixed by comprehensive cleanup: ResetFailedUnit + machinectl terminate
- ✅ **Stale entries in machinectl** - Fixed by calling machinectl terminate during delete/stop

### Open Issues

- **macvlan host visibility** - Host cannot ping macvlan containers (kernel limitation). Mitigated by `host_access` feature.
- **Container network interface naming** - macvlan creates `mv-<interface>`, ipvlan creates `iv-<interface>`. Documented.

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Networking complexity | High | Start with `host` mode (rootless), document root requirement |
| Image compatibility | Medium | Test images early, validate init system |
| Privilege model confusion | Medium | Clear error messages, document which commands need root |
| systemd version differences | Low | Test on multiple distros, document minimum (245+) |
| Slice ordering | Medium | Ensure slice created before machine start |
| btrfs/zfs requirement for snapshots | Low | Thin wrapper, rotation is user's responsibility |
| D-Bus API changes | Low | Use stable machine1 interface |
| User namespace complexity | Medium | Start with `PrivateUsers=no` |
| NetworkManager conflicts | Low | Document manual bridge setup |
| Provisioning idempotency | Medium | Track script hashes, document requirement |
| Boot script ordering | Low | Number scripts sequentially |
| Template rendering errors | Low | Validate templates, show clear errors |

---

## Success Criteria

### V1.5 (Current)

- [x] Provisioning engine (all modes)
- [x] Template variables render correctly
- [x] File references download and validate
- [x] Post-boot provisioning works reliably
- [x] Readiness probes work reliably
- [x] All CLI commands implemented
- [x] All networking modes work
- [x] Idempotent re-runs
- [x] Host-side hooks work
- [x] GPU pass-through works
- [ ] Comprehensive documentation

### V2 (Phase 3: Snapshots)

- [x] Snapshot thin wrapper works (btrfs + LVM)
- [x] Tested on Bazzite/Fedora 44 (btrfs)
- [x] Tested on Debian 13 trixie (LVM)
- [x] Automatic btrfs subvolume creation
- [x] Automatic LVM volume creation (per-machine LV)
- [x] All snapshot commands working (create, list, delete)
- [x] Rollback tested with nested subvolumes (btrfs)
- [x] Rollback tested with LVM merge (LVM)
- [x] Machine delete handles child subvolumes correctly (btrfs)
- [x] Machine delete cleans up LVM volumes correctly (LVM)

### V2+ (Phase 6: ZFS/LVM Snapshots) - DEFERRED

- [ ] Set up fast SSD-based test environment for LVM testing
- [ ] Test LVM thin pool snapshots (create, list, rollback, delete)
- [ ] Re-enable LVM backend in detector.go
- [ ] Set up test environment with ZFS filesystem
- [ ] Test ZFS snapshots (create, list, rollback, delete)
- [ ] Re-enable ZFS backend in detector.go

**Note:** Directory mode snapshots removed (unreliable, slow). Snapshots now require btrfs filesystem.

### V2 (Phase 4: Templates) - COMPLETE ✅

- [x] Docker-in-container template works
- [x] DNS server templates work (pihole, adguard - tested and working)
- [x] Media server templates work (jellyfin and plex tested with GPU pass-through)
- [x] File server templates work (samba, nextcloud - tested and working)
- [x] Base system templates tested (Debian, Ubuntu, Fedora)
- [x] Rootfs ownership fix applied
- [x] EOVERFLOW limitation resolved (using PrivateUsers=identity)
- [x] All templates tested on test machine
- [x] Unit tests added for provision/exec.go, image/fetcher.go, kernel/systemd.go
- [x] GPU pass-through tested on Intel iGPU (Gemini Lake UHD Graphics 600)
- [x] Jellyfin repository GPG key URL fixed

### V2 (Phase 4.5: Code Quality & Security) - COMPLETE ✅

- [x] Path traversal vulnerabilities fixed (data.go, yq.go)
- [x] Command injection in hook scripts fixed
- [x] Rollback protection added to edit command
- [x] Strict machine/snapshot name validation added
- [x] Volume path validation with sensitive path blacklist
- [x] Port range validation (1-65535)
- [x] GPU device path validation (/dev/ prefix required)
- [x] Nil pointer dereference fixed in provision exec
- [x] Download timeout set to 30 minutes
- [x] Cache directory permissions secured (0700)
- [x] Resource leaks fixed (defer mgr.Close() in snapshot commands)

### V2+ (Phase 5: Cockpit)

- [ ] Cockpit plugin mirrors cockpit-machines UX
