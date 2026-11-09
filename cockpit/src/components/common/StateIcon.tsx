import React from 'react';
import { Icon } from "@patternfly/react-core/dist/esm/components/Icon/index.js";
import {
    CheckCircleIcon,
    ExclamationCircleIcon,
    InProgressIcon,
    UnknownIcon
} from '@patternfly/react-icons';

interface StateIconProps {
    state: string;
}

export const StateIcon: React.FC<StateIconProps> = ({ state }) => {
    switch (state) {
        case 'running':
            return (
                <Icon status="success">
                    <CheckCircleIcon />
                </Icon>
            );
        case 'stopped':
            return (
                <Icon status="danger">
                    <ExclamationCircleIcon />
                </Icon>
            );
        case 'creating':
            return (
                <Icon status="info">
                    <InProgressIcon />
                </Icon>
            );
        default:
            return (
                <Icon>
                    <UnknownIcon />
                </Icon>
            );
    }
};
