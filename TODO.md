# TODO

Remaining tasks for lxspawn, organized by priority.

---

## Current Status

**All core functionality is complete and stable for the first public release.**

Completed phases:
- Phase 1 (V1) - Basic machine lifecycle, resource limits, networking, volumes, JSON output, polkit rules
- Phase 2 (V1.5) - Provisioning engine, template variables, readiness probes, hooks, edit commands
- Phase 3 (V2) - Snapshot support (btrfs only)
- Phase 4 (V2) - Templates, adaptive network mode, Docker-in-container, DNS servers, media servers
- Phase 4.5-4.20 (V2) - Code quality, security hardening, bug fixes, template improvements
- Phase 4.21 (V2) - Code Review 2 security fixes (SUGGESTIONS-2.md)
- Phase 4.22 (V2) - Code Review 2 remaining fixes (SUGGESTIONS-2.md)
- Phase 4.23 (V2) - CI/CD and build system hardening (SUGGESTIONS-2.md Section 8)

### Phase 4.21 - Code Review 2 Fixes (2026-01-25)

All critical security issues from SUGGESTIONS-2.md addressed and tested on btrfs test machine:

- [x] **C1/C2: Template and stdin canonical config** - Machines created with `--template` or `-` now have proper canonical config
- [x] **C3: Transactional edits** - Edit commands backup config and restore on failure
- [x] **C4: ParseString defaults** - `ParseString()` now sets same defaults as `Parse()`
- [x] **C5: Apply preserves metadata** - Apply preserves `CreatedAt` and cleans stale files
- [x] **C6: Operation lock** - Per-machine flock prevents concurrent operation races
- [x] **D1/D2/D3: Rollback fixes** - Sudoers rules match, recovery runs, no auto-delete
- [x] **D4: Delete cleanup** - Removes snapshots directory, disables autostart, polls for stopped state
- [x] **O1: WaitForMachine** - Polls for actual container boot, not just machined state
- [x] **O3: Provision restart** - Defers restart if machine was running before provisioning
- [x] **O4: Exec retry logic** - Only retries on connection errors, not script exit codes
- [x] **O5: Script hash state** - Saves provision state after each script, not just at end
- [x] **O6: Timeout handling** - Uses RuntimeMaxSec for proper in-container process termination
- [x] **O7: Typed errors** - All "not found" errors use `NewMachineNotFoundError`
- [x] **O8: Progress stream** - Clean NDJSON output with ProgressMode flag
- [x] **P1: Sudoers rules** - Tightened patterns, removed LVM, exact arguments
- [x] **P3: Sudo -n flag** - All sudo calls use non-interactive mode
- [x] **P4: Setup fails loudly** - Returns non-zero on sudoers/polkit failures
- [x] **P5: State file permissions** - Changed to 0660 with lxspawn group
- [x] **P6: Scripts via stdin** - Provision scripts passed via stdin, not argv
- [x] **P7: Secret permissions** - Boot scripts 0700, config 0600
- [x] **V1/V2/V3: Injection fixes** - Strict validation for GPU devices, tmpfs options, paths
- [x] **V5: GPU error handling** - Returns error when GPU detection fails with gpu.enabled
- [x] **X1: Tar extraction** - Real filesystem symlink checks prevent escapes
- [x] **X2: Image verification** - SHA256 verification against SHA256SUMS
- [x] **X4: XZ error handling** - Named return captures close errors
- [x] **X5: HTTPS validation** - Reject redirects to HTTP

### Phase 4.22 - Code Review 2 Remaining Fixes (2026-01-25)

Additional fixes from SUGGESTIONS-2.md:

- [x] **O2/X3: Rootfs ownership** - Removed `fixRootfsOwnership` chown -R 0:0 which broke setuid/capabilities. Tar extraction now preserves ownership correctly.
- [x] **V6: Template param injection** - Templates now use `$LXSPAWN_PARAM_*` env vars for secrets instead of direct interpolation. Added bounded probe loops.
- [x] **O9: Disk usage** - `readDiskUsage` now uses btrfs qgroups for accurate per-machine usage on btrfs.
- [x] **O12: GPU/Network detection** - NVIDIA dedup by PCI address, discrete detection by PCI class code, network detection resolves bridge/bond slaves.
- [x] **T1: Show-template command** - Added `lxspawn show-template <name>` command for Cockpit integration.
- [x] **T2: Template fixes** - Fixed ServerName, probe infinite loops in all templates.
- [x] **C7: --name hostname** - `--name` override now always updates hostname.
- [x] **C8: Param validation** - `--param` keys validated against declared `param:` section.
- [x] **D5: Snapshot fixes** - Rollback no longer wipes descriptions, checks starting/stopping states, btrfs delete order fixed.
- [x] **V4: Validation gaps** - Machine names require 2+ chars, env values reject spaces, hostname validation fixed.
- [x] **P8: SELinux persistence** - Hook labelling uses `semanage fcontext` + `restorecon` for persistence.
- [x] **O10: Stop validation** - Stop/ForceStop now validate machine name and check registry.
- [x] **O11: CLI fixes** - exec preserves args before --, logs use JSON output with proper priority, CPU% normalized without quota, confirmation prompts skip in JSON mode.
- [x] **Boot scripts** - Dedup by content hash, `set -e` in service, once-only marker.

### Phase 4.23 - CI/CD and Build System Hardening (2026-01-25)

Final items from SUGGESTIONS-2.md (Section 8: CI, release, build, dependencies):

- [x] **B1: Release test gate** - Added test job with `go mod verify`, `go vet`, `go test -race -cover`, and `govulncheck` before goreleaser runs.
- [x] **B1: Supply-chain checks** - Removed `go mod tidy` from goreleaser before hooks to prevent dependency changes at release time. Added `go mod verify` step.
- [x] **B2: Consolidated test runs** - Combined three separate test runs (short, race, cover) into single `go test -race -cover` run.
- [x] **B3: Makefile fixes** - Added `GOOS=linux` for cross-platform builds, added `-race` flag to test target.
- [x] **B4: Updated dependencies** - Updated cobra v1.8.0 → v1.10.2, golang.org/x/sys v0.17.0 → v0.50.0.
- [x] **Dependabot** - Added `.github/dependabot.yml` for weekly dependency updates (gomod and github-actions).
- [x] **Section 9: Regression tests** - Added tests for V1 (GPU device newline injection) and V2 (tmpfs options validation). C4 tests already exist (TestParseStringDefaults). T3 template tests already exist.
- [x] **Section 10: Docs cleanup** - Fixed TODO C6 (removed outdated Makefile template install note). Old SUGGESTIONS.md already deleted.

**Testing on Bazzite (192.168.0.200) - 2026-01-25:**
- [x] C1/C2 fix verified: `lxspawn create --template debian-base --name test-c1` creates canonical config
- [x] C3 fix verified: `lxspawn edit-resources test-c1 --cpus 2 --memory 1G` works on template-created machine
- [x] V1 fix verified: GPU device with newline injection rejected
- [x] V2 fix verified: tmpfs options with newline injection rejected
- [x] T1 fix verified: `lxspawn show-template debian-base` displays embedded template
- [x] All `lxspawn doctor` checks pass (group, systemd-nspawn, machinectl, polkit, sudoers, btrfs, nftables)

### Known Issues (Open)

- **Data/Boot provisioner permission bug** - Fixed. Provisioners now use `deploy.Sudo()` for writes to root-owned rootfs.

---

## Phase 5 (V2+) - Cockpit Web UI - DEFERRED

**Note:** Cockpit plugin code exists in `cockpit/` directory but is not being actively developed for the first public release.

### Completed (MVP Foundation)

- Project setup (React 18, PatternFly 5, TypeScript, esbuild)
- Core components (MachineList, MachineDetailsPage, CreateMachineWizard)
- API wrapper matching lxspawn JSON format
- Deployment to test machine
- Bug fixes (memory stats, template-based creation)
- CLI: `lxspawn apply` command (declarative config)
- CLI: `restart_required` field in apply/status
- CLI: `restart`, `stop --force`, `autostart` commands
- CLI: `version` and `doctor` commands with JSON output
- CLI: System-wide machine registry (R13)

### Phase 5A: CLI Prerequisites (Unblock Cockpit UI)

**Priority 1: Quick Wins (1-2 hours each)**

- [x] **C12: Snapshot description** - Add `--description` flag to `snapshot` command. Store in `snapshots.json`. Schema already has field, just never populated.
- [x] **Fix plugin bug: editVolume readonly** - Plugin passes `--readonly true` as two args. Fix: use `--readonly=true` format in `cockpit/src/api/lxspawn.ts`.
- [x] **Fix plugin bug: editResources memory_high** - Add `--memory-high` and `--io-weight` flags to `edit_resources.go`. Update `ResourceJSON` schema.
- [x] **C3: Complete config --json** - Expand `ConfigData` to include: `hooks`, `provision`, `param`, `probes`, `journal`, `resources.memory_high`, `resources.io_weight`. Update `cmd/lxspawn/config.go`.

**Priority 2: Medium Effort (half day each)**

- [x] **C6: Embed templates** - Use `go:embed` to embed `examples/templates/*.yaml` in binary. Update `list_templates.go` to check embedded templates first. No Makefile update needed since templates are embedded.
- [x] **C4: Canonical config copy** - On `create`/`launch`, copy YAML to `/var/lib/lxspawn/<name>/config.yaml`. On `apply`, update canonical copy. `config`/`status` read from canonical copy. State stores `ConfigPath`. Fallback to `SourcePath` for backwards compat.
- [x] **C13: Logs for viewer** - Make JSON and text modes read from same source (container journal with `-M`). Add `--since` and `--priority` flags. Add `--follow` streaming mode. Make `--lines` work in both modes.
- [x] **C14: Batched stats** - Add `stats --all --json` or `list --stats --json`. Return cumulative counters with timestamp. UI computes rates from consecutive samples (no 100ms sleep). Eliminates N processes per polling cycle.

**Priority 3: Large Effort (full day+ each)**

- [x] **C5: Create flow** - `create -` / `launch -` reads YAML from stdin. `create --template NAME --name N` uses embedded template. `--param key=value` (repeatable) overrides template params. Refactor create/launch to share config-loading logic.
- [x] **C9: Structured error codes** - Define error codes: `MACHINE_NOT_FOUND`, `MACHINE_RUNNING`, `PERMISSION_DENIED`, `VALIDATION_FAILED`, `STORAGE_ERROR`, `PROVISION_FAILED`. Replace `COMMAND_FAILED`. Add field-level validation errors. Map to distinct exit codes.
- [x] **C10: Progress stream** - Add `--progress=json` flag to `create`/`launch`. Newline-delimited JSON events: `downloading`, `extracting`, `provisioning`, `probe`, `done`. Add `creating` state. Operation lock. Cancellation via context.

**Priority 4: Testability (completed)**

- [x] **Configurable filesystem paths (PathConfig)** - All paths configurable via `deploy.SetPaths()`. Enables NixOS support and testing with temp dirs.
- [x] **Dependency injection for Manager** - `NewManagerWithDeps()` accepts interfaces. `Manager` fields use `StateStore` and `SystemdClient` interfaces.
- [x] **Fake implementations** - `FakeSystemdClient` and `FakeStorageBackend` in `internal/testutil`.
- [x] **Integration tests** - 20+ lifecycle tests in `manager_integration_test.go` using fakes.
- [x] **Fix ParseString() defaults bug** - `ParseString()` now sets same defaults as `Parse()` (Seccomp, IOWeight, Storage.Size).
- [x] **Split manager.go** - Extracted `stats.go` and `shell.go` from the 1143-line god object.

### Phase 5B: Cockpit UI Components

**Priority 1: Quick Wins (1-2 hours each)**

- [ ] **DeleteDialog component** - Modal confirmation with PatternFly `Modal`. Replace browser `confirm()`. Option to preserve volumes (bind mounts).
- [ ] **EditCPUModal component** - Dialog to edit CPU count using `NumberInput`. Calls `editResources` with `--cpus`.
- [ ] **EditMemoryModal component** - Dialog to edit memory, memory_high, swap using `NumberInput` and unit selector. Calls `editResources`.
- [ ] **CreateSnapshotModal component** - Dialog with name and description fields. Calls `snapshot` with `--description`.
- [ ] **RevertSnapshotModal component** - Confirmation dialog with data loss warning. Calls `rollback`.

**Priority 2: Medium Effort (half day each)**

- [ ] **AddVolumeModal component** - Dialog to add bind mounts: host path, container path, readonly toggle. Calls `add-volume`.
- [ ] **EditVolumeModal component** - Dialog to edit existing bind mount. Calls `edit-volume`.
- [ ] **EditNetworkModal component** - Dialog to edit network mode, bridge, interface, port forwarding. Calls `edit-network`.
- [ ] **SnapshotsCard component** - Card showing snapshot list with create/revert/delete actions. Uses `list-snapshots --json`.
- [ ] **NetworkCard component** - Detailed network info: mode, IPs, ports, edit button. Uses `config --json`.
- [ ] **ConsoleCard component** - Embedded terminal using Cockpit's terminal component or link to `machinectl shell`.

**Priority 3: Large Effort (full day+ each)**

- [ ] **ProvisioningCard component** - Shows provisioning status, last run, phases completed, probes passed, re-run button. Uses `provision-status --json`.
- [ ] **SecurityCard component** - Shows capabilities, dropped capabilities, seccomp, no_new_privileges, inaccessible paths. Edit button opens modal.
- [ ] **GPUCard component** - Shows GPU enabled/disabled, type, devices. Edit button opens modal.
- [ ] **HooksCard component** - Shows hook scripts (pre_start, post_start, post_stop) with edit button. Uses `TextArea` for scripts.
- [ ] **AdvancedCard component** - Shows ephemeral, read_only, hostname, environment variables. Edit button opens modal.
- [ ] **EditSecurityModal component** - Dialog to edit capabilities, seccomp, no_new_privileges, inaccessible paths. Uses `DualListSelector` for capabilities.
- [ ] **EditGPUModal component** - Dialog to enable/disable GPU, select type (auto, intel, amd, nvidia, mali, adreno).
- [ ] **EditHooksModal component** - Dialog to edit pre_start, post_start, post_stop scripts using `TextArea`.
- [ ] **EditAdvancedModal component** - Dialog for ephemeral, read_only, hostname, environment variables. Uses `Switch` for toggles, key-value editor for env vars.

### Phase 5C: Template Enhancements

- [ ] **Template customization in UI** - Allow users to modify template parameters before creation. Form fields for `param:` values.
- [ ] **Template grouping and categorization** - Base Systems, DNS Servers, Media Servers, etc. Add `category:` field to templates.
- [ ] **Template metadata enhancement** - Add `category`, `priority`, `tags`, `icons` to template metadata.

### Phase 5D: Testing & Polish

- [ ] **Test plugin in browser** - Verify all fixes work on test machine (192.168.0.200 or 192.168.0.235).
- [ ] **Test create wizard with template selection** - Verify embedded templates load, params work.
- [ ] **Test machine details page** - Verify all cards display correctly, edit modals work.
- [ ] **Fix remaining UI/UX issues** - Address any issues found during testing.
- [ ] **Real-time updates via D-Bus signals** - Subscribe to machined D-Bus signals instead of polling (future enhancement).

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

## Known Issues

### Data/Boot Provisioner Permission Bug

**Status:** Fixed. Provisioners now use `deploy.Sudo()` for writes to root-owned rootfs.

When running `lxspawn provision` on a machine with rootfs owned by root (typical after image extraction), the data and boot provisioners previously failed with "permission denied" when trying to write files to the rootfs. This has been fixed by using `deploy.Sudo()` for all privileged rootfs writes.

### machinectl Shell PTY Errors

Intermittent "Failed to get shell PTY: Protocol error" or "There is no system bus in container" when using `lxspawn shell` or post-boot provisioning.

**Status:** Partially fixed. Post-boot provisioning, readiness probes, and `lxspawn exec` now use `systemd-run -M` for non-interactive execution, which avoids PTY allocation issues entirely. Interactive shell (`lxspawn shell`) still uses `machinectl shell` with retry logic for reliability.

### Macvlan Host Visibility

Host cannot ping macvlan containers (expected limitation of macvlan). Container can communicate with other network devices on the LAN.

**Workaround:** Use `host_access` configuration or `adaptive` network mode to create a host-side macvlan interface.

### Container Network Interface Naming

- macvlan creates `mv-<interface>` in container
- ipvlan creates `iv-<interface>` in container
- Network config must match these names

---

## Future Enhancements (Backlog)

These are ideas for future development, not currently scheduled:

- [ ] Live migration support (requires checkpoint/restore)
- [ ] Machine cloning (copy existing machine with new name)
- [ ] Machine import/export (tarball format for portability)
- [ ] Resource monitoring dashboard (CLI-based, not just Cockpit)
- [ ] Automatic backup scheduling
- [ ] Machine templates from running containers (snapshot + export)
- [ ] Multi-host management (manage machines across multiple hosts)
- [ ] Integration with container registries (pull pre-built machine images)
- [ ] Support for additional init systems (OpenRC, runit) beyond systemd
- [ ] Windows container support (if technically feasible)
