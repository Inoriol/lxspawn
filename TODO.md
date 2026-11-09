# TODO

Remaining tasks for lxspawn, organized by priority.

---

## Current Status

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
