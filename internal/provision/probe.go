package provision

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/logger"
)

const defaultProbeTimeout = 10 * time.Second

type ProbeRunner struct {
	machineName string
}

func NewProbeRunner(machineName string) *ProbeRunner {
	return &ProbeRunner{machineName: machineName}
}

func (p *ProbeRunner) RunProbes(probes []config.Probe, ctx *TemplateContext) ([]string, error) {
	var passedProbes []string
	
	for i, probe := range probes {
		if probe.Mode != "readiness" {
			continue
		}

		rendered, err := RenderTemplate(probe.Script, ctx)
		if err != nil {
			return passedProbes, fmt.Errorf("probe[%d]: failed to render template: %w", i, err)
		}

		logger.Action(fmt.Sprintf("Waiting for %s", probe.Description))

		if err := p.runProbe(rendered, probe.Hint, ctx); err != nil {
			return passedProbes, fmt.Errorf("probe[%d] failed: %w", i, err)
		}

		passedProbes = append(passedProbes, probe.Description)
		logger.Success(fmt.Sprintf("Probe passed: %s", probe.Description))
	}

	return passedProbes, nil
}

func (p *ProbeRunner) runProbe(script string, hint string, ctx *TemplateContext) error {
	maxAttempts := 30
	for attempt := 0; attempt < maxAttempts; attempt++ {
		timeoutSec := int(defaultProbeTimeout.Seconds())
		args := []string{
			"systemd-run", "-M", p.machineName, "--pipe", "--wait",
			fmt.Sprintf("--property=RuntimeMaxSec=%d", timeoutSec),
			"--", "/bin/bash", "-s",
		}
		cmd := deploy.Sudo(args...)
		cmd.Stdin = strings.NewReader(script)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err == nil {
			return nil
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			if exitCode == 137 || exitCode == 143 {
				if attempt < maxAttempts-1 {
					logger.Warn(fmt.Sprintf("Probe attempt %d timed out after %v, retrying...", attempt+1, defaultProbeTimeout))
					time.Sleep(2 * time.Second)
					continue
				}
			}
		}

		if attempt < maxAttempts-1 {
			time.Sleep(2 * time.Second)
		}
	}

	if hint != "" {
		renderedHint, _ := RenderTemplate(hint, ctx)
		logger.Warn(fmt.Sprintf("Hint: %s", renderedHint))
	}

	return fmt.Errorf("probe timed out after %d attempts", maxAttempts)
}
