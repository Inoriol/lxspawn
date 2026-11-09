export interface Machine {
    name: string;
    distro: string;
    state: "running" | "stopped" | "creating";
    network: string;
    resources: {
        cpus: number;
        memory: string;
    };
    created_at: string;
    source_path: string;
}

export interface MachineStatus {
    name: string;
    distro: string;
    state: string;
    created_at: string;
    source_path: string;
    files: string[];
}

export interface MachineStats {
    name: string;
    state: string;
    cpu_usage_percent: number;
    memory_used_bytes: number;
    memory_total_bytes: number;
    memory_usage_percent: number;
    disk_used_bytes: number;
    disk_total_bytes: number;
}

export interface MachineConfig {
    name: string;
    distro: string;
    resources: {
        cpus: number;
        memory: string;
        memory_high?: string;
        swap?: string;
        tasks?: number | string;
        io_weight?: number;
    };
    network: {
        mode: string;
        bridge?: string;
        interface?: string;
        zone?: string;
        ports?: string[];
    };
    volumes?: Array<{
        host?: string;
        mount: string;
        readonly?: boolean;
        type?: string;
        options?: string;
    }>;
    security?: {
        no_new_privileges?: boolean;
        seccomp?: boolean;
        capabilities?: string[];
        drop_capabilities?: string[];
        inaccessible?: string[];
    };
    gpu?: {
        enabled: boolean;
        type: string;
    };
    advanced?: {
        ephemeral?: boolean;
        read_only?: boolean;
        hostname?: string;
        environment?: Record<string, string>;
    };
    storage?: {
        size?: string;
    };
    journal?: {
        link?: string;
    };
    hooks?: {
        pre_start?: string;
        post_start?: string;
        post_stop?: string;
    };
    provision?: Array<{
        mode: string;
        script?: string;
        path?: string;
        content?: string;
    }>;
    param?: Record<string, string>;
    probes?: Array<{
        mode: string;
        description?: string;
        script: string;
        hint?: string;
    }>;
}

export interface Distro {
    name: string;
    description?: string;
}

export interface Template {
    name: string;
    description: string;
    path: string;
}

export interface Snapshot {
    name: string;
    created_at: string;
    description?: string;
}

export interface ProvisionStatus {
    last_run: string;
    phases_completed: string[];
    probes_passed: string[];
    script_hashes: Record<string, string>;
}
