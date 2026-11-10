package deploy

import (
	"context"
	"fmt"
	"strings"
	"time"

	systemdbus "github.com/coreos/go-systemd/v22/dbus"
	godbus "github.com/godbus/dbus/v5"
)

const (
	unitStartTimeout = 5 * time.Minute
	unitStopTimeout  = 30 * time.Second
	waitForUnitPoll  = 500 * time.Millisecond
)

type SystemdManager struct {
	conn    *systemdbus.Conn
	sysConn *godbus.Conn
}

func NewSystemdManager() (*SystemdManager, error) {
	conn, err := systemdbus.NewSystemConnectionContext(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to systemd bus: %w", err)
	}

	sysConn, err := godbus.SystemBus()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to connect to system bus: %w", err)
	}

	return &SystemdManager{conn: conn, sysConn: sysConn}, nil
}

func (s *SystemdManager) Close() error {
	s.conn.Close()
	s.sysConn.Close()
	return nil
}

func (s *SystemdManager) ReloadDaemon() error {
	if err := s.conn.Reload(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}
	return nil
}

func (s *SystemdManager) StartUnit(unitName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), unitStartTimeout)
	defer cancel()

	ch := make(chan string, 1)
	_, err := s.conn.StartUnitContext(ctx, unitName, "replace", ch)
	if err != nil {
		return fmt.Errorf("failed to enqueue start job for unit %s: %w", unitName, err)
	}

	select {
	case result := <-ch:
		if result != "done" {
			return fmt.Errorf("unit %s failed to start, job result: %s", unitName, result)
		}
	case <-ctx.Done():
		return fmt.Errorf("timed out waiting for unit %s to start", unitName)
	}

	return nil
}

func (s *SystemdManager) StopUnit(unitName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), unitStopTimeout)
	defer cancel()

	ch := make(chan string, 1)
	_, err := s.conn.StopUnitContext(ctx, unitName, "replace", ch)
	if err != nil {
		return fmt.Errorf("failed to enqueue stop job for unit %s: %w", unitName, err)
	}

	select {
	case result := <-ch:
		if result != "done" {
			return fmt.Errorf("unit %s failed to stop, job result: %s", unitName, result)
		}
	case <-ctx.Done():
		return fmt.Errorf("timed out waiting for unit %s to stop", unitName)
	}

	return nil
}

func (s *SystemdManager) ResetFailedUnit(unitName string) error {
	if err := s.conn.ResetFailedUnit(unitName); err != nil {
		return fmt.Errorf("failed to reset failed state for unit %s: %w", unitName, err)
	}
	return nil
}

func (s *SystemdManager) RestartUnit(unitName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), unitStopTimeout)
	defer cancel()

	ch := make(chan string, 1)
	_, err := s.conn.RestartUnitContext(ctx, unitName, "replace", ch)
	if err != nil {
		return fmt.Errorf("failed to enqueue restart job for unit %s: %w", unitName, err)
	}

	select {
	case result := <-ch:
		if result != "done" {
			return fmt.Errorf("unit %s failed to restart, job result: %s", unitName, result)
		}
	case <-ctx.Done():
		return fmt.Errorf("timed out waiting for unit %s to restart", unitName)
	}

	return nil
}

func (s *SystemdManager) ListUnitsByNames(unitNames []string) ([]systemdbus.UnitStatus, error) {
	return s.conn.ListUnitsByNamesContext(context.Background(), unitNames)
}

func (s *SystemdManager) ListAllUnits() ([]systemdbus.UnitStatus, error) {
	return s.conn.ListUnitsContext(context.Background())
}

func (s *SystemdManager) GetUnitProperties(unitName string) (map[string]interface{}, error) {
	return s.conn.GetUnitPropertiesContext(context.Background(), unitName)
}

type RunningMachineInfo struct {
	Name      string
	Class     string
	Service   string
	OS        string
	Version   string
	Addresses []string
}

func (s *SystemdManager) ListRunningMachines() ([]RunningMachineInfo, error) {
	obj := s.sysConn.Object("org.freedesktop.machine1", "/org/freedesktop/machine1")
	call := obj.Call("org.freedesktop.machine1.Manager.ListMachines", 0)
	if call.Err != nil {
		return nil, fmt.Errorf("failed to list machines: %w", call.Err)
	}

	var machines [][]interface{}
	if err := call.Store(&machines); err != nil {
		return nil, fmt.Errorf("failed to parse machines: %w", err)
	}

	var result []RunningMachineInfo
	for _, m := range machines {
		if len(m) < 4 {
			continue
		}
		name, _ := m[0].(string)
		class, _ := m[1].(string)
		service, _ := m[2].(string)
		os_, _ := m[3].(string)

		info := RunningMachineInfo{
			Name:    name,
			Class:   class,
			Service: service,
			OS:      os_,
		}

		if len(m) > 4 {
			if version, ok := m[4].(string); ok {
				info.Version = version
			}
		}

		addrs := s.getMachineAddresses(name)
		info.Addresses = addrs

		result = append(result, info)
	}

	return result, nil
}

func (s *SystemdManager) GetMachineState(name string) (string, []string, error) {
	obj := s.sysConn.Object("org.freedesktop.machine1", godbus.ObjectPath("/org/freedesktop/machine1/machine/"+machineName(name)))

	prop, err := obj.GetProperty("org.freedesktop.machine1.Machine.State")
	if err != nil {
		if isMachineNotFound(err) {
			return "stopped", nil, nil
		}
		return "", nil, fmt.Errorf("failed to get machine state: %w", err)
	}

	state, _ := prop.Value().(string)

	addresses := s.getMachineAddresses(name)

	return state, addresses, nil
}

func isMachineNotFound(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "UnknownMachine") ||
		strings.Contains(errStr, "unknown machine") ||
		strings.Contains(errStr, "org.freedesktop.DBus.Error.UnknownObject")
}

func (s *SystemdManager) GetMachineLeader(name string) (int, error) {
	obj := s.sysConn.Object("org.freedesktop.machine1", godbus.ObjectPath("/org/freedesktop/machine1/machine/"+machineName(name)))

	prop, err := obj.GetProperty("org.freedesktop.machine1.Machine.Leader")
	if err != nil {
		return 0, fmt.Errorf("failed to get leader property: %w", err)
	}

	leader, ok := prop.Value().(uint32)
	if !ok {
		return 0, fmt.Errorf("failed to parse leader property")
	}

	return int(leader), nil
}

func (s *SystemdManager) getMachineAddresses(name string) []string {
	obj := s.sysConn.Object("org.freedesktop.machine1", godbus.ObjectPath("/org/freedesktop/machine1/machine/"+machineName(name)))

	prop, err := obj.GetProperty("org.freedesktop.machine1.Machine.Address")
	if err != nil {
		return nil
	}

	var addrs [][]byte
	if err := prop.Store(&addrs); err != nil {
		return nil
	}

	var result []string
	for _, addr := range addrs {
		if ip := formatIP(addr); ip != "" {
			result = append(result, ip)
		}
	}

	return result
}

func machineName(name string) string {
	result := make([]byte, 0, len(name)*2)
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			// Escape leading digit (D-Bus object paths cannot start with a digit)
			if i == 0 && c >= '0' && c <= '9' {
				result = append(result, '_', hexDigit(c>>4), hexDigit(c&0x0f))
			} else {
				result = append(result, c)
			}
		} else {
			result = append(result, '_', hexDigit(c>>4), hexDigit(c&0x0f))
		}
	}
	return string(result)
}

func hexDigit(b byte) byte {
	if b < 10 {
		return '0' + b
	}
	return 'a' + b - 10
}

func formatIP(b []byte) string {
	switch len(b) {
	case 4:
		return fmt.Sprintf("%d.%d.%d.%d", b[0], b[1], b[2], b[3])
	case 16:
		return fmt.Sprintf("%x:%x:%x:%x:%x:%x:%x:%x",
			uint16(b[0])<<8|uint16(b[1]),
			uint16(b[2])<<8|uint16(b[3]),
			uint16(b[4])<<8|uint16(b[5]),
			uint16(b[6])<<8|uint16(b[7]),
			uint16(b[8])<<8|uint16(b[9]),
			uint16(b[10])<<8|uint16(b[11]),
			uint16(b[12])<<8|uint16(b[13]),
			uint16(b[14])<<8|uint16(b[15]))
	default:
		return ""
	}
}
