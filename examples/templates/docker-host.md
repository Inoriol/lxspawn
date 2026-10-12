# Docker-in-Container Template

## Overview
This template creates a Debian container with Docker installed and configured. It uses bridge networking for proper iptables support and disables seccomp filtering to allow Docker's container operations.

## Requirements
- Bridge interface `br0` must exist on the host
- User must be in `lxspawn` group

## Usage
```bash
lxspawn launch examples/templates/docker-host.yaml
```

## Testing Checklist
- [ ] Can pull images: `docker pull alpine`
- [ ] Can run containers: `docker run --rm alpine echo hello`
- [ ] Docker networks work (bridge, host)
- [ ] Docker volumes persist across container restarts
- [ ] Can build images: `docker build`
- [ ] Container logs visible via `journalctl -M docker-host`
- [ ] Docker Compose works

## Configuration Notes
- **Network mode:** bridge (required for Docker's iptables rules)
- **Seccomp:** disabled (required for nested containers)
- **Capabilities:** CAP_NET_ADMIN, CAP_NET_RAW, CAP_SYS_ADMIN
- **Tasks:** infinity (Docker spawns many processes)

## Known Issues
- Cgroup delegation may need manual configuration on some hosts
- Nested containers (docker-in-docker) may have performance overhead
- Host cannot access container IPs directly (bridge networking limitation)

## Cleanup
```bash
lxspawn stop docker-host
lxspawn delete docker-host -y
```
