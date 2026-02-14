package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/idebeijer/kubert/internal/config"
)

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
	config := api.NewConfig()

	// Cluster 1 data
	config.Clusters["shared-cluster"] = &api.Cluster{Server: "https://shared.example.com"}

	config.AuthInfos["user-1"] = &api.AuthInfo{Token: "token-1"}
	config.Contexts["ctx-1"] = &api.Context{Cluster: "shared-cluster", AuthInfo: "user-1", Namespace: "ns-1"}

	config.AuthInfos["user-2"] = &api.AuthInfo{Username: "admin", Password: "password"}
	config.Contexts["ctx-2"] = &api.Context{Cluster: "shared-cluster", AuthInfo: "user-2", Namespace: "default"}

	// Cluster 2 data
	config.Clusters["other-cluster"] = &api.Cluster{Server: "https://other.example.com"}
	config.AuthInfos["user-3"] = &api.AuthInfo{ClientCertificate: "/path/to/cert"}
	config.Contexts["ctx-3"] = &api.Context{Cluster: "other-cluster", AuthInfo: "user-3"}

	// Write this original config to disk
	if err := clientcmd.WriteToFile(*config, tempFile.Name()); err != nil {
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

		if err := runNamespaceCommand([]string{"default"}); err != nil {
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
