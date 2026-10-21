package output

const APIVersion = "1.0"

type Envelope struct {
	Version string      `json:"version"`
	Data    interface{} `json:"data"`
}

type ErrorEnvelope struct {
	Version string     `json:"version"`
	Error   *ErrorInfo `json:"error"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type MachineListData struct {
	Machines []MachineJSON `json:"machines"`
}

type MachineJSON struct {
	Name       string        `json:"name"`
	Distro     string        `json:"distro"`
	State      string        `json:"state"`
	Network    string        `json:"network"`
	Addresses  []string      `json:"addresses,omitempty"`
	Resources  *ResourceJSON `json:"resources,omitempty"`
	CreatedAt  string        `json:"created_at"`
	SourcePath string        `json:"source_path,omitempty"`
}

type ResourceJSON struct {
	CPUs   float64 `json:"cpus,omitempty"`
	Memory string  `json:"memory,omitempty"`
	Swap   string  `json:"swap,omitempty"`
	Tasks  string  `json:"tasks,omitempty"`
}

type MachineStatusData struct {
	Name       string        `json:"name"`
	Distro     string        `json:"distro"`
	State      string        `json:"state"`
	Leader     int           `json:"leader,omitempty"`
	Addresses  []string      `json:"addresses,omitempty"`
	Network    string        `json:"network"`
	Resources  *ResourceJSON `json:"resources,omitempty"`
	CreatedAt  string        `json:"created_at"`
	SourcePath string        `json:"source_path,omitempty"`
	Files      []string      `json:"files,omitempty"`
	Ephemeral  bool          `json:"ephemeral,omitempty"`
	ReadOnly   bool          `json:"read_only,omitempty"`
}

type LifecycleData struct {
	Action  string `json:"action"`
	Machine string `json:"machine"`
	Success bool   `json:"success"`
}

type CreateData struct {
	Machine      string   `json:"machine"`
	SourcePath   string   `json:"source_path"`
	FilesWritten []string `json:"files_written"`
	Success      bool     `json:"success"`
}

type LogsData struct {
	Machine string         `json:"machine"`
	Entries []LogEntryJSON `json:"entries"`
}

type LogEntryJSON struct {
	Timestamp string `json:"timestamp"`
	Priority  int    `json:"priority"`
	Message   string `json:"message"`
}

type SetupData struct {
	GroupCreated     bool     `json:"group_created"`
	DirsCreated      []string `json:"dirs_created"`
	PolkitInstalled  bool     `json:"polkit_installed"`
	SudoersInstalled bool     `json:"sudoers_installed"`
	Success          bool     `json:"success"`
}

type SnapshotJSON struct {
	Name        string `json:"name"`
	Machine     string `json:"machine"`
	Backend     string `json:"backend"`
	CreatedAt   string `json:"created_at"`
	Path        string `json:"path,omitempty"`
	Dataset     string `json:"dataset,omitempty"`
	LV          string `json:"lv,omitempty"`
	Description string `json:"description,omitempty"`
}

type SnapshotListData struct {
	Machine   string         `json:"machine"`
	Backend   string         `json:"backend"`
	Snapshots []SnapshotJSON `json:"snapshots"`
}

type SnapshotData struct {
	Action   string `json:"action"`
	Machine  string `json:"machine"`
	Snapshot string `json:"snapshot"`
	Backend  string `json:"backend"`
	Success  bool   `json:"success"`
}

type StatsData struct {
	Name              string  `json:"name"`
	State             string  `json:"state"`
	CPUUsagePercent   float64 `json:"cpu_usage_percent"`
	MemoryUsedBytes   int64   `json:"memory_used_bytes"`
	MemoryTotalBytes  int64   `json:"memory_total_bytes"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	DiskUsedBytes     int64   `json:"disk_used_bytes,omitempty"`
	DiskTotalBytes    int64   `json:"disk_total_bytes,omitempty"`
}

type NetworkConfigJSON struct {
	Mode      string   `json:"mode"`
	Bridge    string   `json:"bridge,omitempty"`
	Zone      string   `json:"zone,omitempty"`
	Interface string   `json:"interface,omitempty"`
	Ports     []string `json:"ports,omitempty"`
}

type VolumeJSON struct {
	Host     string `json:"host,omitempty"`
	Mount    string `json:"mount"`
	ReadOnly bool   `json:"readonly"`
	Type     string `json:"type,omitempty"`
	Options  string `json:"options,omitempty"`
}

type SecurityJSON struct {
	NoNewPrivileges  bool     `json:"no_new_privileges"`
	Seccomp          bool     `json:"seccomp"`
	Capabilities     []string `json:"capabilities,omitempty"`
	DropCapabilities []string `json:"drop_capabilities,omitempty"`
	Inaccessible     []string `json:"inaccessible,omitempty"`
}

type GPUJSON struct {
	Enabled bool     `json:"enabled"`
	Type    string   `json:"type"`
	Devices []string `json:"devices,omitempty"`
}

type AdvancedJSON struct {
	Ephemeral   bool              `json:"ephemeral"`
	ReadOnly    bool              `json:"read_only"`
	Hostname    string            `json:"hostname,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

type ConfigData struct {
	Name      string             `json:"name"`
	Distro    string             `json:"distro"`
	Resources *ResourceJSON      `json:"resources,omitempty"`
	Storage   *StorageJSON       `json:"storage,omitempty"`
	Network   *NetworkConfigJSON `json:"network,omitempty"`
	Volumes   []VolumeJSON       `json:"volumes,omitempty"`
	Security  *SecurityJSON      `json:"security,omitempty"`
	GPU       *GPUJSON           `json:"gpu,omitempty"`
	Advanced  *AdvancedJSON      `json:"advanced,omitempty"`
}

type StorageJSON struct {
	Size string `json:"size,omitempty"`
}

type ProvisionStatusData struct {
	Name           string            `json:"name"`
	LastRun        string            `json:"last_run,omitempty"`
	PhasesComplete []string          `json:"phases_completed,omitempty"`
	ProbesPassed   []string          `json:"probes_passed,omitempty"`
	ScriptHashes   map[string]string `json:"script_hashes,omitempty"`
}

type DistroInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type DistroListData struct {
	Distros []DistroInfo `json:"distros"`
}
