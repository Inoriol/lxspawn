# lxspawn Template Specification

This document defines the complete YAML schema for lxspawn machine configurations.

## Sample Manifest

The following example uses every available field. Using all fields together does not represent a realistic configuration; it exists solely to demonstrate the full schema.

```yaml
name: full-example
distro: debian:bookworm

resources:
  cpus: 2
  memory: 2G
  memory_high: 1.5G
  swap: 512M
  tasks: 512
  io_weight: 100

storage:
  size: 10G

network:
  mode: bridge
  bridge: br0
  zone: dev
  interface: eth0
  ports:
    - "tcp:8080:80"
    - "udp:5353:53"
    - "tcp:443"

volumes:
  - host: /mnt/data
    mount: /data
    readonly: false
  - host: /srv/www
    mount: /var/www/html
    readonly: true
  - type: tmpfs
    mount: /tmp
    options: "mode=1777,size=2G"

security:
  no_new_privileges: true
  private_users: "no"
  seccomp: true
  capabilities:
    - CAP_NET_ADMIN
    - CAP_NET_BIND_SERVICE
  drop_capabilities:
    - CAP_SYS_RAWIO
    - CAP_MKNOD
  inaccessible:
    - /etc/shadow
    - /etc/gshadow
  system_call_filter:
    - "@basic-io"
    - "@file-io"

advanced:
  ephemeral: false
  read_only: false
  hostname: myhost
  environment:
    FOO: bar
    BAZ: qux

journal:
  link: try-guest

hooks:
  pre_start: |
    #!/bin/bash
    set -euo pipefail
    modprobe br_netfilter
    echo 1 > /proc/sys/net/ipv4/ip_forward
  post_start: |
    #!/bin/bash
    ip link set dev ve-full-example master br2
    ip link set dev ve-full-example up
  post_stop: |
    echo "Container stopped"

provision:
  - mode: boot
    script: |
      #!/bin/bash
      systemctl mask systemd-udevd
      systemctl disable systemd-modules-load.service
  - mode: data
    path: /etc/myapp/config.yaml
    content: |
      hostname: {{.Name}}
      port: {{.Param.port}}
    owner: "root:root"
    permissions: "0644"
    overwrite: false
  - mode: data
    path: /etc/myapp/remote.conf
    file:
      url: https://example.com/config.conf
      digest: "sha256:abc123..."
  - mode: yq
    path: /etc/docker/daemon.json
    expression: '.storage-options |= . + ["overlay2"]'
    format: json
  - mode: system
    script: |
      #!/bin/bash
      set -eux -o pipefail
      apt-get update && apt-get install -y nginx
      systemctl enable --now nginx
  - mode: system
    file:
      url: setup.sh
      digest: "sha256:def456..."
  - mode: user
    script: |
      #!/bin/bash
      mkdir -p ~/.config/myapp
      cat > ~/.config/myapp/settings.json <<EOF
      {"machine": "{{.Name}}"}
      EOF

param:
  port: "8080"
  env: "production"

probes:
  - mode: readiness
    description: "nginx to be serving requests"
    script: |
      #!/bin/bash
      until curl -sf http://localhost/health; do sleep 2; done
    hint: |
      nginx may not have started. Check: journalctl -M {{.Name}} -u nginx
```

---

## Field Reference

### Top-Level Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | yes | — | Machine name. Must not contain spaces, slashes, or backslashes. Used as the systemd machine identifier and in all generated unit names. |
| `distro` | string | yes | — | Distribution image in `distro:release` format (e.g. `debian:bookworm`, `ubuntu:noble`, `fedora:44`). Used to fetch the rootfs from `images.linuxcontainers.org`. **Note:** Arch Linux is not supported on btrfs hosts due to filesystem compatibility issues (see TESTING-ISSUES.md). |
| `resources` | object | no | `{}` | Resource limits applied via a systemd slice unit. |
| `network` | object | no | `{mode: host}` | Network configuration for the container. |
| `volumes` | list | no | `[]` | Bind mounts and tmpfs mounts into the container. |
| `security` | object | no | see below | Security settings controlling capabilities, privileges, and seccomp. |
| `gpu` | object | no | `{enabled: false}` | GPU pass-through configuration for hardware acceleration. |
| `advanced` | object | no | `{}` | Advanced container options (ephemeral, read-only, hostname, environment). |
| `journal` | object | no | `{link: try-guest}` | Journal integration between host and container. |
| `hooks` | object | no | `{}` | Host-side lifecycle scripts executed at container start/stop. |
| `provision` | list | no | `[]` | Provisioning steps executed pre-boot or post-boot. |
| `param` | map | no | `{}` | User-defined key-value pairs available as template variables in provision scripts and probes. |
| `probes` | list | no | `[]` | Readiness probes that run after post-boot provisioning to verify services are available. |

---

### `resources`

All fields are optional. When any field is set, a systemd slice unit (`machine-<name>.slice`) is generated.

| Field | Type | Default | systemd mapping | Description |
|-------|------|---------|-----------------|-------------|
| `cpus` | float | `0` | `CPUQuota=<cpus*100>%` | CPU core limit. `2` becomes `CPUQuota=200%`. |
| `memory` | string | `""` | `MemoryMax=` | Hard memory limit. Container is OOM-killed if exceeded. Accepts systemd size suffixes (`K`, `M`, `G`, `T`). |
| `memory_high` | string | `""` | `MemoryHigh=` | Soft memory limit. Container is throttled before reaching the hard limit. |
| `swap` | string | `""` | `MemorySwapMax=` | Maximum swap usage. |
| `tasks` | string | `""` | `TasksMax=` | Maximum number of processes and threads. Accepts numeric values (e.g. `512`) or special values like `infinity` (required for Docker-in-container). |
| `io_weight` | int | `100` | `IOWeight=` | IO scheduling priority (1–10000). Default 100 is the systemd default; only written to the slice if set to a non-default value. |

---

### `storage`

All fields are optional. Controls storage allocation for the machine rootfs.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `size` | string | `"10G"` | Size of the storage volume for the machine rootfs. Used for LVM thin provisioning. Accepts size suffixes: `G` (gigabytes), `M` (megabytes), `T` (terabytes). Examples: `10G`, `500M`, `1T`. |

**Storage Backends:**

lxspawn automatically detects and uses the best storage backend based on your filesystem:

- **btrfs:** Machine rootfs is automatically created as a btrfs subvolume. The `storage.size` field has no effect. Snapshots use btrfs copy-on-write.
- **LVM with thin pool:** Machine rootfs is created as a thin volume from the thin pool. The `storage.size` field specifies the virtual size (can be larger than available space due to over-provisioning). Snapshots use LVM thin snapshots.
- **LVM without thin pool:** Falls back to directory mode with a warning. For efficient space usage, create a thin pool: `sudo lvcreate -L 50G -n lxspawn-pool <vg-name> && sudo lvconvert --type thin-pool <vg-name>/lxspawn-pool`
- **Other filesystems:** Uses directory-based storage in `/var/lib/machines/<name>/`. The `storage.size` field has no effect. Snapshots are not supported for directory mode.

**Notes:**

- The size must be large enough to hold the rootfs image plus any additional packages or data. A typical Debian rootfs is ~200MB; allow extra space for growth.
- On LVM thin, space is allocated on-demand, so you can specify a larger virtual size than physically available (over-provisioning).
- The selected backend is logged at startup: `Using storage backend: <type>`

---

### `network`

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `mode` | string | no | `host` | Network mode. One of: `host`, `private`, `bridge`, `zone`, `macvlan`, `ipvlan`, `adaptive`. |
| `bridge` | string | when `mode: bridge` | — | Name of the host bridge interface to attach to. The bridge must already exist on the host. For `adaptive` mode, specifies preferred bridge. |
| `zone` | string | when `mode: zone` | — | Network zone name. systemd-nspawn auto-creates a bridge named `vz-<zone>`. |
| `interface` | string | when `mode: macvlan` or `ipvlan` | — | Host network interface to create the macvlan/ipvlan on (e.g. `eth0`, `enp2s0`). For `adaptive` mode, use `auto` to use the interface detected by `lxspawn setup network`. |
| `ports` | list of strings | no | `[]` | Port forwarding rules. Each entry follows the format `[proto:]host_port:container_port` (e.g. `tcp:8080:80`, `udp:5353:53`). Requires a non-host network mode. |
| `host_access` | object | no | see below | Configuration for host-side macvlan interface when using `adaptive` or `macvlan` mode. Allows the host to reach macvlan containers. |

**Network modes:**

| Mode | Isolation | LAN access | Root required | Notes |
|------|-----------|------------|---------------|-------|
| `host` | None | Yes | No | Container shares host network namespace. Default. |
| `private` | Full | No | Yes | Creates a veth pair (`ve-<name>` on host, `host0` in container). |
| `bridge` | Partial | Yes | Yes | Attaches veth to a pre-existing host bridge. |
| `zone` | Partial | Zone only | Yes | Auto-managed bridge `vz-<zone>`. Containers in the same zone can communicate. |
| `macvlan` | Partial | Yes | Yes | Container appears as a separate device on the LAN with its own MAC. Host cannot ping the container (kernel limitation) unless `host_access` is enabled. |
| `ipvlan` | Partial | Yes | Yes | Container shares the host MAC address but gets its own IP. |
| `adaptive` | Varies | Varies | Varies | Automatically selects the best network mode. If `bridge` is specified and exists, uses bridge mode. Otherwise, falls back to macvlan with `host_access` enabled. Requires `lxspawn setup network` to have been run if `interface` is `auto` or omitted. |

#### `host_access`

When using `adaptive` mode (which falls back to macvlan) or explicit `macvlan` mode, the host cannot reach the container due to a kernel limitation. The `host_access` configuration creates a host-side macvlan interface to solve this.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` (explicit macvlan) / `true` (adaptive fallback) | Whether to create a host-side macvlan interface. Automatically enabled when `adaptive` mode falls back to macvlan. |
| `interface` | string | `mv-lxspawn` | Name of the host-side macvlan interface to create. |
| `ip` | string | auto-generated | IP address to assign to the host-side interface. When using `adaptive` mode, an IP is automatically generated in the same subnet as the host interface (using a deterministic hash of the machine name). If a static IP is provided (e.g. `192.168.0.201/24`), it is used instead. If set to `dhcp`, the interface is created without an IP (requires external DHCP). |

**Example with adaptive mode:**

```yaml
network:
  mode: adaptive
  bridge: br0        # Used if bridge exists, otherwise falls back to macvlan
  interface: auto    # Uses interface from `lxspawn setup network`
  # host_access is automatically enabled when falling back to macvlan
```

**Example with explicit macvlan and host access:**

```yaml
network:
  mode: macvlan
  interface: enp2s0
  host_access:
    enabled: true
    interface: mv-mgmt
    ip: 192.168.0.201/24  # Static IP for host-side interface
```

**Note:** The host-side macvlan interface is created in the `post_start` hook and removed in the `post_stop` hook. If you have custom hooks, the host access setup is prepended to your `post_start` hook and the cleanup is prepended to your `post_stop` hook.

---

### `volumes`

Each entry is one of a bind mount or a tmpfs mount.

#### Bind mount

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `host` | string | yes | — | Absolute path on the host. |
| `mount` | string | yes | — | Absolute path inside the container. |
| `readonly` | bool | no | `false` | If `true`, the mount is read-only (`BindReadOnly=`). |

#### Tmpfs mount

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | string | yes | — | Must be `tmpfs`. |
| `mount` | string | yes | — | Absolute path inside the container. |
| `options` | string | no | `""` | Tmpfs mount options (e.g. `mode=1777,size=2G`). Appended after the mount point, separated by `:`. |

---

### `security`

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `no_new_privileges` | bool | `true` | Sets `NoNewPrivileges=yes` in the `.nspawn` file. Prevents processes from gaining additional privileges via setuid/setgid. |
| `private_users` | string | auto | Value for `PrivateUsers=` in the `.nspawn` file. Options: `yes`, `no`, `identity`, `managed`, or a numeric UID base. When not set, lxspawn automatically selects the appropriate value based on network mode and systemd version (see below). |
| `seccomp` | bool | `true` | When `false`, writes an empty `SystemCallFilter=` to clear any seccomp filtering. When `true`, no seccomp filter is applied by default (systemd-nspawn does not enable seccomp on its own). |
| `capabilities` | list of strings | `[]` | Capabilities to add (`Capability=` in `.nspawn`). Example: `CAP_NET_ADMIN`, `CAP_NET_BIND_SERVICE`. |
| `drop_capabilities` | list of strings | `[]` | Capabilities to drop (`DropCapability=` in `.nspawn`). |
| `inaccessible` | list of strings | `[]` | Paths to mask inside the container (`Inaccessible=` in `.nspawn`). Each entry is over-mounted with an empty node with restrictive permissions. |
| `system_call_filter` | list of strings | `[]` | Seccomp system call filter (`SystemCallFilter=` in `.nspawn`). Accepts system call names and predefined sets (e.g. `@basic-io`, `@network-io`). Prefix with `~` to deny. |

**Automatic `PrivateUsers` selection:**

When `private_users` is not explicitly set, lxspawn automatically selects the appropriate value:

1. **Docker-in-container** (`seccomp: false`): Uses `PrivateUsers=no` to disable user namespacing, allowing nested containers and Docker operations.

2. **Macvlan/IPvlan networking**: Uses `PrivateUsers=no` to avoid compatibility issues between user namespaces and macvlan/ipvlan network interfaces.

3. **Other network modes** (host, bridge, private, zone): Uses `PrivateUsers=identity` on systemd 249+ for correct UID handling with btrfs idmapped mounts. Falls back to no `PrivateUsers` setting on older systemd versions.

**Why this matters:**

- `PrivateUsers=no`: No user namespace. Container UIDs are host UIDs. Required for Docker-in-container and macvlan/ipvlan networking.
- `PrivateUsers=identity`: User namespace with 1:1 UID mapping. Fixes EOVERFLOW errors on btrfs when installing packages. Requires systemd 249+.
- Mapped UIDs (default `-U`): Container UIDs are mapped to a high range on the host. Can cause EOVERFLOW errors on btrfs with large inode numbers.

**Docker-in-container requirements:**

When running Docker inside a container, the following configuration is required:

```yaml
security:
  seccomp: false
  capabilities:
    - CAP_NET_ADMIN
    - CAP_NET_RAW
    - CAP_SYS_ADMIN
    - CAP_BPF

resources:
  tasks: infinity

network:
  mode: adaptive
  interface: auto

hooks:
  pre_start: |
    #!/bin/bash
    set -euo pipefail
    echo 1 > /proc/sys/net/ipv4/ip_forward
    modprobe br_netfilter || true
    echo 1 > /proc/sys/net/bridge/bridge-nf-call-iptables 2>/dev/null || true
```

**Why these settings are needed:**

- `seccomp: false` - Disables seccomp filtering to allow Docker's syscalls. Also triggers:
  - `Delegate=yes` in the drop-in for cgroup delegation
  - `PrivateUsers=no` in the .nspawn file to disable user namespacing (required for nested containers)
- `CAP_BPF` - Required for Docker's cgroup device control via BPF
- `CAP_SYS_ADMIN` - Required for namespace operations
- `tasks: infinity` - Docker needs unlimited processes
- `pre_start` hook - Enables IP forwarding and bridge netfilter for Docker networking

**Additional Docker configuration:**

The provisioning script should configure Docker to use the `cgroupfs` driver instead of `systemd`:

```yaml
provision:
  - mode: system
    script: |
      #!/bin/bash
      set -eux -o pipefail
      
      # Install Docker
      apt-get update
      apt-get install -y ca-certificates curl gnupg
      install -m 0755 -d /etc/apt/keyrings
      curl -fsSL https://download.docker.com/linux/debian/gpg -o /tmp/docker.gpg.asc
      gpg --batch --no-tty --dearmor -o /etc/apt/keyrings/docker.gpg /tmp/docker.gpg.asc
      chmod a+r /etc/apt/keyrings/docker.gpg
      
      echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian bookworm stable" > /etc/apt/sources.list.d/docker.list
      
      apt-get update
      apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
      
      # Configure Docker to use cgroupfs instead of systemd
      mkdir -p /etc/docker
      cat > /etc/docker/daemon.json <<'EOF'
      {
        "exec-opts": ["native.cgroupdriver=cgroupfs"],
        "storage-driver": "overlay2"
      }
      EOF
      
      systemctl enable docker || true
```

**Note:** The `--batch --no-tty` flags are required for GPG when running in nsenter context (no TTY available).

---

### `gpu`

GPU pass-through configuration for hardware acceleration in containers. This enables shared GPU access (not full PCI passthrough) for use cases like media transcoding, compute workloads, and 3D rendering.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable GPU pass-through to the container. |
| `type` | string | `auto` | GPU vendor to use. Options: `auto`, `intel`, `amd`, `nvidia`, `mali`, `adreno`. When `auto`, prefers discrete GPUs over integrated GPUs. |
| `devices` | list of strings | `[]` | Explicit list of device nodes to bind mount (e.g., `/dev/dri/card1`, `/dev/dri/renderD128`). When specified, overrides auto-detection. |

**Supported GPU vendors:**

| Vendor | Device Nodes | Kernel Drivers | Use Cases |
|--------|--------------|----------------|-----------|
| Intel | `/dev/dri/card*`, `/dev/dri/renderD*` | `i915`, `xe` | Video decode (VA-API), 3D rendering, OpenCL compute |
| AMD | `/dev/dri/card*`, `/dev/dri/renderD*`, `/dev/kfd` | `amdgpu` | Video decode (VA-API), 3D rendering, ROCm compute |
| NVIDIA | `/dev/nvidia*`, `/dev/nvidiactl`, `/dev/nvidia-uvm` | `nvidia` | CUDA compute, NVENC/NVDEC, 3D rendering |
| ARM Mali | `/dev/mali*` | `panfrost` | 3D rendering (limited) |
| Qualcomm Adreno | `/dev/kgsl-3d0`, `/dev/dri/renderD*` | `msm` | 3D rendering (limited) |

**Auto-detection priority:**
1. Discrete GPUs (PCIe, dedicated VRAM)
2. Integrated GPUs (iGPU)
3. First detected GPU if no discrete/iGPU distinction

**Container requirements:**
- GPU device nodes are bind-mounted into the container
- Cgroup `DeviceAllow` rules are added automatically
- Required capabilities (e.g., `CAP_SYS_ADMIN` for NVIDIA) are added automatically
- Userspace drivers must be installed in the container (see provisioning examples)

**NVIDIA-specific considerations:**
- Driver version in container MUST match host exactly
- No driver virtualization (unlike AMD/Intel)
- Proprietary license restrictions apply

**Example with auto-detection:**

```yaml
gpu:
  enabled: true
  type: auto  # Auto-detect and prefer discrete GPU
```

**Example with explicit vendor:**

```yaml
gpu:
  enabled: true
  type: intel  # Use Intel GPU specifically
```

**Example with explicit devices:**

```yaml
gpu:
  enabled: true
  devices:
    - /dev/dri/card1
    - /dev/dri/renderD128
```

**Example with provisioning for GPU drivers:**

```yaml
gpu:
  enabled: true
  type: auto

provision:
  - mode: system
    script: |
      #!/bin/bash
      set -eux -o pipefail
      
      # Install GPU drivers based on detected hardware
      # Intel (intel-media-va-driver is in Debian repos):
      apt-get install -y intel-media-va-driver || true
      # intel-compute-runtime not in Debian bookworm, optional:
      apt-get install -y intel-compute-runtime || echo "intel-compute-runtime not available"
      
      # AMD:
      apt-get install -y mesa-va-drivers || true
      apt-get install -y rocm-opencl-runtime || true
      
      # NVIDIA (must match host version):
      # apt-get install -y nvidia-driver
```

**Note:** GPU driver packages may not be available in all distributions. Use `|| true` to make them optional and fall back to software decoding.

---

### `advanced`

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `ephemeral` | bool | `false` | When `true`, the container runs from a temporary snapshot. All changes are lost when the container stops (`Ephemeral=yes` in `.nspawn`). |
| `read_only` | bool | `false` | When `true`, the container rootfs is mounted read-only (`ReadOnly=yes` in `.nspawn`). |
| `hostname` | string | value of `name` | Hostname set inside the container (`Hostname=` in `.nspawn`). |
| `environment` | map | `{}` | Environment variables set for the container's init process (`Environment=` in `.nspawn`). |

---

### `journal`

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `link` | string | `try-guest` | Controls how the host and container journals are linked (`LinkJournal=` in `.nspawn`). Options: `auto`, `no`, `host`, `try-host`, `guest`, `try-guest`. |

---

### `hooks`

Host-side lifecycle scripts. Each script is written to `/var/lib/lxspawn/<name>/.hooks/` and referenced via drop-in overrides (`ExecStartPre`, `ExecStartPost`, `ExecStopPost`).

| Field | Type | Description |
|-------|------|-------------|
| `pre_start` | string | Script executed before the container starts. Use for loading kernel modules, setting sysctls, configuring bridges. |
| `post_start` | string | Script executed after the container starts. Use for attaching extra veth interfaces, configuring host-side networking. |
| `post_stop` | string | Script executed after the container stops. Use for cleanup. |

Scripts without a shebang (`#!`) automatically get `#!/bin/bash` and `set -euo pipefail` prepended.

---

### `provision`

Each entry defines a provisioning step. Steps are executed in order.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mode` | string | yes | One of: `boot`, `data`, `yq`, `system`, `user`. |
| `script` | string | depends on mode | Inline script content. |
| `file` | object | depends on mode | Remote script/file reference (alternative to `script` or `content`). |
| `path` | string | `data`/`yq` only | Absolute path inside the container. |
| `content` | string | `data` only | File content to write (supports template variables). |
| `owner` | string | no | File owner in `user:group` format (default `root:root`). |
| `permissions` | string | no | File permissions in octal (default `0644`). |
| `overwrite` | bool | no | If `false`, skip writing when the file already exists (default `true`). |
| `expression` | string | `yq` only | yq expression to apply to the file. See [yq expression limitations](#yq-expression-limitations) below. |
| `format` | string | no | File format for yq: `auto`, `json`, `yaml`, `toml`, etc. |

#### `yq` expression limitations

The built-in yq provisioner supports a **subset** of yq expression syntax. It is not a full yq implementation.

**Supported operations:**

| Syntax | Description | Example |
|--------|-------------|---------|
| `.path = value` | Set a value at a path | `.server.port = 8080` |
| `.path \|= . + value` | Append to an array | `.storage-options \|= . + ["overlay2"]` |
| `.path += value` | Append to an array | `.items += ["new-item"]` |

**Value types** are auto-detected: strings (quoted), numbers, booleans (`true`/`false`), `null`, JSON arrays/objects.

**Not supported:** Complex yq expressions such as conditionals (`select()`), multi-document processing, array slicing, custom functions, pipes, or tag manipulation. For advanced use cases, use a `system` or `boot` provision step to invoke the `yq` binary directly inside the container.

#### `file` reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | yes | URL or relative path to the script/file. Relative paths are resolved against the directory containing the YAML config. **HTTPS only** - HTTP URLs are rejected for security. |
| `digest` | string | no | Expected digest in `algorithm:hex` format (e.g. `sha256:abc123...`). Validated after download. Strongly recommended for remote files to prevent tampering. |

**Security restrictions:**
- Only HTTPS URLs are allowed (HTTP rejected with error)
- Download size limited to 2GB maximum
- Digest verification recommended for all remote files
- Files cached locally in `~/.cache/lxspawn/scripts/` with permissions `0700`

**Provisioning modes:**

| Mode | Phase | Where it runs | Purpose |
|------|-------|---------------|---------|
| `boot` | Pre-boot | Host writes to rootfs | Early setup before systemd starts. Script is written to `<rootfs>/etc/lxspawn/boot/` and executed by a oneshot systemd service on first boot. |
| `data` | Pre-boot | Host writes to rootfs | Write config files directly into the rootfs. No shell needed. |
| `yq` | Pre-boot | Host edits rootfs | Edit existing config files (JSON/YAML/TOML) using yq expressions. |
| `system` | Post-boot | `nsenter` as root | Package installation, service management. Uses `nsenter` to enter the container's namespaces directly, avoiding PTY allocation issues. |
| `user` | Post-boot | `nsenter` as user | User-level setup, dotfiles. Uses `nsenter` with `su` to run as the specified user. |

**DNS Configuration with systemd-resolved:**

Some distributions (e.g., Fedora) use systemd-resolved, which manages `/etc/resolv.conf`. Masking the service in `boot` mode doesn't prevent it from overwriting DNS configuration during boot. To configure DNS in such containers, overwrite `/etc/resolv.conf` in a `system` provision script (post-boot), after systemd-resolved has started:

```yaml
provision:
  - mode: boot
    script: |
      #!/bin/bash
      systemctl mask systemd-resolved.service
  - mode: system
    script: |
      #!/bin/bash
      # Overwrite resolv.conf AFTER systemd-resolved has started
      cat > /etc/resolv.conf << 'EOF'
      nameserver 8.8.8.8
      nameserver 1.1.1.1
      nameserver 8.8.4.4
      EOF
```

**Template variables** (available in all `script`, `content`, `hint`, and `expression` fields):

| Variable | Value |
|----------|-------|
| `{{.Name}}` | Machine name |
| `{{.Hostname}}` | Container hostname |
| `{{.Distro}}` | Distro:release string |
| `{{.User}}` | Host username |
| `{{.UID}}` | Host user ID |
| `{{.Home}}` | Host home directory |
| `{{.Dir}}` | Directory containing the YAML file |
| `{{.Param.Key}}` | Value of a custom parameter |

---

### `param`

A flat map of string key-value pairs. Keys are referenced in templates as `{{.Param.Key}}`. Also exported as `LXSPAWN_PARAM_Key` environment variables inside provision scripts.

```yaml
param:
  port: "8080"
  env: "production"
```

---

### `probes`

Readiness probes run after post-boot provisioning to verify services are available.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mode` | string | yes | Must be `readiness` (only supported mode). |
| `description` | string | yes | Human-readable description of what the probe waits for. |
| `script` | string | yes | Script executed inside the container via `nsenter`. Must exit 0 when the service is ready. |
| `hint` | string | no | Message displayed if the probe fails. Supports template variables. |

---

## Defaults Summary

| Setting | Default |
|---------|---------|
| `network.mode` | `host` |
| `advanced.hostname` | same as `name` |
| `security.no_new_privileges` | `true` |
| `security.seccomp` | `true` |
| `resources.io_weight` | `100` |
| `journal.link` | `try-guest` |
