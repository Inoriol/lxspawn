# TODO

Remaining tasks for lxspawn, organized by phase.

---

## Current Status (2026-10-12)

**Phase 4 (V2) - Templates - In Progress (BLOCKED)**

⚠️ **Phase 4 cannot be marked complete until all templates are tested.**

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
- ✅ Macvlan/ipvlan crash fixed (using PrivateUsers=no for these modes)
- ❌ Arch Linux not supported (btrfs/pacman incompatibility)
- ❌ **Plex template - NOT TESTED (needs GPU testing)**
- ❌ **Jellyfin repository GPG key URL returns 404 (needs fix)**
- ✅ Added unit tests for provision/exec.go, image/fetcher.go, kernel/systemd.go (332 tests total)

**Phase 3 (V2) - Snapshots - Complete (btrfs only)**
- ✅ Snapshot thin wrapper works (btrfs only)
- ✅ Tested on Bazzite/Fedora 44 (btrfs)
- ✅ Automatic btrfs subvolume creation
- ✅ All snapshot commands working (create, list, delete)
- ✅ Rollback tested with nested subvolumes
- ✅ Machine delete handles child subvolumes correctly
- ⏸️ ZFS and LVM backends disabled (untested, moved to Phase 6)

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

## Phase 3: Snapshot Support (V2) - Complete ✅ (btrfs only)

### Snapshot Support

**Design philosophy:** Thin wrapper around existing tools (btrfs).

**Status:** Complete for btrfs. ZFS and LVM moved to Phase 6.

**Implementation:**
1. Detect filesystem type of `/var/lib/machines/<name>/`
2. If btrfs: use `btrfs subvolume snapshot`
3. Otherwise: error with helpful message

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

## Phase 4: Template Creation & Testing (V2) - In Progress (BLOCKED)

⚠️ **This phase cannot be marked complete until all templates are fully tested.**

### Goal

Create and test turnkey templates for common LXC use cases.

### Completion Criteria

Phase 4 is **NOT complete** until:
1. [ ] Plex template is tested with GPU pass-through
2. [ ] Jellyfin repository GPG key URL is fixed (currently returns 404)
3. [ ] All template testing checklists are verified (see below)

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

**Not Tested (BLOCKING):**
- ❌ **plex: Template created but NOT TESTED - requires GPU testing**
  - Test machine has Intel UHD Graphics 600 iGPU available
  - Template provisioning logic is correct (same as jellyfin)
  - Must test before Phase 4 can be marked complete

**Template Fixes Applied:**
- ✅ pihole.yaml: Added `dnsutils` to apt-get install (needed for `dig` in readiness probe)
- ✅ adguard.yaml: Added `dnsutils` to apt-get install (needed for `dig` in readiness probe)
- ✅ nextcloud.yaml: Added `bzip2` to apt-get install (needed for `tar -xjf` extraction)
- ✅ jellyfin.yaml: Added Intel GPU driver installation (intel-media-va-driver, vainfo)

**Notes:**
- Bind mount host paths must exist before launching (samba: /mnt/data, nextcloud: /var/lib/nextcloud)
- Adaptive network mode works correctly with macvlan fallback and host-side interface creation
- All templates using adaptive mode tested successfully on test machine (192.168.0.200)
- GPU pass-through verified on test machine (Intel UHD Graphics 600, Gemini Lake)
- Jellyfin repository GPG key URL issue (404 error) - repository structure may have changed

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

## Phase 4.5: Code Quality & Security Issues

**Status:** Not started. Identified during code review (2026-10-12).

**Priority:** High - Security vulnerabilities should be addressed before Phase 5 (Cockpit UI).

### Critical Issues (5)

1. **Path Traversal in Data Provisioning** (`internal/provision/data.go:41`)
   - Target path constructed without validation: `filepath.Join(d.rootfsPath, prov.Path)`
   - Can write files anywhere on host filesystem
   - Fix: Add path sanitization and verify resolved path stays under rootfs

2. **Path Traversal in YQ Provisioning** (`internal/provision/yq.go:24`)
   - Same vulnerability as data.go
   - Fix: Apply same path validation

3. **Command Injection in Hook Scripts** (`internal/compiler/slice.go:169-193`)
   - Unsanitized user input in shell commands: `fmt.Sprintf("ip link add %s link %s type macvlan mode bridge", hostIface, iface)`
   - Malicious interface names could execute arbitrary commands
   - Fix: Validate interface names with regex `^[a-zA-Z0-9_-]+$`

4. **Destructive Edit Without Rollback** (`cmd/lxspawn/edit.go:63-69`)
   - Machine deleted before new one launches
   - If launch fails, machine is permanently lost
   - Fix: Create backup snapshot before deletion, or use temporary name and swap atomically

5. **Unsafe Tar Extraction** (`internal/image/fetcher.go:216-273`)
   - TOCTOU race condition with symlinks
   - Between path check and file creation, attacker could replace directory with symlink
   - Fix: Extract to temp directory first, then atomically move

### High Issues (12)

6. **Resource Leaks in Snapshot Commands** (`cmd/lxspawn/delete_snapshot.go:35-38`, `list_snapshots.go:20-23`, `rollback.go:37-40`, `snapshot.go:21-24`)
   - `snapshot.NewManager()` called but `mgr.Close()` never called
   - Inconsistent with machine manager usage (all use `defer mgr.Close()`)
   - Fix: Add `defer mgr.Close()` after manager creation

7. **Insufficient Machine Name Validation** (`internal/config/parser.go:54-66`)
   - Only checks for spaces, slashes, backslashes
   - Allows dangerous characters: `;`, `|`, `&`, `$`, backticks
   - Fix: Use strict regex `^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]$`

8. **Insufficient Snapshot Name Validation** (`internal/snapshot/snapshot.go:44-54`)
   - Same issue as machine names
   - Fix: Apply same strict regex

9. **Unvalidated Volume Paths** (`internal/config/parser.go:178-200`)
   - Only checks for absolute paths
   - Could expose sensitive directories like `/etc/shadow`
   - Fix: Add whitelist of allowed base directories, check for symlinks

10. **Insecure Default Network Mode** (`internal/config/parser.go:29-31`)
    - Defaults to "host" mode (no network isolation)
    - Fix: Default to "private" or require explicit specification

11. **Missing Port Range Validation** (`internal/config/parser.go:169-176`)
    - Only checks format, not valid range (0-65535)
    - Ports like `tcp:99999:80` would pass validation
    - Fix: Parse and validate port numbers

12. **Missing Input Validation for Resource Values** (`internal/config/parser.go`)
    - Memory, MemoryHigh, Swap, Tasks not validated for format
    - Invalid values only fail when systemd parses them
    - Fix: Add regex validation for systemd-compatible formats

13. **Nil Pointer Dereference in Provision Exec** (`internal/provision/exec.go:53-72`)
    - `retryCmd` could be nil if switch doesn't match any case
    - Fix: Add default case returning error

14. **Hardcoded DNS Servers** (`internal/image/cleanup.go:32-36`)
    - Hardcoded to 8.8.8.8, 1.1.1.1, 8.8.4.4
    - Bypasses local DNS, leaks queries to external servers
    - Fix: Read from host or make configurable

15. **File Cache Directory Permissions** (`internal/provision/file.go:31-34`)
    - Cache directory created with 0755 (world-readable)
    - Other users could read or replace cached scripts
    - Fix: Use 0700 permissions

16. **Race Condition in State Manager** (`internal/deploy/state.go:59-95, 152-176`)
    - Nested locking with mutex + file lock
    - Complex locking hierarchy could lead to deadlocks
    - Fix: Use single synchronization mechanism or document hierarchy

17. **No Timeout on Downloads** (`internal/image/fetcher.go:27-29`)
    - HTTP client has `Timeout: 0` (no timeout)
    - Downloads can hang indefinitely
    - Fix: Set reasonable timeout (30 minutes)

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

**Status:** Not started. Blocked on Phase 4 (Templates) completion.

**Design:** Thin wrapper over `lxspawn --json` commands, mirroring cockpit-machines UX.

**Key principles:**
1. Framework agnostic (don't lock into React)
2. Thin wrapper (all data via `cockpit.spawn(["lxspawn", "command", "--json"])`)
3. Leverage Cockpit built-ins (terminal, journal, file APIs)
4. Mirror cockpit-machines UX patterns

---

## Phase 6: ZFS and LVM Snapshot Support (V2+)

### Goal

Test and enable ZFS and LVM snapshot backends that are currently disabled.

### Status

**Not started.** Requires VM setup with ZFS and LVM filesystems.

### Tasks

- [ ] Set up VM with ZFS filesystem
- [ ] Test ZFS snapshots (create, list, rollback, delete)
- [ ] Set up VM with LVM (XFS) filesystem
- [ ] Test LVM snapshots (create, list, rollback, delete)
- [ ] Document any backend-specific issues found
- [ ] Re-enable ZFS and LVM backends in `internal/snapshot/detector.go`
- [ ] Update tests in `internal/snapshot/detector_test.go`
- [ ] Update documentation to reflect full snapshot support

### Implementation Notes

ZFS backend (`internal/snapshot/zfs.go`):
- Uses `zfs snapshot`, `zfs list`, `zfs rollback`, `zfs destroy`
- Requires ZFS dataset for machine rootfs

LVM backend (`internal/snapshot/lvm.go`):
- Uses `lvcreate --snapshot`, `lvs`, `lvconvert --merge`, `lvremove`
- Requires LVM logical volume for machine rootfs
- Note: LVM snapshot merge requires next activation (not immediate)

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

### Unit Tests (332 tests, all passing)

- Config parsing and validation (40 tests)
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

- [x] Snapshot thin wrapper works (btrfs only)
- [x] Tested on Bazzite/Fedora 44 (btrfs)
- [x] Automatic btrfs subvolume creation
- [x] All snapshot commands working (create, list, delete)
- [x] Rollback tested with nested subvolumes
- [x] Machine delete handles child subvolumes correctly

### V2+ (Phase 6: ZFS/LVM Snapshots)

- [ ] Set up VM with ZFS filesystem
- [ ] Test ZFS snapshots (create, list, rollback, delete)
- [ ] Set up VM with LVM (XFS) filesystem
- [ ] Test LVM snapshots (create, list, rollback, delete)
- [ ] Document any backend-specific issues found
- [ ] Re-enable ZFS and LVM backends in detector.go

### V2 (Phase 4: Templates) - NOT COMPLETE

- [x] Docker-in-container template works
- [x] DNS server templates work (pihole, adguard - tested and working)
- [ ] Media server templates work (jellyfin tested, **plex NOT TESTED**)
- [x] File server templates work (samba, nextcloud - tested and working)
- [x] Base system templates tested (Debian, Ubuntu, Fedora)
- [x] Rootfs ownership fix applied
- [x] EOVERFLOW limitation resolved (using PrivateUsers=identity)
- [ ] All templates tested on test machine (**plex missing**)
- [x] Unit tests added for provision/exec.go, image/fetcher.go, kernel/systemd.go
- [x] GPU pass-through tested on Intel iGPU (Gemini Lake UHD Graphics 600)
- [ ] Fix Jellyfin repository GPG key URL (currently returns 404)

**⚠️ Phase 4 cannot be marked complete until:**
1. Plex template is tested with GPU pass-through
2. Jellyfin repository GPG key URL is fixed
3. All template testing checklists are verified

### V2+ (Phase 5: Cockpit)

- [ ] Cockpit plugin mirrors cockpit-machines UX
