package provision

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/logger"
)

type Engine struct {
	machineName string
	rootfsPath  string
	statePath   string
	fileHandler *FileHandler
}

type ProvisionState struct {
	LastRun        string            `json:"last_run"`
	PhasesComplete []string          `json:"phases_completed"`
	ProbesPassed   []string          `json:"probes_passed"`
	ScriptHashes   map[string]string `json:"script_hashes"`
}

func NewEngine(machineName string) (*Engine, error) {
	rootfsPath := deploy.MachineRootfsPath(machineName)
	lxspawnDir := deploy.LxspawnMachineDir(machineName)
	statePath := filepath.Join(lxspawnDir, "provision-state.json")

	fileHandler, err := NewFileHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to create file handler: %w", err)
	}

	return &Engine{
		machineName: machineName,
		rootfsPath:  rootfsPath,
		statePath:   statePath,
		fileHandler: fileHandler,
	}, nil
}

func (e *Engine) Provision(cfg *config.Config, sourcePath string) error {
	logger.Action("Starting provisioning")

	// Create template context
	ctx, err := NewTemplateContext(
		cfg.Name,
		cfg.Advanced.Hostname,
		cfg.Distro,
		filepath.Dir(sourcePath),
		cfg.Param,
	)
	if err != nil {
		return fmt.Errorf("failed to create template context: %w", err)
	}

	state := &ProvisionState{
		PhasesComplete: []string{},
		ProbesPassed:   []string{},
		ScriptHashes:   make(map[string]string),
	}

	// Pre-boot phase: boot, data, yq
	for i, prov := range cfg.Provision {
		switch prov.Mode {
		case config.ProvisionBoot:
			provisioner := NewBootProvisioner(e.rootfsPath)
			if err := provisioner.Provision(prov, ctx, e.fileHandler, filepath.Dir(sourcePath)); err != nil {
				return fmt.Errorf("boot provision[%d] failed: %w", i, err)
			}
		case config.ProvisionData:
			provisioner := NewDataProvisioner(e.rootfsPath)
			if err := provisioner.Provision(prov, ctx, e.fileHandler, filepath.Dir(sourcePath)); err != nil {
				return fmt.Errorf("data provision[%d] failed: %w", i, err)
			}
		case config.ProvisionYQ:
			provisioner := NewYQProvisioner(e.rootfsPath)
			if err := provisioner.Provision(prov, ctx, e.fileHandler, filepath.Dir(sourcePath)); err != nil {
				return fmt.Errorf("yq provision[%d] failed: %w", i, err)
			}
		}
	}

	state.PhasesComplete = append(state.PhasesComplete, "pre-boot")
	if err := e.saveState(state); err != nil {
		logger.Warn(fmt.Sprintf("failed to save state: %v", err))
	}

	logger.Success("Pre-boot provisioning complete")
	logger.Info("Start the machine to continue with post-boot provisioning")

	return nil
}

func (e *Engine) ProvisionPostBoot(cfg *config.Config, sourcePath string) error {
	logger.Action("Starting post-boot provisioning")

	ctx, err := NewTemplateContext(
		cfg.Name,
		cfg.Advanced.Hostname,
		cfg.Distro,
		filepath.Dir(sourcePath),
		cfg.Param,
	)
	if err != nil {
		return fmt.Errorf("failed to create template context: %w", err)
	}

	state, err := e.loadState()
	if err != nil {
		state = &ProvisionState{
			PhasesComplete: []string{},
			ProbesPassed:   []string{},
			ScriptHashes:   make(map[string]string),
		}
	}

	for i, prov := range cfg.Provision {
		switch prov.Mode {
		case config.ProvisionSystem, config.ProvisionUser:
			script := prov.Script
			if prov.File != nil {
				resolvedPath, err := e.fileHandler.ResolveFileRef(prov.File.URL, prov.File.Digest, filepath.Dir(sourcePath))
				if err != nil {
					return fmt.Errorf("failed to resolve file: %w", err)
				}
				data, err := os.ReadFile(resolvedPath)
				if err != nil {
					return fmt.Errorf("failed to read file: %w", err)
				}
				script = string(data)
			}

			rendered, err := RenderTemplate(script, ctx)
			if err != nil {
				return fmt.Errorf("failed to render template: %w", err)
			}

			scriptKey := fmt.Sprintf("%s:%d", prov.Mode, i)
			currentHash := computeHash(rendered)
			
			if storedHash, exists := state.ScriptHashes[scriptKey]; exists && storedHash == currentHash {
				logger.Info(fmt.Sprintf("Skipping %s provision[%d] (unchanged)", prov.Mode, i))
				continue
			}

			provisioner := NewExecProvisioner(e.machineName)
			provCopy := prov
			provCopy.Script = rendered
			if err := provisioner.Provision(provCopy, ctx, e.fileHandler, filepath.Dir(sourcePath)); err != nil {
				return fmt.Errorf("%s provision[%d] failed: %w", prov.Mode, i, err)
			}

			state.ScriptHashes[scriptKey] = currentHash
		}
	}

	state.PhasesComplete = append(state.PhasesComplete, "post-boot")

	if len(cfg.Probes) > 0 {
		probeRunner := NewProbeRunner(e.machineName)
		passedProbes, err := probeRunner.RunProbes(cfg.Probes, ctx)
		if err != nil {
			return fmt.Errorf("readiness probes failed: %w", err)
		}
		state.ProbesPassed = passedProbes
		state.PhasesComplete = append(state.PhasesComplete, "probes")
	}

	state.LastRun = time.Now().UTC().Format(time.RFC3339)
	if err := e.saveState(state); err != nil {
		logger.Warn(fmt.Sprintf("failed to save state: %v", err))
	}

	logger.Success("Post-boot provisioning complete")
	return nil
}

func computeHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(hash[:])
}

func (e *Engine) loadState() (*ProvisionState, error) {
	data, err := os.ReadFile(e.statePath)
	if err != nil {
		return nil, err
	}

	var state ProvisionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func (e *Engine) saveState(state *ProvisionState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return deploy.WriteFileAtomic(e.statePath, data, 0600)
}

func (e *Engine) GetState() (*ProvisionState, error) {
	return e.loadState()
}
