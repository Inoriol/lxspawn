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

- [x] **C6: Embed templates** - Use `go:embed` to embed `examples/templates/*.yaml` in binary. Update `list_templates.go` to check embedded templates first. Update Makefile to install templates to `/usr/share/lxspawn/templates/`.
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
