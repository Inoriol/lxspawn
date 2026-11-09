/*
 * SPDX-License-Identifier: MIT
 *
 * Cockpit plugin for managing LXC-style system containers powered by systemd-nspawn
 */

import React, { useState, useEffect } from 'react';
import { Page } from '@patternfly/react-core/dist/esm/components/Page/index.js';

import cockpit from 'cockpit';

import { MachineList } from './components/MachineList';
import { MachineDetailsPage } from './components/MachineDetailsPage';
import { CreateMachineWizard } from './components/CreateMachineWizard';

const _ = cockpit.gettext;

export const Application = () => {
    const [currentView, setCurrentView] = useState<'list' | 'details' | 'create'>('list');
    const [selectedMachine, setSelectedMachine] = useState<string | null>(null);

    useEffect(() => {
        const handleLocation = () => {
            const path = cockpit.location.path;
            if (path.length === 0) {
                setCurrentView('list');
                setSelectedMachine(null);
            } else if (path[0] === 'create') {
                setCurrentView('create');
            } else if (path.length >= 1) {
                setCurrentView('details');
                setSelectedMachine(path[0]);
            }
        };

        handleLocation();
        cockpit.addEventListener('locationchanged', handleLocation);
        return () => cockpit.removeEventListener('locationchanged', handleLocation);
    }, []);

    const navigateToMachine = (machineName: string) => {
        if (machineName === 'create') {
            cockpit.location.go(['create']);
        } else {
            cockpit.location.go([machineName]);
        }
    };

    const navigateBack = () => {
        cockpit.location.go([]);
    };

    return (
        <Page className='pf-m-no-sidebar'>
            {currentView === 'list' && (
                <MachineList onNavigate={navigateToMachine} />
            )}
            {currentView === 'details' && selectedMachine && (
                <MachineDetailsPage
                    machineName={selectedMachine}
                    onNavigateBack={navigateBack}
                />
            )}
            {currentView === 'create' && (
                <CreateMachineWizard
                    onComplete={navigateBack}
                    onCancel={navigateBack}
                />
            )}
        </Page>
    );
};
