package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/kubeconfig"
	"github.com/idebeijer/kubert/internal/kubert"
	"github.com/idebeijer/kubert/internal/state"
)

// setupTestXDGDataHome sets xdg.DataHome to a temp directory and returns a cleanup function
// that restores the original value.
// nolint:unparam
func setupTestXDGDataHome(t *testing.T) string {
	t.Helper()
	original := xdg.DataHome
	tempDir := t.TempDir()
	xdg.DataHome = tempDir
	t.Cleanup(func() { xdg.DataHome = original })
	return tempDir
}

func TestContextOptions_Complete_SetsConfig(t *testing.T) {
	original := config.Cfg
	defer func() { config.Cfg = original }()

	config.Cfg = config.Config{
		KubeconfigPaths: config.KubeconfigPaths{
			Include: []string{"/some/path"},
		},
	}

	o := NewContextOptions()
	cmd := &cobra.Command{}

	if len(o.Config.KubeconfigPaths.Include) != 0 {
		t.Error("Config should not be set before Complete()")
	}

	_ = o.Complete(cmd, []string{})

	if len(o.Config.KubeconfigPaths.Include) != 1 || o.Config.KubeconfigPaths.Include[0] != "/some/path" {
		t.Errorf("Complete() should set Config from config.Cfg, got: %+v", o.Config.KubeconfigPaths)
	}
}

func TestCreateTempKubeconfigFile_Isolation(t *testing.T) {
	// Create a temp file to act as the "original" complex kubeconfig
	tempFile, err := os.CreateTemp("", "original-kubeconfig-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Fatal(err)
		}
	}()

	// Define a complex configuration with multiple contexts, clusters, and users
	cfg := api.NewConfig()

	// Cluster 1 data
	cfg.Clusters["shared-cluster"] = &api.Cluster{Server: "https://shared.example.com"}

	cfg.AuthInfos["user-1"] = &api.AuthInfo{Token: "token-1"}
	cfg.Contexts["ctx-1"] = &api.Context{Cluster: "shared-cluster", AuthInfo: "user-1", Namespace: "ns-1"}

	cfg.AuthInfos["user-2"] = &api.AuthInfo{Username: "admin", Password: "password"}
	cfg.Contexts["ctx-2"] = &api.Context{Cluster: "shared-cluster", AuthInfo: "user-2", Namespace: "default"}

	// Cluster 2 data
	cfg.Clusters["other-cluster"] = &api.Cluster{Server: "https://other.example.com"}
	cfg.AuthInfos["user-3"] = &api.AuthInfo{ClientCertificate: "/path/to/cert"}
	cfg.Contexts["ctx-3"] = &api.Context{Cluster: "other-cluster", AuthInfo: "user-3"}

	// Write this original config to disk
	if err := clientcmd.WriteToFile(*cfg, tempFile.Name()); err != nil {
		t.Fatal(err)
	}

	// Now try to isolate "ctx-2"
	isolatedFile, cleanup, err := createTempKubeconfigFile(tempFile.Name(), "ctx-2", "")
	if err != nil {
		t.Fatalf("createTempKubeconfigFile failed: %v", err)
	}
	defer cleanup()

	// Load the generated isolated config to verify contents
	isolatedConfig, err := clientcmd.LoadFromFile(isolatedFile.Name())
	if err != nil {
		t.Fatalf("Failed to load generated isolated config: %v", err)
	}

	// Verify Context correctness
	if len(isolatedConfig.Contexts) != 1 {
		t.Errorf("Expected exactly 1 context, got %d", len(isolatedConfig.Contexts))
	}
	ctx, exists := isolatedConfig.Contexts["ctx-2"]
	if !exists {
		t.Fatal("Target context 'ctx-2' missing from isolated config")
	}
	if ctx.Cluster != "shared-cluster" {
		t.Errorf("Context cluster mismatch. Got %s, want shared-cluster", ctx.Cluster)
	}
	if ctx.AuthInfo != "user-2" {
		t.Errorf("Context user mismatch. Got %s, want user-2", ctx.AuthInfo)
	}

	// Verify Cluster correctness
	if len(isolatedConfig.Clusters) != 1 {
		t.Errorf("Expected exactly 1 cluster, got %d", len(isolatedConfig.Clusters))
	}
	cluster, exists := isolatedConfig.Clusters["shared-cluster"]
	if !exists {
		t.Fatal("Target cluster 'shared-cluster' missing from isolated config")
	}
	if cluster.Server != "https://shared.example.com" {
		t.Errorf("Cluster server mismatch. Got %s", cluster.Server)
	}

	// Verify User correctness
	if len(isolatedConfig.AuthInfos) != 1 {
		t.Errorf("Expected exactly 1 user, got %d", len(isolatedConfig.AuthInfos))
	}
	user, exists := isolatedConfig.AuthInfos["user-2"]
	if !exists {
		t.Fatal("Target user 'user-2' missing from isolated config")
	}
	if user.Username != "admin" {
		t.Errorf("User username mismatch. Got %s", user.Username)
	}

	// Verify leakage (ensure other data is NOT present)
	if _, exists := isolatedConfig.Clusters["other-cluster"]; exists {
		t.Error("Leaked 'other-cluster' into isolated config")
	}
	if _, exists := isolatedConfig.AuthInfos["user-3"]; exists {
		t.Error("Leaked 'user-3' into isolated config")
	}
}

// Note: This is an experimental test that simulates launching a shell with the modified kubeconfig.
// Skipped for now.
// nolint
func TestLaunchShellWithKubeconfig(t *testing.T) {
	t.Skip()

	// Part 1: The child process logic (the code that runs when the shell is launched)
	if os.Getenv("GO_HELPER_PROCESS") == "1" {
		// 1. Verify Environment
		if os.Getenv("KUBERT_SHELL_ACTIVE") != "1" {
			fmt.Fprintf(os.Stderr, "Error: KUBERT_SHELL_ACTIVE not set\n")
			os.Exit(1)
		}

		// 2. TODO: simulate running kubert commands either by binary or by directly calling the logic

		o := NewNamespaceOptions()
		o.Args = []string{"default"}
		if err := o.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to run namespace command: %v\n", err)
			os.Exit(1)
		}

		// Success
		os.Exit(0)
		return
	}

	// Part 2: Setup parent process logic (sets up env with kubeconfigs and starts the shell)

	// 1. Setup Safe Paths
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "kubeconfig")
	originalKubeconfigPath := filepath.Join(tempDir, "orig_kubeconfig")

	// Create dummy files so the tool doesn't crash
	createTestKubeconfig(t, originalKubeconfigPath, "test-context", "test-cluster", "test-user")

	// Create a copy for the temp kubeconfig (TODO: maybe use actual `kubert context` logic)
	createTestKubeconfig(t, kubeconfigPath, "test-context", "test-cluster", "test-user")

	// 2. Create the Wrapper Script (The "Adapter")
	// This forces the shell to run ONLY this test function
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	wrapperScript := filepath.Join(tempDir, "mock_shell.sh")
	scriptContent := fmt.Sprintf("#!/bin/sh\n%s -test.run=TestLaunchShellWithKubeconfig", testBinary)
	if err := os.WriteFile(wrapperScript, []byte(scriptContent), 0o755); err != nil {
		t.Fatal(err)
	}

	// 3. Configure the Environment
	// Tell the child logic to activate
	os.Setenv("GO_HELPER_PROCESS", "1")
	defer os.Unsetenv("GO_HELPER_PROCESS")

	// Point SHELL to our wrapper script
	os.Setenv("SHELL", wrapperScript)

	// 4. Run the Function
	cfg := config.Config{}
	err = launchShellWithKubeconfig(kubeconfigPath, originalKubeconfigPath, "ctx", cfg)
	if err != nil {
		t.Fatalf("Function failed: %v", err)
	}
}

// Note: This test is experimental and more of an integration test which may be flaky and should
// be run inside the ./testdata/Dockerfile container.
// Running locally would require all shells to be installed.
// nolint
func TestLaunchShells(t *testing.T) {
	if os.Getenv("RUN_SHELL_TESTS") != "true" {
		t.Skip("Skipping shell tests")
	}

	// 1. Compile binary for testing
	tempBinDir := t.TempDir() // Automatically cleaned up by Go
	kubertBinary := filepath.Join(tempBinDir, "kubert")

	buildCmd := exec.Command("go", "build", "-o", kubertBinary, ".")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build kubert for testing: %s\n%s", err, out)
	}

	// 2. Prepare PATH
	originalPath := os.Getenv("PATH")
	newPath := tempBinDir + string(os.PathListSeparator) + originalPath

	// Set the PATH for the current process (the test runner).
	// Your launchShellWithKubeconfig function inherits this environment,
	// so the child shells (bash/zsh/fish) will inherit it too.
	os.Setenv("PATH", newPath)
	defer os.Setenv("PATH", originalPath) // Restore after test

	shells := []string{"/bin/sh", "/bin/bash", "/bin/zsh", "/usr/bin/fish"}

	for _, shellBin := range shells {
		t.Run(filepath.Base(shellBin), func(t *testing.T) {
			if _, err := exec.LookPath(shellBin); err != nil {
				t.Skipf("%s not found", shellBin)
			}

			os.Setenv("SHELL", shellBin)

			r, w, _ := os.Pipe()
			os.Stdin = r

			// 3. Define test logic to run inside the shell
			go func() {
				defer w.Close()

				// TODO: add tests and configure namespace switch to allow 'offline' mode
				// to prevent calls to a fake kubernetes cluster.

				cmd := `
                # Run your tool
                echo "Running kubert..."
                kubert version

                kubert ns switch default

                # Verify execution
                echo "KUBERT_TEST_COMPLETE"
                exit
                `
				w.Write([]byte(cmd))
			}()

			// Capture output
			var stdout, stderr bytes.Buffer

			opts := ShellOptions{
				Stdin:  r,
				Stdout: &stdout,
				Stderr: &stderr,
			}

			cfg := config.Config{}
			err := launchShellWithKubeconfig("/tmp/kube", "/orig", "ctx", cfg, opts)
			if err != nil {
				t.Errorf("Shell %s crashed: %v", shellBin, err)
			}

			// 4. Verify output
			if !strings.Contains(stdout.String(), "KUBERT_TEST_COMPLETE") {
				t.Errorf("Expected 'KUBERT_TEST_COMPLETE' in stdout, got: %s", stdout.String())
			}
		})
	}
}

func TestContextOptions_Complete(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedArgs []string
	}{
		{
			name:         "with context name",
			args:         []string{"my-cluster"},
			expectedArgs: []string{"my-cluster"},
		},
		{
			name:         "with previous context flag",
			args:         []string{"-"},
			expectedArgs: []string{"-"},
		},
		{
			name:         "no args",
			args:         []string{},
			expectedArgs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := NewContextOptions()
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

func TestContextOptions_Run_WithContextName(t *testing.T) {
	var buf bytes.Buffer
	shellLauncherCalled := false
	tempFileCreated := false

	o := &ContextOptions{
		Out:    &buf,
		ErrOut: &buf,
		Args:   []string{"test-cluster"},
		Config: config.Config{},
		ContextLoader: func() ([]kubeconfig.Context, error) {
			return []kubeconfig.Context{
				{Name: "test-cluster", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config"}},
			}, nil
		},
		StateManager: func() (*state.Manager, error) {
			setupTestXDGDataHome(t)
			return state.NewManager()
		},
		Selector: func(items []string) (string, error) {
			t.Error("Selector should not be called when context name is provided")
			return "", nil
		},
		IsInteractive: func() bool {
			return true
		},
		ShellLauncher: func(kubeconfigPath, originalPath, contextName string, cfg config.Config) error {
			shellLauncherCalled = true
			if contextName != "test-cluster" {
				t.Errorf("Expected context name 'test-cluster', got '%s'", contextName)
			}
			return nil
		},
		TempFileWriter: func(kubeconfigPath, contextName, namespace string) (*os.File, func(), error) {
			tempFileCreated = true
			tempFile, _ := os.CreateTemp("", "test-*.yaml")
			cleanup := func() {
				_ = tempFile.Close()
				_ = os.Remove(tempFile.Name())
			}
			return tempFile, cleanup, nil
		},
	}

	err := o.Run()
	if err != nil {
		t.Errorf("Run() returned unexpected error: %v", err)
	}

	if !shellLauncherCalled {
		t.Error("ShellLauncher should have been called")
	}

	if !tempFileCreated {
		t.Error("TempFileWriter should have been called")
	}
}

func TestContextOptions_Run_PreviousContext(t *testing.T) {
	var buf bytes.Buffer
	shellLauncherCalled := false

	setupTestXDGDataHome(t)

	sm, err := state.NewManager()
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	_ = sm.SetLastContext("previous-cluster")

	o := &ContextOptions{
		Out:    &buf,
		ErrOut: &buf,
		Args:   []string{"-"},
		Config: config.Config{},
		ContextLoader: func() ([]kubeconfig.Context, error) {
			return []kubeconfig.Context{
				{Name: "previous-cluster", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config"}},
			}, nil
		},
		StateManager: func() (*state.Manager, error) {
			return sm, nil
		},
		ShellLauncher: func(kubeconfigPath, originalPath, contextName string, cfg config.Config) error {
			shellLauncherCalled = true
			if contextName != "previous-cluster" {
				t.Errorf("Expected context name 'previous-cluster', got '%s'", contextName)
			}
			return nil
		},
		TempFileWriter: func(kubeconfigPath, contextName, namespace string) (*os.File, func(), error) {
			tempFile, _ := os.CreateTemp("", "test-*.yaml")
			cleanup := func() {
				_ = tempFile.Close()
				_ = os.Remove(tempFile.Name())
			}
			return tempFile, cleanup, nil
		},
	}

	err = o.Run()
	if err != nil {
		t.Errorf("Run() returned unexpected error: %v", err)
	}

	if !shellLauncherCalled {
		t.Error("ShellLauncher should have been called")
	}
}

func TestContextOptions_Run_NoPreviousContext(t *testing.T) {
	var buf bytes.Buffer

	setupTestXDGDataHome(t)

	sm, err := state.NewManager()
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	o := &ContextOptions{
		Out:    &buf,
		ErrOut: &buf,
		Args:   []string{"-"},
		Config: config.Config{},
		ContextLoader: func() ([]kubeconfig.Context, error) {
			return []kubeconfig.Context{
				{Name: "test-cluster", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config"}},
			}, nil
		},
		StateManager: func() (*state.Manager, error) {
			return sm, nil
		},
	}

	err = o.Run()
	if err == nil {
		t.Error("Expected error for no previous context, got nil")
		return
	}
	if !strings.Contains(err.Error(), "no previous context") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestContextOptions_Run_InteractiveSelection(t *testing.T) {
	var buf bytes.Buffer
	selectorCalled := false
	shellLauncherCalled := false

	o := &ContextOptions{
		Out:    &buf,
		ErrOut: &buf,
		Args:   []string{},
		Config: config.Config{},
		ContextLoader: func() ([]kubeconfig.Context, error) {
			return []kubeconfig.Context{
				{Name: "cluster-1", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config"}},
				{Name: "cluster-2", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config"}},
			}, nil
		},
		StateManager: func() (*state.Manager, error) {
			setupTestXDGDataHome(t)
			return state.NewManager()
		},
		Selector: func(items []string) (string, error) {
			selectorCalled = true
			return "cluster-1", nil
		},
		IsInteractive: func() bool {
			return true
		},
		ShellLauncher: func(kubeconfigPath, originalPath, contextName string, cfg config.Config) error {
			shellLauncherCalled = true
			return nil
		},
		TempFileWriter: func(kubeconfigPath, contextName, namespace string) (*os.File, func(), error) {
			tempFile, _ := os.CreateTemp("", "test-*.yaml")
			cleanup := func() {
				_ = tempFile.Close()
				_ = os.Remove(tempFile.Name())
			}
			return tempFile, cleanup, nil
		},
	}

	err := o.Run()
	if err != nil {
		t.Errorf("Run() returned unexpected error: %v", err)
	}

	if !selectorCalled {
		t.Error("Selector should have been called in interactive mode")
	}

	if !shellLauncherCalled {
		t.Error("ShellLauncher should have been called")
	}
}

func TestContextOptions_Run_NonInteractivePrintOnly(t *testing.T) {
	var buf bytes.Buffer

	o := &ContextOptions{
		Out:    &buf,
		ErrOut: &buf,
		Args:   []string{},
		Config: config.Config{},
		ContextLoader: func() ([]kubeconfig.Context, error) {
			return []kubeconfig.Context{
				{Name: "cluster-1", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config"}},
				{Name: "cluster-2", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config"}},
			}, nil
		},
		StateManager: func() (*state.Manager, error) {
			setupTestXDGDataHome(t)
			return state.NewManager()
		},
		Selector: func(items []string) (string, error) {
			t.Error("Selector should not be called in non-interactive mode")
			return "", nil
		},
		IsInteractive: func() bool {
			return false
		},
		ShellLauncher: func(kubeconfigPath, originalPath, contextName string, cfg config.Config) error {
			t.Error("ShellLauncher should not be called when printing only")
			return nil
		},
	}

	err := o.Run()
	if err != nil {
		t.Errorf("Run() returned unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "cluster-1") {
		t.Error("Expected cluster-1 in output")
	}
	if !strings.Contains(output, "cluster-2") {
		t.Error("Expected cluster-2 in output")
	}
}

func TestContextOptions_Run_ContextNotFound(t *testing.T) {
	var buf bytes.Buffer

	o := &ContextOptions{
		Out:    &buf,
		ErrOut: &buf,
		Args:   []string{"nonexistent-cluster"},
		Config: config.Config{},
		ContextLoader: func() ([]kubeconfig.Context, error) {
			return []kubeconfig.Context{
				{Name: "cluster-1", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config"}},
			}, nil
		},
		StateManager: func() (*state.Manager, error) {
			setupTestXDGDataHome(t)
			return state.NewManager()
		},
	}

	err := o.Run()
	if err == nil {
		t.Error("Expected error for nonexistent context, got nil")
	}
	if !strings.Contains(err.Error(), "context nonexistent-cluster not found") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestContextOptions_Run_ContextLoaderError(t *testing.T) {
	var buf bytes.Buffer

	o := &ContextOptions{
		Out:    &buf,
		ErrOut: &buf,
		Args:   []string{"test-cluster"},
		StateManager: func() (*state.Manager, error) {
			setupTestXDGDataHome(t)
			return state.NewManager()
		},
		ContextLoader: func() ([]kubeconfig.Context, error) {
			return nil, fmt.Errorf("failed to load kubeconfig")
		},
	}

	err := o.Run()
	if err == nil {
		t.Error("Expected error from context loader, got nil")
		return
	}
	if !strings.Contains(err.Error(), "error loading contexts") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestContextOptions_Run_StateManagerError(t *testing.T) {
	var buf bytes.Buffer

	o := &ContextOptions{
		Out:    &buf,
		ErrOut: &buf,
		Args:   []string{"test-cluster"},
		StateManager: func() (*state.Manager, error) {
			return nil, fmt.Errorf("state manager initialization failed")
		},
	}

	err := o.Run()
	if err == nil {
		t.Error("Expected error from state manager, got nil")
	}
	if !strings.Contains(err.Error(), "error creating state manager") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestGetContextNames(t *testing.T) {
	contexts := []kubeconfig.Context{
		{Name: "cluster-1"},
		{Name: "cluster-2"},
		{Name: "cluster-3"},
	}

	names := getContextNames(contexts)

	if len(names) != 3 {
		t.Errorf("Expected 3 names, got %d", len(names))
	}

	expectedNames := map[string]bool{"cluster-1": true, "cluster-2": true, "cluster-3": true}
	for _, name := range names {
		if !expectedNames[name] {
			t.Errorf("Unexpected name: %s", name)
		}
	}
}

func TestFindContextByName(t *testing.T) {
	contexts := []kubeconfig.Context{
		{Name: "cluster-1", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config1"}},
		{Name: "cluster-2", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config2"}},
	}

	t.Run("context found", func(t *testing.T) {
		ctx, found := findContextByName(contexts, "cluster-1")
		if !found {
			t.Error("Expected to find cluster-1")
		}
		if ctx.Name != "cluster-1" {
			t.Errorf("Expected name 'cluster-1', got '%s'", ctx.Name)
		}
		if ctx.FilePath != "/tmp/config1" {
			t.Errorf("Expected path '/tmp/config1', got '%s'", ctx.FilePath)
		}
	})

	t.Run("context not found", func(t *testing.T) {
		_, found := findContextByName(contexts, "nonexistent")
		if found {
			t.Error("Should not find nonexistent context")
		}
	})
}

func TestContextOptions_Run_WarningSuppressedAfterMax(t *testing.T) {
	setupTestXDGDataHome(t)
	sm, err := state.NewManager()
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	existingKubeconfig, err := os.CreateTemp("", "kubert-existing-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		_ = existingKubeconfig.Close()
		_ = os.Remove(existingKubeconfig.Name())
	}()

	t.Setenv(kubert.ShellActiveEnvVar, "1")
	t.Setenv(kubert.ShellKubeconfigEnvVar, existingKubeconfig.Name())

	makeOpts := func(errBuf *bytes.Buffer) *ContextOptions {
		return &ContextOptions{
			Out:    &bytes.Buffer{},
			ErrOut: errBuf,
			Args:   []string{"ctx-b"},
			Config: config.Config{},
			ContextLoader: func() ([]kubeconfig.Context, error) {
				return []kubeconfig.Context{
					{Name: "ctx-b", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config"}},
				}, nil
			},
			StateManager:   func() (*state.Manager, error) { return sm, nil },
			IsInteractive:  func() bool { return false },
			ShellLauncher:  func(_, _, _ string, _ config.Config) error { return nil },
			TempFileWriter: func(_, _, _ string) (*os.File, func(), error) { return nil, nil, nil },
			InPlaceWriter:  func(_, _, _, _ string) error { return nil },
		}
	}

	// First 3 invocations should print a notice to stderr.
	for i := 1; i <= state.InPlaceSwitchWarnMax; i++ {
		var errBuf bytes.Buffer
		if err := makeOpts(&errBuf).Run(); err != nil {
			t.Fatalf("Run() invocation %d: %v", i, err)
		}
		if !strings.Contains(errBuf.String(), "Warning:") {
			t.Errorf("invocation %d: expected warning in stderr, got: %q", i, errBuf.String())
		}
	}

	// 4th invocation should produce no notice.
	var errBuf bytes.Buffer
	if err := makeOpts(&errBuf).Run(); err != nil {
		t.Fatalf("Run() invocation 4: %v", err)
	}
	if strings.Contains(errBuf.String(), "Warning:") {
		t.Errorf("invocation 4: warning should not appear after max, got: %q", errBuf.String())
	}
}

func TestContextOptions_Run_InPlaceSwitch(t *testing.T) {
	var buf bytes.Buffer
	inPlaceWriterCalled := false
	shellLauncherCalled := false

	// Create a real temp file to act as the existing kubeconfig managed by kubert.
	existingKubeconfig, err := os.CreateTemp("", "kubert-existing-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		_ = existingKubeconfig.Close()
		_ = os.Remove(existingKubeconfig.Name())
	}()

	t.Setenv(kubert.ShellActiveEnvVar, "1")
	t.Setenv(kubert.ShellKubeconfigEnvVar, existingKubeconfig.Name())

	o := &ContextOptions{
		Out:    &buf,
		ErrOut: &buf,
		Args:   []string{"ctx-b"},
		Config: config.Config{},
		ContextLoader: func() ([]kubeconfig.Context, error) {
			return []kubeconfig.Context{
				{Name: "ctx-b", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config"}},
			}, nil
		},
		StateManager: func() (*state.Manager, error) {
			setupTestXDGDataHome(t)
			return state.NewManager()
		},
		IsInteractive: func() bool { return false },
		ShellLauncher: func(_, _, _ string, _ config.Config) error {
			shellLauncherCalled = true
			return nil
		},
		TempFileWriter: func(_, _, _ string) (*os.File, func(), error) {
			t.Error("TempFileWriter should not be called for in-place switch")
			return nil, nil, nil
		},
		InPlaceWriter: func(_, contextName, _, targetPath string) error {
			inPlaceWriterCalled = true
			if contextName != "ctx-b" {
				t.Errorf("Expected context name 'ctx-b', got '%s'", contextName)
			}
			if targetPath != existingKubeconfig.Name() {
				t.Errorf("Expected target path %q, got %q", existingKubeconfig.Name(), targetPath)
			}
			return nil
		},
	}

	if err := o.Run(); err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}

	if !inPlaceWriterCalled {
		t.Error("InPlaceWriter should have been called for in-place switch")
	}
	if shellLauncherCalled {
		t.Error("ShellLauncher should not have been called for in-place switch")
	}
	if !strings.Contains(buf.String(), "ctx-b") {
		t.Errorf("Expected output to mention ctx-b, got: %q", buf.String())
	}
}

func TestContextOptions_Run_InPlaceSwitch_Nested(t *testing.T) {
	shellLauncherCalled := false

	existingKubeconfig, err := os.CreateTemp("", "kubert-existing-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		_ = existingKubeconfig.Close()
		_ = os.Remove(existingKubeconfig.Name())
	}()

	t.Setenv(kubert.ShellActiveEnvVar, "1")
	t.Setenv(kubert.ShellKubeconfigEnvVar, existingKubeconfig.Name())

	o := &ContextOptions{
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
		Args:   []string{"ctx-b"},
		Nested: true,
		Config: config.Config{},
		ContextLoader: func() ([]kubeconfig.Context, error) {
			return []kubeconfig.Context{
				{Name: "ctx-b", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config"}},
			}, nil
		},
		StateManager: func() (*state.Manager, error) {
			setupTestXDGDataHome(t)
			return state.NewManager()
		},
		IsInteractive: func() bool { return false },
		ShellLauncher: func(_, _, _ string, _ config.Config) error {
			shellLauncherCalled = true
			return nil
		},
		TempFileWriter: func(_, _, _ string) (*os.File, func(), error) {
			f, err := os.CreateTemp("", "test-*.yaml")
			if err != nil {
				return nil, nil, err
			}
			return f, func() { _ = os.Remove(f.Name()) }, nil
		},
		InPlaceWriter: func(_, _, _, _ string) error {
			t.Error("InPlaceWriter should not be called when --nested is set")
			return nil
		},
	}

	if err := o.Run(); err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}

	if !shellLauncherCalled {
		t.Error("ShellLauncher should have been called when --nested is set")
	}
}

func TestContextOptions_Run_InPlaceSwitch_HooksFire(t *testing.T) {
	preHookFired := false
	postHookFired := false

	existingKubeconfig, err := os.CreateTemp("", "kubert-existing-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		_ = existingKubeconfig.Close()
		_ = os.Remove(existingKubeconfig.Name())
	}()

	t.Setenv(kubert.ShellActiveEnvVar, "1")
	t.Setenv(kubert.ShellKubeconfigEnvVar, existingKubeconfig.Name())

	preFile := filepath.Join(t.TempDir(), "pre-fired")
	postFile := filepath.Join(t.TempDir(), "post-fired")

	o := &ContextOptions{
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
		Args:   []string{"ctx-b"},
		Config: config.Config{
			Hooks: config.Hooks{
				PreShell:  fmt.Sprintf("touch %s", preFile),
				PostShell: fmt.Sprintf("touch %s", postFile),
			},
		},
		ContextLoader: func() ([]kubeconfig.Context, error) {
			return []kubeconfig.Context{
				{Name: "ctx-b", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config"}},
			}, nil
		},
		StateManager: func() (*state.Manager, error) {
			setupTestXDGDataHome(t)
			return state.NewManager()
		},
		IsInteractive: func() bool { return false },
		ShellLauncher: func(_, _, _ string, _ config.Config) error {
			t.Error("ShellLauncher should not be called for in-place switch")
			return nil
		},
		TempFileWriter: func(_, _, _ string) (*os.File, func(), error) {
			t.Error("TempFileWriter should not be called for in-place switch")
			return nil, nil, nil
		},
		InPlaceWriter: func(_, _, _, _ string) error { return nil },
	}

	if err := o.Run(); err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}

	if _, err := os.Stat(preFile); err == nil {
		preHookFired = true
	}
	if _, err := os.Stat(postFile); err == nil {
		postHookFired = true
	}

	if !preHookFired {
		t.Error("pre-context hook should have fired on in-place switch")
	}
	if !postHookFired {
		t.Error("post-context hook should have fired on in-place switch")
	}
}

func TestContextOptions_Run_InPlaceSwitch_MissingKubeconfigEnvVar(t *testing.T) {
	t.Setenv(kubert.ShellActiveEnvVar, "1")
	// Deliberately not setting KUBERT_SHELL_KUBECONFIG.

	o := &ContextOptions{
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
		Args:   []string{"ctx-b"},
		Config: config.Config{},
		ContextLoader: func() ([]kubeconfig.Context, error) {
			return []kubeconfig.Context{
				{Name: "ctx-b", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config"}},
			}, nil
		},
		StateManager: func() (*state.Manager, error) {
			setupTestXDGDataHome(t)
			return state.NewManager()
		},
		IsInteractive: func() bool { return false },
	}

	err := o.Run()
	if err == nil {
		t.Fatal("Expected error when KUBERT_SHELL_KUBECONFIG is not set, got nil")
	}
	if !strings.Contains(err.Error(), "KUBERT_SHELL_KUBECONFIG not set") {
		t.Errorf("Unexpected error message: %v", err)
	}
}
