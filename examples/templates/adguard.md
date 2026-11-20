# AdGuard Home DNS Server Template

## Overview
This template creates an AdGuard Home DNS server with a dedicated IP on your LAN using adaptive networking. AdGuard Home provides network-wide ad blocking, privacy protection, and parental control.

## Requirements
- Available IP address on your LAN (default: `192.168.0.101`)
- User must be in `lxspawn` group

## Configuration
Edit the `param:` section in `adguard.yaml` to customize:
- `adguard_ip`: Static IP for AdGuard (must be available on your LAN)
- `gateway`: Your router's IP
- `dns1`, `dns2`: Upstream DNS servers
- `admin_pass`: Web admin password (required, change from CHANGE_ME)

## Usage
```bash
# Edit parameters (set admin_pass)
vim examples/templates/adguard.yaml

# Launch AdGuard
lxspawn launch examples/templates/adguard.yaml

# Access web admin (pre-configured, no wizard needed)
# http://192.168.0.101/
```

## Network Notes
- Uses adaptive mode (bridge if available, else macvlan)
- **Host cannot ping AdGuard in macvlan mode** (kernel limitation)
- Other LAN devices can reach AdGuard at the configured IP
- Web interface runs on port 80

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
