# Debian Base System Template

## Overview
Minimal Debian base system with common utilities. Use as a starting point for custom workloads.

## Features
- Debian Bookworm (stable)
- Host networking (simple, works rootless)
- Common utilities pre-installed (curl, wget, vim, htop, net-tools)
- Masked unnecessary services for faster boot

## Usage
```bash
lxspawn launch examples/templates/debian-base.yaml
lxspawn shell debian-base
```

## Testing Checklist
- [ ] Image downloads successfully
- [ ] systemd starts correctly
- [ ] Network connectivity works
- [ ] Package manager works (apt)
- [ ] Journal integration works
- [ ] Shell access works

## Customization
Add your own provisioning scripts or modify resources:
```yaml
resources:
  cpus: 2
  memory: 1G

provision:
  - mode: system
    script: |
      #!/bin/bash
      # Add your custom setup here
```

## Cleanup
```bash
lxspawn stop debian-base
lxspawn delete debian-base -y
```
