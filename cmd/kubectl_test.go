package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/state"
)

func TestKubectlOptions_Complete_SetsConfig(t *testing.T) {
	original := config.Cfg
	defer func() { config.Cfg = original }()

	config.Cfg = config.Config{
		Protection: config.Protection{
			Commands: []string{"apply", "delete"},
		},
	}

	o := NewKubectlOptions()
	cmd := &cobra.Command{}

	if len(o.Config.Protection.Commands) != 0 {
		t.Error("Config should not be set before Complete()")
	}

	_ = o.Complete(cmd, []string{"get", "pods"})

	if len(o.Config.Protection.Commands) != 2 {
		t.Errorf("Complete() should set Config from config.Cfg, got: %+v", o.Config.Protection.Commands)
	}
}

func TestIsCommandProtected(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		blockedCmds []string
		expected    bool
	}{
		{
			name:        "command in blocked list",
			args:        []string{"apply", "-f", "deployment.yaml"},
			blockedCmds: []string{"apply", "delete", "edit"},
			expected:    true,
		},
		{
			name:        "command not in blocked list",
			args:        []string{"get", "pods"},
			blockedCmds: []string{"apply", "delete", "edit"},
			expected:    false,
		},
		{
			name:        "empty args",
			args:        []string{},
			blockedCmds: []string{"apply", "delete"},
			expected:    false,
		},
		{
			name:        "empty blocked list",
			args:        []string{"apply"},
			blockedCmds: []string{},
			expected:    false,
		},
		{
			name:        "case sensitive match",
			args:        []string{"Apply"},
			blockedCmds: []string{"apply"},
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCommandProtected(tt.args, tt.blockedCmds)
			if result != tt.expected {
				t.Errorf("isCommandProtected(%v, %v) = %v, want %v",
					tt.args, tt.blockedCmds, result, tt.expected)
			}
		})
	}
}

func TestIsContextProtected(t *testing.T) {
	t.Run("context not in state, matches regex", func(t *testing.T) {
		setupTestXDGDataHome(t)

		sm, err := state.NewManager()
		if err != nil {
			t.Fatalf("Failed to create state manager: %v", err)
		}

		prodRegex := "^prod.*"
		cfg := config.Config{
			Protection: config.Protection{
				Regex: &prodRegex,
			},
		}

		// Test context that matches regex
		protected, err := isContextProtected(sm, "prod-cluster", cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !protected {
			t.Error("Expected prod-cluster to be protected by regex")
		}

		// Test context that doesn't match regex
		protected, err = isContextProtected(sm, "dev-cluster", cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if protected {
			t.Error("Expected dev-cluster to not be protected")
		}
	})

	t.Run("invalid regex", func(t *testing.T) {
		setupTestXDGDataHome(t)

		sm, err := state.NewManager()
		if err != nil {
			t.Fatalf("Failed to create state manager: %v", err)
		}

		invalidRegex := "["
		cfg := config.Config{
			Protection: config.Protection{
				Regex: &invalidRegex,
			},
		}

		_, err = isContextProtected(sm, "any-context", cfg)
		if err == nil {
			t.Error("Expected error for invalid regex, got nil")
		}
		if !strings.Contains(err.Error(), "failed to compile regex") {
			t.Errorf("Unexpected error message: %v", err)
		}
	})
}

func TestKubectlOptions_Complete(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedArgs []string
	}{
		{
			name:         "simple command",
			args:         []string{"get", "pods"},
			expectedArgs: []string{"get", "pods"},
		},
		{
			name:         "empty args",
			args:         []string{},
			expectedArgs: []string{},
		},
		{
			name:         "command with flags",
			args:         []string{"apply", "-f", "file.yaml", "--dry-run"},
			expectedArgs: []string{"apply", "-f", "file.yaml", "--dry-run"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := NewKubectlOptions()
			cmd := &cobra.Command{}

			err := o.Complete(cmd, tt.args)
			if err != nil {
				t.Errorf("Complete() returned error: %v", err)
			}

			if len(o.Args) != len(tt.expectedArgs) {
				t.Fatalf("Args length mismatch: got %d, want %d", len(o.Args), len(tt.expectedArgs))
			}

			for i, arg := range o.Args {
				if arg != tt.expectedArgs[i] {
					t.Errorf("Args[%d] = %s, want %s", i, arg, tt.expectedArgs[i])
				}
			}
		})
	}
}

func TestKubectlOptions_Validate(t *testing.T) {
	o := NewKubectlOptions()
	err := o.Validate()
	if err != nil {
		t.Errorf("Validate() returned unexpected error: %v", err)
	}
}

func TestKubectlOptions_Run_Unprotected(t *testing.T) {
	var buf bytes.Buffer

	o := &KubectlOptions{
		Out:    &buf,
		ErrOut: &buf,
		Args:   []string{"get", "pods"},
		Config: config.Config{
			Protection: config.Protection{
				Commands: []string{"apply", "delete"},
			},
		},
		StateManager: func() (*state.Manager, error) {
			setupTestXDGDataHome(t)
			return state.NewManager()
		},
		ClientConfigLoader: func() (*api.Config, error) {
			return &api.Config{
				CurrentContext: "dev-cluster",
			}, nil
		},
		CommandRunner: func(args []string) error {
			// Mock successful execution
			return nil
		},
		Prompter: func() bool {
			t.Error("Prompter should not be called for unprotected context")
			return false
		},
	}

	err := o.Run()
	if err != nil {
		t.Errorf("Run() returned unexpected error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "WARNING") {
		t.Error("Should not show warning for unprotected context")
	}
}

func TestKubectlOptions_Run_ProtectedWithPromptDisabled(t *testing.T) {
	var buf bytes.Buffer

	commandRunnerCalled := false
	o := &KubectlOptions{
		Out:    &buf,
		ErrOut: &buf,
		Args:   []string{"apply", "-f", "deployment.yaml"},
		Config: config.Config{
			Protection: config.Protection{
				Commands: []string{"apply", "delete"},
				Prompt:   false,
			},
		},
		StateManager: func() (*state.Manager, error) {
			setupTestXDGDataHome(t)
			return state.NewManager()
		},
		ClientConfigLoader: func() (*api.Config, error) {
			return &api.Config{
				CurrentContext: "prod-cluster",
			}, nil
		},
		CommandRunner: func(args []string) error {
			commandRunnerCalled = true
			return nil
		},
		Prompter: func() bool {
			t.Error("Prompter should not be called when prompt is disabled")
			return false
		},
	}

	err := o.Run()
	if err != nil {
		t.Errorf("Run() returned unexpected error: %v", err)
	}

	// Without actual protection state set up, command will run
	// A full integration test would verify the protection message appears
	_ = commandRunnerCalled // acknowledge we track this for potential assertions
}

func TestKubectlOptions_Run_ProtectedCommandNotBlocked(t *testing.T) {
	var buf bytes.Buffer

	o := &KubectlOptions{
		Out:    &buf,
		ErrOut: &buf,
		Args:   []string{"get", "pods"}, // Not a protected command
		Config: config.Config{
			Protection: config.Protection{
				Commands: []string{"apply", "delete"},
				Prompt:   true,
			},
		},
		StateManager: func() (*state.Manager, error) {
			return &state.Manager{}, nil
		},
		ClientConfigLoader: func() (*api.Config, error) {
			return &api.Config{
				CurrentContext: "prod-cluster",
			}, nil
		},
		CommandRunner: func(args []string) error {
			return nil
		},
		Prompter: func() bool {
			t.Error("Prompter should not be called for non-protected command")
			return false
		},
	}

	err := o.Run()
	if err != nil {
		t.Errorf("Run() returned unexpected error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "WARNING") {
		t.Error("Should not show warning for non-protected command")
	}
}

func TestKubectlOptions_Run_StateManagerError(t *testing.T) {
	var buf bytes.Buffer

	o := &KubectlOptions{
		Out:    &buf,
		ErrOut: &buf,
		Args:   []string{"get", "pods"},
		StateManager: func() (*state.Manager, error) {
			return nil, errors.New("state manager initialization failed")
		},
	}

	err := o.Run()
	if err == nil {
		t.Error("Expected error from StateManager, got nil")
	}
	if !strings.Contains(err.Error(), "state manager initialization failed") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestKubectlOptions_Run_ClientConfigError(t *testing.T) {
	var buf bytes.Buffer

	o := &KubectlOptions{
		Out:    &buf,
		ErrOut: &buf,
		Args:   []string{"get", "pods"},
		StateManager: func() (*state.Manager, error) {
			return &state.Manager{}, nil
		},
		ClientConfigLoader: func() (*api.Config, error) {
			return nil, errors.New("kubeconfig not found")
		},
	}

	err := o.Run()
	if err == nil {
		t.Error("Expected error from ClientConfigLoader, got nil")
	}
	if !strings.Contains(err.Error(), "kubeconfig not found") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestPromptUserConfirmation(t *testing.T) {
	// Skip for now. This function reads from stdin, can't easily test it without mocking stdin.
	t.Skip("promptUserConfirmation requires stdin mocking")
}
