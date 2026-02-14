package config

import (
	"strings"
	"testing"
)

func TestSetDefaults(t *testing.T) {
	// DefaultCfg is populated via init() which calls setDefaults()

	t.Run("kubeconfig include defaults", func(t *testing.T) {
		include := DefaultCfg.KubeconfigPaths.Include
		if len(include) != 3 {
			t.Fatalf("expected 3 default include paths, got %d: %v", len(include), include)
		}

		expected := []string{
			"~/.kube/config",
			"~/.kube/*.yml",
			"~/.kube/*.yaml",
		}
		for i, want := range expected {
			if include[i] != want {
				t.Errorf("include[%d] = %q, want %q", i, include[i], want)
			}
		}
	})

	t.Run("kubeconfig exclude defaults empty", func(t *testing.T) {
		if len(DefaultCfg.KubeconfigPaths.Exclude) != 0 {
			t.Errorf("expected empty exclude paths, got %v", DefaultCfg.KubeconfigPaths.Exclude)
		}
	})

	t.Run("interactive defaults to true", func(t *testing.T) {
		if !DefaultCfg.Interactive {
			t.Error("expected Interactive to default to true")
		}
	})

	t.Run("protection regex defaults to nil", func(t *testing.T) {
		if DefaultCfg.Protection.Regex != nil {
			t.Errorf("expected Protection.Regex to be nil, got %q", *DefaultCfg.Protection.Regex)
		}
	})

	t.Run("protection commands defaults", func(t *testing.T) {
		cmds := DefaultCfg.Protection.Commands
		expectedCmds := []string{
			"delete", "edit", "exec", "drain", "scale",
			"autoscale", "replace", "apply", "patch", "set",
		}

		if len(cmds) != len(expectedCmds) {
			t.Fatalf("expected %d default protection commands, got %d: %v", len(expectedCmds), len(cmds), cmds)
		}

		for i, want := range expectedCmds {
			if cmds[i] != want {
				t.Errorf("commands[%d] = %q, want %q", i, cmds[i], want)
			}
		}
	})

	t.Run("protection prompt defaults to true", func(t *testing.T) {
		if !DefaultCfg.Protection.Prompt {
			t.Error("expected Protection.Prompt to default to true")
		}
	})

	t.Run("hooks default to empty", func(t *testing.T) {
		if DefaultCfg.Hooks.PreShell != "" {
			t.Errorf("expected Hooks.PreShell to be empty, got %q", DefaultCfg.Hooks.PreShell)
		}
		if DefaultCfg.Hooks.PostShell != "" {
			t.Errorf("expected Hooks.PostShell to be empty, got %q", DefaultCfg.Hooks.PostShell)
		}
	})

	t.Run("fzf opts default to empty", func(t *testing.T) {
		if DefaultCfg.Fzf.Opts != "" {
			t.Errorf("expected Fzf.Opts to be empty, got %q", DefaultCfg.Fzf.Opts)
		}
	})
}

func TestGenerateDefaultYAML(t *testing.T) {
	output, err := GenerateDefaultYAML()
	if err != nil {
		t.Fatalf("GenerateDefaultYAML() error = %v", err)
	}

	if output == "" {
		t.Fatal("GenerateDefaultYAML() returned empty string")
	}

	t.Run("contains kubeconfig include paths", func(t *testing.T) {
		for _, path := range []string{"~/.kube/config", "~/.kube/*.yml", "~/.kube/*.yaml"} {
			if !strings.Contains(output, path) {
				t.Errorf("YAML missing include path %q:\n%s", path, output)
			}
		}
	})

	t.Run("contains protection commands", func(t *testing.T) {
		for _, cmd := range []string{"delete", "apply", "exec", "drain"} {
			if !strings.Contains(output, cmd) {
				t.Errorf("YAML missing protection command %q:\n%s", cmd, output)
			}
		}
	})

	t.Run("contains interactive setting", func(t *testing.T) {
		if !strings.Contains(output, "interactive") {
			t.Errorf("YAML missing 'interactive' key:\n%s", output)
		}
	})

	t.Run("contains expected sections", func(t *testing.T) {
		for _, section := range []string{"kubeconfigs:", "protection:", "hooks:", "fzf:"} {
			if !strings.Contains(output, section) {
				t.Errorf("YAML missing %q section:\n%s", section, output)
			}
		}
	})
}

func TestDefaultCfg_NotAffectedByGlobalCfg(t *testing.T) {
	originalInclude := make([]string, len(DefaultCfg.KubeconfigPaths.Include))
	copy(originalInclude, DefaultCfg.KubeconfigPaths.Include)

	Cfg.KubeconfigPaths.Include = []string{"/custom/path"}

	if len(DefaultCfg.KubeconfigPaths.Include) != len(originalInclude) {
		t.Error("modifying Cfg affected DefaultCfg")
	}

	// Restore
	Cfg.KubeconfigPaths.Include = nil
}
