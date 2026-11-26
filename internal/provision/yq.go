package provision

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Inoriol/lxspawn/internal/config"
	"github.com/Inoriol/lxspawn/internal/logger"
	"gopkg.in/yaml.v3"
)

type YQProvisioner struct {
	rootfsPath string
}

func NewYQProvisioner(rootfsPath string) *YQProvisioner {
	return &YQProvisioner{rootfsPath: rootfsPath}
}

func (y *YQProvisioner) Provision(prov config.Provision, ctx *TemplateContext, fileHandler *FileHandler, baseDir string) error {
	relPath := prov.Path
	if !filepath.IsAbs(relPath) {
		return fmt.Errorf("path must be absolute: %s", relPath)
	}

	data, err := rootfsReadFile(y.rootfsPath, relPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", prov.Path, err)
	}

	format := prov.Format
	if format == "" {
		format = detectFormat(prov.Path)
	}

	var content interface{}
	switch format {
	case "json":
		if err := json.Unmarshal(data, &content); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &content); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	rendered, err := RenderTemplate(prov.Expression, ctx)
	if err != nil {
		return fmt.Errorf("failed to render expression: %w", err)
	}

	if err := applyExpression(content, rendered, format); err != nil {
		return fmt.Errorf("failed to apply expression: %w", err)
	}

	var output []byte
	switch format {
	case "json":
		output, err = json.MarshalIndent(content, "", "  ")
	case "yaml", "yml":
		output, err = yaml.Marshal(content)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	if err := rootfsWriteFile(y.rootfsPath, relPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", prov.Path, err)
	}

	logger.Debug(fmt.Sprintf("Modified %s", prov.Path))
	return nil
}

func detectFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	default:
		return "yaml"
	}
}

func applyExpression(content interface{}, expression string, format string) error {
	expression = strings.TrimSpace(expression)

	path, operator, value, err := parseOperator(expression)
	if err != nil {
		return err
	}

	switch operator {
	case "|=":
		return applyUpdateExpression(content, path, value)
	case "+=":
		return applyAppendExpression(content, path, value)
	case "=":
		return applySetExpression(content, path, value)
	default:
		return fmt.Errorf("unsupported operator: %s", operator)
	}
}

func parseOperator(expression string) (path, operator, value string, err error) {
	operators := []string{"|=", "+=", "="}

	for i := 0; i < len(expression); i++ {
		c := expression[i]
		if c == '"' || c == '\'' {
			quote := c
			i++
			for i < len(expression) && expression[i] != quote {
				if expression[i] == '\\' {
					i++
				}
				i++
			}
			continue
		}

		for _, op := range operators {
			if i+len(op) <= len(expression) && expression[i:i+len(op)] == op {
				if op == "=" && i+1 < len(expression) && expression[i+1] == '=' {
					continue
				}
				path = strings.TrimSpace(expression[:i])
				value = strings.TrimSpace(expression[i+len(op):])
				return path, op, value, nil
			}
		}
	}

	return "", "", "", fmt.Errorf("no operator found in expression: %s", expression)
}

func applySetExpression(content interface{}, path string, value string) error {
	path = strings.TrimPrefix(path, ".")
	if path == "" {
		return fmt.Errorf("empty path")
	}
	components := strings.Split(path, ".")

	current, ok := content.(map[string]interface{})
	if !ok {
		return fmt.Errorf("content is not a map")
	}

	for i := 0; i < len(components)-1; i++ {
		key := components[i]
		if _, exists := current[key]; !exists {
			current[key] = make(map[string]interface{})
		}
		next, ok := current[key].(map[string]interface{})
		if !ok {
			return fmt.Errorf("path component %s is not a map", key)
		}
		current = next
	}

	finalKey := components[len(components)-1]
	current[finalKey] = parseValue(value)

	return nil
}

func applyUpdateExpression(content interface{}, path string, updateExpr string) error {
	path = strings.TrimPrefix(path, ".")
	if path == "" {
		return fmt.Errorf("empty path")
	}
	components := strings.Split(path, ".")

	current, ok := content.(map[string]interface{})
	if !ok {
		return fmt.Errorf("content is not a map")
	}

	for i := 0; i < len(components)-1; i++ {
		key := components[i]
		if _, exists := current[key]; !exists {
			current[key] = make(map[string]interface{})
		}
		next, ok := current[key].(map[string]interface{})
		if !ok {
			return fmt.Errorf("path component %s is not a map", key)
		}
		current = next
	}

	finalKey := components[len(components)-1]
	currentValue := current[finalKey]

	if !strings.HasPrefix(updateExpr, ". +") {
		return fmt.Errorf("unsupported update expression (only '. +' is supported): %s", updateExpr)
	}

	valuePart := strings.TrimSpace(strings.TrimPrefix(updateExpr, ". +"))
	newValue := parseValue(valuePart)

	if arr, ok := currentValue.([]interface{}); ok {
		if newArr, ok := newValue.([]interface{}); ok {
			current[finalKey] = append(arr, newArr...)
		} else {
			current[finalKey] = append(arr, newValue)
		}
	} else if currentValue == nil {
		if newArr, ok := newValue.([]interface{}); ok {
			current[finalKey] = newArr
		} else {
			current[finalKey] = []interface{}{newValue}
		}
	} else {
		return fmt.Errorf("cannot update non-array value with +=")
	}

	return nil
}

func applyAppendExpression(content interface{}, path string, value string) error {
	path = strings.TrimPrefix(path, ".")
	if path == "" {
		return fmt.Errorf("empty path")
	}
	components := strings.Split(path, ".")

	current, ok := content.(map[string]interface{})
	if !ok {
		return fmt.Errorf("content is not a map")
	}

	for i := 0; i < len(components)-1; i++ {
		key := components[i]
		if _, exists := current[key]; !exists {
			current[key] = make(map[string]interface{})
		}
		next, ok := current[key].(map[string]interface{})
		if !ok {
			return fmt.Errorf("path component %s is not a map", key)
		}
		current = next
	}

	finalKey := components[len(components)-1]
	currentValue := current[finalKey]
	newValue := parseValue(value)

	if arr, ok := currentValue.([]interface{}); ok {
		if newArr, ok := newValue.([]interface{}); ok {
			current[finalKey] = append(arr, newArr...)
		} else {
			current[finalKey] = append(arr, newValue)
		}
	} else if currentValue == nil {
		if newArr, ok := newValue.([]interface{}); ok {
			current[finalKey] = newArr
		} else {
			current[finalKey] = []interface{}{newValue}
		}
	} else {
		return fmt.Errorf("cannot append to non-array value")
	}

	return nil
}

func parseValue(value string) interface{} {
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}
	if value == "null" {
		return nil
	}

	if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
		return value[1 : len(value)-1]
	}

	if !strings.Contains(value, ".") {
		var num int64
		if _, err := fmt.Sscanf(value, "%d", &num); err == nil {
			if fmt.Sprintf("%d", num) == value {
				return num
			}
		}
	}

	if strings.Count(value, ".") == 1 {
		var num float64
		if _, err := fmt.Sscanf(value, "%f", &num); err == nil {
			if fmt.Sprintf("%g", num) == value || fmt.Sprintf("%f", num) == value {
				return num
			}
		}
	}

	if strings.HasPrefix(value, "[") || strings.HasPrefix(value, "{") {
		var parsed interface{}
		if err := json.Unmarshal([]byte(value), &parsed); err == nil {
			return parsed
		}
	}

	return value
}
