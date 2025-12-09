package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// MockProvider is a mock implementation of the Provider interface for testing.
type MockProvider struct {
	kubeconfigs []WithPath
}

func (m *MockProvider) Load() ([]WithPath, error) {
	return m.kubeconfigs, nil
}

func TestFileSystemProvider_Load(t *testing.T) {
	dir, err := os.MkdirTemp("", "kubeconfig")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	kubeconfigPath := filepath.Join(dir, "config")
	kubeconfig := &api.Config{}
	if err := clientcmd.WriteToFile(*kubeconfig, kubeconfigPath); err != nil {
		t.Fatal(err)
	}

	includePatterns := []string{fmt.Sprintf("%s/*", dir)}

	provider := NewFileSystemProvider(includePatterns, nil)
	kubeconfigs, err := provider.Load()
	if err != nil {
		t.Fatal(err)
	}

	// Filter out the default kubeconfig if present
	var filteredKubeconfigs []WithPath
	for _, k := range kubeconfigs {
		if k.FilePath == kubeconfigPath {
			filteredKubeconfigs = append(filteredKubeconfigs, k)
		}
	}
	kubeconfigs = filteredKubeconfigs

	// Check that the correct kubeconfig file was loaded.
	if len(kubeconfigs) != 1 || kubeconfigs[0].FilePath != kubeconfigPath {
		t.Errorf("Load() = %v, want %v", kubeconfigs, []WithPath{{Config: kubeconfig, FilePath: kubeconfigPath}})
	}
}

func TestLoader_LoadAll(t *testing.T) {
	kubeconfig := &api.Config{}
	mockProvider := &MockProvider{kubeconfigs: []WithPath{{Config: kubeconfig, FilePath: "config"}}}

	loader := NewLoader(WithProvider(mockProvider))
	kubeconfigs, err := loader.LoadAll()
	if err != nil {
		t.Fatal(err)
	}

	// Check that the correct kubeconfig files were loaded.
	if len(kubeconfigs) != 1 || kubeconfigs[0].FilePath != "config" {
		t.Errorf("LoadAll() = %v, want %v", kubeconfigs, mockProvider.kubeconfigs)
	}
}

func TestFileSystemProvider_Load_NoFilesMatchIncludePatterns(t *testing.T) {
	provider := NewFileSystemProvider([]string{"nonexistent"}, nil)
	kubeconfigs, err := provider.Load()
	if err != nil {
		t.Fatal(err)
	}

	// Filter out the default kubeconfig if present
	var filteredKubeconfigs []WithPath
	for _, k := range kubeconfigs {
		if k.FilePath != clientcmd.RecommendedHomeFile {
			filteredKubeconfigs = append(filteredKubeconfigs, k)
		}
	}
	kubeconfigs = filteredKubeconfigs

	if len(kubeconfigs) != 0 {
		t.Errorf("Load() = %v, want %v", kubeconfigs, []WithPath{})
	}
}

func TestFileSystemProvider_Load_AllFilesMatchExcludePatterns(t *testing.T) {
	dir, err := os.MkdirTemp("", "kubeconfig")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	kubeconfigPath := filepath.Join(dir, "config")
	kubeconfig := &api.Config{}
	if err := clientcmd.WriteToFile(*kubeconfig, kubeconfigPath); err != nil {
		t.Fatal(err)
	}

	excludePatterns := []string{fmt.Sprintf("%s/*", dir)}

	provider := NewFileSystemProvider([]string{fmt.Sprintf("%s/*", dir)}, excludePatterns)
	kubeconfigs, err := provider.Load()
	if err != nil {
		t.Fatal(err)
	}

	// Filter out the default kubeconfig if present
	var filteredKubeconfigs []WithPath
	for _, k := range kubeconfigs {
		if k.FilePath != clientcmd.RecommendedHomeFile {
			filteredKubeconfigs = append(filteredKubeconfigs, k)
		}
	}
	kubeconfigs = filteredKubeconfigs

	if len(kubeconfigs) != 0 {
		t.Errorf("Load() = %v, want %v", kubeconfigs, []WithPath{})
	}
}

func TestLoader_LoadContexts_DuplicateDetection(t *testing.T) {
	kubeconfig1 := &api.Config{
		Contexts: map[string]*api.Context{
			"prod-cluster": {Cluster: "cluster1"},
			"dev-cluster":  {Cluster: "cluster2"},
		},
	}

	kubeconfig2 := &api.Config{
		Contexts: map[string]*api.Context{
			"prod-cluster": {Cluster: "cluster3"}, // Duplicate name
			"staging":      {Cluster: "cluster4"},
		},
	}

	mockProvider := &MockProvider{kubeconfigs: []WithPath{
		{Config: kubeconfig1, FilePath: "/path/to/config1"},
		{Config: kubeconfig2, FilePath: "/path/to/config2"},
	}}

	loader := NewLoader(WithProvider(mockProvider))
	_, err := loader.LoadContexts()

	if err == nil {
		t.Fatal("LoadContexts() should return error for duplicate context names")
	}

	expectedError := "duplicate context name \"prod-cluster\" found"
	if !contains(err.Error(), expectedError) {
		t.Errorf("LoadContexts() error = %v, want error containing %q", err, expectedError)
	}

	if !contains(err.Error(), "/path/to/config1") {
		t.Errorf("LoadContexts() error should mention first file path, got: %v", err)
	}

	if !contains(err.Error(), "/path/to/config2") {
		t.Errorf("LoadContexts() error should mention second file path, got: %v", err)
	}
}

func TestLoader_LoadContexts_NoDuplicates(t *testing.T) {
	kubeconfig1 := &api.Config{
		Contexts: map[string]*api.Context{
			"prod-cluster": {Cluster: "cluster1"},
			"dev-cluster":  {Cluster: "cluster2"},
		},
	}

	kubeconfig2 := &api.Config{
		Contexts: map[string]*api.Context{
			"staging-cluster": {Cluster: "cluster3"},
			"test-cluster":    {Cluster: "cluster4"},
		},
	}

	mockProvider := &MockProvider{kubeconfigs: []WithPath{
		{Config: kubeconfig1, FilePath: "/path/to/config1"},
		{Config: kubeconfig2, FilePath: "/path/to/config2"},
	}}

	loader := NewLoader(WithProvider(mockProvider))
	contexts, err := loader.LoadContexts()
	if err != nil {
		t.Fatalf("LoadContexts() unexpected error: %v", err)
	}

	if len(contexts) != 4 {
		t.Errorf("LoadContexts() returned %d contexts, want 4", len(contexts))
	}

	expectedNames := map[string]bool{
		"prod-cluster":    false,
		"dev-cluster":     false,
		"staging-cluster": false,
		"test-cluster":    false,
	}

	for _, ctx := range contexts {
		if _, exists := expectedNames[ctx.Name]; !exists {
			t.Errorf("unexpected context name: %s", ctx.Name)
		}
		expectedNames[ctx.Name] = true
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("expected context %s not found", name)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
