# Nextcloud File Server Template

## Overview
Nextcloud self-hosted file sync and share server. Uses adaptive networking for LAN access with MariaDB backend.

## Requirements
- Data directory at `/var/lib/nextcloud` (adjust in YAML if different)
- User must be in `lxspawn` group

## Usage
```bash
# Edit parameters (set passwords and trusted_domain)
vim examples/templates/nextcloud.yaml

# Launch Nextcloud
lxspawn launch examples/templates/nextcloud.yaml

# Access web interface
# http://<container-ip>/
```

## Configuration
- **Network:** adaptive (bridge if available, else macvlan)
- **Memory:** 2G (adjust based on user count)
- **Volumes:** 
  - `/var/lib/nextcloud` → `/var/www/html/nextcloud/data` (user data)
- **Database:** MariaDB (included)

## Parameters
Edit the `param:` section in the YAML:
- `db_pass`: Database password (required, change from CHANGE_ME_DB)
- `admin_pass`: Admin password (required, change from CHANGE_ME_ADMIN)
- `trusted_domain`: Set to container IP or hostname for LAN access (default: localhost)

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
