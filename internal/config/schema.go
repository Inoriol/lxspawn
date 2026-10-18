package config

type NetworkMode string

const (
	NetworkHost     NetworkMode = "host"
	NetworkPrivate  NetworkMode = "private"
	NetworkBridge   NetworkMode = "bridge"
	NetworkZone     NetworkMode = "zone"
	NetworkMacvlan  NetworkMode = "macvlan"
	NetworkIpvlan   NetworkMode = "ipvlan"
	NetworkAdaptive NetworkMode = "adaptive"
)

type ProvisionMode string

const (
	ProvisionBoot   ProvisionMode = "boot"
	ProvisionData   ProvisionMode = "data"
	ProvisionYQ     ProvisionMode = "yq"
	ProvisionSystem ProvisionMode = "system"
	ProvisionUser   ProvisionMode = "user"
)

type Config struct {
	Name      string            `yaml:"name"`
	Distro    string            `yaml:"distro"`
	Resources Resources         `yaml:"resources"`
	Storage   Storage           `yaml:"storage"`
	Network   Network           `yaml:"network"`
	Volumes   []Volume          `yaml:"volumes"`
	Provision []Provision       `yaml:"provision"`
	Probes    []Probe           `yaml:"probes"`
	Hooks     Hooks             `yaml:"hooks"`
	Security  Security          `yaml:"security"`
	GPU       GPU               `yaml:"gpu"`
	Param     map[string]string `yaml:"param"`
	Journal   Journal           `yaml:"journal"`
	Advanced  Advanced          `yaml:"advanced"`
}

type Resources struct {
	CPUs       float64 `yaml:"cpus"`
	Memory     string  `yaml:"memory"`
	MemoryHigh string  `yaml:"memory_high"`
	Swap       string  `yaml:"swap"`
	Tasks      string  `yaml:"tasks"`
	IOWeight   int     `yaml:"io_weight"`
}

type Storage struct {
	Size string `yaml:"size"`
}

type Network struct {
	Mode       NetworkMode `yaml:"mode"`
	Bridge     string      `yaml:"bridge"`
	Zone       string      `yaml:"zone"`
	Interface  string      `yaml:"interface"`
	Ports      []string    `yaml:"ports"`
	HostAccess HostAccess  `yaml:"host_access"`
}

type HostAccess struct {
	Enabled   bool   `yaml:"enabled"`
	Interface string `yaml:"interface"` // default: "mv-lxspawn"
	IP        string `yaml:"ip"`        // static IP or "dhcp"
}

type Volume struct {
	Host     string `yaml:"host"`
	Mount    string `yaml:"mount"`
	ReadOnly bool   `yaml:"readonly"`
	Type     string `yaml:"type"`
	Options  string `yaml:"options"`
}

type Hooks struct {
	PreStart  string `yaml:"pre_start,omitempty"`
	PostStart string `yaml:"post_start,omitempty"`
	PostStop  string `yaml:"post_stop,omitempty"`
}

type Security struct {
	NoNewPrivileges  *bool    `yaml:"no_new_privileges,omitempty"`
	PrivateUsers     string   `yaml:"private_users,omitempty"`
	Capabilities     []string `yaml:"capabilities,omitempty"`
	DropCapabilities []string `yaml:"drop_capabilities,omitempty"`
	Inaccessible     []string `yaml:"inaccessible,omitempty"`
	Seccomp          *bool    `yaml:"seccomp,omitempty"`
	SystemCallFilter []string `yaml:"system_call_filter,omitempty"`
}

type Provision struct {
	Mode        ProvisionMode `yaml:"mode"`
	Script      string        `yaml:"script,omitempty"`
	File        *FileRef      `yaml:"file,omitempty"`
	Path        string        `yaml:"path,omitempty"`
	Content     string        `yaml:"content,omitempty"`
	Owner       string        `yaml:"owner,omitempty"`
	Permissions string        `yaml:"permissions,omitempty"`
	Overwrite   *bool         `yaml:"overwrite,omitempty"`
	Expression  string        `yaml:"expression,omitempty"`
	Format      string        `yaml:"format,omitempty"`
}

type FileRef struct {
	URL    string `yaml:"url"`
	Digest string `yaml:"digest,omitempty"`
}

type Probe struct {
	Mode        string `yaml:"mode"`
	Description string `yaml:"description"`
	Script      string `yaml:"script"`
	Hint        string `yaml:"hint,omitempty"`
}

type Journal struct {
	Link string `yaml:"link"`
}

type Advanced struct {
	Ephemeral   bool              `yaml:"ephemeral"`
	ReadOnly    bool              `yaml:"read_only"`
	Hostname    string            `yaml:"hostname"`
	Environment map[string]string `yaml:"environment"`
}

type GPU struct {
	Enabled bool     `yaml:"enabled"`
	Type    string   `yaml:"type"`
	Devices []string `yaml:"devices,omitempty"`
}
