package cmd

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/state"
)

func TestNamespaceOptions_Complete_SetsConfig(t *testing.T) {
	original := config.Cfg
	defer func() { config.Cfg = original }()

	config.Cfg = config.Config{
		KubeconfigPaths: config.KubeconfigPaths{
			Include: []string{"/some/path"},
		},
	}

	o := NewNamespaceOptions()
	cmd := &cobra.Command{}

	if len(o.Config.KubeconfigPaths.Include) != 0 {
		t.Error("Config should not be set before Complete()")
	}

	if err := o.Complete(cmd, nil); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if len(o.Config.KubeconfigPaths.Include) != 1 || o.Config.KubeconfigPaths.Include[0] != "/some/path" {
		t.Errorf("Config was not set from config.Cfg in Complete(), got %v", o.Config.KubeconfigPaths.Include)
	}
}

func TestNamespaceOptions_Run_WithArg(t *testing.T) {
	setupTestXDGDataHome(t)

	namespaces := []string{"default", "kube-system", "kube-public"}
	switchCalled := false
	var switchedNamespace string

	o := &NamespaceOptions{
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
		Args:   []string{"kube-system"},
		NamespaceLister: func(ctx context.Context) ([]string, error) {
			return namespaces, nil
		},
		StateManager: state.NewManager,
		NamespaceSwitcher: func(sm *state.Manager, namespace string, ns []string) error {
			switchCalled = true
			switchedNamespace = namespace
			return nil
		},
		IsInteractive: func() bool { return true },
	}

	if err := o.Run(); err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	if !switchCalled {
		t.Error("NamespaceSwitcher was not called")
	}
	if switchedNamespace != "kube-system" {
		t.Errorf("switched to %q, want %q", switchedNamespace, "kube-system")
	}
}

func TestNamespaceOptions_Run_Interactive(t *testing.T) {
	setupTestXDGDataHome(t)

	namespaces := []string{"default", "kube-system", "monitoring"}
	switchCalled := false
	var switchedNamespace string

	o := &NamespaceOptions{
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
		Args:   nil,
		NamespaceLister: func(ctx context.Context) ([]string, error) {
			return namespaces, nil
		},
		IsInteractive: func() bool { return true },
		Selector: func(items []string) (string, error) {
			return "monitoring", nil
		},
		StateManager: state.NewManager,
		NamespaceSwitcher: func(sm *state.Manager, namespace string, ns []string) error {
			switchCalled = true
			switchedNamespace = namespace
			return nil
		},
	}

	if err := o.Run(); err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	if !switchCalled {
		t.Error("NamespaceSwitcher was not called")
	}
	if switchedNamespace != "monitoring" {
		t.Errorf("switched to %q, want %q", switchedNamespace, "monitoring")
	}
}

func TestNamespaceOptions_Run_NonInteractive_PrintsNamespaces(t *testing.T) {
	namespaces := []string{"default", "kube-system", "production"}
	var out bytes.Buffer

	o := &NamespaceOptions{
		Out:    &out,
		ErrOut: &bytes.Buffer{},
		Args:   nil,
		NamespaceLister: func(ctx context.Context) ([]string, error) {
			return namespaces, nil
		},
		IsInteractive: func() bool { return false },
		StateManager:  state.NewManager,
		NamespaceSwitcher: func(sm *state.Manager, namespace string, ns []string) error {
			t.Error("NamespaceSwitcher should not be called in non-interactive mode without args")
			return nil
		},
	}

	if err := o.Run(); err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	output := out.String()
	for _, ns := range namespaces {
		if !strings.Contains(output, ns) {
			t.Errorf("output missing namespace %q, got: %s", ns, output)
		}
	}
}

func TestNamespaceOptions_Run_ListerError(t *testing.T) {
	o := &NamespaceOptions{
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
		NamespaceLister: func(ctx context.Context) ([]string, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	err := o.Run()
	if err == nil {
		t.Fatal("Run() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("error = %v, want to contain 'connection refused'", err)
	}
}

func TestNamespaceOptions_Run_SelectorError(t *testing.T) {
	o := &NamespaceOptions{
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
		Args:   nil,
		NamespaceLister: func(ctx context.Context) ([]string, error) {
			return []string{"default"}, nil
		},
		IsInteractive: func() bool { return true },
		Selector: func(items []string) (string, error) {
			return "", fmt.Errorf("fzf not found")
		},
	}

	err := o.Run()
	if err == nil {
		t.Fatal("Run() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "fzf not found") {
		t.Errorf("error = %v, want to contain 'fzf not found'", err)
	}
}

func TestNamespaceOptions_Run_StateManagerError(t *testing.T) {
	o := &NamespaceOptions{
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
		Args:   []string{"default"},
		NamespaceLister: func(ctx context.Context) ([]string, error) {
			return []string{"default", "kube-system"}, nil
		},
		StateManager: func() (*state.Manager, error) {
			return nil, fmt.Errorf("state dir not writable")
		},
	}

	err := o.Run()
	if err == nil {
		t.Fatal("Run() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "state dir not writable") {
		t.Errorf("error = %v, want to contain 'state dir not writable'", err)
	}
}

func TestNamespaceOptions_Run_SwitcherError(t *testing.T) {
	setupTestXDGDataHome(t)

	o := &NamespaceOptions{
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
		Args:   []string{"default"},
		NamespaceLister: func(ctx context.Context) ([]string, error) {
			return []string{"default"}, nil
		},
		StateManager: state.NewManager,
		NamespaceSwitcher: func(sm *state.Manager, namespace string, ns []string) error {
			return fmt.Errorf("write failed")
		},
	}

	err := o.Run()
	if err == nil {
		t.Fatal("Run() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "write failed") {
		t.Errorf("error = %v, want to contain 'write failed'", err)
	}
}

func TestSwitchNamespace(t *testing.T) {
	setupTestXDGDataHome(t)

	// Create a kubeconfig file with a context
	kubeconfigPath := filepath.Join(t.TempDir(), "kubeconfig")
	kubecfg := api.NewConfig()
	kubecfg.CurrentContext = "test-context"
	kubecfg.Contexts["test-context"] = &api.Context{
		Cluster:   "test-cluster",
		AuthInfo:  "test-user",
		Namespace: "default",
	}
	kubecfg.Clusters["test-cluster"] = &api.Cluster{Server: "https://localhost:6443"}
	kubecfg.AuthInfos["test-user"] = &api.AuthInfo{Token: "test-token"}

	if err := clientcmd.WriteToFile(*kubecfg, kubeconfigPath); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	// Set KUBECONFIG env var for switchNamespace
	t.Setenv("KUBECONFIG", kubeconfigPath)

	sm, err := state.NewManager()
	if err != nil {
		t.Fatalf("failed to create state manager: %v", err)
	}

	namespaces := []string{"default", "kube-system", "production"}

	t.Run("switch to existing namespace", func(t *testing.T) {
		if err := switchNamespace(sm, "kube-system", namespaces); err != nil {
			t.Fatalf("switchNamespace() unexpected error: %v", err)
		}

		// Verify the kubeconfig was updated
		cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
		if err != nil {
			t.Fatalf("failed to load kubeconfig: %v", err)
		}
		if cfg.Contexts["test-context"].Namespace != "kube-system" {
			t.Errorf("namespace = %q, want %q", cfg.Contexts["test-context"].Namespace, "kube-system")
		}
	})

	t.Run("switch to non-existing namespace", func(t *testing.T) {
		err := switchNamespace(sm, "nonexistent", namespaces)
		if err == nil {
			t.Fatal("switchNamespace() expected error, got nil")
		}
		if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("error = %v, want to contain 'does not exist'", err)
		}
	})
}

func TestSwitchNamespace_InvalidKubeconfig(t *testing.T) {
	setupTestXDGDataHome(t)

	t.Setenv("KUBECONFIG", "/nonexistent/path/kubeconfig")

	sm, err := state.NewManager()
	if err != nil {
		t.Fatalf("failed to create state manager: %v", err)
	}

	err = switchNamespace(sm, "default", []string{"default"})
	if err == nil {
		t.Fatal("switchNamespace() expected error for invalid kubeconfig path")
	}
}

func TestListNamespaces(t *testing.T) {
	t.Run("returns namespace names", func(t *testing.T) {
		clientset := fake.NewClientset(
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "production"}},
		)

		names, err := listNamespaces(context.Background(), clientset)
		if err != nil {
			t.Fatalf("listNamespaces() error = %v", err)
		}

		if len(names) != 3 {
			t.Fatalf("expected 3 namespaces, got %d: %v", len(names), names)
		}

		expected := map[string]bool{"default": true, "kube-system": true, "production": true}
		for _, name := range names {
			if !expected[name] {
				t.Errorf("unexpected namespace %q", name)
			}
		}
	})

	t.Run("returns empty slice for no namespaces", func(t *testing.T) {
		clientset := fake.NewClientset()

		names, err := listNamespaces(context.Background(), clientset)
		if err != nil {
			t.Fatalf("listNamespaces() error = %v", err)
		}

		if len(names) != 0 {
			t.Errorf("expected 0 namespaces, got %d: %v", len(names), names)
		}
	})
}
