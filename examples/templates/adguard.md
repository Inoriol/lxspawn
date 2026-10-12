# AdGuard Home DNS Server Template

## Overview
This template creates an AdGuard Home DNS server with a dedicated IP on your LAN using macvlan networking. AdGuard Home provides network-wide ad blocking, privacy protection, and parental control.

## Requirements
- Physical network interface (default: `enp2s0`)
- Available IP address on your LAN (default: `192.168.0.101`)
- User must be in `lxspawn` group

## Configuration
Edit the `param:` section in `adguard.yaml` to customize:
- `adguard_ip`: Static IP for AdGuard (must be available on your LAN)
- `gateway`: Your router's IP
- `dns1`, `dns2`: Upstream DNS servers

## Usage
```bash
# Edit parameters first
vim examples/templates/adguard.yaml

# Launch AdGuard
lxspawn launch examples/templates/adguard.yaml

# Access web admin (first-time setup wizard)
# http://192.168.0.101:3000
```

## Testing Checklist
- [ ] DNS queries work from LAN clients
- [ ] Web admin interface accessible at http://<adguard_ip>:3000
- [ ] Initial setup wizard completes
- [ ] Blocklists update correctly
- [ ] Query logs show DNS requests
- [ ] Logs visible on host via `journalctl -M adguard`

## Network Notes
- Uses macvlan for dedicated LAN IP
- Container interface name: `mv-enp2s0`
- **Host cannot ping AdGuard** (macvlan limitation)
- Other LAN devices can reach AdGuard at the configured IP
- Web interface runs on port 3000 (first-time setup)

## Customization
To use a different network interface:
```yaml
network:
  mode: macvlan
  interface: <your-interface>  # e.g., eth0, ens33
```

Update the network config path and interface name in provision data mode accordingly.

## Cleanup
```bash
lxspawn stop adguard
lxspawn delete adguard -y
```

## Troubleshooting
- If DNS doesn't work, check: `journalctl -M adguard -u AdGuardHome`
- If web interface doesn't load, verify IP is not in use: `ping <adguard_ip>`
- To view AdGuard logs: `lxspawn exec adguard -- journalctl -u AdGuardHome -f`
