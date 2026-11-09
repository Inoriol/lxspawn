# Cockpit-Machines UI Analysis for lxspawn

This document analyzes the cockpit-machines plugin screens and options, and identifies what's applicable to lxspawn's Cockpit UI (Phase 5).

## Overview

cockpit-machines is a Cockpit plugin for managing virtual machines via libvirt. It provides a comprehensive web UI for VM lifecycle management, resource configuration, and monitoring.

**Key Characteristics:**
- React/PatternFly-based UI
- Communicates with libvirt via D-Bus API
- Supports both system and session connections
- Real-time monitoring and state management
- Comprehensive VM configuration options

---

## Screen Inventory

### 1. Main VM List Screen (`hostvmslist.tsx`)

**Purpose:** Dashboard showing all VMs on the host

**Features:**
- **Aggregate Status Cards** (top)
  - Storage pools count (active/inactive)
  - Networks count (active/inactive)
  - Clickable links to storage pools and networks pages
  
- **VM Table** with columns:
  - Name (clickable link to details)
  - Connection (system/session)
  - State (running, shut off, paused, etc.) with state icon
  - CPU usage (progress bar, only for running VMs)
  - Memory usage (progress bar, only for running VMs)
  - Actions (context menu)

- **Toolbar:**
  - Filter by name (text search)
  - Filter by state (dropdown: All, running, shut off, paused, etc.)
  - Create VM button
  - Import VM button

- **State Indicators:**
  - Download progress (for VMs being created)
  - "Creating VM" status
  - Needs shutdown indicator (pending configuration changes)
  - Uses SPICE indicator (legacy display protocol warning)

**Applicable to lxspawn:**
- ✅ Aggregate status cards (adapt for lxspawn resources)
- ✅ VM list table with state, CPU, memory
- ✅ Filter by name and state
- ✅ Create machine button
- ❌ Connection selector (lxspawn doesn't have system/session concept)
- ❌ Import VM (lxspawn uses YAML configs, not disk images)

---

### 2. VM Details Page (`vmDetailsPage.tsx`)

**Purpose:** Detailed view and management of a single VM

**Layout:** Breadcrumb navigation + action bar + card grid

**Cards:**

#### 2.1 Overview Card (`vmOverviewCard.tsx`)

**General Section:**
- Connection (system/session)
- State (with state icon)
- Memory (editable via modal)
- CPU (vCPU count, mode, model - editable via modal)
- Boot order (editable via modal)
- Autostart (toggle switch)
- Watchdog (editable via modal, with info popover)
- Vsock (editable via modal, with info popover)

**Hypervisor Details Section:**
- Emulated machine type
- Firmware (BIOS/UEFI - editable via modal)

**Applicable to lxspawn:**
- ✅ State display
- ✅ Memory display and edit
- ✅ CPU display and edit
- ✅ Autostart toggle
- ❌ Boot order (lxspawn doesn't have boot devices)
- ❌ Watchdog (not applicable to containers)
- ❌ Vsock (not applicable to containers)
- ❌ Firmware (not applicable to containers)
- ❌ Emulated machine (not applicable to containers)

#### 2.2 Usage Card (`vmUsageCard.tsx`)

**Features:**
- CPU usage graph over time
- Memory usage graph over time
- Real-time polling (configurable)

**Applicable to lxspawn:**
- ✅ CPU usage graph (via systemd-cgtop or cgroup stats)
- ✅ Memory usage graph (via cgroup stats)
- ✅ Real-time polling

#### 2.3 Console Card (`consoles.tsx`)

**Features:**
- Multiple console types:
  - VNC (graphical console)
  - SPICE (graphical console, legacy)
  - Serial (text console via PTY)
- Expand/collapse toggle
- Detached window mode
- Console type selector

**Applicable to lxspawn:**
- ❌ VNC/SPICE (no graphical console in containers)
- ✅ Serial console (adapt as "shell" access)
- ✅ Alternative: Embed Cockpit's terminal component
- ✅ Alternative: Link to `machinectl shell` or `nsenter`

#### 2.4 Disks Card (`vmDisksCard.tsx`)

**Features:**
- Table of virtual disks:
  - Device
  - Source (storage pool/volume, file path, URL)
  - Target (virtual device name)
  - Bus type (virtio, ide, scsi, etc.)
  - Driver type (qcow2, raw, etc.)
  - Used/Capacity (for running VMs)
  - Access mode (readonly, shared)
  - Cache mode
  - Discard mode
  - Serial number
- Actions: Add disk, Edit, Delete, Eject (for CD-ROM)

**Applicable to lxspawn:**
- ❌ Virtual disks (lxspawn uses bind mounts, not virtual disks)
- ✅ Adapt as "Volumes" card showing bind mounts
- ✅ Show host path, container path, readonly status
- ✅ Add/edit/delete bind mounts

#### 2.5 Network Interfaces Card (`vmNicsCard.tsx`)

**Features:**
- Table of virtual network interfaces:
  - Interface type (network, bridge, device)
  - Source (network name, bridge name, device name)
  - Model (virtio, e1000, rtl8139, etc.)
  - MAC address
  - IP addresses (for running VMs, via DHCP lease lookup)
  - Port forwarding rules
  - State (up/down)
- Actions: Add interface, Edit, Delete

**Applicable to lxspawn:**
- ✅ Network mode display (host, private, bridge, macvlan, ipvlan, zone)
- ✅ Network interface details
- ✅ IP address display
- ✅ Port forwarding rules display
- ✅ Add/edit/delete network configuration
- ❌ Model selection (not applicable to containers)
- ❌ MAC address editing (systemd-nspawn manages this)

#### 2.6 Host Devices Card (`hostDevCard.tsx`)

**Features:**
- Table of PCI/USB devices passed through to VM
- Actions: Add host device, Delete

**Applicable to lxspawn:**
- ✅ GPU pass-through display (already in lxspawn)
- ✅ Add/edit GPU configuration
- ❌ General PCI passthrough (not applicable to containers)

#### 2.7 Snapshots Card (`vmSnapshotsCard.tsx`)

**Features:**
- Table of snapshots:
  - Name
  - Description
  - Creation time (relative and absolute)
  - VM state (running, shut off, disk-snapshot)
  - Parent snapshot
  - Current snapshot indicator
- Actions: Create snapshot, Revert, Delete

**Applicable to lxspawn:**
- ✅ Snapshot list (already in lxspawn CLI)
- ✅ Snapshot creation dialog
- ✅ Snapshot revert
- ✅ Snapshot delete
- ✅ Description field
- ✅ Creation time display
- ❌ Parent snapshot (lxspawn snapshots are flat, not hierarchical)
- ❌ VM state in snapshot (lxspawn doesn't save VM state)

#### 2.8 Shared Directories Card (`vmFilesystemsCard.tsx`)

**Features:**
- Table of shared directories (virtiofs):
  - Mount tag
  - Source path (host)
  - Mount point (guest)
  - Driver type (virtiofs)
- Actions: Add, Edit, Delete
- Info popover with mount instructions

**Applicable to lxspawn:**
- ✅ Bind mounts display (similar concept)
- ✅ Already covered in Volumes card
- ❌ virtiofs (lxspawn uses bind mounts, not virtiofs)

---

### 3. Create VM Dialog (`createVmDialog.tsx`)

**Purpose:** Create or import a new VM

**Tabs:**

#### 3.1 Details Tab (`details.tsx`)

**Fields:**
- **Name:** VM name (auto-generated from OS if empty)
- **Connection:** system/session selector
- **Source type:**
  - Download an OS (from libosinfo database)
  - Cloud image (with cloud-init support)
  - Local install media (ISO)
  - URL (network installation)
  - PXE (network boot, system only)
  - Existing disk image (import mode)
- **OS:** Operating system selector (from libosinfo)
  - Auto-detect from ISO
  - Filter by vendor, OS family
  - Show EOL warnings
- **Memory:** Size and unit (GiB, MiB)
- **Storage:**
  - Storage pool selector
  - Volume selector (for existing volumes)
  - New volume size (for new volumes)
  - Volume format (qcow2, raw)
- **Profile:** Desktop/Server selection (for cloud images)

#### 3.2 Automation Tab (`automation.tsx`)

**Fields:**
- **Root password:** For unattended installation
- **User login:** Username for unattended installation
- **User password:** Password for unattended installation
- **SSH keys:** List of SSH public keys to inject
- **Profile:** Desktop/Server (for cloud images)
- **Extra arguments:** Additional virt-install arguments

**Applicable to lxspawn:**
- ✅ Name field
- ❌ Connection selector (not applicable)
- ❌ Source type (lxspawn uses YAML, not ISO/images)
- ✅ OS/Distro selector (adapt for lxspawn distro list)
- ✅ Memory field
- ❌ Storage pool (lxspawn uses bind mounts)
- ❌ Automation tab (lxspawn uses YAML provisioning)
- ✅ Alternative: YAML editor or file upload
- ✅ Alternative: Form-based YAML generator

---

### 4. Storage Pools List (`storagePoolList.tsx`)

**Purpose:** Manage storage pools (libvirt concept)

**Features:**
- Table of storage pools:
  - Name
  - Size (used/total)
  - Connection (system/session)
  - State (active/inactive)
  - Actions (start, stop, delete, refresh)
- Create storage pool button

**Applicable to lxspawn:**
- ❌ Storage pools (not applicable to containers)
- ❌ lxspawn uses host filesystem, not storage pools

---

### 5. Storage Pool Details

**Tabs:**

#### 5.1 Overview Tab (`storagePoolOverviewTab.tsx`)

**Features:**
- Pool type (dir, netfs, iscsi, disk, logical, fs, etc.)
- Source (host, path, format, etc.)
- Target path
- Size (used/total/available)
- State (active/inactive)
- Autostart toggle

**Applicable to lxspawn:**
- ❌ Not applicable

#### 5.2 Volumes Tab (`storagePoolVolumesTab.tsx`)

**Features:**
- Table of volumes in pool:
  - Name
  - Size
  - Format
  - Used by (VMs)
- Actions: Create volume, Delete volume

**Applicable to lxspawn:**
- ❌ Not applicable (lxspawn doesn't manage volumes)

---

### 6. Create Storage Pool Dialog (`createStoragePoolDialog.tsx`)

**Fields:**
- Name
- Type (dir, netfs, iscsi, disk, logical, fs, iscsi-direct)
- Target path (for dir, netfs, iscsi, disk, fs)
- Host (for netfs, iscsi, iscsi-direct)
- Source path (for netfs, iscsi)
- Source format (for disk, fs)
- Initiator (for iscsi-direct)
- Autostart toggle

**Applicable to lxspawn:**
- ❌ Not applicable

---

### 7. Networks List (`networkList.tsx`)

**Purpose:** Manage virtual networks (libvirt concept)

**Features:**
- Table of networks:
  - Name
  - Device (bridge name)
  - Connection (system/session)
  - Forwarding mode (nat, open, none, bridge, etc.)
  - State (active/inactive)
  - Actions (start, stop, delete)
- Create network button

**Applicable to lxspawn:**
- ❌ Virtual networks (lxspawn uses host networking, bridges, macvlan)
- ✅ Alternative: Show host network interfaces
- ✅ Alternative: Show bridge devices
- ✅ Alternative: Network mode configuration

---

### 8. Network Details

**Features:**
- Overview tab:
  - Network name
  - Device (bridge)
  - Forwarding mode
  - IPv4 configuration (address, netmask, DHCP range)
  - IPv6 configuration (address, prefix, DHCP range)
  - State (active/inactive)
  - Autostart toggle

**Applicable to lxspawn:**
- ❌ Not applicable (lxspawn doesn't create virtual networks)

---

### 9. Create Network Dialog (`createNetworkDialog.tsx`)

**Fields:**
- Name
- Forward mode (nat, open, none)
- Device (for bridge mode)
- IPv4 configuration:
  - Network address
  - Netmask or prefix length
  - DHCP enabled toggle
  - DHCP range start/end
- IPv6 configuration:
  - Network address
  - Prefix length
  - DHCP enabled toggle
  - DHCP range start/end

**Applicable to lxspawn:**
- ❌ Not applicable (lxspawn uses existing host networks)

---

### 10. VM Action Menus (`vmActions.tsx`)

**Actions:**
- **Run** (when shut off)
- **Install** (when has install phase)
- **Shutdown** (graceful, when running)
- **Force off** (when running)
- **Reboot** (graceful, when running)
- **Force reboot** (when running)
- **Pause** (when running)
- **Resume** (when paused)
- **Send non-maskable interrupt** (when running)
- **Clone** (when shut off)
- **Migrate** (when running or shut off)
- **Rename** (when shut off)
- **Delete** (when shut off)
- **Edit description** (any state)
- **Add TPM** (when shut off)
- **Replace SPICE** (when uses SPICE)

**Applicable to lxspawn:**
- ✅ Start (lxspawn start)
- ✅ Stop (lxspawn stop)
- ✅ Force stop (lxspawn stop --force)
- ✅ Reboot (lxspawn exec -- systemctl reboot)
- ✅ Delete (lxspawn delete)
- ✅ Edit (lxspawn edit)
- ❌ Clone (not in lxspawn, but could be added)
- ❌ Migrate (not applicable to containers)
- ❌ Rename (not in lxspawn, but could be added)
- ❌ Pause/Resume (not applicable to containers)
- ❌ Send NMI (not applicable to containers)
- ❌ Add TPM (not applicable to containers)
- ❌ Replace SPICE (not applicable to containers)

---

### 11. Dialogs and Modals

#### 11.1 Delete Dialog (`deleteDialog.tsx`)

**Options:**
- Delete storage (checkbox)
- Delete associated storage volumes (checkbox)

**Applicable to lxspawn:**
- ✅ Delete machine confirmation
- ✅ Option to preserve volumes (bind mounts)

#### 11.2 Clone Dialog (`vmCloneDialog.tsx`)

**Fields:**
- New VM name
- Clone storage (checkbox)

**Applicable to lxspawn:**
- ❌ Not in lxspawn (could be added as "copy config")

#### 11.3 Migrate Dialog (`vmMigrateDialog.tsx`)

**Fields:**
- Destination host
- Connection (system/session)
- Temporary/permanent migration
- Auto-converge toggle

**Applicable to lxspawn:**
- ❌ Not applicable (containers don't migrate)

#### 11.4 Rename Dialog (`vmRenameDialog.tsx`)

**Fields:**
- New name

**Applicable to lxspawn:**
- ✅ Rename machine (not in lxspawn, but could be added)

#### 11.5 Edit Description Dialog (`vmEditDescriptionDialog.tsx`)

**Fields:**
- Description (text area)

**Applicable to lxspawn:**
- ✅ Edit machine description (not in lxspawn, but could be added)

#### 11.6 CPU Modal (`cpuModal.tsx`)

**Fields:**
- vCPU count
- CPU model
- CPU mode (host-passthrough, host-model, custom)

**Applicable to lxspawn:**
- ✅ Edit CPU count (resources.cpus)
- ❌ CPU model (not applicable to containers)

#### 11.7 Memory Modal (`memoryModal.tsx`)

**Fields:**
- Current memory
- Maximum memory
- Unit (GiB, MiB)

**Applicable to lxspawn:**
- ✅ Edit memory (resources.memory, resources.memory_high)

#### 11.8 Boot Order Modal (`bootOrder.tsx`)

**Fields:**
- List of boot devices (disks, network, etc.)
- Drag-and-drop reordering

**Applicable to lxspawn:**
- ❌ Not applicable (containers don't have boot devices)

#### 11.9 Firmware Modal (`firmware.tsx`)

**Fields:**
- Firmware type (BIOS, UEFI)
- Secure boot toggle

**Applicable to lxspawn:**
- ❌ Not applicable (containers use host kernel)

#### 11.10 Watchdog Modal (`watchdog.tsx`)

**Fields:**
- Watchdog model (i6300esb, ib700, etc.)
- Action (reset, shutdown, poweroff, pause, none)

**Applicable to lxspawn:**
- ❌ Not applicable (containers don't have watchdog devices)

#### 11.11 Vsock Modal (`vsock.tsx`)

**Fields:**
- CID (Context Identifier)
- Auto-assign toggle

**Applicable to lxspawn:**
- ❌ Not applicable (containers don't use vsock)

#### 11.12 Add Disk Dialog (`diskAdd.tsx`)

**Fields:**
- Source (existing volume, new volume, file, URL)
- Device (disk, cdrom, floppy)
- Bus type (virtio, ide, scsi, etc.)
- Target (device name)
- Driver type (qcow2, raw, etc.)
- Cache mode
- Discard mode
- Serial number
- Size (for new volumes)
- Format (for new volumes)
- Storage pool (for new volumes)

**Applicable to lxspawn:**
- ✅ Adapt as "Add Volume" dialog
- ✅ Host path
- ✅ Container path
- ✅ Readonly toggle
- ❌ Bus type, driver type, cache mode (not applicable)

#### 11.13 Edit Disk Dialog (`diskEdit.tsx`)

**Fields:**
- Similar to Add Disk, but for editing existing disk

**Applicable to lxspawn:**
- ✅ Adapt as "Edit Volume" dialog

#### 11.14 Add Network Interface Dialog (`nicAdd.tsx`)

**Fields:**
- Interface type (network, bridge, device)
- Source (network name, bridge name, device name)
- Model (virtio, e1000, rtl8139, etc.)
- MAC address (optional)

**Applicable to lxspawn:**
- ✅ Network mode selector
- ✅ Bridge name (for bridge mode)
- ✅ Interface name (for macvlan/ipvlan mode)
- ❌ Model selection (not applicable)
- ❌ MAC address (not applicable)

#### 11.15 Edit Network Interface Dialog (`nicEdit.tsx`)

**Fields:**
- Similar to Add Interface, but for editing existing interface

**Applicable to lxspawn:**
- ✅ Edit network configuration

#### 11.16 Create Snapshot Modal (`vmSnapshotsCreateModal.tsx`)

**Fields:**
- Name
- Description
- Memory dump toggle (save VM state)
- Disk-only toggle

**Applicable to lxspawn:**
- ✅ Snapshot name
- ✅ Snapshot description
- ❌ Memory dump (not applicable to containers)
- ❌ Disk-only (all lxspawn snapshots are disk-only)

#### 11.17 Revert Snapshot Modal (`vmSnapshotsRevertModal.tsx`)

**Options:**
- Confirmation message
- Warning about data loss

**Applicable to lxspawn:**
- ✅ Revert confirmation
- ✅ Warning about data loss

---

## Applicable Features Summary

### High Priority (Direct Mapping)

1. **Main Dashboard**
   - Machine list with state, CPU, memory
   - Filter by name and state
   - Aggregate status cards (adapt for lxspawn resources)

2. **Machine Details Page**
   - Overview card (state, CPU, memory, autostart)
   - Usage card (CPU/memory graphs)
   - Volumes card (bind mounts)
   - Network card (network mode, interfaces, IPs, ports)
   - Snapshots card (list, create, revert, delete)
   - Console card (adapt as terminal/shell access)

3. **Machine Creation**
   - Name field
   - Distro selector
   - Resource configuration (CPU, memory)
   - Network configuration
   - Volume configuration
   - Alternative: YAML editor or file upload

4. **Machine Actions**
   - Start, Stop, Force stop, Reboot, Delete, Edit

5. **Dialogs**
   - Delete confirmation
   - CPU/Memory edit
   - Volume add/edit
   - Network configuration
   - Snapshot create/revert/delete

### Medium Priority (Adaptation Needed)

1. **GPU Pass-through**
   - Display GPU configuration
   - Add/edit GPU settings

2. **Provisioning**
   - Display provisioning status
   - Show provisioning scripts
   - Re-run provisioning

3. **Hooks**
   - Display hook scripts
   - Edit hooks

4. **Security**
   - Display security settings
   - Edit capabilities
   - Edit seccomp settings

5. **Advanced Settings**
   - Ephemeral mode
   - Read-only rootfs
   - Hostname
   - Environment variables

### Low Priority (Nice to Have)

1. **Machine Clone**
   - Copy configuration to new machine

2. **Machine Rename**
   - Rename existing machine

3. **Machine Description**
   - Add/edit description

4. **Host Devices**
   - Display host devices (for reference)

### Not Applicable

1. **Storage Pools** (libvirt concept)
2. **Virtual Networks** (libvirt concept)
3. **Boot Order** (containers don't have boot devices)
4. **Firmware** (containers use host kernel)
5. **Watchdog** (not applicable to containers)
6. **Vsock** (not applicable to containers)
7. **Migration** (containers don't migrate)
8. **Pause/Resume** (not applicable to containers)
9. **Send NMI** (not applicable to containers)
10. **Add TPM** (not applicable to containers)
11. **SPICE/VNC** (no graphical console in containers)

---

## Recommended lxspawn UI Structure

### 1. Main Dashboard

**Aggregate Status Cards:**
- Machines (total/running/stopped)
- Snapshots (total)
- Storage (btrfs/zfs/lvm status)

**Machine List:**
- Name (link to details)
- Distro
- State (running, stopped, creating, etc.)
- CPU usage (progress bar)
- Memory usage (progress bar)
- Network mode
- Actions (context menu)

**Toolbar:**
- Filter by name
- Filter by state
- Create machine button (opens wizard or YAML editor)

### 2. Machine Details Page

**Breadcrumb:** Virtual machines > {machine-name}

**Action Bar:**
- Machine name
- Actions (Start, Stop, Delete, Edit, etc.)
- State indicator

**Cards:**

**Overview Card:**
- State
- Distro
- CPU (count, editable)
- Memory (size, editable)
- Network mode
- Autostart toggle
- Created time
- Source YAML path

**Usage Card:**
- CPU usage graph
- Memory usage graph

**Console Card:**
- Embedded terminal (Cockpit terminal component)
- Or link to `machinectl shell`

**Volumes Card:**
- Table of bind mounts:
  - Host path
  - Container path
  - Readonly
  - Actions (edit, delete)
- Add volume button

**Network Card:**
- Network mode
- Interface details
- IP addresses
- Port forwarding rules
- Edit network button

**Snapshots Card:**
- Table of snapshots:
  - Name
  - Description
  - Created time
  - Actions (revert, delete)
- Create snapshot button

**Provisioning Card:**
- Provisioning status
- Last run time
- Phases completed
- Probes passed
- Re-run provisioning button

**Security Card:**
- Capabilities
- Dropped capabilities
- Seccomp status
- No new privileges
- Inaccessible paths
- Edit security button

**GPU Card:**
- GPU enabled/disabled
- GPU type
- GPU devices
- Edit GPU button

**Hooks Card:**
- Pre-start hook
- Post-start hook
- Post-stop hook
- Edit hooks button

**Advanced Card:**
- Ephemeral mode
- Read-only rootfs
- Hostname
- Environment variables
- Edit advanced button

### 3. Create Machine Wizard

**Option A: Form-Based**
- Step 1: Name and distro
- Step 2: Resources (CPU, memory)
- Step 3: Network (mode, bridge, interface, ports)
- Step 4: Volumes (add bind mounts)
- Step 5: Security (capabilities, seccomp)
- Step 6: GPU (enable, type)
- Step 7: Provisioning (scripts, probes)
- Step 8: Review and create

**Option B: YAML Editor**
- Text editor with syntax highlighting
- Template selector (pre-filled YAML)
- Validation
- Create button

**Option C: Hybrid**
- Form for basic settings
- YAML editor for advanced settings
- Toggle between form and YAML

### 4. Dialogs

**Delete Machine:**
- Confirmation message
- Option to preserve volumes
- Delete button

**Edit CPU:**
- CPU count
- Save button

**Edit Memory:**
- Memory size
- Memory high (soft limit)
- Swap limit
- Save button

**Edit Network:**
- Network mode
- Bridge name (if bridge mode)
- Interface name (if macvlan/ipvlan)
- Port forwarding rules
- Save button

**Add/Edit Volume:**
- Host path
- Container path
- Readonly toggle
- Save button

**Create Snapshot:**
- Name
- Description
- Create button

**Revert Snapshot:**
- Confirmation message
- Warning about data loss
- Revert button

**Edit Security:**
- Capabilities (add/remove)
- Dropped capabilities (add/remove)
- Seccomp toggle
- No new privileges toggle
- Inaccessible paths (add/remove)
- Save button

**Edit GPU:**
- Enable/disable
- GPU type (auto, intel, amd, nvidia, mali, adreno)
- Save button

**Edit Hooks:**
- Pre-start script (text area)
- Post-start script (text area)
- Post-stop script (text area)
- Save button

**Edit Advanced:**
- Ephemeral toggle
- Read-only rootfs toggle
- Hostname
- Environment variables (key-value pairs)
- Save button

---

## Implementation Recommendations

### Phase 5a: Core UI (MVP)

1. Main dashboard with machine list
2. Machine details page with overview, usage, volumes, network cards
3. Basic actions (start, stop, delete)
4. Create machine wizard (form-based or YAML editor)
5. Basic dialogs (delete, edit CPU/memory)

### Phase 5b: Advanced Features

1. Snapshots card with create/revert/delete
2. Console card (embedded terminal)
3. Advanced dialogs (edit network, volumes, security)
4. Provisioning card
5. GPU card

### Phase 5c: Polish

1. Hooks card
2. Advanced settings card
3. Machine clone/rename
4. Machine description
5. Real-time updates via D-Bus signals

---

## Technical Considerations

### Data Source

**cockpit-machines:** Uses libvirt D-Bus API
**lxspawn:** Use `lxspawn --json` commands via `cockpit.spawn()`

### State Management

**cockpit-machines:** Redux-like state with D-Bus event listeners
**lxspawn:** Simpler state with periodic polling or D-Bus signals (future)

### Real-Time Updates

**cockpit-machines:** D-Bus signals for state changes
**lxspawn:** Periodic polling (every 5-10 seconds) or D-Bus signals (future enhancement)

### Terminal Integration

**cockpit-machines:** VNC/SPICE/Serial consoles
**lxspawn:** Cockpit terminal component or link to `machinectl shell`

### File Upload

**cockpit-machines:** ISO upload for installation
**lxspawn:** YAML file upload for machine creation

### Validation

**cockpit-machines:** Client-side validation with libvirt capabilities
**lxspawn:** Client-side validation + `lxspawn create --dry-run` (future)

---

## Conclusion

cockpit-machines provides a comprehensive reference for building a VM management UI. While many features are specific to libvirt/KVM, the overall structure, UX patterns, and UI components are highly applicable to lxspawn.

The key differences are:
- lxspawn uses YAML configs, not ISO/images
- lxspawn uses bind mounts, not virtual disks
- lxspawn uses host networking, not virtual networks
- lxspawn doesn't have boot devices, firmware, watchdog, etc.
- lxspawn has provisioning, hooks, and templates (unique features)

By adapting the cockpit-machines UI patterns and focusing on lxspawn-specific features, we can create a powerful and intuitive web UI for managing system containers.
