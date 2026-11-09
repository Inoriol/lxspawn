import React, { useState, useEffect } from 'react';
import cockpit from 'cockpit';
import { Page, PageSection } from "@patternfly/react-core/dist/esm/components/Page/index.js";
import { Breadcrumb, BreadcrumbItem } from "@patternfly/react-core/dist/esm/components/Breadcrumb/index.js";
import { Card, CardBody, CardHeader, CardTitle } from "@patternfly/react-core/dist/esm/components/Card/index.js";
import { Gallery } from "@patternfly/react-core/dist/esm/layouts/Gallery/index.js";
import { Button } from "@patternfly/react-core/dist/esm/components/Button/index.js";
import { DescriptionList, DescriptionListTerm, DescriptionListGroup, DescriptionListDescription } from "@patternfly/react-core/dist/esm/components/DescriptionList/index.js";
import { Progress, ProgressVariant } from "@patternfly/react-core/dist/esm/components/Progress/index.js";

import { StateIcon } from './common/StateIcon';
import { getMachineStatus, getMachineConfig, getMachineStats, startMachine, stopMachine, deleteMachine } from '../api/lxspawn';
import { formatBytes, formatPercent, formatCPU, getStateLabel } from '../utils/format';
import type { MachineStatus, MachineConfig, MachineStats } from '../api/types';

const _ = cockpit.gettext;

interface MachineDetailsPageProps {
    machineName: string;
    onNavigateBack: () => void;
}

export const MachineDetailsPage: React.FC<MachineDetailsPageProps> = ({ machineName, onNavigateBack }) => {
    const [status, setStatus] = useState<MachineStatus | null>(null);
    const [config, setConfig] = useState<MachineConfig | null>(null);
    const [stats, setStats] = useState<MachineStats | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const fetchData = async () => {
        try {
            setLoading(true);
            const [statusData, configData, statsData] = await Promise.all([
                getMachineStatus(machineName),
                getMachineConfig(machineName),
                getMachineStats(machineName).catch(() => null)
            ]);

            setStatus(statusData);
            setConfig(configData);
            if (statsData) {
                setStats(statsData);
            }
            setError(null);
        } catch (err: any) {
            setError(err.message || _("Failed to load machine details"));
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchData();
        const interval = setInterval(() => {
            if (status?.state === 'running') {
                getMachineStats(machineName)
                    .then(statsData => setStats(statsData))
                    .catch(() => {});
            }
        }, 5000);

        return () => clearInterval(interval);
    }, [machineName, status?.state]);

    const handleStart = async () => {
        await startMachine(machineName);
        fetchData();
    };

    const handleStop = async () => {
        await stopMachine(machineName);
        fetchData();
    };

    const handleDelete = async () => {
        if (confirm(cockpit.format(_("Are you sure you want to delete machine '$0'?"), machineName))) {
            await deleteMachine(machineName);
            onNavigateBack();
        }
    };

    if (loading) {
        return <Page><PageSection><p>{_("Loading...")}</p></PageSection></Page>;
    }

    if (error) {
        return <Page><PageSection><p style={{ color: 'red' }}>{error}</p></PageSection></Page>;
    }

    const cpuPercent = stats?.cpu_usage_percent || 0;
    const memPercent = stats?.memory_usage_percent || 0;
    const memUsed = stats?.memory_used_bytes || 0;
    const memTotal = stats?.memory_total_bytes || 0;

    return (
        <Page>
            <PageSection>
                <Breadcrumb>
                    <BreadcrumbItem to="#" onClick={onNavigateBack}>
                        {_("Machines")}
                    </BreadcrumbItem>
                    <BreadcrumbItem isActive>{machineName}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <PageSection>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
                    <h1>{machineName}</h1>
                    <div style={{ display: 'flex', gap: '8px' }}>
                        {status?.state !== 'running' && (
                            <Button variant="primary" onClick={handleStart}>
                                {_("Start")}
                            </Button>
                        )}
                        {status?.state === 'running' && (
                            <Button variant="danger" onClick={handleStop}>
                                {_("Stop")}
                            </Button>
                        )}
                        <Button variant="danger" onClick={handleDelete}>
                            {_("Delete")}
                        </Button>
                    </div>
                </div>
            </PageSection>
            <PageSection>
                <Gallery hasGutter>
                    <Card>
                        <CardTitle>{_("Overview")}</CardTitle>
                        <CardBody>
                            <DescriptionList>
                                <DescriptionListGroup>
                                    <DescriptionListTerm>{_("State")}</DescriptionListTerm>
                                    <DescriptionListDescription>
                                        <span style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                            <StateIcon state={status?.state || 'unknown'} />
                                            {getStateLabel(status?.state || 'unknown')}
                                        </span>
                                    </DescriptionListDescription>
                                </DescriptionListGroup>
                                <DescriptionListGroup>
                                    <DescriptionListTerm>{_("Distro")}</DescriptionListTerm>
                                    <DescriptionListDescription>{config?.distro}</DescriptionListDescription>
                                </DescriptionListGroup>
                                <DescriptionListGroup>
                                    <DescriptionListTerm>{_("CPU")}</DescriptionListTerm>
                                    <DescriptionListDescription>
                                        {config?.resources.cpus ? formatCPU(config.resources.cpus) : '-'}
                                    </DescriptionListDescription>
                                </DescriptionListGroup>
                                <DescriptionListGroup>
                                    <DescriptionListTerm>{_("Memory")}</DescriptionListTerm>
                                    <DescriptionListDescription>{config?.resources.memory || '-'}</DescriptionListDescription>
                                </DescriptionListGroup>
                                <DescriptionListGroup>
                                    <DescriptionListTerm>{_("Network")}</DescriptionListTerm>
                                    <DescriptionListDescription>{config?.network.mode || '-'}</DescriptionListDescription>
                                </DescriptionListGroup>
                            </DescriptionList>
                        </CardBody>
                    </Card>

                    <Card>
                        <CardTitle>{_("Usage")}</CardTitle>
                        <CardBody>
                            {status?.state === 'running' && stats ? (
                                <>
                                    <Progress
                                        value={cpuPercent}
                                        className="pf-m-sm"
                                        min={0}
                                        max={100}
                                        variant={cpuPercent > 90 ? ProgressVariant.danger : undefined}
                                        title={_("CPU")}
                                        label={formatPercent(cpuPercent)}
                                    />
                                    <br />
                                    <Progress
                                        value={memPercent}
                                        className="pf-m-sm"
                                        min={0}
                                        max={100}
                                        variant={memPercent > 90 ? ProgressVariant.danger : undefined}
                                        title={_("Memory")}
                                        label={cockpit.format("$0 / $1", formatBytes(memUsed), formatBytes(memTotal))}
                                    />
                                </>
                            ) : (
                                <p>{_("Machine is not running")}</p>
                            )}
                        </CardBody>
                    </Card>

                    {config?.volumes && config.volumes.length > 0 && (
                        <Card>
                            <CardTitle>{_("Volumes")}</CardTitle>
                            <CardBody>
                                <DescriptionList>
                                    {config.volumes.map((vol, idx) => (
                                        <DescriptionListGroup key={idx}>
                                            <DescriptionListTerm>{vol.mount}</DescriptionListTerm>
                                            <DescriptionListDescription>
                                                {vol.host || vol.type} {vol.readonly ? _("(read-only)") : ""}
                                            </DescriptionListDescription>
                                        </DescriptionListGroup>
                                    ))}
                                </DescriptionList>
                            </CardBody>
                        </Card>
                    )}
                </Gallery>
            </PageSection>
        </Page>
    );
};
