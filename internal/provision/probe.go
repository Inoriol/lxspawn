package provision

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/logger"
)

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
		leaderPID, err := p.getLeaderPID()
		if err != nil {
			if attempt < maxAttempts-1 {
				time.Sleep(2 * time.Second)
				continue
			}
			return fmt.Errorf("failed to get leader PID: %w", err)
		}

		args := []string{"nsenter", "-t", strconv.Itoa(leaderPID), "-m", "-u", "-i", "-n", "-p", "--", "/bin/bash", "-c", script}
		cmd := deploy.Sudo(args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err == nil {
			return nil
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

func (p *ProbeRunner) getLeaderPID() (int, error) {
	systemd, err := deploy.NewSystemdManager()
	if err != nil {
		return 0, fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer systemd.Close()

	leader, err := systemd.GetMachineLeader(p.machineName)
	if err != nil {
		return 0, fmt.Errorf("failed to get leader PID: %w", err)
	}

	return leader, nil
}
