import React from 'react';
import cockpit from 'cockpit';
import { Card, CardHeader, CardTitle } from "@patternfly/react-core/dist/esm/components/Card/index.js";
import { Flex, FlexItem } from "@patternfly/react-core/dist/esm/layouts/Flex/index.js";
import { Divider } from "@patternfly/react-core/dist/esm/components/Divider/index.js";
import { ServerIcon, CubeIcon } from '@patternfly/react-icons';
import type { Machine } from '../../api/types';

const _ = cockpit.gettext;

interface AggregateStatusCardsProps {
    machines: Machine[];
}

export const AggregateStatusCards: React.FC<AggregateStatusCardsProps> = ({ machines }) => {
    const runningCount = machines.filter(m => m.state === 'running').length;
    const stoppedCount = machines.filter(m => m.state === 'stopped').length;
    const totalCount = machines.length;

    return (
        <>
            <Card id="card-lxspawn-machines" className="ct-card-info">
                <CardHeader>
                    <Flex alignItems={{ default: 'alignItemsCenter' }}>
                        <ServerIcon />
                        <CardTitle>
                            {cockpit.format(cockpit.ngettext("$0 Machine", "$0 Machines", totalCount), totalCount)}
                        </CardTitle>
                    </Flex>
                    <Flex alignItems={{ default: 'alignItemsCenter' }}>
                        <FlexItem className="active-resources">
                            <CubeIcon />
                            {cockpit.format(_("Running: $0"), runningCount)}
                        </FlexItem>
                        <Divider orientation={{ default: "vertical" }} />
                        <FlexItem className="active-resources">
                            {cockpit.format(_("Stopped: $0"), stoppedCount)}
                        </FlexItem>
                    </Flex>
                </CardHeader>
            </Card>
        </>
    );
};
