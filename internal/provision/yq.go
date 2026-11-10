package provision

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

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
	targetPath, err := validatePath(y.rootfsPath, prov.Path)
	if err != nil {
		return fmt.Errorf("invalid path %q: %w", prov.Path, err)
	}

	data, err := os.ReadFile(targetPath)
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

	f, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_TRUNC|syscall.O_NOFOLLOW, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", prov.Path, err)
	}
	if _, err := f.Write(output); err != nil {
		f.Close()
		return fmt.Errorf("failed to write file %s: %w", prov.Path, err)
	}
	f.Close()

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
		return "yaml" // Default to YAML
	}
}

func applyExpression(content interface{}, expression string, format string) error {
	expression = strings.TrimSpace(expression)
	
	// Parse operators more carefully to avoid false matches
	// Order matters: check |= and += before = 
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

// parseOperator extracts the path, operator, and value from an expression
func parseOperator(expression string) (path, operator, value string, err error) {
	// Find the first unquoted operator
	operators := []string{"|=", "+=", "="}
	
	for i := 0; i < len(expression); i++ {
		c := expression[i]
		// Skip quoted strings
		if c == '"' || c == '\'' {
			quote := c
			i++
			for i < len(expression) && expression[i] != quote {
				if expression[i] == '\\' {
					i++ // Skip escaped character
				}
				i++
			}
			continue
		}
		
		// Check for operators at this position
		for _, op := range operators {
			if i+len(op) <= len(expression) && expression[i:i+len(op)] == op {
				// Make sure we don't match == as =
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

	// Only support ". +" syntax for update
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
	// Try to parse as different types
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}
	if value == "null" {
		return nil
	}

	// Remove quotes if present (check before number parsing)
	if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
		return value[1 : len(value)-1]
	}

	// Try as integer first (no decimal point)
	if !strings.Contains(value, ".") {
		var num int64
		if _, err := fmt.Sscanf(value, "%d", &num); err == nil {
			// Verify the entire string was consumed
			if fmt.Sprintf("%d", num) == value {
				return num
			}
		}
	}

	// Try as float (must have exactly one decimal point and be a valid number)
	if strings.Count(value, ".") == 1 {
		var num float64
		if _, err := fmt.Sscanf(value, "%f", &num); err == nil {
			// Verify it's not something like "10.0.0.1" (IP address)
			// by checking that the string representation matches
			if fmt.Sprintf("%g", num) == value || fmt.Sprintf("%f", num) == value {
				return num
			}
		}
	}

	// Try as JSON array/object
	if strings.HasPrefix(value, "[") || strings.HasPrefix(value, "{") {
		var parsed interface{}
		if err := json.Unmarshal([]byte(value), &parsed); err == nil {
			return parsed
		}
	}

	// Return as string
	return value
}
