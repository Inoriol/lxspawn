package provision

import (
	"context"
	"fmt"
	"os"
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
		args := []string{"systemd-run", "-M", p.machineName, "--pipe", "--wait", "--", "/bin/bash", "-c", script}
		cmd := deploy.Sudo(args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		ctxTimeout, cancel := context.WithTimeout(context.Background(), defaultProbeTimeout)
		
		err := cmd.Start()
		if err != nil {
			cancel()
			if attempt < maxAttempts-1 {
				time.Sleep(2 * time.Second)
				continue
			}
			return fmt.Errorf("probe failed to start after %d attempts: %w", maxAttempts, err)
		}

		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()

		select {
		case <-ctxTimeout.Done():
			cancel()
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
			if attempt < maxAttempts-1 {
				logger.Warn(fmt.Sprintf("Probe attempt %d timed out after %v, retrying...", attempt+1, defaultProbeTimeout))
				time.Sleep(2 * time.Second)
				continue
			}
		case err := <-done:
			cancel()
			if err == nil {
				return nil
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
