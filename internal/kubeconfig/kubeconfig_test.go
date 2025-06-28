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

	if len(kubeconfigs) != 0 {
		t.Errorf("Load() = %v, want %v", kubeconfigs, []WithPath{})
	}
}
