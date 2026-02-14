package fzf

import (
	"slices"
	"testing"

	"github.com/idebeijer/kubert/internal/config"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "single arg",
			input: "--ansi",
			want:  []string{"--ansi"},
		},
		{
			name:  "multiple args",
			input: "--ansi --multi --height=50%",
			want:  []string{"--ansi", "--multi", "--height=50%"},
		},
		{
			name:  "double quoted string",
			input: "--preview \"cat {}\"",
			want:  []string{"--preview", "cat {}"},
		},
		{
			name:  "single quoted string",
			input: "--preview 'cat {}'",
			want:  []string{"--preview", "cat {}"},
		},
		{
			name:  "multiple spaces between args",
			input: "--ansi   --multi",
			want:  []string{"--ansi", "--multi"},
		},
		{
			name:  "quoted string with spaces",
			input: "--header \"select a context\"",
			want:  []string{"--header", "select a context"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseArgs(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseArgs(%q) = %v (len %d), want %v (len %d)", tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseArgs(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuildFzfArgs(t *testing.T) {
	tests := []struct {
		name         string
		fzfOpts      string
		multiSelect  bool
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:         "default single select",
			fzfOpts:      "",
			multiSelect:  false,
			wantContains: []string{"--ansi"},
			wantAbsent:   []string{"--multi"},
		},
		{
			name:         "multi select adds --multi",
			fzfOpts:      "",
			multiSelect:  true,
			wantContains: []string{"--ansi", "--multi"},
		},
		{
			name:         "custom opts included",
			fzfOpts:      "--height=50% --reverse",
			multiSelect:  false,
			wantContains: []string{"--ansi", "--height=50%", "--reverse"},
			wantAbsent:   []string{"--multi"},
		},
		{
			name:         "custom opts with --ansi not duplicated",
			fzfOpts:      "--ansi --height=50%",
			multiSelect:  false,
			wantContains: []string{"--ansi", "--height=50%"},
		},
		{
			name:         "custom opts with --multi not duplicated",
			fzfOpts:      "--multi --height=50%",
			multiSelect:  true,
			wantContains: []string{"--multi", "--height=50%", "--ansi"},
		},
		{
			name:         "custom opts with -m shorthand not duplicated",
			fzfOpts:      "-m --height=50%",
			multiSelect:  true,
			wantContains: []string{"-m", "--height=50%", "--ansi"},
			wantAbsent:   []string{"--multi"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := config.Cfg
			defer func() { config.Cfg = original }()

			config.Cfg.Fzf.Opts = tt.fzfOpts

			got := buildFzfArgs(tt.multiSelect)

			for _, want := range tt.wantContains {
				found := slices.Contains(got, want)
				if !found {
					t.Errorf("buildFzfArgs() = %v, missing expected %q", got, want)
				}
			}

			for _, absent := range tt.wantAbsent {
				for _, arg := range got {
					if arg == absent {
						t.Errorf("buildFzfArgs() = %v, should not contain %q", got, absent)
					}
				}
			}
		})
	}
}
