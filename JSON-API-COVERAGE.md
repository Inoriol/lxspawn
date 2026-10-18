# JSON API Coverage Analysis for Cockpit UI

This document analyzes whether lxspawn's `--json` output provides sufficient coverage for building a Cockpit UI based on the cockpit-machines reference.

---

## Current JSON Coverage

### Commands with JSON Support ✅

| Command | JSON Output | Coverage | Notes |
|---------|-------------|----------|-------|
| `list` | `MachineListData` | ✅ Complete | Name, distro, state, network, addresses, resources, created_at, source_path |
| `status` | `MachineStatusData` | ✅ Complete | Same as list + leader PID, files array |
| `create` | `CreateData` | ✅ Complete | Machine name, source_path, files_written, success |
| `launch` | `CreateData` | ✅ Complete | Same as create |
| `start` | `LifecycleData` | ✅ Complete | Action, machine, success |
| `stop` | `LifecycleData` | ✅ Complete | Action, machine, success |
| `delete` | `LifecycleData` | ✅ Complete | Action, machine, success |
| `provision` | `LifecycleData` | ✅ Complete | Action, machine, success |
| `show` | `map[string]string` | ✅ Complete | nspawn, slice, drop-in, hooks file contents |
| `edit` | `CreateData` | ✅ Complete | Same as create |
| `snapshot` | `SnapshotData` | ✅ Complete | Action, machine, snapshot, backend, success |
| `list-snapshots` | `SnapshotListData` | ✅ Complete | Machine, backend, snapshots array |
| `rollback` | `SnapshotData` | ✅ Complete | Action, machine, snapshot, backend, success |
| `delete-snapshot` | `SnapshotData` | ✅ Complete | Action, machine, snapshot, backend, success |
| `setup` | `SetupData` | ✅ Complete | group_created, dirs_created, polkit_installed, sudoers_installed, success |

### Commands without JSON Support ❌

| Command | Reason | Impact on Cockpit UI |
|---------|--------|----------------------|
| `shell` | Interactive command | ❌ Not needed (use Cockpit terminal) |
| `exec` | Interactive command | ⚠️ May need for provisioning status |
| `logs` | Streaming output | ⚠️ May need for log viewer |

---

## JSON Schema Analysis

### Current Schemas (from `internal/output/schemas.go`)

```go
// Machine list item
type MachineJSON struct {
    Name       string        `json:"name"`
    Distro     string        `json:"distro"`
    State      string        `json:"state"`
    Network    string        `json:"network"`
    Addresses  []string      `json:"addresses,omitempty"`
    Resources  *ResourceJSON `json:"resources,omitempty"`
    CreatedAt  string        `json:"created_at"`
    SourcePath string        `json:"source_path,omitempty"`
}

// Machine status (detailed)
type MachineStatusData struct {
    Name       string        `json:"name"`
    Distro     string        `json:"distro"`
    State      string        `json:"state"`
    Leader     int           `json:"leader,omitempty"`
    Addresses  []string      `json:"addresses,omitempty"`
    Network    string        `json:"network"`
    Resources  *ResourceJSON `json:"resources,omitempty"`
    CreatedAt  string        `json:"created_at"`
    SourcePath string        `json:"source_path,omitempty"`
    Files      []string      `json:"files,omitempty"`
}

// Resources
type ResourceJSON struct {
    CPUs   float64 `json:"cpus,omitempty"`
    Memory string  `json:"memory,omitempty"`
    Swap   string  `json:"swap,omitempty"`
    Tasks  string  `json:"tasks,omitempty"`
}

// Snapshot
type SnapshotJSON struct {
    Name      string `json:"name"`
    Machine   string `json:"machine"`
    Backend   string `json:"backend"`
    CreatedAt string `json:"created_at"`
    Path      string `json:"path,omitempty"`
    Dataset   string `json:"dataset,omitempty"`
    LV        string `json:"lv,omitempty"`
}
```

---

## Cockpit UI Requirements vs JSON Coverage

### 1. Main Dashboard (Machine List)

**Required Data:**
- ✅ Machine name → `MachineJSON.Name`
- ✅ Distro → `MachineJSON.Distro`
- ✅ State → `MachineJSON.State`
- ✅ Network mode → `MachineJSON.Network`
- ✅ IP addresses → `MachineJSON.Addresses`
- ✅ CPU usage → ❌ **MISSING** (need runtime stats)
- ✅ Memory usage → ❌ **MISSING** (need runtime stats)
- ✅ Resources config → `MachineJSON.Resources`
- ✅ Created time → `MachineJSON.CreatedAt`

**Verdict:** ⚠️ **80% covered** - Missing runtime CPU/memory usage stats

---

### 2. Machine Details Page

#### 2.1 Overview Card

**Required Data:**
- ✅ State → `MachineStatusData.State`
- ✅ Distro → `MachineStatusData.Distro`
- ✅ CPU count → `MachineStatusData.Resources.CPUs`
- ✅ Memory → `MachineStatusData.Resources.Memory`
- ✅ Network mode → `MachineStatusData.Network`
- ✅ IP addresses → `MachineStatusData.Addresses`
- ✅ Created time → `MachineStatusData.CreatedAt`
- ✅ Source YAML path → `MachineStatusData.SourcePath`
- ✅ Generated files → `MachineStatusData.Files`

**Verdict:** ✅ **100% covered**

#### 2.2 Usage Card (CPU/Memory Graphs)

**Required Data:**
- ❌ CPU usage % → **MISSING** (need cgroup stats)
- ❌ Memory usage bytes → **MISSING** (need cgroup stats)
- ❌ Memory total → **MISSING** (need cgroup stats)
- ❌ Historical data → **MISSING** (need polling)

**Verdict:** ❌ **0% covered** - Need new command for runtime stats

#### 2.3 Console Card

**Required Data:**
- ❌ Terminal access → **NOT APPLICABLE** (use Cockpit terminal component)

**Verdict:** N/A - Use Cockpit's built-in terminal

#### 2.4 Volumes Card (Bind Mounts)

**Required Data:**
- ❌ Host paths → **MISSING**
- ❌ Container paths → **MISSING**
- ❌ Readonly flag → **MISSING**

**Verdict:** ❌ **0% covered** - Need to parse YAML or add to status

#### 2.5 Network Card

**Required Data:**
- ✅ Network mode → `MachineStatusData.Network`
- ✅ IP addresses → `MachineStatusData.Addresses`
- ❌ Bridge name → **MISSING**
- ❌ Interface name → **MISSING**
- ❌ Port forwarding rules → **MISSING**

**Verdict:** ⚠️ **40% covered** - Missing detailed network config

#### 2.6 Snapshots Card

**Required Data:**
- ✅ Snapshot name → `SnapshotJSON.Name`
- ✅ Created time → `SnapshotJSON.CreatedAt`
- ✅ Backend type → `SnapshotJSON.Backend`
- ❌ Description → **MISSING**

**Verdict:** ⚠️ **75% covered** - Missing description field

#### 2.7 Provisioning Card

**Required Data:**
- ❌ Provisioning status → **MISSING**
- ❌ Last run time → **MISSING**
- ❌ Phases completed → **MISSING**
- ❌ Probes passed → **MISSING**
- ❌ Script hashes → **MISSING**

**Verdict:** ❌ **0% covered** - Need new command for provisioning state

#### 2.8 Security Card

**Required Data:**
- ❌ Capabilities → **MISSING**
- ❌ Dropped capabilities → **MISSING**
- ❌ Seccomp status → **MISSING**
- ❌ No new privileges → **MISSING**
- ❌ Inaccessible paths → **MISSING**

**Verdict:** ❌ **0% covered** - Need to parse YAML or add to status

#### 2.9 GPU Card

**Required Data:**
- ❌ GPU enabled → **MISSING**
- ❌ GPU type → **MISSING**
- ❌ GPU devices → **MISSING**

**Verdict:** ❌ **0% covered** - Need to parse YAML or add to status

#### 2.10 Hooks Card

**Required Data:**
- ❌ Pre-start script → **MISSING** (available in `show` command)
- ❌ Post-start script → **MISSING** (available in `show` command)
- ❌ Post-stop script → **MISSING** (available in `show` command)

**Verdict:** ⚠️ **50% covered** - Available via `show --json` but not structured

#### 2.11 Advanced Card

**Required Data:**
- ❌ Ephemeral mode → **MISSING**
- ❌ Read-only rootfs → **MISSING**
- ❌ Hostname → **MISSING**
- ❌ Environment variables → **MISSING**

**Verdict:** ❌ **0% covered** - Need to parse YAML or add to status

---

### 3. Create Machine Wizard

**Required Data:**
- ✅ Create from YAML → `create` command
- ❌ List available distros → **MISSING**
- ❌ Validate YAML → **MISSING** (need dry-run)
- ❌ Preview generated config → ✅ Available via `show`

**Verdict:** ⚠️ **50% covered** - Missing distro list and validation

---

### 4. Dialogs

#### 4.1 Delete Machine

**Required Data:**
- ✅ Delete machine → `delete` command
- ✅ Confirmation → `LifecycleData.Success`

**Verdict:** ✅ **100% covered**

#### 4.2 Edit CPU/Memory

**Required Data:**
- ❌ Edit resources → **MISSING** (need `edit-resources` command)

**Verdict:** ❌ **0% covered** - Need new command

#### 4.3 Edit Network

**Required Data:**
- ❌ Edit network config → **MISSING** (need `edit-network` command)

**Verdict:** ❌ **0% covered** - Need new command

#### 4.4 Add/Edit Volume

**Required Data:**
- ❌ Add volume → **MISSING** (need `add-volume` command)
- ❌ Edit volume → **MISSING** (need `edit-volume` command)

**Verdict:** ❌ **0% covered** - Need new commands

#### 4.5 Snapshot Operations

**Required Data:**
- ✅ Create snapshot → `snapshot` command
- ✅ List snapshots → `list-snapshots` command
- ✅ Revert snapshot → `rollback` command
- ✅ Delete snapshot → `delete-snapshot` command

**Verdict:** ✅ **100% covered**

---

## Missing JSON Outputs

### High Priority (Needed for MVP)

1. **Runtime Stats Command**
   ```bash
   lxspawn stats <name> --json
   ```
   **Output:**
   ```json
   {
     "name": "myserver",
     "cpu_usage_percent": 15.5,
     "memory_used_bytes": 536870912,
     "memory_total_bytes": 2147483648,
     "memory_usage_percent": 25.0,
     "disk_used_bytes": 1073741824,
     "disk_total_bytes": 10737418240
   }
   ```

2. **Detailed Config Command**
   ```bash
   lxspawn config <name> --json
   ```
   **Output:**
   ```json
   {
     "name": "myserver",
     "distro": "debian:bookworm",
     "resources": {
       "cpus": 2,
       "memory": "2G",
       "memory_high": "1.5G",
       "swap": "512M",
       "tasks": 512
     },
     "network": {
       "mode": "bridge",
       "bridge": "br0",
       "interface": null,
       "ports": ["tcp:8080:80"]
     },
     "volumes": [
       {
         "host": "/mnt/data",
         "mount": "/data",
         "readonly": false
       }
     ],
     "security": {
       "no_new_privileges": true,
       "seccomp": true,
       "capabilities": ["CAP_NET_ADMIN"],
       "drop_capabilities": ["CAP_SYS_RAWIO"],
       "inaccessible": ["/etc/shadow"]
     },
     "gpu": {
       "enabled": true,
       "type": "intel"
     },
     "advanced": {
       "ephemeral": false,
       "read_only": false,
       "hostname": "myserver",
       "environment": {"FOO": "bar"}
     }
   }
   ```

3. **Provisioning State Command**
   ```bash
   lxspawn provision-status <name> --json
   ```
   **Output:**
   ```json
   {
     "name": "myserver",
     "last_run": "2024-01-15T10:30:00Z",
     "phases_completed": ["pre-boot", "post-boot", "probes"],
     "probes_passed": ["nginx", "postgresql"],
     "probes_failed": [],
     "script_hashes": {
       "system:0": "sha256:abc..."
     }
   }
   ```

4. **List Distros Command**
   ```bash
   lxspawn list-distros --json
   ```
   **Output:**
   ```json
   {
     "distros": [
       {"name": "debian:bookworm", "description": "Debian 12 (stable)"},
       {"name": "debian:trixie", "description": "Debian 13 (testing)"},
       {"name": "ubuntu:noble", "description": "Ubuntu 24.04 LTS"},
       {"name": "fedora:40", "description": "Fedora 40"}
     ]
   }
   ```

### Medium Priority (Needed for Advanced Features)

5. **Edit Resources Command**
   ```bash
   lxspawn edit-resources <name> --cpus 4 --memory 4G --json
   ```

6. **Edit Network Command**
   ```bash
   lxspawn edit-network <name> --mode bridge --bridge br0 --json
   ```

7. **Add Volume Command**
   ```bash
   lxspawn add-volume <name> --host /mnt/data --mount /data --json
   ```

8. **Edit Volume Command**
   ```bash
   lxspawn edit-volume <name> --host /mnt/data --mount /data --readonly --json
   ```

9. **Remove Volume Command**
   ```bash
   lxspawn remove-volume <name> --mount /data --json
   ```

### Low Priority (Nice to Have)

10. **Logs Command (JSON)**
    ```bash
    lxspawn logs <name> --json --lines 100
    ```
    **Output:**
    ```json
    {
      "name": "myserver",
      "entries": [
        {
          "timestamp": "2024-01-15T10:30:00Z",
          "priority": 6,
          "message": "Started nginx.service"
        }
      ]
    }
    ```

11. **Validate YAML Command**
    ```bash
    lxspawn validate myserver.yaml --json
    ```
    **Output:**
    ```json
    {
      "valid": true,
      "errors": [],
      "warnings": []
    }
    ```

12. **List Templates Command**
    ```bash
    lxspawn list-templates --json
    ```
    **Output:**
    ```json
    {
      "templates": [
        {
          "name": "docker-host",
          "description": "Docker-in-container with full support",
          "path": "/usr/share/lxspawn/templates/docker-host.yaml"
        }
      ]
    }
    ```

---

## Implementation Plan

### Phase 1: Core JSON API (MVP)

**New Commands:**
1. `lxspawn stats <name> --json` - Runtime CPU/memory stats
2. `lxspawn config <name> --json` - Detailed config (volumes, network, security, GPU, advanced)
3. `lxspawn list-distros --json` - Available distros

**Enhancements:**
1. Add `description` field to `SnapshotJSON`
2. Add `ephemeral`, `read_only` to `MachineStatusData`

**Coverage:** 80% of MVP requirements

### Phase 2: Advanced JSON API

**New Commands:**
1. `lxspawn provision-status <name> --json` - Provisioning state
2. `lxspawn edit-resources <name> --json` - Edit CPU/memory
3. `lxspawn edit-network <name> --json` - Edit network config
4. `lxspawn add-volume <name> --json` - Add bind mount
5. `lxspawn edit-volume <name> --json` - Edit bind mount
6. `lxspawn remove-volume <name> --json` - Remove bind mount

**Coverage:** 95% of advanced features

### Phase 3: Polish JSON API

**New Commands:**
1. `lxspawn logs <name> --json` - Structured log output
2. `lxspawn validate <yaml> --json` - YAML validation
3. `lxspawn list-templates --json` - Available templates

**Coverage:** 100% of requirements

---

## Alternative Approach: Parse YAML Directly

Instead of adding many new commands, the Cockpit UI could:

1. Read the source YAML file directly (path available in `MachineStatusData.SourcePath`)
2. Parse YAML on the client side (JavaScript YAML parser)
3. Extract volumes, network, security, GPU, advanced settings

**Pros:**
- Fewer new commands needed
- YAML is the source of truth
- Easier to maintain

**Cons:**
- Client-side YAML parsing adds complexity
- Need to handle YAML schema changes
- Doesn't solve runtime stats (still need `stats` command)

**Recommended Hybrid Approach:**
- Use `config` command for parsed/validated config
- Use `stats` command for runtime data
- Use `show` command for generated systemd files
- Parse YAML only for editing (send updated YAML to `edit` command)

---

## Conclusion

**Current Coverage:** ~40% of Cockpit UI requirements

**What's Missing:**
1. ❌ Runtime CPU/memory stats (critical for dashboard)
2. ❌ Detailed config (volumes, network, security, GPU, advanced)
3. ❌ Provisioning status
4. ❌ Edit commands for resources, network, volumes
5. ❌ Distro list for create wizard

**What's Already Covered:**
1. ✅ Machine list with state, network, resources
2. ✅ Machine status with leader PID, files
3. ✅ Lifecycle operations (start, stop, delete, provision)
4. ✅ Snapshot operations (create, list, rollback, delete)
5. ✅ Generated config files (show command)

**Recommendation:**

**YES, we can add sufficient `--json` outputs to cover the Cockpit UI**, but we need:

1. **3-4 new commands** for MVP (stats, config, list-distros)
2. **6-7 new commands** for advanced features (edit-*, add-volume, etc.)
3. **Enhance existing schemas** (add description to snapshots, etc.)

**Estimated Effort:**
- Phase 1 (MVP): 2-3 days
- Phase 2 (Advanced): 3-4 days
- Phase 3 (Polish): 1-2 days

**Total:** ~1-2 weeks of development

This is very achievable and aligns with the Phase 5 (Cockpit UI) roadmap in TODO.md.
