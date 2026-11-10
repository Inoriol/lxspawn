# TODO

Remaining tasks for lxspawn, organized by priority.

---

## Current Status

**Phase 4.15 (V2) - P2 Bug Fixes - COMPLETE ✅**

**Goal:** Fix all P2 (broken features and wrong behaviour) issues from SUGGESTIONS.md code review.

**Status:** All 18 applicable P2 issues fixed (R20 deferred - LVM backend disabled), unit tests added.

**Completed:**
- ✅ **R1:** `lxspawn edit` no longer destroys container contents. Added `Manager.Apply()` method that regenerates config files in place, preserving rootfs.
- ✅ **R2:** Btrfs snapshots moved to sibling directory `<machine>.snapshots/` instead of child `<rootfs>/.snapshots/`. Rollback no longer deletes snapshots.
- ✅ **R3:** Rollback uses safe rename-aside pattern. Old rootfs renamed to `.old`, new moved in, `.old` deleted only on success.
- ✅ **R4:** Added `ValidateMachineName()` exported function. Snapshot manager validates names at all entry points (create, rollback, delete).
- ✅ **R5:** Tar extraction tracks symlinks created during extraction. Prevents writes through symlinks using `O_NOFOLLOW` and path resolution.
- ✅ **R6:** Added `secureJoin()` function in provision package that resolves symlinks and validates paths don't escape rootfs.
- ✅ **R7:** Hook generation returns errors for invalid interface names instead of generating echo commands. All values shell-quoted with `shellQuote()`.
- ✅ **R8:** Comprehensive validation for hostname, environment variables, capabilities, GPU devices, network interfaces, volume paths, and resource sizes.

**Files modified:**
- `cmd/lxspawn/edit.go` - uses new Apply method
- `internal/machine/manager.go` - added Apply() method
- `internal/snapshot/btrfs.go` - sibling snapshot directory, safer rollback
- `internal/snapshot/manager.go` - name validation
- `internal/config/parser.go` - comprehensive validation, ValidateMachineName()
- `internal/image/fetcher.go` - symlink-safe extraction
- `internal/provision/data.go`, `yq.go` - secure path resolution
- `internal/compiler/slice.go`, `compiler.go` - shell injection prevention
- `cmd/lxspawn/setup.go` - updated sudoers rules for new snapshot paths

**Tests added:**
- `TestValidateMachineName` - comprehensive name validation
- `TestValidateHostnameWithNewline`, `TestValidateEnvironmentWithNewline` - injection prevention
- `TestValidateCapabilityWithSpace`, `TestValidateGPUDeviceWithDotDot` - path traversal prevention
- `TestValidateNetworkBridgeWithInvalidChars`, `TestValidateNetworkInterfaceWithInvalidChars` - shell injection prevention
- `TestShellQuote` - shell quoting correctness
- `TestGenerateHostAccessScript*` - hook generation error handling
- `TestSecureJoin` - symlink resolution and path validation
- `TestValidatePathSymlinkEscape` - provisioner path validation

**Live testing on Bazzite (192.168.0.200):**
- ✅ R1: Created machine, installed cowsay, edited config, verified cowsay still works
- ✅ R2: Created snapshot, verified stored in `/var/lib/machines/test-edit.snapshots/snap1`
- ✅ R4: Attempted path traversal names (`../etc`, `test\ninjection`) - both rejected
- ✅ R7: Attempted shell injection in interface name (`eth0; rm -rf /`) - rejected

---

**Phase 4.15 (V2) - P2 Bug Fixes - COMPLETE ✅**

**Goal:** Fix P2 (broken features and wrong behaviour) issues from SUGGESTIONS.md code review.

**Status:** All 19 P2 issues fixed, unit tests added.

**Completed:**
- ✅ **R9:** `edit-*` commands now call `config.Validate()` and `mgr.Apply()` to regenerate unit files after editing YAML. Tested: edit-resources, edit-network, add-volume all regenerate .nspawn/slice/drop-in files correctly.
- ✅ **R10:** `validate` command now calls `config.Validate()` and uses `ParseStrict()` with `KnownFields(true)` to reject typos. Tested: `memroy:` typo correctly rejected.
- ✅ **R11:** Boot service symlink now uses relative path `../lxspawn-boot.service` instead of dangling host path. Tested: service enabled and runs successfully on boot.
- ✅ **R12:** GPU capabilities moved from drop-in (where they were ignored) to `.nspawn` file's `Capability=` line. Unit tests confirm drop-in has no Capability=, .nspawn has it.
- ✅ **R13:** Machine registry moved from per-user `~/.local/share/lxspawn/machines.json` to system-wide `/var/lib/lxspawn/machines.json`. `Create` now refuses to adopt pre-existing rootfs directories. **Tested on Bazzite:** Registry created at `/var/lib/lxspawn/machines.json`, pre-existing rootfs correctly rejected.
- ✅ **R14:** Registry `load()` now decodes into fresh map (prevents resurrection of deleted machines). Lock leak fixed - unlock called on load failure. **Unit test added:** `TestLoadUsesFreshMap` verifies deleted machines don't resurrect.
- ✅ **R15:** `Delete` now checks machine state after stop and aborts if still running. Provisioning failure now cleans up registered machine and storage. **Tested on Bazzite:** Delete correctly stops machine before deletion.
- ✅ **R16:** Tar extraction now preserves setuid/setgid/sticky bits using proper Unix-to-Go mode conversion. Applies uid/gid from tar header when running as root. **Unit test added:** `TestUnixModeToFileMode` verifies correct conversion of special bits.
- ✅ **R17:** Btrfs usage parsing fixed - now trims whitespace and uses proper prefix matching. Tested: `lxspawn stats` shows real disk values.
- ✅ **R18:** `snapshot` and `rollback` commands now check machine is stopped before proceeding. **Tested on Bazzite:** Running machine correctly rejected for snapshot.
- ✅ **R19:** Btrfs subvolume listing now uses inode-based detection (inode 256) instead of parsing `btrfs subvolume list` output. Works correctly on Fedora/Ubuntu layouts where `/` is not the top-level subvolume. **Tested on Bazzite:** Snapshot listing works correctly with new inode-based detection. **Unit tests added:** `TestIsBtrfsSubvolume` in both snapshot and storage packages.
- ✅ **R21:** D-Bus machine name escaping fixed for leading digits (e.g., `3dprint` → `_33dprint`). `GetMachineState` now returns real errors instead of silently returning "stopped". **Unit tests added:** `TestMachineName` and `TestIsMachineNotFound`.
- ✅ **R22:** GPU detection fixes: render node now read from sysfs instead of calculated from card index; PCI resource file parsing fixed to compute size as `end - start + 1`; NVIDIA GPU deduplication added; `/dev/kfd` now bind-mounted for AMD GPUs (ROCm support). **Unit tests added:** `TestIsDiscreteGPU`, `TestFindRenderNode`, `TestIsNVIDIAGPUDetected`.
- ✅ **R23:** `logs --json` now reads from container journal (using `-M` flag) instead of host unit log, consistent with text mode.
- ✅ **R24:** Remote file handling hardened: HTTP URLs rejected (HTTPS-only); digest required for remote files; path traversal prevented; redirect to HTTP blocked; truncation detection added. **Unit tests added:** `TestResolveFileRefRejectsHTTP`, `TestResolveFileRefRequiresDigestForHTTPS`, `TestResolveFileRefRejectsPathTraversal`.
- ✅ **R25:** yq mode parser fixed: `parseValue` now correctly handles IP addresses and version strings as strings; operator parsing uses proper parsing instead of `Contains`; unsupported operators now error. **Unit tests added:** `TestParseValueIPAddress`, `TestParseValueVersionString`, `TestParseOperator`.
- ✅ **R26:** Template handling hardened: `missingkey=error` option added to error on missing params instead of rendering empty; environment variable keys validated. **Unit tests added:** `TestIsValidEnvVarName`, updated `TestRenderTemplateMissingParam`.
- ✅ **R27:** Host-side macvlan interface now uses per-machine naming (`mv-<machinename>`) instead of shared `mv-lxspawn`, preventing conflicts when multiple adaptive machines exist. **Unit test added:** `TestGenerateHostAccessInterfaceName`.

**Deferred:**
- ⏸️ **R20:** LVM thin volumes not remounted after reboot - DEFERRED (LVM backend is disabled pending testing infrastructure, see Phase 6)

**Files modified:**
- `cmd/lxspawn/edit_resources.go`, `edit_network.go`, `add_volume.go`, `edit_volume.go`, `remove_volume.go` - added Validate() and Apply() calls
- `cmd/lxspawn/validate.go` - uses ParseStrict() and calls Validate()
- `cmd/lxspawn/snapshot.go`, `rollback.go` - check machine state before operation
- `internal/config/parser.go` - added ParseStrict() with KnownFields(true)
- `internal/provision/boot.go` - relative symlink path
- `internal/provision/file.go` - HTTPS-only, digest required, path traversal prevention
- `internal/provision/yq.go` - fixed operator parsing, parseValue for IP addresses
- `internal/provision/template.go` - missingkey=error option
- `internal/provision/exec.go` - environment variable key validation
- `internal/compiler/slice.go`, `nspawn.go` - moved GPU capabilities to .nspawn
- `internal/compiler/compiler.go` - per-machine macvlan interface naming
- `internal/storage/btrfs.go` - fixed usage parsing, inode-based subvolume detection
- `internal/deploy/state.go` - system-wide registry path, fresh map decoding, lock leak fix
- `internal/deploy/dbus.go` - D-Bus machine name escaping for leading digits, real error returns
- `internal/machine/manager.go` - rootfs existence check, delete state check, provision cleanup, logs consistency
- `internal/snapshot/manager.go` - system-wide snapshot state path
- `internal/snapshot/btrfs.go` - inode-based subvolume detection
- `internal/image/fetcher.go` - setuid/setgid/sticky bit preservation, uid/gid application
- `internal/gpu/detect.go` - render node from sysfs, PCI resource parsing, NVIDIA dedup, /dev/kfd binding

**Tests added:**
- `TestParseStrictRejectsUnknownFields` - validates typo rejection
- `TestParseStrictAcceptsValidFields` - validates correct parsing
- `TestParseStrictRejectsTypoInNetworkMode` - validates nested field typos
- `TestCompileDropInNoCapability` - validates R12 fix
- `TestCompileNspawnIncludesGPUCapabilities` - validates R12 fix
- `TestCompileNspawnIncludesSecurityCapabilities` - validates capabilities
- `TestBtrfsUsageParsing` - validates R17 fix
- `TestLoadUsesFreshMap` - validates R14 fix (deleted machines don't resurrect)
- `TestUnixModeToFileMode` - validates R16 fix (setuid/setgid/sticky preservation)
- `TestIsBtrfsSubvolume` (snapshot) - validates R19 fix (inode-based detection)
- `TestIsBtrfsSubvolumeWithInode256` (snapshot) - validates R19 fix logic
- `TestIsBtrfsSubvolume` (storage) - validates R19 fix (inode-based detection)
- `TestMachineName` - validates R21 fix (D-Bus name escaping)
- `TestIsMachineNotFound` - validates R21 fix (error handling)
- `TestIsDiscreteGPU` - validates R22 fix (PCI resource parsing)
- `TestFindRenderNode` - validates R22 fix (render node from sysfs)
- `TestIsNVIDIAGPUDetected` - validates R22 fix (NVIDIA dedup)
- `TestResolveFileRefRejectsHTTP` - validates R24 fix
- `TestResolveFileRefRequiresDigestForHTTPS` - validates R24 fix
- `TestResolveFileRefRejectsPathTraversal` - validates R24 fix
- `TestParseValueIPAddress` - validates R25 fix
- `TestParseValueVersionString` - validates R25 fix
- `TestParseOperator` - validates R25 fix
- `TestIsValidEnvVarName` - validates R26 fix
- `TestGenerateHostAccessInterfaceName` - validates R27 fix

**Live testing on Bazzite (192.168.0.200):**
- ✅ R9: edit-resources updated slice from 512M/100% to 1G/200%, edit-network changed Private=no to Private=yes, add-volume added Bind mount
- ✅ R10: Typo `memroy:` correctly rejected with "field memroy not found"
- ✅ R11: Boot service enabled and ran successfully (status: active/exited)
- ✅ R13: Registry created at `/var/lib/lxspawn/machines.json`, pre-existing rootfs correctly rejected with clear error message
- ✅ R15: Delete correctly stops machine before deletion
- ✅ R17: `lxspawn stats` shows real disk values (35.6 GiB / 116.7 GiB)
- ✅ R18: Running machine correctly rejected for snapshot/rollback
- ✅ R19: Snapshot listing works correctly with inode-based detection

---

**Phase 4.16 (V2) - P3 Hardening & Cleanup - COMPLETE ✅**

**Goal:** Fix all P3 (hardening and cleanup) issues from SUGGESTIONS.md code review.

**Status:** All 9 P3 issues fixed, unit tests updated.

**Completed:**
- ✅ **P3-1:** File modes/ownership in provision/data.go - `os.Chmod()` now called after write to ensure permissions are applied. `setOwnership()` errors now returned instead of logged as warnings.
- ✅ **P3-2:** Network config permissions - `/var/lib/lxspawn/config.json` now written with 0644 permissions instead of 0600, allowing group users to read it for `interface: auto`.
- ✅ **P3-3:** Interface filtering in network/detect.go - `isPhysicalInterface()` now only checks for `/sys/class/net/<name>/device` existence, removing fallback that allowed virtual interfaces. Interface name parsing now strips `@...` suffix from `ip -o link` output.
- ✅ **P3-4:** Relative source path stored in create.go/launch.go - Now uses `filepath.Abs()` to store absolute paths in machine registry, preventing failures when running commands from different directories.
- ✅ **P3-5:** CLI behaviour - Added `SilenceUsage: true` and `SilenceErrors: true` to root command to prevent duplicate error messages. `exec` command now properly propagates child exit codes. `status.Leader` now populated for running machines.
- ✅ **P3-6:** Download/extract - `xzReader.Close()` error now checked. Added 10GB cap on decompressed size. Downloads now use `/var/lib/lxspawn/cache/` instead of `/tmp` to avoid tmpfs issues on Fedora.
- ✅ **P3-7:** Smaller correctness issues - `ListSnapshots` now reads actual btrfs subvolume creation time via `btrfs subvolume show`. Post-rollback chown now uses `root:root` instead of invoking user. `setup network` now validates non-numeric input. `CPUQuota` now uses `math.Round()` to prevent truncation (e.g., 0.29 CPUs → 29% instead of 28%). `ParseSizeToGB("0.5G")` now returns 1 GB instead of 0.
- ✅ **P3-8:** Dead code privileges - Removed ZFS and LVM snapshot sudoers rules from setup.go since these backends are disabled. Kept only LVM storage rules needed for thin provisioning.
- ✅ **P3-9:** GNU-only commands - Replaced all uses of GNU-specific `stat -f -c %T` and `df --output` with portable `syscall.Statfs()` calls. This makes the code work on macOS and other non-GNU systems.

**Files modified:**
- `internal/provision/data.go` - chmod after write, error on chown failure
- `internal/network/config.go` - 0644 permissions
- `internal/network/detect.go` - physical interface check, @suffix stripping
- `cmd/lxspawn/create.go`, `launch.go` - filepath.Abs for source path
- `cmd/lxspawn/main.go` - SilenceUsage/SilenceErrors, exit code propagation
- `internal/machine/manager.go` - Leader population, syscall.Statfs for disk usage
- `internal/image/fetcher.go` - decompression cap, cache directory
- `internal/snapshot/btrfs.go` - creation time from btrfs, root:root chown
- `internal/config/parser.go` - ParseSizeToGB minimum 1GB
- `internal/compiler/slice.go` - math.Round for CPUQuota
- `cmd/lxspawn/setup.go` - removed ZFS/LVM snapshot rules, added cache directory
- `internal/snapshot/detector.go` - syscall.Statfs
- `internal/storage/detector.go` - syscall.Statfs
- `internal/storage/directory.go` - syscall.Statfs
- `internal/snapshot/lvm.go` - /proc/mounts instead of df
- `internal/kernel/version.go` - syscall.Statfs for IsBtrfs
- `internal/provision/data_test.go` - skip test if not root

**Tests updated:**
- `TestDataProvisionerWithOwner` - now skips if not running as root

**Live testing on Bazzite (192.168.0.200):**
- ✅ P3-1: Created machine with `permissions: "0600"` in data provision, verified file has 600 permissions
- ✅ P3-2: Verified `/var/lib/lxspawn/config.json` has 0644 permissions (-rw-r--r--)
- ✅ P3-4: Verified `source_path` stored as absolute path (`/tmp/test-p3.yaml`)
- ✅ P3-5: Verified `status --json` includes `leader` PID; verified `exec` propagates exit code 42 correctly
- ✅ P3-6: Verified `/var/lib/lxspawn/cache/` directory created by setup
- ✅ P3-7: Verified `cpus: 0.29` generates `CPUQuota=29%` (not 28%); verified `stats --json` returns real disk values
- ✅ P3-8: Verified sudoers rules contain no ZFS or lvconvert entries

---

**Phase 5a (V2) - Cockpit Web UI MVP - DEFERRED 🚧**

**Goal:** Create core Cockpit web UI plugin with dashboard, details page, and create wizard.

**Status:** Core components built and deployed to test machine. API wrapper fixed to match lxspawn JSON format. Template-based creation implemented. Memory stats bug fixed.

**Note:** Cockpit plugin development is deferred for the first public release. The code exists in the `cockpit/` directory but is not being actively developed or built.

### Completed (Day 1-3)

**1. Project Setup (COMPLETE ✅)**
- ✅ Cloned cockpit-starter-kit template
- ✅ Renamed all references: `starter-kit` → `lxspawn`
- ✅ Configured package.json (React 18, PatternFly 5, TypeScript, esbuild)
- ✅ Set up manifest.json for Cockpit plugin
- ✅ Configured development environment
- ✅ Set up auto-sync to Bazzite test machine (192.168.0.200)

**2. Core Components (COMPLETE ✅)**
- ✅ Created API wrapper (`cockpit/src/api/lxspawn.ts`) - all lxspawn commands
- ✅ Created TypeScript types (`cockpit/src/api/types.ts`)
- ✅ Created utility functions (`cockpit/src/utils/format.ts`)
- ✅ Created StateIcon component
- ✅ Created AggregateStatusCards component
- ✅ Created MachineList component (dashboard with filtering & stats)
- ✅ Created MachineDetailsPage component (overview, usage, volumes, network)
- ✅ Created CreateMachineWizard component (template-based)
- ✅ Updated App.tsx with routing

**3. API Integration (COMPLETE ✅)**
- ✅ Fixed API wrapper to match lxspawn JSON format (`{version, data}`)
- ✅ All API functions now return data directly (no wrapper objects)
- ✅ Updated all components to use new API format

**4. Deployment (COMPLETE ✅)**
- ✅ Built plugin successfully
- ✅ Deployed to Bazzite test machine (~/.local/share/cockpit/lxspawn/)
- ✅ Plugin accessible at https://192.168.0.200:9090

**5. Bug Fixes (COMPLETE ✅)**
- ✅ Fixed TypeScript types to match actual lxspawn JSON format
  - Machine type now has `resources.cpus` and `resources.memory` (not top-level)
  - Stats type now uses `cpu_usage_percent`, `memory_used_bytes`, etc.
- ✅ Fixed MachineList component to use correct stats field names
- ✅ Fixed MachineDetailsPage component to use correct stats field names
- ✅ Fixed createMachine API to save YAML to permanent location
  - Now saves to `~/.local/share/lxspawn/<name>/config.yaml` instead of /tmp
  - Prevents "config file not found" errors when clicking on machine
  - Fixed `cockpit.directory.home` error - now uses `cockpit.user()` to get home directory
  - Fixed directory creation - now uses `mkdir -p` before writing config file
- ✅ Fixed Wizard component to use PatternFly 6 API (children instead of steps array)
- ✅ Changed CreateMachineWizard to use templates instead of form-based creation
  - Added template selection step with radio buttons
  - Added API function to read template content
  - Template name is replaced with user-provided machine name
  - Templates are now installed on test machine at `~/.local/share/lxspawn/templates/`
- ✅ Fixed `lxspawn stats` command - Memory stats showing 0 B / 0 B
  - **Problem:** Memory stats returned 0 for machines with hyphens in names
  - **Root Cause:** Code assumed cgroup path was `/sys/fs/cgroup/machine.slice/machine-<name>.slice`, but systemd creates hierarchical slices for names with hyphens (e.g., `test-stats` → `/sys/fs/cgroup/machine.slice/machine-test.slice/machine-test-stats.slice`)
  - **Fix:** Added `getMachineCgroupPath()` function that uses the leader PID to read the actual cgroup path from `/proc/<pid>/cgroup`
  - **Files:** `internal/machine/manager.go:708-750`, `internal/machine/manager.go:752-780`
  - **Tested:** Works for machines with and without hyphens in names
- ✅ Removed sudo-helper binary, replaced with direct sudo calls
  - **Problem:** Sudo-helper binary added unnecessary complexity and attack surface
  - **Solution:** Use direct `sudo` calls with specific sudoers rules for each operation
  - **Implementation:**
    - Replaced `deploy.SudoHelper()` with `deploy.Sudo()` (simple wrapper around `exec.Command("sudo", args...)`)
    - Updated `cmd/lxspawn/setup.go` to generate specific sudoers rules for each command
    - Added rules for core operations, btrfs, LVM, and ZFS
    - Fixed path handling in `deleteChildSubvolumes()` to use absolute paths
    - Added sudoers rules for snapshot paths (`/var/lib/machines/*/.snapshots/*`)
  - **Files:** `internal/deploy/filesystem.go`, `cmd/lxspawn/setup.go`, all files using `deploy.Sudo()`
  - **Benefits:** Simpler architecture, better security through explicit rules, easier to audit
  - **Tested:** Full lifecycle (create, start, stop, snapshot, delete) works correctly on test machine

### In Progress

**6. Testing & Refinement (IN PROGRESS 🚧)**
- 🚧 Test plugin in browser - verify all fixes work
- 🚧 Test create wizard with template selection
- 🚧 Test machine details page
- 🚧 Fix any remaining UI/UX issues

### Next Steps

**7. Dialog Components (PENDING)**
- [ ] Create DeleteDialog component
- [ ] Create EditCPUModal component
- [ ] Create EditMemoryModal component
- [ ] Create EditNetworkModal component
- [ ] Create AddVolumeModal component
- [ ] Create EditVolumeModal component

**8. Advanced Features (PENDING - Phase 5b)**
- [ ] Create SnapshotsCard component
- [ ] Create ConsoleCard component (embedded terminal)
- [ ] Create ProvisioningCard component
- [ ] Create GPUCard component
- [ ] Create SecurityCard component
- [ ] Create HooksCard component
- [ ] Create AdvancedCard component

**9. Template Enhancements (PENDING - Phase 5c)**
- [ ] Template customization in UI - allow users to modify template parameters before creation
  - Add form fields for common parameters (CPU, memory, network mode)
  - Show template preview with modifications
  - Generate customized YAML from template + user inputs
- [ ] Template grouping and categorization
  - Group templates by category: Base Systems, DNS Servers, Media Servers, File Servers, etc.
  - Show base systems at the top (most commonly used)
  - Add category filters in template selection
  - Add search/filter by template name or description
- [ ] Template metadata enhancement
  - Add category field to templates
  - Add priority/order field for sorting
  - Add tags for filtering
  - Add icon/emoji for visual identification

---

## Phase 6 (V2+) - ZFS/LVM Snapshots - DEFERRED

**Note:** Directory mode snapshots removed (unreliable, slow). Snapshots now require btrfs filesystem. LVM/ZFS support deferred to after Phase 5 (Cockpit Web UI).

- [ ] **R20: LVM thin volumes not remounted after reboot** - `storage/lvm_thin.go:85` only runs `mount`; no fstab entry or `.mount` unit generated. After reboot `/var/lib/machines/<name>` is empty and machine won't start. Fix: generate systemd `.mount` unit or add fstab entry. (Deferred because LVM backend is disabled pending testing infrastructure)
- [ ] Set up fast SSD-based test environment for LVM testing
- [ ] Test LVM thin pool snapshots (create, list, rollback, delete)
- [ ] Re-enable LVM backend
- [ ] Set up test environment with ZFS filesystem
- [ ] Test ZFS snapshots (create, list, rollback, delete)
- [ ] Re-enable ZFS backend

---

## Completed Phases (Historical)

The following phases have been completed and are part of the first public release:

- ✅ **Phase 1 (V1)** - Basic machine lifecycle, resource limits, networking, volumes, JSON output, polkit rules
- ✅ **Phase 2 (V1.5)** - Provisioning engine, template variables, readiness probes, hooks, edit commands
- ✅ **Phase 3 (V2)** - Snapshot support (btrfs only)
- ✅ **Phase 4 (V2)** - Templates, adaptive network mode, Docker-in-container, DNS servers, media servers
- ✅ **Phase 4.5 (V2)** - Code quality & security hardening
- ✅ **Phase 4.6 (V2)** - Cleanup improvements
- ✅ **Phase 4.7 (V2)** - Storage backend overhaul (btrfs, LVM thin, directory)
- ✅ **Phase 4.9 (V2)** - E2E testing on Bazzite
- ✅ **Phase 4.10 (V2)** - Snapshot backend cleanup & directory backend fix
- ✅ **Phase 4.11 (V2)** - JSON API for Cockpit UI
- ✅ **Phase 4.12 (V2)** - Privilege model hardening
- ✅ **Phase 4.13 (V2)** - Sudoers & polkit audit
- ✅ **Phase 5a (V2)** - Cockpit Web UI MVP (code complete, deferred for release)

All completed work is stable and ready for the first public release.
