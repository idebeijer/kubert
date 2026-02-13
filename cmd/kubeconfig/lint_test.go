package kubeconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestValidateKubeconfig(t *testing.T) {
	tests := []struct {
		name     string
		config   *api.Config
		wantErrs int
		wantWarn int
		contains string
	}{
		{
			name: "valid kubeconfig",
			config: &api.Config{
				Clusters:       map[string]*api.Cluster{"cluster": {Server: "https://localhost:6443"}},
				AuthInfos:      map[string]*api.AuthInfo{"user": {}},
				Contexts:       map[string]*api.Context{"ctx": {Cluster: "cluster", AuthInfo: "user"}},
				CurrentContext: "ctx",
			},
			wantErrs: 0,
			wantWarn: 0,
		},
		{
			name: "missing cluster reference",
			config: &api.Config{
				Clusters:  map[string]*api.Cluster{},
				AuthInfos: map[string]*api.AuthInfo{"user": {}},
				Contexts:  map[string]*api.Context{"ctx": {Cluster: "missing", AuthInfo: "user"}},
			},
			wantErrs: 1,
			wantWarn: 1,
			contains: "non-existent cluster",
		},
		{
			name: "invalid current-context",
			config: &api.Config{
				Clusters:       map[string]*api.Cluster{"cluster": {Server: "https://localhost:6443"}},
				AuthInfos:      map[string]*api.AuthInfo{"user": {}},
				Contexts:       map[string]*api.Context{"ctx": {Cluster: "cluster", AuthInfo: "user"}},
				CurrentContext: "missing",
			},
			wantErrs: 1,
			contains: "does not exist",
		},
		{
			name: "cluster missing server",
			config: &api.Config{
				Clusters:  map[string]*api.Cluster{"cluster": {Server: ""}},
				AuthInfos: map[string]*api.AuthInfo{"user": {}},
				Contexts:  map[string]*api.Context{"ctx": {Cluster: "cluster", AuthInfo: "user"}},
			},
			wantErrs: 1,
			contains: "no server URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &lintResult{Errors: []string{}, Warnings: []string{}}
			validateKubeconfig(tt.config, result)

			if len(result.Errors) != tt.wantErrs {
				t.Errorf("got %d errors, want %d. Errors: %v", len(result.Errors), tt.wantErrs, result.Errors)
			}
			if len(result.Warnings) != tt.wantWarn {
				t.Errorf("got %d warnings, want %d. Warnings: %v", len(result.Warnings), tt.wantWarn, result.Warnings)
			}
			if tt.contains != "" && !containsAny(append(result.Errors, result.Warnings...), tt.contains) {
				t.Errorf("expected message containing %q, got: %v", tt.contains, append(result.Errors, result.Warnings...))
			}
		})
	}
}

func TestLintFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Valid config
	validPath := filepath.Join(tmpDir, "valid.yaml")
	valid := &api.Config{
		Clusters:  map[string]*api.Cluster{"cluster": {Server: "https://localhost:6443"}},
		AuthInfos: map[string]*api.AuthInfo{"user": {}},
		Contexts:  map[string]*api.Context{"ctx": {Cluster: "cluster", AuthInfo: "user"}},
	}
	if err := clientcmd.WriteToFile(*valid, validPath); err != nil {
		t.Fatal(err)
	}

	// Invalid config
	invalidPath := filepath.Join(tmpDir, "invalid.yaml")
	invalid := &api.Config{
		Clusters:  map[string]*api.Cluster{},
		AuthInfos: map[string]*api.AuthInfo{"user": {}},
		Contexts:  map[string]*api.Context{"ctx": {Cluster: "missing", AuthInfo: "user"}},
	}
	if err := clientcmd.WriteToFile(*invalid, invalidPath); err != nil {
		t.Fatal(err)
	}

	// Test multiple files
	results := lintFiles([]string{validPath, invalidPath})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Valid file should have no errors
	for _, r := range results {
		if r.FilePath == validPath && len(r.Errors) > 0 {
			t.Errorf("valid file has errors: %v", r.Errors)
		}
		if r.FilePath == invalidPath && len(r.Errors) == 0 {
			t.Error("invalid file has no errors")
		}
	}

	// Test non-existent file
	results = lintFiles([]string{filepath.Join(tmpDir, "missing.yaml")})
	if len(results) != 1 || len(results[0].Errors) == 0 {
		t.Error("non-existent file should produce error")
	}
}

func TestExpandGlobs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	for _, name := range []string{"config1.yaml", "config2.yaml", "other.txt"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Test glob expansion
	results, err := expandGlobs([]string{filepath.Join(tmpDir, "*.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 yaml files, got %d: %v", len(results), results)
	}

	// Test tilde expansion
	home, _ := os.UserHomeDir()
	results, err = expandGlobs([]string{"~/.kube/config"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !strings.HasPrefix(results[0], home) {
		t.Errorf("tilde expansion failed, got: %v", results)
	}
}

func containsAny(slice []string, substr string) bool {
	for _, s := range slice {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
