# Code Review 2 (2026-11-24)

Second review of the whole repo at commit `425051f`, written fresh from the code, not from `SUGGESTIONS.md`.

**Method and limits.** The code was read, not run. It is Linux-only: `GOOS=linux go build ./... && go vet ./...` pass, but `go build` fails on macOS (`stat.Type` is `uint32`, not `int64`), and the tests were not executed. Findings were gathered by four independent read-only reviews (privilege, lifecycle/CLI, image/storage/snapshot, config/provision/CI) and then merged.

**Confidence tags.**
- **[V]** = I re-read the code and confirmed it.
- **[R]** = reported by a reviewer from reading the code and not re-checked. Treat it as a strong lead.
- **[verify]** = depends on kernel or distro behaviour that needs a real machine.

---

## Summary: fix these first

1. **Machines created with `--template` or `-` (stdin) have no usable config** (C1). This breaks the Cockpit create flow end to end.
2. **Rollback is broken in two ways** (D1, D2). The sudoers rules don't match the `.rollback-*` paths, and the failure-recovery commands are built but never run, so a failed rename can destroy the original rootfs.
3. **Tar extraction can write outside the rootfs, and images are never verified** (X1, X2). Together, a bad mirror gets host file writes.
4. **INI/unit injection through `gpu.devices` and tmpfs `options`** (V1, V2). Newlines are not rejected, so a crafted YAML can add `ExecStartPre=` and run as root.
5. **Sudoers still has loose and mismatched rules** (P1-P3). The generator and the Go call sites have drifted apart, and nothing tests them against each other.
6. **No operation lock** (C6). Two concurrent creates of one name can destroy each other's rootfs.
7. **Secrets are visible** in `argv`, the journal and world-readable files (P6, P7).

---

## 1. Config persistence (the canonical-config work is half-finished)

### C1. Template and stdin machines have no canonical config. P1 [V] **[DONE]**
`config_loader.go:39,58` sets `sourcePath` to `"template:<name>"` or `"stdin"`. `manager.go:186` then calls `copyFile(sourcePath, canonical)`, which does `os.ReadFile("template:nextcloud")`, fails, and only logs a warning. `Apply` has the same problem for `apply -`.

Result: `config`, `status`, `edit`, `provision`, `add-volume`, `edit-network` and `edit-resources` all fall back to or read `info.SourcePath` (`edit.go:51`, `provision.go:34`, `add_volume.go:115`, `edit_resources.go:30-37`) and fail with "failed to read config file".

**Fix.** Never copy from `sourcePath`. Serialize the effective `*config.Config` (after `--name` and `--param` overrides) and write it to `CanonicalConfigPath` with `WriteFileAtomic(..., 0600)`. Make the write a hard error in `Create`. Add one `loadMachineConfig(info)` helper that prefers `ConfigPath` and use it in every command that currently calls `config.Parse(info.SourcePath)`.

### C2. `--param` and `--name` overrides are lost. P1 [R] **[DONE]**
Even for file sources, the canonical copy is the raw YAML. After `launch x.yaml --param db_pass=…`, a later `provision` re-renders with the template default `CHANGE_ME_DB`, and the template's own guard aborts. Fixed by the same change as C1.

### C3. Edit commands rewrite the user's original file and are not transactional. P1 [R] **[DONE]**
`edit_resources.go:95-113` writes the YAML before `Validate` or `Apply` run, so `--memory banana` leaves an invalid file that breaks the next command. `edit.go:70-86` stops the machine, and if `Apply` then fails the machine stays stopped.

**Fix.** Edit the canonical copy, validate, then write, and keep a backup that is restored on `Validate` or `Apply` failure. Only stop the machine after validation passes.

### C4. `ParseString` skips defaults that `parse()` applies. P2 [V] **[DONE]**
`parser.go:47-77` (string path) sets only network mode, hostname and `NoNewPrivileges`. `parser.go:79-127` (file path) also sets Seccomp, IOWeight and `Storage.Size` (`10G`). On the template and stdin paths `Storage.Size` is `""`, so `ParseSizeToGB("")` at `manager.go:88` fails. Nextcloud has no `storage.size`, so `create --template nextcloud` should fail; this needs a run to confirm.

**Fix.** One `applyDefaults(cfg)` called by both paths. Add a test that parses every embedded template through `ParseString` and runs `Validate`.

### C5. `Apply` loses fields and leaves stale files. P2 [V for C5a] **[DONE]**
- **C5a.** `Apply` builds a fresh `MachineInfo` without `CreatedAt` (`manager.go:1093-1104`), and `RegisterMachine` replaces the entry, so `created_at` becomes empty after every apply, edit or `edit-resources`. Load the existing entry and update only what changed.
- **C5b.** `Apply` writes new files but never removes stale ones. Removing `hooks:` or `resources:` from the config leaves `lxspawn-hooks.conf` and `machine-<n>.slice` in place, so the old hooks and limits keep applying. Delete artifacts that are no longer compiled, after writing the new ones.

### C6. No operation lock. P2 [R] **[DONE]**
`Create` checks existence against state loaded at startup, spends minutes fetching, and `RegisterMachine` overwrites without re-checking (`manager.go:61,64,204`, `state.go:149-158`). Two concurrent `create foo` calls both pass; the first to fail runs `storage.DeleteMachine("foo")` and destroys the other's rootfs. Delete and Apply have no lock either, although TODO C10 says an operation lock was added.

**Fix.** Take a per-machine `flock` at the start of every mutating command. Make `RegisterMachine` fail if the name already exists (separate "update" from "register").

### C7. `--name` on a file or stdin source keeps the old hostname. P3 [R]
The parser already defaulted the hostname to the YAML's `name`, so the `Hostname == ""` check in `config_loader.go:51-54,65-68` never fires. Track whether the hostname was set explicitly.

### C8. `--param` keys are not checked against the declared `param:`. P3 [R]
A typo such as `--param ipp=…` is silently accepted and the template default is used. Reject undeclared keys.

---

## 2. Snapshots and rollback

### D1. Rollback and snapshot listing don't match the generated sudoers rules. P1 [R] **[DONE]**
`RollbackSnapshot` (`snapshot/btrfs.go:210-223`) runs `btrfs subvolume snapshot` to `<name>.rollback-temp`, `mv` to `<name>.rollback-old`, and `btrfs subvolume delete` on those paths. Every rule pins the destination to `[a-zA-Z0-9][a-zA-Z0-9_-]*$` (no `.`), so none match and a non-root rollback fails at the first sudo call. `getBtrfsSubvolumeCreatedAt` (`:91`) runs bare `sudo btrfs subvolume show <name>.snapshots/<snap>`, which no rule allows, so `list-snapshots` stops for a password per snapshot. Nested-subvolume deletes (`:169`, `storage/btrfs.go:156`) are also not covered.

**Fix.** Add exact rules for these paths (see P1 for how to generate them from one definition), and add a test that checks every `deploy.Sudo(...)` call against the generated rules.

### D2. Rollback failure recovery never runs, and leftovers are auto-deleted. P1 [V] **[DONE]**
`btrfs.go:217` and `:223` build `deploy.Sudo("btrfs", "subvolume", "delete", tempPath)` and `deploy.Sudo("mv", oldPath, rootfsPath)` but never call `.Run()`. If the second `mv` fails, the original rootfs stays at `<name>.rollback-old`, and the next rollback treats it as "leftover" and deletes it (`:203-208`). That is data loss.

**Fix.** Run the recovery commands and check their errors. Never auto-delete `.rollback-old`; refuse and tell the user to inspect it.

### D3. Rollback flattens all ownership. P1 [R, verify] **[DONE]**
`btrfs.go:231` runs `chown -R 0:0` on the rootfs after every rollback. With `PrivateUsers=identity`, container uids are real host uids, so `/var/lib/mysql` and `/var/www` become root-owned and MariaDB/Apache fail after rolling back the Nextcloud template. Drop the chown, or only apply it when uid shifting is in use. See also O2 for the same pattern at create time.

### D4. Delete leaves snapshots, autostart links and orphaned config. P2 [R] **[DONE]** **[DONE]**
- `storage/btrfs.go:80` only scans inside the machine path, and nothing removes the sibling `<name>.snapshots`. A new machine with the same name inherits old snapshots, so `snapshot create` says "already exists" and rollback restores stale data.
- `Delete` never runs `systemctl disable`, so a dangling `systemd-nspawn@<n>.service` enable link makes a later same-name machine autostart.
- `manager.go:409-416` removes config files before deleting storage. If the storage delete fails, the machine is registered but has no config and can't start. Delete storage first.
- The running check is fail-open: a D-Bus error skips it (`:398`), and it runs right after `machinectl terminate` with no wait (`:398-401`), so a spurious "still running" is possible. Poll for the stopped state with a timeout.

### D5. Smaller snapshot issues. P3 [R]
- `snapshot/manager.go:133`: rollback wipes `snapshots.json`, which loses descriptions even though the snapshots survive.
- `rollback.go:35`: only `running` blocks rollback; `starting` and `stopping` should too.
- `storage/btrfs.go:154`: the reversed loop over a post-order list deletes parents first, which contradicts the "deepest first" comment.

---

## 3. Image fetching and extraction

### X1. Tar extraction can write outside the rootfs. P1 [R] **[DONE]**
`image/fetcher.go:301-372`: `resolvePath` checks symlinks lexically, and absolute link targets are re-rooted (`Join(targetDir, "/etc")`) although the real filesystem does not re-root them. `isSymlinkInPath` is only called for `TypeReg` and `TypeLink`. `TypeDir` (`MkdirAll`, `Lchown`, `Chmod`), `TypeSymlink` (`os.Remove`, `os.Symlink`) and `TypeLink` follow symlinked parents. An archive with `x -> /etc/systemd/nspawn` followed by `x/evil` creates or removes host files; a `..` chain also passes the lexical check.

**Fix.** `Lstat` every parent component of each entry and refuse if any is a symlink, for all entry types. Better, extract with `openat2(RESOLVE_IN_ROOT)`, or hand extraction to `importctl`/`bsdtar`.

### X2. Downloaded images are never verified. P1 [R] **[DONE]**
Only HTTPS protects the download (`fetcher.go:157-245`). There is no sha256 or signature check. Fetch the index or `SHA256SUMS`, verify the hash (and ideally the GPG signature) before extracting. With X1 unfixed, a compromised mirror gets host write access.

### X3. Extraction does not preserve ownership, modes or capabilities. P2 [R, verify]
- `fetcher.go:315,339,368`: `Lchown` and `Chmod` only run as root, and `fixRootfsOwnership` then runs `chown -R 0:0` (see O2). Check `stat /tmp /etc/shadow` and `find <rootfs> -perm -4000` in a fresh machine.
- PAX xattrs are ignored (`fetcher.go`), so file capabilities such as `ping`'s `cap_net_raw` are lost.

### X4. xz exit status is dropped. P2 [R] **[DONE]**
`fetcher.go:258-262`: the deferred close error goes to a non-returned `err`, and `tar.Reader` accepts a clean EOF at a header boundary. A truncated stream can leave a partial rootfs that passes `ValidateInit`. Use a named return and check `cmd.Wait()`.

### X5. Smaller fetcher issues. P3 [R] **[DONE]**
- HTTPS is only checked on the first URL, and Go follows https-to-http redirects. Add `CheckRedirect`. `distro:release` goes into the URL unvalidated (`:206,70-83,162`).
- The newest dated directory is chosen without checking it contains `rootfs.tar.xz` (`:166-203`). An in-progress build gives a 404 with no fallback.
- Temp files in `cache/` leak on SIGINT, and nothing is actually cached (`:225`).
- `image/cleanup.go:32-36` hardcodes 8.8.8.8, 1.1.1.1 and 8.8.4.4, which overrides LAN DNS such as Pi-hole/AdGuard. `archToLXC` lacks `386->i386` and `ppc64le->ppc64el`.
- Host-side writes into the rootfs (`provision/boot.go:68`, `cleanup.go:37`, `engine.go`) use plain `os.WriteFile` and `MkdirAll`, so they follow symlinks planted in an image. `data.go` already has a `secureJoin`; use it everywhere. **Fixed: All rootfs writes now use `secureJoin` and `deploy.Sudo()` for proper permission handling.**

---

## 4. Injection and validation

### V1. `gpu.devices` allows newlines. P1 [V] **[DONE]**
`parser.go:525-545` checks the `/dev/` prefix, `filepath.Clean` and `..`, but not control characters. `/dev/x\nExecStartPre=…` passes and is written into `Bind=` (`compiler/nspawn.go:117`) and `DeviceAllow=` (`slice.go:88`). With a YAML from Cockpit or stdin, that is root command execution.

**Fix.** Require `^/dev/[A-Za-z0-9._/-]+$` and reject `:`.

### V2. tmpfs `options` is unvalidated, and colons in volume paths. P2 [R] **[DONE]**
`parser.go:405-412`, `nspawn.go:124-128`: only `Mount` is newline-checked. `Options` can inject INI lines, and `:` in `Host` or `Mount` changes `Bind=` option parsing. Validate `Options` against `^[A-Za-z0-9=,._-]*$` and reject `:` in paths.

### V3. Sensitive-path check is bypassed by parent directories and symlinks. P2 [R] **[DONE]**
`parser.go:393-440` blocks descendants of `/proc`, `/sys` and similar, but `host: /` , `/etc`, `/root`, `/var/lib/machines` pass, and symlinks are not resolved. Reject `/` and any ancestor of a sensitive path, run `EvalSymlinks`, and add `/etc`, `/root`, `/boot` and `/var/lib/lxspawn`.

### V4. Smaller validation gaps. P3 [R]
- One-character hostnames skip the regex (`parser.go:194`, `len > 1`), so `$` or `\` passes.
- `Environment=K=V` values with spaces are not quoted, and nspawn splits unquoted values on whitespace (not run-tested).
- `ValidateMachineName` accepts one-character names (`parser.go:20`), but the polkit regex needs two or more (`setup.go:25`), so such a machine can never start. Reject them.

### V5. GPU detection errors are swallowed. P2 [V] **[DONE]** **[DONE]**
`compiler/nspawn.go:67-71` and `:114-118` use `if err == nil`. With `gpu.enabled: true` and no GPU found, the machine starts with no devices and no warning. Return the error, or at least warn.

### V6. Template params are interpolated unescaped into shell scripts. P2 [V]
`{{.Param.x}}` is substituted straight into `bash` and `mysql -e` commands (for example `nextcloud.yaml:61-63,72-76`). A param containing `'` breaks the script or injects commands. This matters once `--param` is fed by a UI. Add a `shquote` template function, use it in the embedded templates, and consider passing params as environment variables (the engine already exports `LXSPAWN_PARAM_*`).

---

## 5. Privilege model and setup

The trust model is documented honestly in `ARCHITECTURE.md` (group = root-equivalent). The items below are bugs in the rules and in setup itself, not in that decision. If the group is root-equivalent anyway, either tighten the rules fully or stop listing regex restriction as a mitigation.

### P1. Sudoers rules are still loose. P2 [V] **[DONE]**
`setup.go:215,243-258`:
- LVM block: `mkfs.ext4 ^-q /dev/.*/lxspawn-…$` matches `-q /dev/sda1 /dev/x/lxspawn-y` (`.*` spans spaces). `lvs`, `lvremove`, `lvcreate` use the same `.*`.
- `systemd-run ^-M <name> --pipe --wait -- /bin/bash -c .*$` allows any command in any container.
- The LVM backend is documented as disabled, but its rules are still emitted. Gate them.
- Unescaped `.` in `.service` and `.snapshots` (`:218,221,222,230-236`).
- `chmod ^[0-7]{3,4}` allows setuid modes (`4755`) on a machine directory.

**Fix.** Build every rule from one table with an `escapeRegex()` helper and shared name/path fragments. Use exact argument lists. Emit LVM rules only when that backend is enabled.

### P2. Sudoers and Go call sites have drifted. P2 [V, R] **[DONE]**
Beyond D1: `Stop` and `ForceStop` fall back to `sudo systemctl stop` and `sudo systemctl kill` (`manager.go:311,345`), but only `systemctl enable|disable` are allowed. The btrfs backend falls back to `sudo mkdir -p <name>.snapshots` (`btrfs.go:36-39`), which the `mkdir` rule rejects because of the `.`.

**Fix.** A unit test that extracts every `deploy.Sudo(...)` invocation shape and asserts a matching rule exists. Better, put the allowed commands in one table that both the generator and the callers use.

### P3. `deploy.Sudo` has no `-n`. P3 [R] **[DONE]**
`filesystem.go:19`. An unmatched rule hangs on a TTY, or fails with "a terminal is required" under Cockpit. Use `sudo -n`.

### P4. `setup` reports success when it fails. P2 [R, verify] **[DONE]**
`setup.go:98-108`: sudoers and polkit failures only log a warning, and the command still prints "Setup complete" and JSON `Success: true`. The sudo version gate (`:190`, 1.9.10+) excludes Ubuntu 22.04 (1.9.9) and RHEL/Alma/Rocky 9 (1.9.5), and `getSudoVersion` probably fails on `sudo-rs` output. The user then hits password prompts later with no rules installed.

**Fix.** Return a non-zero exit with a list of failed items and the exact next step. Decide explicitly what to support on older sudo (for example, install the broad rules with a loud warning, or refuse).

### P5. State is effectively single-user. P2 [R] **[DONE]**
`deploy/state.go:57,89,125`: `machines.json` and its lock are 0600 and owned by whoever wrote last, so a second group member gets EACCES and `NewStateManager` fails. Use 0660 with group `lxspawn`, or keep state root-owned behind a narrow writer.

### P6. Provisioning secrets are passed as argv. P2 [R] **[DONE]**
`provision/exec.go:121`, `probe.go:51`, `manager.go:451`: the rendered script (including `db_pass` and `admin_pass`) goes in `sudo systemd-run … bash -c <script>`. It is visible in `/proc/<pid>/cmdline` and logged by sudo. Run `bash -s` and send the script on stdin (`--pipe` forwards it). Then pin the sudoers rule to exact arguments, which also removes the `.*`.

### P7. Secrets are written world-readable. P2 [R] **[DONE]**
`manager.go:142-186` writes the YAML (with `param:` secrets) to `/var/lib/lxspawn/<name>/config.yaml` at 0644, and `provision/boot.go:68` writes rendered boot scripts at 0755 in the rootfs. Use 0600 and 0700.

### P8. Hook labelling is not persistent. P3 [R]
`filesystem.go:124`, `manager.go:162,1079`: `chcon -t bin_t` is lost on `restorecon` or a relabel, and then hooks stop executing. Use `semanage fcontext` plus `restorecon`.

---

## 6. Lifecycle, provisioning and CLI behaviour

### O1. `WaitForMachine` returns before the container has booted. P2 [R] **[DONE]**
`manager.go:690-703` returns as soon as machined reports `running`, which is when the leader registers. Post-boot provisioning and probes race against boot, which likely contributes to the "no system bus in container" errors in Known Issues. Poll `systemd-run -M <n> --wait --pipe systemctl is-system-running --wait` (or `/bin/true`) until it succeeds, with a timeout.

### O2. `fixRootfsOwnership` runs `chown -R 0:0` on every new machine. P2 [R, verify]
`manager.go:230,786`. The kernel clears setuid/setgid bits and file capabilities on chown, which would break `sudo`, `su`, `passwd` and `ping` in the container, and it flattens image ownership (`_apt`, `systemd-*`, groups). Check with `find <rootfs> -perm -4000` after create. Extract with correct ownership (see X3) and chown only what lxspawn itself created.

### O3. `Provision` can leave a service down and says "complete" when it skipped work. P2 [R] **[DONE]** **[DONE]**
`manager.go:709-778`: it stops a running machine and returns on pre-boot failure without restarting it. For a stopped machine it skips post-boot scripts yet prints "Provisioning complete". Restart in a `defer` when `wasRunning`, and report skipped work.

### O4. Provision scripts run up to three times on failure. P2 [R] **[DONE]** **[DONE]**
`provision/exec.go:49-103` retries the whole script on any non-zero exit. A script that fails after a partial install repeats from the top. Retry only when `systemd-run` fails to connect (D-Bus not ready); fail immediately on script exit codes.

### O5. Script hashes are saved only after all scripts pass. P2 [R] **[DONE]** **[DONE]**
`engine.go:155-197`. If script 2 fails, script 1's hash is never written, so a re-run repeats finished scripts. Save state after each script.

### O6. Probe and exec timeouts don't stop the real process. P2 [R] **[DONE]** **[DONE]**
`probe.go:70`, `exec.go:92`: `cmd.Process.Kill()` targets `sudo` (root), so the unprivileged kill gets EPERM (ignored), and the in-container unit keeps running. The nextcloud probe is an infinite `until curl` loop, so up to 30 of them pile up. Use `--unit=lxspawn-probe-<n>` plus `systemctl stop`, or `--property=RuntimeMaxSec=`, and drop infinite loops from probe scripts.

### O7. Typed errors exist but are not used. P2 [V] **[DONE]**
`NewMachineNotFoundError` and `NewMachineRunningError` (`errors/errors.go:46,60`) have no callers. `manager.go` (about `:280,359,380,432`) and `apply.go:38` return `fmt.Errorf("machine '%s' not found")`, so JSON consumers get `COMMAND_FAILED` and exit code 1, never `MACHINE_NOT_FOUND`. `COMMAND_FAILED` is also not in the declared enum. This undoes the structured-error work the Cockpit plugin needs.

### O8. `--progress` is not a clean NDJSON stream. P2 [R] **[DONE]**
`create.go:64-67`, `launch.go:64-81`, `logger.go:123-127`: the logger still prints human text on stdout unless `--json` or `-q` is also set. Only `provisioning`, `starting`, `done` and `error` events exist; `downloading`, `extracting` and `probe` are never emitted. `launch` also emits `starting` before anything is created. Make `--progress` imply quiet logging (or log to stderr) and emit events from the fetcher and probe code.

### O9. `readDiskUsage` reports the whole host filesystem. P2 [R]
`manager.go:1020-1031` runs `Statfs` on the rootfs path, so every machine shows identical host used/total. Use btrfs qgroups or `btrfs filesystem du`, or label the field as host disk usage.

### O10. Stop/ForceStop skip validation. P3 [R]
`manager.go:322-353`, `stop.go`: no registry check and no `ValidateMachineName`, and the name goes straight to `machinectl`, so `stop -f -- --host=x` passes an option. Validate and check the registry, as `Start` and `Restart` do. (Also see P2: the sudo fallbacks don't match the rules.)

### O11. Smaller CLI issues. P3 [R]
- `exec.go:19-36`: the first `--` wins and anything between the machine name and `--` is dropped (`exec m foo -- bar` runs `bar`).
- `logs`: JSON priority is hard-coded to 6 and `--follow` is ignored in JSON mode (`manager.go:577-585`). Use `journalctl -o json` and read `PRIORITY`, `MESSAGE` and `__REALTIME_TIMESTAMP`.
- `delete.go:22`, `rollback.go:45`: the confirmation prompt prints to stdout even in `--json` mode.
- `manager.go:955-961`: CPU percent is only normalised when a quota is set, so values above 100% appear.
- `provision/boot.go:62` names scripts `len(entries)`, so re-running `provision` adds duplicates, and the boot loop (`boot.go:15`) has no `set -e` or once-only marker.
- `exec.go:62`: `su - <user>` resets the environment, so `LXSPAWN_PARAM_*` exports are lost in user-mode scripts. Use `su -p` or inject exports inside the `-c` string.
- `compiler` host-access IP (`~:170`) derives the last octet from a hash with no collision check against the gateway, DHCP range or other machines.
- Fixed `time.Sleep` calls remain in `Stop`, `ForceStop`, `Restart` and `engine.go:~118`; poll for state instead.

### O12. GPU and network detection. P2/P3 [R]
- `gpu/detect.go:65-70,97-110`: NVIDIA dedup never matches (DRM lists `/dev/dri/*`, NVIDIA lists `/dev/nvidiaN`). With modeset the GPU appears twice, and `auto` or `nvidia` can bind only `/dev/dri/cardN`, without `nvidiactl` or `nvidia-uvm`. Dedup by PCI address and merge device lists.
- `network/detect.go:41`: `isPhysicalInterface` needs `/sys/class/net/X/device`, so a default route on `br0`, a bond or a VLAN gives "no primary interface found" and adaptive mode fails on typical homelab hosts. Resolve bridge/bond slaves.
- `gpu/detect.go:229-261`: "discrete" means any BAR above 256 MB, which misclassifies resizable-BAR iGPUs; virtio-gpu and unknown vendors are dropped. `:362` silently grants NVIDIA `CAP_SYS_ADMIN`; document it or make it opt-in.
- `kernel` `SystemdVersion`: after a first failure, later calls return `(0, nil)` (the error is per-call).

---

## 7. Templates

- **T1.** Embedded templates report `path: "embedded"`, and the Cockpit wrapper's `getTemplateContent` calls `cockpit.file("embedded").read()` (`cockpit/src/api/lxspawn.ts`, `list_templates.go:31`). Add `lxspawn show-template NAME` and make the wizard use `create --template` instead of writing YAML to `~/.local/share`. [R]
- **T2.** `nextcloud.yaml`: secrets are in `param:` (C2/P6/P7 apply), `ServerName localhost` is hard-coded in the vhost, and the probe is an infinite loop (O6). The data directory is `/var/lib/nextcloud` on the host and is created by a `pre_start` hook; check ownership against `www-data` inside the container when `PrivateUsers=identity` is used. [V for the first two, verify for ownership]
- **T3.** Add a CI test that, for every embedded template, runs `ParseString`, `Validate` and `compiler.Compile` with default params. It would catch C4 and any future template regressions. [V] **[DONE]**

---

## 8. CI, release, build, dependencies

### B1. Release has no test gate and no supply-chain checks. P2 [V]
`release.yml` publishes on any `v*` tag without running tests. `.goreleaser.yml` runs `go mod tidy` in `before:`, so dependencies can change at release time. There is no signing, SBOM or `govulncheck`, and the root `.github` has only `workflows` (no `dependabot.yml`; `cockpit/.github/` is inert). Run tests in the release job, use `go mod verify`, sign `checksums.txt` with cosign, and add `govulncheck` and dependabot.

### B2. `test.yml` runs the suite three times. P3 [V]
`-short`, `-race` and `-cover` each run all tests. Use one `go test -race -cover ./...`. Actions are pinned to tags, not SHAs.

### B3. Makefile. P3 [V/R]
No `GOOS=linux`, so `make build` and `make test` fail on macOS; `test` lacks `-race`; and `install` only installs the binary. Either document the Linux requirement or add `GOOS=linux` to `build` and `vet` and run tests in a container (`docker run golang:…`).

### B4. Dependencies are dated. P3 [R]
`cobra 1.8.0`, `golang.org/x/sys 0.17.0` (the package that defines the `Statfs` field types behind the macOS build failure).

---

## 9. Tests

- `cmd/lxspawn` has about 4k lines and one test file; `internal/machine` (about 1.6k lines) has one; `output` and `logger` have none.
- Nothing checks the sudoers rules against the real call sites or against bypass attempts (this is why D1, P1 and P2 went unnoticed). Add a unit test for the call-site cross-check and an integration job (a privileged container with a throwaway user) that installs the rules and asserts the denied commands fail.
- Add a Linux test run that works from macOS (Docker), so the loop is local.
- Add regression tests for C1 (create from template, then `provision`/`edit-resources`), C4, D2 (failure injection on the second `mv`) and V1/V2.

---

## 10. Docs and scope

- **`SUGGESTIONS.md`** (50 KB) describes the old glob-style rules and is now misleading. Archive it (for example into `docs/archive/`) or delete it before the public announcement.
- **README, ARCHITECTURE, SPEC, TODO, CHANGELOG** total about 1,400 lines of overlapping text. Keep README short and tested (install, `setup`, `launch`, `shell`, `delete`); move the rest behind links.
- **`ARCHITECTURE.md`** says rules "restrict arguments to valid machine names" as a mitigation (about line 925). Only keep that sentence once P1 and P2 are fixed.
- **TODO C6** says the Makefile installs templates; it doesn't (and doesn't need to now they are embedded). Update it.
- **Cockpit.** The plugin is a starter-kit fork with a long unchecked UI backlog (Phase 5B-D) and an error wrapper that drops structured error codes (`spawnLxspawn`; `createMachine` also uses an unvalidated `name` in a filesystem path). Until O7 and C1 are fixed, the plugin cannot work reliably. I'd put real CLI users first and the UI after.

---

## Suggested order of work

1. **C1, C2, C4, T3** — one change: persist the effective config, share defaults, add the template test. Unblocks everything in the Cockpit flow.
2. **D1, D2, D3** — rollback correctness and data safety.
3. **V1, V2, V3** — injection fixes (small and local).
4. **X1, X2** — safe extraction and verified downloads.
5. **P1, P2, P3, P4** — regenerate sudoers from one table; add the call-site test; make `setup` fail loudly.
6. **C3, C5, C6** — transactional edit, apply cleanup, operation lock.
7. **O7, O8, O1** — typed errors, clean progress stream, real readiness check.
8. **P5, P6, P7, O2/X3** — state ownership, secret handling, ownership flattening.
9. Everything P3 and the remaining O/D/X items; then B1-B4 and docs cleanup before announcing.
