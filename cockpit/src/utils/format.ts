import cockpit from 'cockpit';

const _ = cockpit.gettext;

export function formatBytes(bytes: number, decimals: number = 2): string {
    if (bytes === 0) return '0 B';

    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];

    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

export function formatCPU(cpus: number): string {
    return cockpit.format(cockpit.ngettext("$0 vCPU", "$0 vCPUs", cpus), cpus);
}

export function formatMemory(memoryStr: string): string {
    return memoryStr;
}

export function formatPercent(value: number, decimals: number = 1): string {
    return value.toFixed(decimals) + '%';
}

export function formatRelativeTime(dateStr: string): string {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffSecs = Math.floor(diffMs / 1000);
    const diffMins = Math.floor(diffSecs / 60);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffSecs < 60) {
        return _("just now");
    } else if (diffMins < 60) {
        return cockpit.format(cockpit.ngettext("$0 minute ago", "$0 minutes ago", diffMins), diffMins);
    } else if (diffHours < 24) {
        return cockpit.format(cockpit.ngettext("$0 hour ago", "$0 hours ago", diffHours), diffHours);
    } else if (diffDays < 7) {
        return cockpit.format(cockpit.ngettext("$0 day ago", "$0 days ago", diffDays), diffDays);
    } else {
        return date.toLocaleDateString();
    }
}

export function getStateColor(state: string): 'success' | 'danger' | 'warning' | 'info' | 'default' {
    switch (state) {
        case 'running':
            return 'success';
        case 'stopped':
            return 'danger';
        case 'creating':
            return 'info';
        default:
            return 'default';
    }
}

export function getStateLabel(state: string): string {
    switch (state) {
        case 'running':
            return _("Running");
        case 'stopped':
            return _("Stopped");
        case 'creating':
            return _("Creating");
        default:
            return state;
    }
}
