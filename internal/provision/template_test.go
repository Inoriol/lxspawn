package provision

import (
	"strings"
	"testing"
)

func testContext() *TemplateContext {
	return &TemplateContext{
		Name:     "myserver",
		Hostname: "myhost",
		Distro:   "debian:bookworm",
		User:     "testuser",
		UID:      "1000",
		Home:     "/home/testuser",
		Dir:      "/home/testuser/projects",
		Param: map[string]string{
			"port": "8080",
			"env":  "production",
		},
	}
}

func TestRenderTemplateName(t *testing.T) {
	ctx := testContext()
	got, err := RenderTemplate("name={{.Name}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "name=myserver" {
		t.Errorf("got %q, want %q", got, "name=myserver")
	}
}

func TestRenderTemplateHostname(t *testing.T) {
	ctx := testContext()
	got, err := RenderTemplate("host={{.Hostname}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "host=myhost" {
		t.Errorf("got %q, want %q", got, "host=myhost")
	}
}

func TestRenderTemplateDistro(t *testing.T) {
	ctx := testContext()
	got, err := RenderTemplate("distro={{.Distro}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "distro=debian:bookworm" {
		t.Errorf("got %q, want %q", got, "distro=debian:bookworm")
	}
}

func TestRenderTemplateUser(t *testing.T) {
	ctx := testContext()
	got, err := RenderTemplate("user={{.User}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "user=testuser" {
		t.Errorf("got %q, want %q", got, "user=testuser")
	}
}

func TestRenderTemplateUID(t *testing.T) {
	ctx := testContext()
	got, err := RenderTemplate("uid={{.UID}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "uid=1000" {
		t.Errorf("got %q, want %q", got, "uid=1000")
	}
}

func TestRenderTemplateHome(t *testing.T) {
	ctx := testContext()
	got, err := RenderTemplate("home={{.Home}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "home=/home/testuser" {
		t.Errorf("got %q, want %q", got, "home=/home/testuser")
	}
}

func TestRenderTemplateDir(t *testing.T) {
	ctx := testContext()
	got, err := RenderTemplate("dir={{.Dir}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "dir=/home/testuser/projects" {
		t.Errorf("got %q, want %q", got, "dir=/home/testuser/projects")
	}
}

func TestRenderTemplateParam(t *testing.T) {
	ctx := testContext()
	got, err := RenderTemplate("port={{.Param.port}} env={{.Param.env}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "port=8080 env=production" {
		t.Errorf("got %q, want %q", got, "port=8080 env=production")
	}
}

func TestRenderTemplateMultipleVars(t *testing.T) {
	ctx := testContext()
	got, err := RenderTemplate("{{.Name}}:{{.Distro}}:{{.Param.port}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "myserver:debian:bookworm:8080" {
		t.Errorf("got %q", got)
	}
}

func TestRenderTemplateMultiline(t *testing.T) {
	ctx := testContext()
	input := `server {
    listen {{.Param.port}};
    server_name {{.Name}};
    root /var/www/{{.Name}};
}`
	got, err := RenderTemplate(input, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "listen 8080") {
		t.Error("missing rendered port")
	}
	if !strings.Contains(got, "server_name myserver") {
		t.Error("missing rendered name")
	}
}

func TestRenderTemplateNoVars(t *testing.T) {
	ctx := testContext()
	got, err := RenderTemplate("no variables here", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "no variables here" {
		t.Errorf("got %q", got)
	}
}

func TestRenderTemplateEmpty(t *testing.T) {
	ctx := testContext()
	got, err := RenderTemplate("", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestRenderTemplateInvalidTemplate(t *testing.T) {
	ctx := testContext()
	_, err := RenderTemplate("{{.Invalid", ctx)
	if err == nil {
		t.Fatal("expected error for invalid template")
	}
}

func TestRenderTemplateMissingParam(t *testing.T) {
	ctx := testContext()
	_, err := RenderTemplate("{{.Param.nonexistent}}", ctx)
	if err == nil {
		t.Fatal("expected error for missing param")
	}
	if !strings.Contains(err.Error(), "map has no entry") {
		t.Errorf("expected map has no entry error, got: %v", err)
	}
}

func TestRenderTemplateNilParam(t *testing.T) {
	ctx := &TemplateContext{
		Name:  "test",
		Param: nil,
	}
	_, err := RenderTemplate("{{.Param.port}}", ctx)
	if err == nil {
		t.Fatal("expected error for nil param")
	}
}

func TestRenderTemplateSpecialChars(t *testing.T) {
	ctx := testContext()
	got, err := RenderTemplate("echo '{{.Name}}'", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "echo 'myserver'" {
		t.Errorf("got %q", got)
	}
}

func TestRenderTemplateParamWithSpaces(t *testing.T) {
	ctx := &TemplateContext{
		Name: "test",
		Param: map[string]string{
			"greeting": "hello world",
		},
	}
	got, err := RenderTemplate("{{.Param.greeting}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello world" {
		t.Errorf("got %q", got)
	}
}

func TestShellEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", "''"},
		{"simple alphanumeric", "hello", "hello"},
		{"with spaces", "hello world", "'hello world'"},
		{"with single quote", "it's", "'it'\"'\"'s'"},
		{"with semicolon", "cmd1; cmd2", "'cmd1; cmd2'"},
		{"with dollar sign", "$HOME", "'$HOME'"},
		{"with backtick", "test`cmd`", "'test`cmd`'"},
		{"path is safe", "/usr/bin/test", "/usr/bin/test"},
		{"equals is safe", "KEY=value", "KEY=value"},
		{"with pipe", "a | b", "'a | b'"},
		{"with newline", "line1\nline2", "'line1\nline2'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShellEscape(tt.input)
			if result != tt.expected {
				t.Errorf("ShellEscape(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRenderTemplateShellEscapeFunc(t *testing.T) {
	ctx := &TemplateContext{
		Name: "test",
		Param: map[string]string{
			"value": "hello world",
		},
	}
	got, err := RenderTemplate(`echo {{shellEscape .Param.value}}`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "echo 'hello world'" {
		t.Errorf("got %q, want %q", got, "echo 'hello world'")
	}
}

func TestRenderTemplateShellEscapeFuncSafe(t *testing.T) {
	ctx := &TemplateContext{
		Name: "test",
		Param: map[string]string{
			"value": "simple",
		},
	}
	got, err := RenderTemplate(`echo {{shellEscape .Param.value}}`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "echo simple" {
		t.Errorf("got %q, want %q", got, "echo simple")
	}
}

func TestIsValidEnvVarName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid simple", "port", true},
		{"valid underscore", "my_var", true},
		{"valid uppercase", "MY_VAR", true},
		{"valid with numbers", "var123", true},
		{"valid underscore prefix", "_var", true},
		{"empty", "", false},
		{"starts with number", "123var", false},
		{"contains dash", "my-var", false},
		{"contains dot", "my.var", false},
		{"contains space", "my var", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidEnvVarName(tt.input)
			if got != tt.want {
				t.Errorf("isValidEnvVarName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
