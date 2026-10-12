# Jellyfin Media Server Template

## Overview
Jellyfin media server with GPU pass-through support for hardware transcoding. Uses bridge networking for LAN access.

## Requirements
- Bridge interface `br0` must exist on the host
- Media directory at `/mnt/media` (adjust in YAML if different)
- User must be in `lxspawn` group

## GPU Support
The template includes hooks to load GPU drivers:
- **Intel:** `modprobe i915` (enabled by default)
- **AMD:** `modprobe amdgpu` (commented out)
- **Nvidia:** Requires additional configuration

For GPU device binding, you may need to add a post_start hook:
```yaml
hooks:
  post_start: |
    #!/bin/bash
    # Bind GPU devices into container
    # This is experimental and may require manual configuration
```

## Usage
```bash
# Edit volumes if needed
vim examples/templates/jellyfin.yaml

# Launch Jellyfin
lxspawn launch examples/templates/jellyfin.yaml

# Access web interface
# http://<container-ip>:8096
```

## Testing Checklist
- [ ] GPU detected inside container (check `/dev/dri`)
- [ ] Hardware transcoding works (if GPU available)
- [ ] Media library accessible at `/media`
- [ ] Web interface accessible from LAN
- [ ] DLNA discovery works
- [ ] Logs visible via `journalctl -M jellyfin`

## Configuration
- **Network:** bridge (for LAN access)
- **Memory:** 4G (adjust based on library size)
- **Volumes:** 
  - `/mnt/media` → `/media` (read-only)
  - `/var/lib/jellyfin` → `/var/lib/jellyfin` (metadata)

## GPU Troubleshooting
Check if GPU is visible:
```bash
lxspawn exec jellyfin -- ls -la /dev/dri
```

If GPU not detected, you may need to:
1. Add device bind mounts via hooks
2. Install GPU userspace libraries in container
3. Adjust security capabilities

## Cleanup
```bash
lxspawn stop jellyfin
lxspawn delete jellyfin -y
```

## Notes
- Hardware transcoding requires compatible GPU and drivers
- Container needs GPU userspace libraries (included in most distros)
- DLNA/UPnP discovery may not work across network namespaces
