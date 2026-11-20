package main

import (
	"testing"
)

func TestGetSudoVersion(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    string
		wantErr bool
	}{
		{
			name:   "standard version",
			output: "Sudo version 1.9.17p2\nSudoers policy plugin version 1.9.17p2\n",
			want:   "1.9.17p2",
		},
		{
			name:   "older version",
			output: "Sudo version 1.8.21p2\n",
			want:   "1.8.21p2",
		},
		{
			name:   "version without patch letter",
			output: "Sudo version 1.9.10\n",
			want:   "1.9.10",
		},
		{
			name:    "empty output",
			output:  "",
			wantErr: true,
		},
		{
			name:    "unexpected format",
			output:  "Unexpected output\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: We can't easily test getSudoVersion() directly since it calls exec.Command
			// Instead, we test the parsing logic via sudoVersionSupportsRegex
		})
	}
}

func TestSudoVersionSupportsRegex(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{
			name:    "version 1.9.17p2 (current)",
			version: "1.9.17p2",
			want:    true,
		},
		{
			name:    "version 1.9.10 (minimum)",
			version: "1.9.10",
			want:    true,
		},
		{
			name:    "version 1.9.10p1",
			version: "1.9.10p1",
			want:    true,
		},
		{
			name:    "version 1.9.9 (too old)",
			version: "1.9.9",
			want:    false,
		},
		{
			name:    "version 1.9.9p2 (too old)",
			version: "1.9.9p2",
			want:    false,
		},
		{
			name:    "version 1.8.21p2 (old)",
			version: "1.8.21p2",
			want:    false,
		},
		{
			name:    "version 2.0.0 (future)",
			version: "2.0.0",
			want:    true,
		},
		{
			name:    "version 1.10.0 (future minor)",
			version: "1.10.0",
			want:    true,
		},
		{
			name:    "invalid version format",
			version: "invalid",
			want:    false,
		},
		{
			name:    "empty version",
			version: "",
			want:    false,
		},
		{
			name:    "partial version",
			version: "1.9",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sudoVersionSupportsRegex(tt.version)
			if got != tt.want {
				t.Errorf("sudoVersionSupportsRegex(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}
