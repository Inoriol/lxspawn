package templates

import (
	"testing"
)

func TestListEmbeddedTemplateNames(t *testing.T) {
	names := ListEmbeddedTemplateNames()
	if len(names) == 0 {
		t.Fatal("expected at least one embedded template, got none")
	}

	expected := []string{"debian-base", "ubuntu-base", "fedora-base", "docker-host", "pihole", "adguard", "jellyfin", "plex", "samba", "nextcloud"}
	for _, exp := range expected {
		found := false
		for _, name := range names {
			if name == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected template %q not found in embedded templates", exp)
		}
	}
}

func TestGetEmbeddedTemplateContent(t *testing.T) {
	content, ok := GetEmbeddedTemplateContent("debian-base")
	if !ok {
		t.Fatal("expected debian-base template to exist")
	}
	if len(content) == 0 {
		t.Fatal("expected debian-base template to have content")
	}
}

func TestGetEmbeddedTemplateContentNotFound(t *testing.T) {
	_, ok := GetEmbeddedTemplateContent("nonexistent-template")
	if ok {
		t.Fatal("expected nonexistent template to not be found")
	}
}

func TestGetEmbeddedTemplates(t *testing.T) {
	templates := GetEmbeddedTemplates()
	if len(templates) == 0 {
		t.Fatal("expected at least one embedded template")
	}

	for _, tmpl := range templates {
		if tmpl.Name == "" {
			t.Error("template has empty name")
		}
		if len(tmpl.Content) == 0 {
			t.Errorf("template %q has empty content", tmpl.Name)
		}
	}
}
