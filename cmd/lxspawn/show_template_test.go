package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Inoriol/lxspawn/internal/output"
)

func TestShowTemplateCommand(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "valid template",
			templateName: "debian-base",
			wantErr:      false,
		},
		{
			name:         "nonexistent template",
			templateName: "nonexistent-template",
			wantErr:      true,
			errContains:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := showTemplateCmd
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs([]string{tt.templateName})

			err := cmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				output := buf.String()
				if !strings.Contains(output, "name:") {
					t.Errorf("expected template content with 'name:' field, got %q", output)
				}
			}
		})
	}
}

func TestShowTemplateJSON(t *testing.T) {
	// Save original stdout and restore after test
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	output.SetJSONMode(true)
	defer func() {
		output.SetJSONMode(false)
		os.Stdout = oldStdout
	}()

	cmd := showTemplateCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"debian-base"})

	err := cmd.Execute()
	
	w.Close()
	var stdoutBuf bytes.Buffer
	io.Copy(&stdoutBuf, r)
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jsonOutput := stdoutBuf.String()
	if !strings.Contains(jsonOutput, `"name"`) {
		t.Errorf("expected JSON output with 'name' field, got %q", jsonOutput)
	}
	if !strings.Contains(jsonOutput, `"content"`) {
		t.Errorf("expected JSON output with 'content' field, got %q", jsonOutput)
	}
}
