package provision

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Inoriol/lxspawn/internal/config"
	"gopkg.in/yaml.v3"
)

func TestDetectFormatJSON(t *testing.T) {
	if got := detectFormat("/etc/docker/daemon.json"); got != "json" {
		t.Errorf("got %q, want %q", got, "json")
	}
}

func TestDetectFormatYAML(t *testing.T) {
	if got := detectFormat("/etc/config.yaml"); got != "yaml" {
		t.Errorf("got %q, want %q", got, "yaml")
	}
}

func TestDetectFormatYML(t *testing.T) {
	if got := detectFormat("/etc/config.yml"); got != "yaml" {
		t.Errorf("got %q, want %q", got, "yaml")
	}
}

func TestDetectFormatDefault(t *testing.T) {
	if got := detectFormat("/etc/config"); got != "yaml" {
		t.Errorf("got %q, want %q (default)", got, "yaml")
	}
}

func TestDetectFormatCaseInsensitive(t *testing.T) {
	if got := detectFormat("/etc/config.JSON"); got != "json" {
		t.Errorf("got %q, want %q", got, "json")
	}
	if got := detectFormat("/etc/config.YAML"); got != "yaml" {
		t.Errorf("got %q, want %q", got, "yaml")
	}
}

func TestParseValueString(t *testing.T) {
	if got := parseValue("hello"); got != "hello" {
		t.Errorf("got %v (%T), want %q", got, got, "hello")
	}
}

func TestParseValueQuotedString(t *testing.T) {
	if got := parseValue(`"hello"`); got != "hello" {
		t.Errorf("got %v (%T), want %q", got, got, "hello")
	}
}

func TestParseValueSingleQuoted(t *testing.T) {
	if got := parseValue("'hello'"); got != "hello" {
		t.Errorf("got %v (%T), want %q", got, got, "hello")
	}
}

func TestParseValueTrue(t *testing.T) {
	if got := parseValue("true"); got != true {
		t.Errorf("got %v (%T), want true", got, got)
	}
}

func TestParseValueFalse(t *testing.T) {
	if got := parseValue("false"); got != false {
		t.Errorf("got %v (%T), want false", got, got)
	}
}

func TestParseValueNull(t *testing.T) {
	if got := parseValue("null"); got != nil {
		t.Errorf("got %v (%T), want nil", got, got)
	}
}

func TestParseValueInteger(t *testing.T) {
	got := parseValue("42")
	num, ok := got.(int64)
	if !ok {
		t.Fatalf("got %T, want int64", got)
	}
	if num != 42 {
		t.Errorf("got %v, want 42", num)
	}
}

func TestParseValueFloat(t *testing.T) {
	got := parseValue("3.14")
	num, ok := got.(float64)
	if !ok {
		t.Fatalf("got %T, want float64", got)
	}
	if num != 3.14 {
		t.Errorf("got %v, want 3.14", num)
	}
}

func TestParseValueJSONArray(t *testing.T) {
	got := parseValue(`["a","b","c"]`)
	arr, ok := got.([]interface{})
	if !ok {
		t.Fatalf("got %T, want []interface{}", got)
	}
	if len(arr) != 3 {
		t.Errorf("len = %d, want 3", len(arr))
	}
}

func TestParseValueJSONObject(t *testing.T) {
	got := parseValue(`{"key":"value"}`)
	obj, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("got %T, want map[string]interface{}", got)
	}
	if obj["key"] != "value" {
		t.Errorf("key = %v, want %q", obj["key"], "value")
	}
}

func TestParseValueIPAddress(t *testing.T) {
	// IP addresses should be returned as strings, not parsed as numbers
	got := parseValue("10.0.0.1")
	str, ok := got.(string)
	if !ok {
		t.Fatalf("got %T (%v), want string", got, got)
	}
	if str != "10.0.0.1" {
		t.Errorf("got %q, want %q", str, "10.0.0.1")
	}
}

func TestParseValueVersionString(t *testing.T) {
	// Version strings like "1.2.3" should be returned as strings
	got := parseValue("1.2.3")
	str, ok := got.(string)
	if !ok {
		t.Fatalf("got %T (%v), want string", got, got)
	}
	if str != "1.2.3" {
		t.Errorf("got %q, want %q", str, "1.2.3")
	}
}

func TestParseOperator(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		wantPath   string
		wantOp     string
		wantValue  string
		wantErr    bool
	}{
		{
			name:       "simple set",
			expression: ".key = value",
			wantPath:   ".key",
			wantOp:     "=",
			wantValue:  "value",
		},
		{
			name:       "update operator",
			expression: ".arr |= . + [1,2]",
			wantPath:   ".arr",
			wantOp:     "|=",
			wantValue:  ". + [1,2]",
		},
		{
			name:       "append operator",
			expression: ".arr += item",
			wantPath:   ".arr",
			wantOp:     "+=",
			wantValue:  "item",
		},
		{
			name:       "value contains operator",
			expression: `.key = "x|=y"`,
			wantPath:   ".key",
			wantOp:     "=",
			wantValue:  `"x|=y"`,
		},
		{
			name:       "no operator",
			expression: "just a string",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, op, value, err := parseOperator(tt.expression)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOperator() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if path != tt.wantPath {
					t.Errorf("path = %q, want %q", path, tt.wantPath)
				}
				if op != tt.wantOp {
					t.Errorf("op = %q, want %q", op, tt.wantOp)
				}
				if value != tt.wantValue {
					t.Errorf("value = %q, want %q", value, tt.wantValue)
				}
			}
		})
	}
}

func TestApplyExpressionSimpleKey(t *testing.T) {
	content := map[string]interface{}{
		"existing": "value",
	}

	if err := applyExpression(content, ".newkey = hello", "yaml"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if content["newkey"] != "hello" {
		t.Errorf("newkey = %v, want %q", content["newkey"], "hello")
	}
	if content["existing"] != "value" {
		t.Error("existing key was modified")
	}
}

func TestApplyExpressionNestedKey(t *testing.T) {
	content := map[string]interface{}{
		"server": map[string]interface{}{
			"host": "localhost",
		},
	}

	if err := applyExpression(content, ".server.port = 8080", "yaml"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	server := content["server"].(map[string]interface{})
	if server["port"] != int64(8080) {
		t.Errorf("port = %v (%T), want int64(8080)", server["port"], server["port"])
	}
	if server["host"] != "localhost" {
		t.Error("host was modified")
	}
}

func TestApplyExpressionCreateNestedPath(t *testing.T) {
	content := map[string]interface{}{}

	if err := applyExpression(content, ".a.b.c = deep", "yaml"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	a := content["a"].(map[string]interface{})
	b := a["b"].(map[string]interface{})
	if b["c"] != "deep" {
		t.Errorf("c = %v, want %q", b["c"], "deep")
	}
}

func TestApplyExpressionOverwrite(t *testing.T) {
	content := map[string]interface{}{
		"key": "old",
	}

	if err := applyExpression(content, ".key = new", "yaml"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if content["key"] != "new" {
		t.Errorf("key = %v, want %q", content["key"], "new")
	}
}

func TestApplyExpressionInvalidFormat(t *testing.T) {
	content := map[string]interface{}{}

	if err := applyExpression(content, "no-equals-sign", "yaml"); err == nil {
		t.Fatal("expected error for invalid expression")
	}
}

func TestApplyExpressionNonMapContent(t *testing.T) {
	content := "not a map"

	if err := applyExpression(content, ".key = value", "yaml"); err == nil {
		t.Fatal("expected error for non-map content")
	}
}

func TestYQProvisionJSON(t *testing.T) {
	rootfs := t.TempDir()
	configDir := filepath.Join(rootfs, "etc", "docker")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(configDir, "daemon.json")
	original := map[string]interface{}{
		"storage-driver": "overlay2",
	}
	data, _ := json.MarshalIndent(original, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	provisioner := NewYQProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:       config.ProvisionYQ,
		Path:       "/etc/docker/daemon.json",
		Expression: `.log-level = debug`,
		Format:     "json",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	result, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if parsed["log-level"] != "debug" {
		t.Errorf("log-level = %v, want %q", parsed["log-level"], "debug")
	}
	if parsed["storage-driver"] != "overlay2" {
		t.Error("existing key was modified")
	}
}

func TestYQProvisionYAML(t *testing.T) {
	rootfs := t.TempDir()
	configDir := filepath.Join(rootfs, "etc", "myapp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	original := "host: localhost\nport: 80\n"
	if err := os.WriteFile(configPath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	provisioner := NewYQProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:       config.ProvisionYQ,
		Path:       "/etc/myapp/config.yaml",
		Expression: `.port = 8080`,
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	result, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("result is not valid YAML: %v", err)
	}

	if parsed["port"] != 8080 {
		t.Errorf("port = %v, want 8080", parsed["port"])
	}
	if parsed["host"] != "localhost" {
		t.Error("existing key was modified")
	}
}

func TestYQProvisionFileNotFound(t *testing.T) {
	rootfs := t.TempDir()
	provisioner := NewYQProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:       config.ProvisionYQ,
		Path:       "/nonexistent/file.json",
		Expression: `.key = value`,
		Format:     "json",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestYQProvisionUnsupportedFormat(t *testing.T) {
	rootfs := t.TempDir()
	configPath := filepath.Join(rootfs, "config.toml")
	if err := os.WriteFile(configPath, []byte("key = value"), 0644); err != nil {
		t.Fatal(err)
	}

	provisioner := NewYQProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:       config.ProvisionYQ,
		Path:       "/config.toml",
		Expression: `.key = new`,
		Format:     "toml",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

func TestYQProvisionWithTemplateExpression(t *testing.T) {
	rootfs := t.TempDir()
	configDir := filepath.Join(rootfs, "etc")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	provisioner := NewYQProvisioner(rootfs)
	ctx := testContext()
	fh := &FileHandler{CacheDir: t.TempDir()}

	prov := config.Provision{
		Mode:       config.ProvisionYQ,
		Path:       "/etc/config.json",
		Expression: `.port = {{.Param.port}}`,
		Format:     "json",
	}

	if err := provisioner.Provision(prov, ctx, fh, t.TempDir()); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	result, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if parsed["port"] != float64(8080) {
		t.Errorf("port = %v, want 8080", parsed["port"])
	}
}
