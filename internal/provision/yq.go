package provision

import (
	"encoding/json"
	"fmt"
	"os"
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
	targetPath, err := validatePath(y.rootfsPath, prov.Path)
	if err != nil {
		return fmt.Errorf("invalid path %q: %w", prov.Path, err)
	}

	// Read existing file
	data, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", prov.Path, err)
	}

	// Determine format
	format := prov.Format
	if format == "" {
		format = detectFormat(prov.Path)
	}

	// Parse the file
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

	// Apply expression (simplified - supports basic path assignments)
	rendered, err := RenderTemplate(prov.Expression, ctx)
	if err != nil {
		return fmt.Errorf("failed to render expression: %w", err)
	}

	if err := applyExpression(content, rendered, format); err != nil {
		return fmt.Errorf("failed to apply expression: %w", err)
	}

	// Write back
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

	if err := os.WriteFile(targetPath, output, 0644); err != nil {
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
		return "yaml" // Default to YAML
	}
}

func applyExpression(content interface{}, expression string, format string) error {
	expression = strings.TrimSpace(expression)
	
	if strings.Contains(expression, "|=") {
		return applyUpdateExpression(content, expression)
	}
	
	if strings.Contains(expression, "+=") {
		return applyAppendExpression(content, expression)
	}
	
	if strings.Contains(expression, "=") && !strings.Contains(expression, "==") {
		return applySetExpression(content, expression)
	}
	
	return fmt.Errorf("unsupported expression syntax: %s", expression)
}

func applySetExpression(content interface{}, expression string) error {
	parts := strings.SplitN(expression, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid expression format: %s", expression)
	}

	path := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	path = strings.TrimPrefix(path, ".")
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

func applyUpdateExpression(content interface{}, expression string) error {
	parts := strings.SplitN(expression, "|=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid update expression: %s", expression)
	}

	path := strings.TrimSpace(parts[0])
	updateExpr := strings.TrimSpace(parts[1])

	path = strings.TrimPrefix(path, ".")
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

	if strings.Contains(updateExpr, ". +") {
		valuePart := strings.TrimSpace(strings.Split(updateExpr, ". +")[1])
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
		}
	}

	return nil
}

func applyAppendExpression(content interface{}, expression string) error {
	parts := strings.SplitN(expression, "+=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid append expression: %s", expression)
	}

	path := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	path = strings.TrimPrefix(path, ".")
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

	// Try as number
	var num float64
	if _, err := fmt.Sscanf(value, "%f", &num); err == nil {
		return num
	}

	// Try as JSON array/object
	if strings.HasPrefix(value, "[") || strings.HasPrefix(value, "{") {
		var parsed interface{}
		if err := json.Unmarshal([]byte(value), &parsed); err == nil {
			return parsed
		}
	}

	// Remove quotes if present
	if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
		return value[1 : len(value)-1]
	}

	// Return as string
	return value
}
