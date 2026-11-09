import React, { useState } from 'react';
import cockpit from 'cockpit';
import { Page, PageSection } from "@patternfly/react-core/dist/esm/components/Page/index.js";
import { Card, CardBody, CardHeader, CardTitle } from "@patternfly/react-core/dist/esm/components/Card/index.js";
import { Form, FormGroup } from "@patternfly/react-core/dist/esm/components/Form/index.js";
import { TextInput } from "@patternfly/react-core/dist/esm/components/TextInput/index.js";
import { Alert } from "@patternfly/react-core/dist/esm/components/Alert/index.js";
import { Wizard, WizardStep } from "@patternfly/react-core/dist/esm/components/Wizard/index.js";
import { Radio } from "@patternfly/react-core/dist/esm/components/Radio/index.js";

import { createMachine, listTemplates, getTemplateContent } from '../api/lxspawn';
import type { Template } from '../api/types';

const _ = cockpit.gettext;

interface CreateMachineWizardProps {
    onComplete: () => void;
    onCancel: () => void;
}

export const CreateMachineWizard: React.FC<CreateMachineWizardProps> = ({ onComplete, onCancel }) => {
    const [templates, setTemplates] = useState<Template[]>([]);
    const [selectedTemplate, setSelectedTemplate] = useState<string>('');
    const [templateContent, setTemplateContent] = useState<string>('');
    const [machineName, setMachineName] = useState('');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState(false);

    React.useEffect(() => {
        listTemplates().then(t => setTemplates(t)).catch(console.error);
    }, []);

    const handleTemplateSelect = async (templateName: string) => {
        setSelectedTemplate(templateName);
        try {
            const content = await getTemplateContent(templateName);
            setTemplateContent(content);
            // Extract default name from template (remove -template suffix if present)
            const defaultName = templateName.replace(/-template$/, '');
            setMachineName(defaultName);
        } catch (err) {
            console.error("Failed to load template:", err);
        }
    };

    const generateFinalYAML = (): string => {
        if (!templateContent) return '';
        
        // Replace the name field in the YAML
        // This is a simple replacement - in a real implementation, you'd want proper YAML parsing
        const lines = templateContent.split('\n');
        const newLines = lines.map(line => {
            if (line.match(/^name:\s*/)) {
                return `name: ${machineName}`;
            }
            return line;
        });
        
        return newLines.join('\n');
    };

    const handleSave = async () => {
        try {
            setLoading(true);
            setError(null);
            await createMachine(machineName, generateFinalYAML());
            setSuccess(true);
            setTimeout(onComplete, 2000);
        } catch (err: any) {
            setError(err.message || _("Failed to create machine"));
        } finally {
            setLoading(false);
        }
    };

    const isNameValid = !machineName || /^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]$/.test(machineName);

    return (
        <Page>
            <PageSection>
                <Card>
                    <CardHeader>
                        <CardTitle>{_("Create Machine from Template")}</CardTitle>
                    </CardHeader>
                    <CardBody>
                        <Wizard onClose={onCancel} onSave={handleSave} height={600}>
                            <WizardStep name={_("Select Template")} id="step-template">
                                <FormGroup label={_("Available templates")} isRequired>
                                    <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                                        {templates.map(template => (
                                            <Radio
                                                key={template.name}
                                                id={`template-${template.name}`}
                                                name="template-select"
                                                label={
                                                    <div>
                                                        <strong>{template.name}</strong>
                                                        <div style={{ fontSize: '0.9em', color: 'var(--pf-t--global--color--200)' }}>
                                                            {template.description}
                                                        </div>
                                                    </div>
                                                }
                                                isChecked={selectedTemplate === template.name}
                                                onChange={() => handleTemplateSelect(template.name)}
                                            />
                                        ))}
                                    </div>
                                </FormGroup>
                            </WizardStep>
                            
                            <WizardStep name={_("Configure")} id="step-configure" isDisabled={!selectedTemplate}>
                                <Form>
                                    <FormGroup label={_("Machine name")} isRequired fieldId="machine-name">
                                        <TextInput
                                            isRequired
                                            type="text"
                                            id="machine-name"
                                            value={machineName}
                                            onChange={(_, value) => setMachineName(value)}
                                            placeholder="myserver"
                                            validated={isNameValid ? "default" : "error"}
                                        />
                                        {!isNameValid && (
                                            <Alert variant="danger" isInline title={_("Invalid name")} style={{ marginTop: '8px' }}>
                                                {_("Name must be 2-64 characters, alphanumeric with hyphens/underscores")}
                                            </Alert>
                                        )}
                                    </FormGroup>
                                </Form>
                            </WizardStep>
                            
                            <WizardStep name={_("Review")} id="step-review" isDisabled={!selectedTemplate}>
                                {success ? (
                                    <Alert variant="success" isInline title={_("Machine created successfully")}>
                                        {_("The machine is being created and will be available shortly.")}
                                    </Alert>
                                ) : error ? (
                                    <Alert variant="danger" isInline title={_("Error creating machine")}>
                                        {error}
                                    </Alert>
                                ) : (
                                    <>
                                        <h3>{_("Configuration Summary")}</h3>
                                        <p style={{ marginBottom: '16px' }}>
                                            <strong>{_("Template:")}</strong> {selectedTemplate}
                                        </p>
                                        <pre style={{
                                            backgroundColor: 'var(--pf-t--global--background--color--secondary--default)',
                                            padding: '16px',
                                            borderRadius: '4px',
                                            overflow: 'auto',
                                            maxHeight: '400px'
                                        }}>
                                            {generateFinalYAML()}
                                        </pre>
                                    </>
                                )}
                            </WizardStep>
                        </Wizard>
                    </CardBody>
                </Card>
            </PageSection>
        </Page>
    );
};
