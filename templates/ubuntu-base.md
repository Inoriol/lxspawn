# Ubuntu Base System Template

## Overview
Minimal Ubuntu base system with common utilities. Use as a starting point for custom workloads.

## Features
- Ubuntu Noble (24.04 LTS)
- Host networking (simple, works rootless)
- Common utilities pre-installed (curl, wget, vim, htop, net-tools)
- Masked unnecessary services for faster boot

## Usage
```bash
lxspawn launch examples/templates/ubuntu-base.yaml
lxspawn shell ubuntu-base
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
lxspawn stop ubuntu-base
lxspawn delete ubuntu-base -y
```
