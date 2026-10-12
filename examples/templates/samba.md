# Samba File Server Template

## Overview
Samba file server for network file sharing. Uses bridge networking for LAN access and provides a shared `data` volume.

## Requirements
- Bridge interface `br0` must exist on the host
- Data directory at `/mnt/data` (adjust in YAML if different)
- User must be in `lxspawn` group

## Usage
```bash
# Edit volumes if needed
vim examples/templates/samba.yaml

# Launch Samba
lxspawn launch examples/templates/samba.yaml

# Access from LAN
# Windows: \\container-ip\data
# Linux: smb://container-ip/data
```

## Testing Checklist
- [ ] Files accessible from LAN clients
- [ ] Permissions preserved correctly
- [ ] Write performance acceptable
- [ ] User authentication works (if configured)
- [ ] Large file handling (>4GB)
- [ ] Logs visible via `journalctl -M fileserver`

## Configuration
- **Network:** bridge (for LAN access)
- **Memory:** 2G
- **Volumes:** 
  - `/mnt/data` → `/data` (read-write)
- **Share:** `data` share with guest access

## Default Credentials
- **Guest access:** enabled (no password required)
- **User:** nobody
- **Password:** samba (change in provision script)

## Customization
To require authentication:
```yaml
provision:
  - mode: system
    script: |
      #!/bin/bash
      # Create specific user
      useradd -M fileuser
      echo -e "password\npassword" | smbpasswd -s -a fileuser
```

Update `smb.conf` to disable guest access:
```ini
[global]
security = user
map to guest = Never

[data]
guest ok = no
valid users = fileuser
```

## Permission Issues
If you encounter permission problems:
1. Check host directory permissions: `ls -la /mnt/data`
2. Ensure container user has access to bind mount
3. Consider using `PrivateUsers=no` for UID mapping

## Cleanup
```bash
lxspawn stop fileserver
lxspawn delete fileserver -y
```

## Notes
- Guest access is enabled by default for simplicity
- For production, configure proper user authentication
- SELinux may require additional context on bind mounts
- File locking works across network (CIFS/SMB protocol)
