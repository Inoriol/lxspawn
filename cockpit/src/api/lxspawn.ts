import cockpit from 'cockpit';
import type {
    Machine,
    MachineStatus,
    MachineStats,
    MachineConfig,
    Distro,
    Template,
    Snapshot,
    ProvisionStatus,
} from './types';

const _ = cockpit.gettext;

interface LxspawnResponse<T> {
    version: string;
    data: T;
}

async function spawnLxspawn(args: string[]): Promise<string> {
    try {
        const output = await cockpit.spawn(["lxspawn", ...args], { err: "message" });
        return output;
    } catch (error: any) {
        console.error("lxspawn command failed:", args, error);
        throw new Error(error.message || _("Command failed"));
    }
}

export async function listMachines(): Promise<Machine[]> {
    const output = await spawnLxspawn(["list", "--json"]);
    const response: LxspawnResponse<{ machines: Machine[] | null }> = JSON.parse(output);
    return response.data.machines || [];
}

export async function getMachineStatus(name: string): Promise<MachineStatus> {
    const output = await spawnLxspawn(["status", name, "--json"]);
    const response: LxspawnResponse<MachineStatus> = JSON.parse(output);
    return response.data;
}

export async function getMachineStats(name: string): Promise<MachineStats> {
    const output = await spawnLxspawn(["stats", name, "--json"]);
    const response: LxspawnResponse<MachineStats> = JSON.parse(output);
    return response.data;
}

export async function getMachineConfig(name: string): Promise<MachineConfig> {
    const output = await spawnLxspawn(["config", name, "--json"]);
    const response: LxspawnResponse<MachineConfig> = JSON.parse(output);
    return response.data;
}

export async function getProvisionStatus(name: string): Promise<ProvisionStatus> {
    const output = await spawnLxspawn(["provision-status", name, "--json"]);
    const response: LxspawnResponse<ProvisionStatus> = JSON.parse(output);
    return response.data;
}

export async function startMachine(name: string): Promise<void> {
    await spawnLxspawn(["start", name]);
}

export async function stopMachine(name: string): Promise<void> {
    await spawnLxspawn(["stop", name]);
}

export async function deleteMachine(name: string): Promise<void> {
    await spawnLxspawn(["delete", name, "-y"]);
}

export async function editResources(
    name: string,
    resources: { cpus?: number; memory?: string; memory_high?: string; swap?: string; tasks?: number }
): Promise<void> {
    const args = ["edit-resources", name];
    if (resources.cpus !== undefined) {
        args.push("--cpus", resources.cpus.toString());
    }
    if (resources.memory) {
        args.push("--memory", resources.memory);
    }
    if (resources.memory_high) {
        args.push("--memory-high", resources.memory_high);
    }
    if (resources.swap) {
        args.push("--swap", resources.swap);
    }
    if (resources.tasks !== undefined) {
        args.push("--tasks", resources.tasks.toString());
    }
    await spawnLxspawn(args);
}

export async function editNetwork(
    name: string,
    network: { mode: string; bridge?: string; interface?: string }
): Promise<void> {
    const args = ["edit-network", name, "--mode", network.mode];
    if (network.bridge) {
        args.push("--bridge", network.bridge);
    }
    if (network.interface) {
        args.push("--interface", network.interface);
    }
    await spawnLxspawn(args);
}

export async function addVolume(
    name: string,
    volume: { host: string; mount: string; readonly?: boolean }
): Promise<void> {
    const args = ["add-volume", name, "--host", volume.host, "--mount", volume.mount];
    if (volume.readonly) {
        args.push("--readonly");
    }
    await spawnLxspawn(args);
}

export async function editVolume(
    name: string,
    volume: { mount: string; host?: string; readonly?: boolean }
): Promise<void> {
    const args = ["edit-volume", name, "--mount", volume.mount];
    if (volume.host) {
        args.push("--host", volume.host);
    }
    if (volume.readonly !== undefined) {
        args.push("--readonly", volume.readonly.toString());
    }
    await spawnLxspawn(args);
}

export async function removeVolume(name: string, mount: string): Promise<void> {
    await spawnLxspawn(["remove-volume", name, "--mount", mount]);
}

export async function listDistros(): Promise<Distro[]> {
    const output = await spawnLxspawn(["list-distros", "--json"]);
    const response: LxspawnResponse<{ distros: Distro[] }> = JSON.parse(output);
    return response.data.distros || [];
}

export async function listTemplates(): Promise<Template[]> {
    const output = await spawnLxspawn(["list-templates", "--json"]);
    const response: LxspawnResponse<{ templates: Template[] | null }> = JSON.parse(output);
    return response.data.templates || [];
}

export async function getTemplateContent(name: string): Promise<string> {
    const output = await spawnLxspawn(["list-templates", "--json"]);
    const response: LxspawnResponse<{ templates: Template[] | null }> = JSON.parse(output);
    const templates = response.data.templates || [];
    const template = templates.find(t => t.name === name);
    
    if (!template) {
        throw new Error(`Template '${name}' not found`);
    }
    
    return await cockpit.file(template.path).read();
}

export async function validateYAML(yamlPath: string): Promise<boolean> {
    const output = await spawnLxspawn(["validate", yamlPath, "--json"]);
    const response: LxspawnResponse<{ valid: boolean }> = JSON.parse(output);
    return response.data.valid;
}

export async function createMachine(name: string, yamlContent: string): Promise<void> {
    const userInfo = await cockpit.user();
    const configDir = `${userInfo.home}/.local/share/lxspawn/${name}`;
    const configPath = `${configDir}/config.yaml`;
    
    try {
        // Create directory first
        await cockpit.spawn(["mkdir", "-p", configDir]);
        // Write config file
        await cockpit.file(configPath).replace(yamlContent);
        // Launch machine
        await spawnLxspawn(["launch", configPath]);
    } catch (error) {
        // Cleanup on error
        try {
            await cockpit.file(configPath, { superuser: 'try' }).replace(null);
        } catch {
            // Ignore cleanup errors
        }
        throw error;
    }
}

export async function listSnapshots(name: string): Promise<Snapshot[]> {
    const output = await spawnLxspawn(["list-snapshots", name, "--json"]);
    const response: LxspawnResponse<{ snapshots: Snapshot[] | null }> = JSON.parse(output);
    return response.data.snapshots || [];
}

export async function createSnapshot(name: string, snapshotName: string, description?: string): Promise<void> {
    const args = ["snapshot", name, snapshotName];
    if (description) {
        args.push("--description", description);
    }
    await spawnLxspawn(args);
}

export async function rollbackSnapshot(name: string, snapshotName: string): Promise<void> {
    await spawnLxspawn(["rollback", name, snapshotName]);
}

export async function deleteSnapshot(name: string, snapshotName: string): Promise<void> {
    await spawnLxspawn(["delete-snapshot", name, snapshotName]);
}

export async function provisionMachine(name: string): Promise<void> {
    await spawnLxspawn(["provision", name]);
}
