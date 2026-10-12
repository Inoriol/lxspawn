# Nextcloud File Server Template

## Overview
Nextcloud self-hosted file sync and share server. Uses bridge networking for LAN access with MariaDB backend.

## Requirements
- Bridge interface `br0` must exist on the host
- Data directory at `/var/lib/nextcloud` (adjust in YAML if different)
- User must be in `lxspawn` group

## Usage
```bash
# Edit volumes if needed
vim examples/templates/nextcloud.yaml

# Launch Nextcloud
lxspawn launch examples/templates/nextcloud.yaml

# Access web interface
# http://<container-ip>/
```

## Testing Checklist
- [ ] Files accessible from LAN clients
- [ ] Web interface accessible
- [ ] File sync works (desktop/mobile clients)
- [ ] User authentication works
- [ ] Large file handling (>4GB)
- [ ] Logs visible via `journalctl -M nextcloud`

## Configuration
- **Network:** bridge (for LAN access)
- **Memory:** 2G (adjust based on user count)
- **Volumes:** 
  - `/var/lib/nextcloud` → `/var/www/html/data` (user data)
- **Database:** MariaDB (included)

## Default Credentials
- **Admin user:** admin
- **Admin password:** admin
- **Database:** nextcloud
- **Database user:** nextcloud
- **Database password:** nextcloud

**Change these immediately after first login!**

## Customization
To use external database:
```yaml
provision:
  - mode: system
    script: |
      #!/bin/bash
      # Skip MariaDB installation
      # Configure external database connection
```

To use PostgreSQL instead:
```yaml
provision:
  - mode: system
    script: |
      #!/bin/bash
      apt-get install -y postgresql php-pgsql
      # Configure PostgreSQL...
```

## Performance Tuning
For better performance:
1. Enable PHP OPcache
2. Configure Redis for file locking
3. Use external database server
4. Increase PHP memory limits

## Backup
Backup these directories:
- `/var/lib/nextcloud` - User data
- `/var/www/html/nextcloud/config` - Configuration
- Database dump

## Cleanup
```bash
lxspawn stop nextcloud
lxspawn delete nextcloud -y
```

## Notes
- Default credentials are for initial setup only
- For production, use HTTPS (requires reverse proxy)
- Consider using external database for better performance
- File versioning and trash bin enabled by default
- Mobile apps available for iOS and Android
