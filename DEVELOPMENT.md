# Development Guide

This document is for developers and AI agents working on lxspawn. It covers building, testing, code conventions, and how to extend the system.

## Prerequisites

- **Go 1.21+** - Primary development language
- **systemd 245+** - Required for systemd-nspawn and machinectl
- **Linux** - Target platform (macOS/Windows not supported)
- **btrfs** (optional) - For snapshot testing
- **LVM** (optional) - For LVM backend testing

## Building from Source

```bash
# Clone the repository
git clone https://github.com/Inoriol/lxspawn
cd lxspawn

# Build the binary
make build

# Install system-wide (requires sudo)
sudo make install

# Or install to custom location
make install PREFIX=/usr/local
```

The binary will be built as `lxspawn` in the project root.

## Running Tests

```bash
# Run all tests (recommended: with race detector)
make test
# or
go test -race ./...

# Run tests with verbose output
go test -race -v ./...

# Run specific package tests
go test ./internal/config/...
go test ./internal/compiler/...
go test ./internal/provision/...

# Run tests with coverage
go test -race -cover ./...

# Run tests and generate HTML coverage report
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Note:** Some tests require root privileges and will be skipped if not run as root. The `-race` flag enables the race detector to catch data races.

## Code Structure

### Package Organization

See [ARCHITECTURE.md](ARCHITECTURE.md#package-structure) for the full package tree.

### Key Interfaces

**StorageBackend** (`internal/storage/backend.go`):
```go
type StorageBackend interface {
    Type() BackendType
    CreateMachine(name string, sizeGB int) error
    DeleteMachine(name string) error
    MachineExists(name string) (bool, error)
    GetMachinePath(name string) string
    GetUsage() (int64, int64, error) // used, total bytes
}
```

**SystemdClient** (`internal/deploy/interfaces.go`):
```go
type SystemdClient interface {
    Close() error
    ReloadDaemon() error
    StartUnit(unitName string) error
    StopUnit(unitName string) error
    RestartUnit(unitName string) error
    ResetFailedUnit(unitName string) error
    ListUnitsByNames(unitNames []string) ([]systemdbus.UnitStatus, error)
    ListAllUnits() ([]systemdbus.UnitStatus, error)
    GetUnitProperties(unitName string) (map[string]interface{}, error)
    ListRunningMachines() ([]RunningMachineInfo, error)
    GetMachineState(name string) (string, []string, error)
    GetMachineLeader(name string) (int, error)
}
```

**StateStore** (`internal/deploy/interfaces.go`):
```go
type StateStore interface {
    GetMachine(name string) (MachineInfo, bool)
    GetStateFilePath() string
    ListMachines() []MachineInfo
    RegisterMachine(machine MachineInfo) error
    UnregisterMachine(name string) error
    Save() error
}
```

**PathConfig** (`internal/deploy/paths.go`):
```go
type PathConfig struct {
    NspawnDir      string  // .nspawn files (default: /etc/systemd/nspawn)
    SystemdDir     string  // slice/drop-in units (default: /etc/systemd/system)
    MachinesDir    string  // machine rootfs (default: /var/lib/machines)
    LxspawnDataDir string  // lxspawn metadata (default: /var/lib/lxspawn)
}

// Configure paths (useful for testing or non-standard layouts like NixOS)
func SetPaths(cfg PathConfig)
func GetPaths() PathConfig
func ResetPaths()  // restore defaults
```

**Manager** (`internal/machine/manager.go`):
```go
type Manager struct { ... }

// Production constructor (creates real dependencies)
func NewManager() (*Manager, error)

// Test constructor (inject fakes)
func NewManagerWithDeps(state StateStore, systemd SystemdClient, storage StorageBackend) *Manager

func (m *Manager) Create(cfg *config.Config, sourcePath string) ([]string, error)
func (m *Manager) Launch(cfg *config.Config, sourcePath string) error
func (m *Manager) Start(name string) error
func (m *Manager) Stop(name string) error
func (m *Manager) Delete(name string) error
func (m *Manager) Close()
```

**SnapshotBackend** (`internal/snapshot/snapshot.go`):
```go
type SnapshotBackend interface {
    CreateSnapshot(machine, snapName, rootfsPath string) error
    ListSnapshots(machine, rootfsPath string) ([]SnapshotInfo, error)
    RollbackSnapshot(machine, snapName, rootfsPath string) error
    DeleteSnapshot(machine, snapName, rootfsPath string) error
    Type() BackendType
}
```

**Snapshot Manager** (`internal/snapshot/manager.go`):
```go
type Manager struct { ... }

func NewManager(machineName string) (*Manager, error)
func (m *Manager) CreateSnapshot(snapName string, description string) error
func (m *Manager) ListSnapshots() ([]SnapshotInfo, error)
func (m *Manager) RollbackSnapshot(snapName string) error
func (m *Manager) DeleteSnapshot(snapName string) error
func (m *Manager) BackendType() BackendType
func (m *Manager) Close()
```

## Adding a New CLI Command

1. **Create command file** in `cmd/lxspawn/`:

```go
// cmd/lxspawn/mycommand.go
package main

import (
    "fmt"
    "github.com/spf13/cobra"
    "lxspawn/internal/machine"
    "lxspawn/internal/output"
)

var mycommandCmd = &cobra.Command{
    Use:   "mycommand <name>",
    Short: "Description of mycommand",
    Args:  cobra.ExactArgs(1),
    RunE:  runMycommand,
}

func init() {
    rootCmd.AddCommand(mycommandCmd)
    mycommandCmd.Flags().Bool("json", false, "Output in JSON format")
}

func runMycommand(cmd *cmd.Command, args []string) error {
    name := args[0]
    jsonOutput, _ := cmd.Flags().GetBool("json")
    
    mgr, err := machine.NewManager()
    if err != nil {
        return err
    }
    defer mgr.Close()
    
    // Do something with the machine
    result, err := mgr.MyOperation(name)
    if err != nil {
        return err
    }
    
    if jsonOutput {
        return output.PrintJSON("1.0", result)
    }
    
    fmt.Printf("Success: %s\n", result.Message)
    return nil
}
```

2. **Add JSON schema** in `internal/output/schemas.go`:

```go
type MyCommandData struct {
    Name    string `json:"name"`
    Message string `json:"message"`
}
```

3. **Add tests** in `cmd/lxspawn/mycommand_test.go` (if needed)

4. **Update README.md** with the new command

5. **Update sudoers rules** in `cmd/lxspawn/setup.go` if the command requires privileged operations

## Adding a New Provision Mode

1. **Add mode to schema** in `internal/config/schema.go`:

```go
type Provision struct {
    Mode       string `yaml:"mode"`
    // ... existing fields
    NewField   string `yaml:"new_field"` // Add new field if needed
}
```

2. **Add validation** in `internal/config/parser.go`:

```go
func validateProvision(provs []Provision) error {
    for i, p := range provs {
        switch p.Mode {
        case "boot", "data", "yq", "system", "user", "newmode":
            // valid
        default:
            return fmt.Errorf("provision[%d]: invalid mode %q", i, p.Mode)
        }
        // Validate new field
        if p.Mode == "newmode" && p.NewField == "" {
            return fmt.Errorf("provision[%d]: new_field is required for newmode", i)
        }
    }
    return nil
}
```

3. **Implement provisioner** in `internal/provision/newmode.go`:

```go
package provision

import (
    "context"
    "fmt"
)

type NewModeProvisioner struct {
    // dependencies
}

func NewNewModeProvisioner() *NewModeProvisioner {
    return &NewModeProvisioner{}
}

func (p *NewModeProvisioner) Provision(ctx context.Context, prov *config.Provision, rootfs string, tmplCtx *TemplateContext) error {
    // Implement provisioning logic
    return nil
}

func (p *NewModeProvisioner) Phase() Phase {
    return PhasePreBoot // or PhasePostBoot
}
```

4. **Register provisioner** in `internal/provision/engine.go`:

```go
func (e *Engine) getProvisioner(mode string) Provisioner {
    switch mode {
    case "boot":
        return e.bootProvisioner
    case "data":
        return e.dataProvisioner
    case "yq":
        return e.yqProvisioner
    case "system", "user":
        return e.execProvisioner
    case "newmode":
        return e.newModeProvisioner
    default:
        return nil
    }
}
```

5. **Add tests** in `internal/provision/newmode_test.go`

6. **Update SPEC.md** with the new mode

## Adding a New Storage Backend

1. **Implement interface** in `internal/storage/newbackend.go`:

```go
package storage

type NewBackend struct {
    basePath string
    // backend-specific fields
}

func NewNewBackend(basePath string) (*NewBackend, error) {
    return &NewBackend{basePath: basePath}, nil
}

func (b *NewBackend) CreateMachine(name string, sizeGB int) error {
    // Implement machine creation
    return nil
}

func (b *NewBackend) DeleteMachine(name string) error {
    // Implement machine deletion
    return nil
}

// ... implement all StorageBackend methods

func (b *NewBackend) Close() error {
    // Cleanup resources
    return nil
}
```

2. **Add detection logic** in `internal/storage/detector.go`:

```go
func DetectBackend(basePath string) (StorageBackend, error) {
    // Check for new backend
    if isNewBackendAvailable(basePath) {
        return NewNewBackend(basePath)
    }
    
    // Fall back to existing backends
    if isBtrfs(basePath) {
        return NewBtrfsBackend(basePath)
    }
    // ...
}
```

3. **Add sudoers rules** in `cmd/lxspawn/setup.go` if the backend requires privileged operations

4. **Add tests** in `internal/storage/newbackend_test.go`

5. **Update README.md** and **ARCHITECTURE.md** with the new backend

## Adding a New Template

1. **Create YAML template** in `templates/`:

```yaml
# templates/mytemplate.yaml
name: mytemplate
distro: debian:bookworm

resources:
  cpus: 2
  memory: 2G

network:
  mode: adaptive
  interface: auto

provision:
  - mode: system
    script: |
      #!/bin/bash
      set -eux -o pipefail
      # Installation steps
      
  - mode: data
    path: /etc/myapp/config.yaml
    content: |
      # Configuration

param:
  param1: "value1"
  param2: "value2"

probes:
  - mode: readiness
    description: "myapp to be ready"
    script: |
      #!/bin/bash
      until curl -sf http://localhost/health; do sleep 2; done
```

2. **Create documentation** in `templates/mytemplate.md`:

```markdown
# MyTemplate

Description of the template.

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| param1    | value1  | Description |
| param2    | value2  | Description |

## Usage

```bash
lxspawn launch --template mytemplate --name mymachine
```

## Post-Installation

Any post-installation steps.
```

3. **Test the template** on a real machine:

```bash
# Copy template to test machine
scp examples/templates/mytemplate.yaml user@testhost:

# Launch on test machine
ssh user@testhost
lxspawn launch mytemplate.yaml
lxspawn status mytemplate
lxspawn shell mytemplate
# Verify everything works
lxspawn delete mytemplate
```

## Code Conventions

### Naming

- **Packages:** lowercase, single word (e.g., `config`, `compiler`, `provision`)
- **Files:** lowercase with underscores (e.g., `parser.go`, `fetcher.go`)
- **Functions:** CamelCase for exported, camelCase for unexported
- **Constants:** CamelCase for exported, camelCase for unexported (no ALL_CAPS)
- **Variables:** camelCase, descriptive names

### Error Handling

Always wrap errors with context:

```go
// Good
if err != nil {
    return fmt.Errorf("failed to create machine %q: %w", name, err)
}

// Bad
if err != nil {
    return err
}
```

Use `errors.Is()` and `errors.As()` for error checking:

```go
if errors.Is(err, os.ErrNotExist) {
    // Handle not found
}

var machineErr *MachineNotFoundError
if errors.As(err, &machineErr) {
    // Handle machine not found
}
```

### Logging

Use the logger package for consistent output:

```go
import "lxspawn/internal/logger"

logger.Info("Machine created: %s", name)
logger.Warn("Machine already exists: %s", name)
logger.Error("Failed to create machine: %v", err)
logger.Debug("Generated .nspawn file: %s", path) // Only shown with --verbose
```

### JSON Output

All commands should support `--json` flag:

```go
if jsonOutput {
    return output.PrintJSON("1.0", data)
}
```

JSON format:
```json
{
  "version": "1.0",
  "data": {
    // Command-specific data
  }
}
```

### Validation

Validate all inputs at the boundary (CLI or API):

```go
func validateMachineName(name string) error {
    if !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]$`).MatchString(name) {
        return fmt.Errorf("invalid machine name: %q", name)
    }
    return nil
}
```

### Security

- **Never log secrets** (passwords, keys, tokens)
- **Validate all paths** to prevent path traversal
- **Use HTTPS-only** for downloads
- **Validate all inputs** before using in shell commands
- **Use sudoers regex** for argument validation
- **Set file permissions** appropriately (0600 for sensitive files)

## Debugging

### Verbose Mode

```bash
lxspawn --verbose launch myserver.yaml
```

Shows:
- YAML parsing details
- Generated file contents
- systemd D-Bus calls
- Image download progress

### Checking Generated Files

```bash
# View .nspawn file
cat /etc/systemd/nspawn/myserver.nspawn

# View slice unit
cat /etc/systemd/system/machine-myserver.slice

# View drop-in
cat /etc/systemd/system/systemd-nspawn@myserver.service.d/lxspawn.conf

# View hooks
ls -la /var/lib/lxspawn/myserver/.hooks/
cat /var/lib/lxspawn/myserver/.hooks/*.sh
```

### Checking State

```bash
# View machine registry
cat /var/lib/lxspawn/machines.json | jq

# View snapshot state
cat /var/lib/lxspawn/myserver/snapshots.json | jq

# View provision state
cat /var/lib/lxspawn/myserver/provision-state.json | jq
```

### Checking Logs

```bash
# Container logs
journalctl -M myserver -f

# systemd service logs
journalctl -u systemd-nspawn@myserver.service -f

# lxspawn logs (if logged to file)
journalctl -u lxspawn.service -f
```

### Debugging systemd-nspawn

```bash
# Verify .nspawn file
systemd-analyze verify /etc/systemd/nspawn/myserver.nspawn

# Check machine status
machinectl status myserver

# Show machine properties
machinectl show myserver
```

### Debugging D-Bus

```bash
# List machines via D-Bus
busctl --system call org.freedesktop.machine1 /org/freedesktop/machine1 org.freedesktop.machine1.Manager ListMachines

# Get machine properties
busctl --system get-property org.freedesktop.machine1 /org/freedesktop/machine1/machine1/myserver org.freedesktop.machine1.Manager State
```

## Common Development Tasks

### Adding a New Field to YAML Schema

1. Add field to struct in `internal/config/schema.go`
2. Add validation in `internal/config/parser.go`
3. Add tests in `internal/config/parser_test.go`
4. Update compiler to use the field in `internal/compiler/`
5. Update SPEC.md with the new field

### Modifying Generated Files

1. Update compiler in `internal/compiler/`:
   - `nspawn.go` for .nspawn files
   - `slice.go` for slice units and drop-ins
2. Update tests in `internal/compiler/compiler_test.go`
3. Test on a real machine to verify systemd accepts the changes

### Changing Provisioning Behavior

1. Update provisioner in `internal/provision/`:
   - `boot.go` for boot mode
   - `data.go` for data mode
   - `yq.go` for yq mode
   - `exec.go` for system/user modes
2. Update tests in `internal/provision/*_test.go`
3. Test on a real machine to verify provisioning works

### Adding a New Networking Mode

1. Add mode to `internal/config/schema.go`
2. Add validation in `internal/config/parser.go`
3. Update compiler in `internal/compiler/nspawn.go` to generate correct .nspawn
4. Update network config injection in `internal/image/network.go`
5. Add tests in `internal/compiler/compiler_test.go` and `internal/image/network_test.go`
6. Update SPEC.md and README.md

## Testing Checklist

Before submitting changes:

- [ ] All existing tests pass: `go test ./...`
- [ ] New tests added for new functionality
- [ ] Code compiles without warnings: `go build ./...`
- [ ] Manual testing on a real machine (if applicable)
- [ ] JSON output works correctly (if applicable)
- [ ] Verbose mode shows useful information (if applicable)
- [ ] Error messages are clear and actionable
- [ ] Documentation updated (README.md, SPEC.md, ARCHITECTURE.md)
- [ ] Sudoers rules updated (if new privileged operations added)
- [ ] No secrets logged or exposed

## Writing Integration Tests

Integration tests use fake implementations from `internal/testutil` to test `Manager` lifecycle methods without requiring real systemd, root privileges, or storage backends.

### Setup Pattern

```go
func setupTestManager(t *testing.T) (*Manager, *deploy.StateManager, *testutil.FakeSystemdClient, *testutil.FakeStorageBackend) {
    t.Helper()

    tmpDir := t.TempDir()

    // Configure paths for this test
    paths := deploy.PathConfig{
        NspawnDir:      filepath.Join(tmpDir, "nspawn"),
        SystemdDir:     filepath.Join(tmpDir, "systemd"),
        MachinesDir:    filepath.Join(tmpDir, "machines"),
        LxspawnDataDir: filepath.Join(tmpDir, "lxspawn"),
    }
    deploy.SetPaths(paths)
    t.Cleanup(deploy.ResetPaths)

    // Create real state manager with temp path
    state := &deploy.StateManager{
        StateFilePath: filepath.Join(tmpDir, "machines.json"),
        Machines:      make(map[string]deploy.MachineInfo),
    }

    // Create fakes
    systemd := testutil.NewFakeSystemdClient()
    storage := testutil.NewFakeStorageBackend(paths.MachinesDir)

    // Inject into manager
    mgr := machine.NewManagerWithDeps(state, systemd, storage)

    return mgr, state, systemd, storage
}
```

### Asserting on Calls

```go
func TestStart_CallsSystemd(t *testing.T) {
    mgr, state, systemd, _ := setupTestManager(t)

    state.Machines["testbox"] = deploy.MachineInfo{Name: "testbox"}

    err := mgr.Start("testbox")
    require.NoError(t, err)

    // Assert ReloadDaemon was called
    reloadCalls := systemd.GetCalls("ReloadDaemon")
    require.Len(t, reloadCalls, 1)

    // Assert StartUnit was called with correct unit name
    startCalls := systemd.GetCalls("StartUnit")
    require.Len(t, startCalls, 1)
    assert.Equal(t, "systemd-nspawn@testbox.service", startCalls[0].Args[0])
}
```

### Injecting Errors

```go
func TestStart_SystemdError(t *testing.T) {
    mgr, state, systemd, _ := setupTestManager(t)

    state.Machines["testbox"] = deploy.MachineInfo{Name: "testbox"}

    // Make StartUnit fail
    systemd.StartUnitFn = func(name string) error {
        return errors.New("systemd start failed")
    }

    err := mgr.Start("testbox")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "failed to start machine")
}
```

### Controlling Machine State

```go
func TestGetStatus_Running(t *testing.T) {
    mgr, state, systemd, _ := setupTestManager(t)

    state.Machines["testbox"] = deploy.MachineInfo{Name: "testbox"}

    // Control what GetMachineState returns
    systemd.MachineState["testbox"] = "running"

    status, err := mgr.GetStatus("testbox")
    require.NoError(t, err)
    assert.Equal(t, "running", status.State)
}
```

## Sudoers Cross-Check Test

The `cmd/lxspawn/sudoers_test.go` file contains a cross-check test that validates every `deploy.Sudo()` call site uses a binary with a corresponding sudoers rule. This prevents drift between the Go code and the generated sudoers rules.

### How It Works

The test uses `go/ast` to parse all `.go` files in the project and:
1. Find every `deploy.Sudo(...)` call and extract the binary name (first argument)
2. Find every `sudoOrDirectForRootfs(...)` wrapper call (which may call `deploy.Sudo()`)
3. Generate sudoers rules with dummy binary paths
4. Verify each binary used in the code has a corresponding rule

### Running the Test

```bash
go test ./cmd/lxspawn -run TestSudoersCrossCheck
```

### Adding New Sudo Calls

When adding new `deploy.Sudo()` calls:
1. Ensure the binary is in the list checked by `findAvailableBinaries()` in `setup.go`
2. Add a corresponding rule in `generateSudoersRules()`
3. The cross-check test will fail if the rule is missing

### Excluded Files

The test excludes disabled backends (LVM, ZFS) which have `deploy.Sudo()` calls but no sudoers rules:
- `internal/snapshot/lvm.go`
- `internal/snapshot/zfs.go`
- `internal/storage/lvm_thin.go`
- `internal/storage/detector.go`

## Code Review Guidelines

When reviewing code:

1. **Security:** Check for path traversal, command injection, privilege escalation
2. **Error handling:** Ensure errors are wrapped with context
3. **Validation:** Verify all inputs are validated at boundaries
4. **Testing:** Check that tests cover happy path and error cases
5. **Documentation:** Ensure public APIs are documented
6. **Consistency:** Follow existing code patterns and conventions
7. **Performance:** Check for unnecessary allocations or slow operations
8. **Resource management:** Ensure resources (files, connections) are properly closed

## Common Pitfalls

### Path Traversal

Always validate paths:

```go
// Good
cleanPath := filepath.Clean(userPath)
if !strings.HasPrefix(cleanPath, rootfs) {
    return fmt.Errorf("path escapes rootfs: %s", userPath)
}

// Bad
fullPath := filepath.Join(rootfs, userPath) // Vulnerable to ../../../etc/passwd
```

### Command Injection

Always quote shell arguments:

```go
// Good
cmd := exec.Command("bash", "-c", script)

// Bad
cmd := exec.Command("bash", "-c", "echo "+userInput) // Vulnerable to ; rm -rf /
```

### Resource Leaks

Always close resources:

```go
// Good
file, err := os.Open(path)
if err != nil {
    return err
}
defer file.Close()

// Bad
file, err := os.Open(path)
// Forgot to close!
```

### Error Swallowing

Always handle or return errors:

```go
// Good
if err != nil {
    return fmt.Errorf("failed to open file: %w", err)
}

// Bad
if err != nil {
    log.Printf("Error: %v", err) // Error swallowed, caller doesn't know
}
```

## Resources

- [Go Documentation](https://go.dev/doc/)
- [systemd-nspawn Manual](https://man7.org/linux/man-pages/man1/systemd-nspawn.1.html)
- [systemd.nspawn Manual](https://man7.org/linux/man-pages/man5/systemd.nspawn.5.html)
- [D-Bus Tutorial](https://dbus.freedesktop.org/doc/dbus-tutorial.html)
- [Cobra Documentation](https://github.com/spf13/cobra)

## Getting Help

- Check ARCHITECTURE.md for design decisions
- Check SPEC.md for YAML schema details
- Check existing code for patterns and conventions
- Run tests to verify your understanding
- Ask in code review if unsure
