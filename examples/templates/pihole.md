# Pi-hole DNS Server Template

## Overview
This template creates a Pi-hole DNS server with a dedicated IP on your LAN using macvlan networking. Pi-hole provides network-wide ad blocking and DNS services.

## Requirements
- Physical network interface (default: `enp2s0`)
- Available IP address on your LAN (default: `192.168.0.100`)
- User must be in `lxspawn` group

## Configuration
Edit the `param:` section in `pihole.yaml` to customize:
- `pihole_ip`: Static IP for Pi-hole (must be available on your LAN)
- `gateway`: Your router's IP
- `dns1`, `dns2`: Upstream DNS servers
- `web_password`: Admin web interface password

## Usage
```bash
# Edit parameters first
vim examples/templates/pihole.yaml

# Launch Pi-hole
lxspawn launch examples/templates/pihole.yaml

# Access web admin
# http://192.168.0.100/admin
```

## Testing Checklist
- [ ] DNS queries work from LAN clients
- [ ] Web admin interface accessible at http://<pihole_ip>/admin
- [ ] Blocklists update correctly
- [ ] DHCP works (if enabled in Pi-hole settings)
- [ ] Logs visible on host via `journalctl -M pihole`

## Network Notes
- Uses macvlan for dedicated LAN IP
- Container interface name: `mv-enp2s0`
- **Host cannot ping Pi-hole** (macvlan limitation)
- Other LAN devices can reach Pi-hole at the configured IP

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
lxspawn stop pihole
lxspawn delete pihole -y
```

## Troubleshooting
- If DNS doesn't work, check: `journalctl -M pihole -u pihole-FTL`
- If web interface doesn't load, verify IP is not in use: `ping <pihole_ip>`
- To reconfigure: `lxspawn exec pihole -- pihole -r`
