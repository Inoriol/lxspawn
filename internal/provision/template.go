package provision

import (
	"bytes"
	"os"
	"os/user"
	"strings"
	"text/template"
)

type TemplateContext struct {
	Name     string
	Hostname string
	Distro   string
	User     string
	UID      string
	Home     string
	Dir      string
	Param    map[string]string
}

func NewTemplateContext(name, hostname, distro, dir string, param map[string]string) (*TemplateContext, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, err
	}

	return &TemplateContext{
		Name:     name,
		Hostname: hostname,
		Distro:   distro,
		User:     currentUser.Username,
		UID:      currentUser.Uid,
		Home:     currentUser.HomeDir,
		Dir:      dir,
		Param:    param,
	}, nil
}

func ShellEscape(s string) string {
	if s == "" {
		return "''"
	}
	needsQuoting := false
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '/' || c == '.' || c == '=' || c == '@' || c == '+' || c == ',') {
			needsQuoting = true
			break
		}
	}
	if !needsQuoting {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

var templateFuncs = template.FuncMap{
	"shellEscape": ShellEscape,
}

func RenderTemplate(text string, ctx *TemplateContext) (string, error) {
	tmpl, err := template.New("provision").Funcs(templateFuncs).Parse(text)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func RenderTemplateFile(path string, ctx *TemplateContext) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return RenderTemplate(string(data), ctx)
}
