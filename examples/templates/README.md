# lxspawn Templates

Turnkey templates for common LXC use cases. These templates serve as:
1. **Reference implementations** - Show users how to configure common workloads
2. **Bug discovery** - Stress-test lxspawn features in real-world scenarios
3. **Documentation** - Working examples that complement the docs

## Template Categories

### High Priority

#### 1. Docker-in-Container (`docker-host.yaml`)
Docker host with full container support. Uses bridge networking and disables seccomp for nested containers.

**Use case:** Running Docker containers inside an lxspawn machine
**Network:** bridge (required for iptables)
**Resources:** 2 CPUs, 2G RAM, unlimited tasks

**Quick start:**
```bash
lxspawn launch examples/templates/docker-host.yaml
lxspawn shell docker-host
docker run --rm alpine echo hello
```

See [docker-host.md](docker-host.md) for details.

---

#### 2. DNS Servers

**Pi-hole (`pihole.yaml`)**
Network-wide ad blocker with dedicated IP using macvlan.

**Use case:** Network-wide ad blocking
**Network:** macvlan (dedicated LAN IP)
**Resources:** 1 CPU, 512M RAM

**Quick start:**
```bash
# Edit IP configuration
vim examples/templates/pihole.yaml

lxspawn launch examples/templates/pihole.yaml
# Access: http://192.168.0.100/admin
```

See [pihole.md](pihole.md) for details.

**AdGuard Home (`adguard.yaml`)**
Alternative DNS server with privacy features.

**Use case:** DNS filtering and privacy
**Network:** macvlan (dedicated LAN IP)
**Resources:** 1 CPU, 512M RAM

**Quick start:**
```bash
# Edit IP configuration
vim examples/templates/adguard.yaml

lxspawn launch examples/templates/adguard.yaml
# Access: http://192.168.0.101:3000
```

See [adguard.md](adguard.md) for details.

---

#### 3. Base Systems

Minimal base systems for custom workloads.

**Debian (`debian-base.yaml`)**
```bash
lxspawn launch examples/templates/debian-base.yaml
```

**Ubuntu (`ubuntu-base.yaml`)**
```bash
lxspawn launch examples/templates/ubuntu-base.yaml
```

**Fedora (`fedora-base.yaml`)**
```bash
lxspawn launch examples/templates/fedora-base.yaml
```

All base templates include:
- Host networking (simple, works rootless)
- Common utilities (curl, wget, vim, htop, net-tools)
- 1 CPU, 512M RAM (adjust as needed)

**Note:** Arch Linux is not supported on btrfs hosts due to filesystem compatibility issues. See TESTING-ISSUES.md for details.

See individual `.md` files for details.

---

### Medium Priority

#### 4. Media Servers

**Jellyfin (`jellyfin.yaml`)**
Media server with GPU pass-through support.

**Use case:** Media streaming with hardware transcoding
**Network:** bridge
**Resources:** 2 CPUs, 4G RAM

**Quick start:**
```bash
# Edit media paths
vim examples/templates/jellyfin.yaml

lxspawn launch examples/templates/jellyfin.yaml
# Access: http://<container-ip>:8096
```

See [jellyfin.md](jellyfin.md) for details.

**Plex (`plex.yaml`)**
Alternative media server.

**Use case:** Media streaming with Plex ecosystem
**Network:** bridge
**Resources:** 2 CPUs, 4G RAM

**Quick start:**
```bash
# Edit media paths
vim examples/templates/plex.yaml

lxspawn launch examples/templates/plex.yaml
# Access: http://<container-ip>:32400/web
```

See [plex.md](plex.md) for details.

---

#### 5. File Servers

**Samba (`samba.yaml`)**
Network file sharing with SMB/CIFS.

**Use case:** Cross-platform file sharing
**Network:** bridge
**Resources:** 2 CPUs, 2G RAM

**Quick start:**
```bash
# Edit data paths
vim examples/templates/samba.yaml

lxspawn launch examples/templates/samba.yaml
# Access: smb://<container-ip>/data
```

See [samba.md](samba.md) for details.

**Nextcloud (`nextcloud.yaml`)**
Self-hosted file sync and share.

**Use case:** Dropbox/Google Drive alternative
**Network:** bridge
**Resources:** 2 CPUs, 2G RAM

**Quick start:**
```bash
# Edit data paths
vim examples/templates/nextcloud.yaml

lxspawn launch examples/templates/nextcloud.yaml
# Access: http://<container-ip>/
```

See [nextcloud.md](nextcloud.md) for details.

---

## Testing

### Test Machine
All templates should be tested on the test machine:
- **IP:** 192.168.0.200
- **User:** inoriol
- **OS:** Bazzite/Fedora 44
- **Interface:** enp2s0

### Test Workflow
1. Deploy template to test machine
2. Run `lxspawn launch <template>.yaml`
3. Verify all checklist items in the template's `.md` file
4. Document any bugs found

### Common Test Commands
```bash
# Deploy binary to test machine
make build
sshpass -p 'Qazwsx12' scp lxspawn inoriol@192.168.0.200:/tmp/lxspawn
sshpass -p 'Qazwsx12' ssh inoriol@192.168.0.200 'echo "Qazwsx12" | sudo -S cp /tmp/lxspawn /usr/local/bin/lxspawn && echo "Qazwsx12" | sudo -S chmod +x /usr/local/bin/lxspawn'

# Test template
ssh inoriol@192.168.0.200
lxspawn launch /path/to/template.yaml
lxspawn status <name>
lxspawn shell <name>
```

## Template Structure

Each template consists of:
- `<name>.yaml` - Machine configuration
- `<name>.md` - Documentation and testing checklist

### YAML Structure
```yaml
name: <template-name>
distro: <distro:release>

resources:
  cpus: <n>
  memory: <size>

network:
  mode: <host|bridge|macvlan|...>
  # mode-specific options

volumes:
  - host: <path>
    mount: <path>

security:
  # capabilities, seccomp, etc.

hooks:
  pre_start: |
    #!/bin/bash
    # Host-side pre-start script

provision:
  - mode: boot|data|yq|system|user
    # provisioning configuration

probes:
  - mode: readiness
    description: "<service> to be ready"
    script: |
      #!/bin/bash
      # readiness check
```

## Known Issues

### Macvlan Host Visibility
Host cannot ping macvlan containers (expected kernel limitation). Container can communicate with other LAN devices.

### Bridge Requirement
Templates using bridge networking require `br0` to exist on the host. Create it first:
```bash
sudo ip link add name br0 type bridge
sudo ip link set br0 up
```

### GPU Pass-Through
GPU device binding is experimental. You may need to manually bind `/dev/dri` or `/dev/nvidia*` devices via hooks.

### Permission Issues
Bind mount permissions can be tricky. Ensure host directories have appropriate ownership and consider using `PrivateUsers=no` if needed.

## Contributing

When creating new templates:
1. Follow the existing structure (`.yaml` + `.md`)
2. Include comprehensive testing checklist
3. Document all configuration options
4. Test on at least 2 distros (Fedora, Ubuntu)
5. Report any bugs discovered

## Bug Tracking

Use GitHub issues with labels:
- `template/docker` - Docker-in-container issues
- `template/dns` - DNS server issues
- `template/media` - Media server issues
- `template/files` - File server issues
- `template/base` - Base system issues

Include:
- Template YAML (redact sensitive info)
- Expected behavior
- Actual behavior
- Logs (`journalctl -M <name>`)
- Host distro and version
