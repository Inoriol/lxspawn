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
	Name      string `json:"name"`
	Machine   string `json:"machine"`
	Backend   string `json:"backend"`
	CreatedAt string `json:"created_at"`
	Path      string `json:"path,omitempty"`
	Dataset   string `json:"dataset,omitempty"`
	LV        string `json:"lv,omitempty"`
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
