import React, { useState, useEffect } from 'react';
import cockpit from 'cockpit';
import { Page, PageSection } from "@patternfly/react-core/dist/esm/components/Page/index.js";
import { Gallery } from "@patternfly/react-core/dist/esm/layouts/Gallery/index.js";
import { Card, CardBody, CardHeader, CardTitle } from "@patternfly/react-core/dist/esm/components/Card/index.js";
import { Button } from "@patternfly/react-core/dist/esm/components/Button/index.js";
import { TextInput } from "@patternfly/react-core/dist/esm/components/TextInput/index.js";
import { Toolbar, ToolbarContent, ToolbarItem } from "@patternfly/react-core/dist/esm/components/Toolbar/index.js";
import { Select, SelectList, SelectOption } from "@patternfly/react-core/dist/esm/components/Select/index.js";
import { MenuToggle, MenuToggleElement } from "@patternfly/react-core/dist/esm/components/MenuToggle/index.js";
import { Progress, ProgressVariant } from "@patternfly/react-core/dist/esm/components/Progress/index.js";
import { ActionsColumn, IAction } from '@patternfly/react-table';
import { Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { Table, TableVariant } from '@patternfly/react-table';

import { AggregateStatusCards } from './cards/AggregateStatusCards';
import { StateIcon } from './common/StateIcon';
import { listMachines, getMachineStats, startMachine, stopMachine, deleteMachine } from '../api/lxspawn';
import { formatBytes, formatPercent, getStateLabel } from '../utils/format';
import type { Machine, MachineStats } from '../api/types';

const _ = cockpit.gettext;

interface MachineRowProps {
    machine: Machine;
    stats?: MachineStats;
    onNavigate: (machineName: string) => void;
    onRefresh: () => void;
}

const MachineRow: React.FC<MachineRowProps> = ({ machine, stats, onNavigate, onRefresh }) => {
    const [isActionsOpen, setIsActionsOpen] = useState(false);

    const rowActions: IAction[] = [
        {
            title: _("Start"),
            onClick: async () => {
                await startMachine(machine.name);
                onRefresh();
            },
            isDisabled: machine.state === 'running'
        },
        {
            title: _("Stop"),
            onClick: async () => {
                await stopMachine(machine.name);
                onRefresh();
            },
            isDisabled: machine.state !== 'running'
        },
        {
            title: _("Delete"),
            onClick: async () => {
                if (confirm(cockpit.format(_("Are you sure you want to delete machine '$0'?"), machine.name))) {
                    await deleteMachine(machine.name);
                    onRefresh();
                }
            }
        }
    ];

    const cpuPercent = stats?.cpu_usage_percent || 0;
    const memPercent = stats?.memory_usage_percent || 0;
    const memUsed = stats?.memory_used_bytes || 0;
    const memTotal = stats?.memory_total_bytes || 0;

    return (
        <Tr>
            <Td dataLabel={_("Name")}>
                <Button variant="link" onClick={() => onNavigate(machine.name)}>
                    {machine.name}
                </Button>
            </Td>
            <Td dataLabel={_("Distro")}>{machine.distro}</Td>
            <Td dataLabel={_("State")}>
                <span style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <StateIcon state={machine.state} />
                    {getStateLabel(machine.state)}
                </span>
            </Td>
            <Td dataLabel={_("CPU")}>
                {machine.state === 'running' && stats ? (
                    <Progress
                        value={cpuPercent}
                        className="pf-m-sm"
                        min={0}
                        max={100}
                        variant={cpuPercent > 90 ? ProgressVariant.danger : undefined}
                        title={_("CPU")}
                        label={formatPercent(cpuPercent)}
                    />
                ) : '-'}
            </Td>
            <Td dataLabel={_("Memory")}>
                {machine.state === 'running' && stats ? (
                    <Progress
                        value={memPercent}
                        className="pf-m-sm"
                        min={0}
                        max={100}
                        variant={memPercent > 90 ? ProgressVariant.danger : undefined}
                        title={_("Memory")}
                        label={cockpit.format("$0 / $1", formatBytes(memUsed), formatBytes(memTotal))}
                    />
                ) : '-'}
            </Td>
            <Td dataLabel={_("Network")}>{machine.network}</Td>
            <Td isActionCell>
                <ActionsColumn
                    items={rowActions}
                    isOpen={isActionsOpen}
                    onToggle={(isOpen) => setIsActionsOpen(isOpen)}
                />
            </Td>
        </Tr>
    );
};

interface MachineListProps {
    onNavigate: (machineName: string) => void;
}

export const MachineList: React.FC<MachineListProps> = ({ onNavigate }) => {
    const [machines, setMachines] = useState<Machine[]>([]);
    const [stats, setStats] = useState<Record<string, MachineStats>>({});
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [nameFilter, setNameFilter] = useState('');
    const [stateFilter, setStateFilter] = useState('All');
    const [stateIsExpanded, setStateIsExpanded] = useState(false);

    const fetchMachines = async () => {
        try {
            setLoading(true);
            const machines = await listMachines();
            setMachines(machines);
            setError(null);
        } catch (err: any) {
            setError(err.message || _("Failed to load machines"));
        } finally {
            setLoading(false);
        }
    };

    const fetchStats = async () => {
        const runningMachines = machines.filter(m => m.state === 'running');
        const newStats: Record<string, MachineStats> = {};

        for (const machine of runningMachines) {
            try {
                const stats = await getMachineStats(machine.name);
                newStats[machine.name] = stats;
            } catch (err) {
                console.error(`Failed to fetch stats for ${machine.name}:`, err);
            }
        }

        setStats(newStats);
    };

    useEffect(() => {
        fetchMachines();
    }, []);

    useEffect(() => {
        if (machines.length > 0) {
            fetchStats();
            const interval = setInterval(fetchStats, 5000);
            return () => clearInterval(interval);
        }
    }, [machines]);

    const filteredMachines = machines.filter(machine => {
        const matchesName = machine.name.toLowerCase().includes(nameFilter.toLowerCase());
        const matchesState = stateFilter === 'All' || machine.state === stateFilter.toLowerCase();
        return matchesName && matchesState;
    });

    const stateOptions = ['All', 'Running', 'Stopped'];

    const toggle = (toggleRef: React.LegacyRef<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            onClick={() => setStateIsExpanded(!stateIsExpanded)}
            isExpanded={stateIsExpanded}
            isFullWidth
        >
            {stateFilter}
        </MenuToggle>
    );

    const columns = [
        { title: _("Name") },
        { title: _("Distro") },
        { title: _("State") },
        { title: _("CPU") },
        { title: _("Memory") },
        { title: _("Network") },
        { title: "" }
    ];

    return (
        <Page>
            <PageSection>
                <Gallery hasGutter>
                    <AggregateStatusCards machines={machines} />
                </Gallery>
            </PageSection>
            <PageSection>
                <Card>
                    <CardHeader>
                        <Toolbar>
                            <ToolbarContent>
                                <ToolbarItem alignSelf="baseline">
                                    <TextInput
                                        name="text-search"
                                        id="text-search"
                                        type="search"
                                        value={nameFilter}
                                        onChange={(_, value) => setNameFilter(value)}
                                        placeholder={_("Filter by name")}
                                    />
                                </ToolbarItem>
                                <ToolbarItem>
                                    <Select
                                        id="state-select"
                                        isOpen={stateIsExpanded}
                                        selected={stateFilter}
                                        onSelect={(_, value) => {
                                            setStateFilter(value as string);
                                            setStateIsExpanded(false);
                                        }}
                                        onOpenChange={(isOpen) => setStateIsExpanded(isOpen)}
                                        toggle={toggle}
                                    >
                                        <SelectList>
                                            {stateOptions.map(option => (
                                                <SelectOption key={option} value={option}>
                                                    {option}
                                                </SelectOption>
                                            ))}
                                        </SelectList>
                                    </Select>
                                </ToolbarItem>
                                <ToolbarItem>
                                    <Button variant="primary" onClick={() => onNavigate('create')}>
                                        {_("Create machine")}
                                    </Button>
                                </ToolbarItem>
                            </ToolbarContent>
                        </Toolbar>
                    </CardHeader>
                    <CardBody>
                        {loading ? (
                            <p>{_("Loading...")}</p>
                        ) : error ? (
                            <p style={{ color: 'red' }}>{error}</p>
                        ) : filteredMachines.length === 0 ? (
                            <p>{_("No machines found")}</p>
                        ) : (
                            <Table variant={TableVariant.compact} aria-label={_("Machines table")}>
                                <Thead>
                                    <Tr>
                                        {columns.map((col, idx) => (
                                            <Th key={idx}>{col.title}</Th>
                                        ))}
                                    </Tr>
                                </Thead>
                                <Tbody>
                                    {filteredMachines.map(machine => (
                                        <MachineRow
                                            key={machine.name}
                                            machine={machine}
                                            stats={stats[machine.name]}
                                            onNavigate={onNavigate}
                                            onRefresh={fetchMachines}
                                        />
                                    ))}
                                </Tbody>
                            </Table>
                        )}
                    </CardBody>
                </Card>
            </PageSection>
        </Page>
    );
};
