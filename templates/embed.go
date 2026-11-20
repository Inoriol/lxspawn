package templates

import (
	"embed"
	"strings"
)

//go:embed *.yaml
var EmbeddedFS embed.FS

type TemplateInfo struct {
	Name    string
	Content []byte
}

func GetEmbeddedTemplates() []TemplateInfo {
	var templates []TemplateInfo

	entries, err := EmbeddedFS.ReadDir(".")
	if err != nil {
		return templates
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		content, err := EmbeddedFS.ReadFile(entry.Name())
		if err != nil {
			continue
		}

		templates = append(templates, TemplateInfo{
			Name:    name,
			Content: content,
		})
	}

	return templates
}

func GetEmbeddedTemplateContent(name string) ([]byte, bool) {
	path := name + ".yaml"
	content, err := EmbeddedFS.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return content, true
}

func ListEmbeddedTemplateNames() []string {
	var names []string
	entries, err := EmbeddedFS.ReadDir(".")
	if err != nil {
		return names
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".yaml"))
	}
	return names
}
