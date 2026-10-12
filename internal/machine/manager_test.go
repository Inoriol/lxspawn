package machine

import (
	"testing"
)

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "''",
		},
		{
			name:     "simple alphanumeric",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "with spaces",
			input:    "hello world",
			expected: "'hello world'",
		},
		{
			name:     "with single quote",
			input:    "it's",
			expected: "'it'\"'\"'s'",
		},
		{
			name:     "with double quote",
			input:    `say "hello"`,
			expected: `'say "hello"'`,
		},
		{
			name:     "with special chars",
			input:    "test$var",
			expected: "'test$var'",
		},
		{
			name:     "path with slashes",
			input:    "/usr/bin/test",
			expected: "/usr/bin/test",
		},
		{
			name:     "with equals",
			input:    "KEY=value",
			expected: "KEY=value",
		},
		{
			name:     "with semicolon",
			input:    "cmd1; cmd2",
			expected: "'cmd1; cmd2'",
		},
		{
			name:     "with pipe",
			input:    "cmd1 | cmd2",
			expected: "'cmd1 | cmd2'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellQuote(tt.input)
			if result != tt.expected {
				t.Errorf("shellQuote(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestShellQuoteArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "single arg",
			args:     []string{"echo"},
			expected: "echo",
		},
		{
			name:     "multiple args",
			args:     []string{"echo", "hello", "world"},
			expected: "echo hello world",
		},
		{
			name:     "args with spaces",
			args:     []string{"echo", "hello world"},
			expected: "echo 'hello world'",
		},
		{
			name:     "args with special chars",
			args:     []string{"sh", "-c", "echo $HOME"},
			expected: "sh -c 'echo $HOME'",
		},
		{
			name:     "empty args",
			args:     []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellQuoteArgs(tt.args)
			if result != tt.expected {
				t.Errorf("shellQuoteArgs(%v) = %q, want %q", tt.args, result, tt.expected)
			}
		})
	}
}
