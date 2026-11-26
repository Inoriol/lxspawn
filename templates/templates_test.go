package templates

import (
	"testing"

	"github.com/Inoriol/lxspawn/internal/compiler"
	"github.com/Inoriol/lxspawn/internal/config"
)

func TestEmbeddedTemplatesParseAndValidate(t *testing.T) {
	templates := GetEmbeddedTemplates()
	if len(templates) == 0 {
		t.Fatal("no embedded templates found")
	}

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			cfg, err := config.ParseString(string(tmpl.Content))
			if err != nil {
				t.Fatalf("ParseString failed for template %q: %v", tmpl.Name, err)
			}

			if cfg.Name == "" {
				cfg.Name = "test-" + tmpl.Name
			}
			if cfg.Advanced.Hostname == "" {
				cfg.Advanced.Hostname = cfg.Name
			}

			if err := config.Validate(cfg); err != nil {
				t.Fatalf("Validate failed for template %q: %v", tmpl.Name, err)
			}

			if cfg.Storage.Size == "" {
				t.Errorf("Storage.Size is empty after ParseString for template %q", tmpl.Name)
			}
			if cfg.Security.Seccomp == nil {
				t.Errorf("Security.Seccomp is nil after ParseString for template %q", tmpl.Name)
			}
			if cfg.Security.NoNewPrivileges == nil {
				t.Errorf("Security.NoNewPrivileges is nil after ParseString for template %q", tmpl.Name)
			}
		})
	}
}

func TestEmbeddedTemplatesCompile(t *testing.T) {
	templates := GetEmbeddedTemplates()
	if len(templates) == 0 {
		t.Fatal("no embedded templates found")
	}

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			cfg, err := config.ParseString(string(tmpl.Content))
			if err != nil {
				t.Fatalf("ParseString failed for template %q: %v", tmpl.Name, err)
			}

			if cfg.Name == "" {
				cfg.Name = "test-" + tmpl.Name
			}
			if cfg.Advanced.Hostname == "" {
				cfg.Advanced.Hostname = cfg.Name
			}

			if err := config.Validate(cfg); err != nil {
				t.Fatalf("Validate failed for template %q: %v", tmpl.Name, err)
			}

			// Skip GPU-enabled templates on CI (no GPU available)
			if cfg.GPU.Enabled {
				t.Skipf("Skipping GPU-enabled template %q (no GPU on CI)", tmpl.Name)
			}

			cfg.Network.Mode = config.NetworkHost

			compiled, err := compiler.Compile(cfg)
			if err != nil {
				t.Fatalf("Compile failed for template %q: %v", tmpl.Name, err)
			}

			if compiled.Nspawn == "" {
				t.Errorf("compiled Nspawn is empty for template %q", tmpl.Name)
			}
			if compiled.DropIn == "" {
				t.Errorf("compiled DropIn is empty for template %q", tmpl.Name)
			}
		})
	}
}
