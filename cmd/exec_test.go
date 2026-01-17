package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/kubeconfig"
	"github.com/idebeijer/kubert/internal/state"
)

func TestGlobToRegex(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		input    string
		expected bool
	}{
		{
			name:     "simple wildcard",
			pattern:  "prod*",
			input:    "prod-east",
			expected: true,
		},
		{
			name:     "wildcard no match",
			pattern:  "prod*",
			input:    "staging-east",
			expected: false,
		},
		{
			name:     "question mark wildcard",
			pattern:  "prod-?",
			input:    "prod-a",
			expected: true,
		},
		{
			name:     "question mark no match",
			pattern:  "prod-?",
			input:    "prod-ab",
			expected: false,
		},
		{
			name:     "multiple wildcards",
			pattern:  "prod-*-db*",
			input:    "prod-east-db-cluster",
			expected: true,
		},
		{
			name:     "exact match",
			pattern:  "production",
			input:    "production",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regexPattern := globToRegex(tt.pattern)
			matched, err := regexp.MatchString(regexPattern, tt.input)
			if err != nil {
				t.Fatalf("failed to compile regex: %v", err)
			}
			if matched != tt.expected {
				t.Errorf("pattern %q against %q: got %v, want %v", tt.pattern, tt.input, matched, tt.expected)
			}
		})
	}
}

func TestFilterContextsByPattern(t *testing.T) {
	contexts := []kubeconfig.Context{
		{Name: "prod-east", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config1"}},
		{Name: "prod-west", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config1"}},
		{Name: "staging-east", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config2"}},
		{Name: "dev-local", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config3"}},
	}

	tests := []struct {
		name          string
		pattern       string
		useRegex      bool
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "glob wildcard matches multiple",
			pattern:       "prod*",
			useRegex:      false,
			expectedCount: 2,
			expectedNames: []string{"prod-east", "prod-west"},
		},
		{
			name:          "glob wildcard matches one",
			pattern:       "dev*",
			useRegex:      false,
			expectedCount: 1,
			expectedNames: []string{"dev-local"},
		},
		{
			name:          "glob wildcard no matches",
			pattern:       "test*",
			useRegex:      false,
			expectedCount: 0,
			expectedNames: []string{},
		},
		{
			name:          "regex pattern",
			pattern:       "^(prod|staging)-.*",
			useRegex:      true,
			expectedCount: 3,
			expectedNames: []string{"prod-east", "prod-west", "staging-east"},
		},
		{
			name:          "glob matches all ending with -east",
			pattern:       "*-east",
			useRegex:      false,
			expectedCount: 2,
			expectedNames: []string{"prod-east", "staging-east"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, err := filterContextsByPattern(contexts, tt.pattern, tt.useRegex)
			if err != nil {
				t.Fatalf("filterContextsByPattern failed: %v", err)
			}

			if len(matched) != tt.expectedCount {
				t.Errorf("expected %d matches, got %d", tt.expectedCount, len(matched))
			}

			matchedNames := make([]string, len(matched))
			for i, ctx := range matched {
				matchedNames[i] = ctx.Name
			}

			if len(tt.expectedNames) > 0 && !stringSlicesEqual(matchedNames, tt.expectedNames) {
				t.Errorf("expected names %v, got %v", tt.expectedNames, matchedNames)
			}
		})
	}
}

func TestFilterContextsByPatterns(t *testing.T) {
	contexts := []kubeconfig.Context{
		{Name: "prod-east", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config1"}},
		{Name: "prod-west", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config1"}},
		{Name: "staging-east", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config2"}},
		{Name: "staging-west", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config2"}},
		{Name: "dev-local", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config3"}},
		{Name: "test-cluster", WithPath: kubeconfig.WithPath{FilePath: "/tmp/config4"}},
	}

	tests := []struct {
		name          string
		patterns      []string
		useRegex      bool
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "single pattern",
			patterns:      []string{"prod*"},
			useRegex:      false,
			expectedCount: 2,
			expectedNames: []string{"prod-east", "prod-west"},
		},
		{
			name:          "multiple patterns",
			patterns:      []string{"prod*", "staging*"},
			useRegex:      false,
			expectedCount: 4,
			expectedNames: []string{"prod-east", "prod-west", "staging-east", "staging-west"},
		},
		{
			name:          "multiple patterns with overlap",
			patterns:      []string{"prod*", "*-east"},
			useRegex:      false,
			expectedCount: 3,
			expectedNames: []string{"prod-east", "prod-west", "staging-east"},
		},
		{
			name:          "multiple patterns no duplicates",
			patterns:      []string{"prod-east", "prod-*"},
			useRegex:      false,
			expectedCount: 2,
			expectedNames: []string{"prod-east", "prod-west"},
		},
		{
			name:          "multiple patterns with regex",
			patterns:      []string{"^prod-.*", "^test-.*"},
			useRegex:      true,
			expectedCount: 3,
			expectedNames: []string{"prod-east", "prod-west", "test-cluster"},
		},
		{
			name:          "all patterns match nothing",
			patterns:      []string{"nonexistent*", "alsonothere*"},
			useRegex:      false,
			expectedCount: 0,
			expectedNames: []string{},
		},
		{
			name:          "mix of matching and non-matching patterns",
			patterns:      []string{"prod*", "nonexistent*", "dev*"},
			useRegex:      false,
			expectedCount: 3,
			expectedNames: []string{"dev-local", "prod-east", "prod-west"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, err := filterContextsByPatterns(contexts, tt.patterns, tt.useRegex)
			if err != nil {
				t.Fatalf("filterContextsByPatterns failed: %v", err)
			}

			if len(matched) != tt.expectedCount {
				t.Errorf("expected %d matches, got %d", tt.expectedCount, len(matched))
			}

			matchedNames := make([]string, len(matched))
			for i, ctx := range matched {
				matchedNames[i] = ctx.Name
			}

			if len(tt.expectedNames) > 0 && !stringSlicesEqual(matchedNames, tt.expectedNames) {
				t.Errorf("expected names %v, got %v", tt.expectedNames, matchedNames)
			}
		})
	}
}

func TestExecuteInContextKubeconfigSetup(t *testing.T) {
	tempDir := t.TempDir()

	kubeconfigPath := filepath.Join(tempDir, "test-config")
	cfg := createTestKubeconfig(t, kubeconfigPath, "test-context", "test-cluster", "test-user")

	contexts := []kubeconfig.Context{
		{
			Name: "test-context",
			WithPath: kubeconfig.WithPath{
				Config:   cfg,
				FilePath: kubeconfigPath,
			},
		},
	}

	sm, err := state.NewManager()
	if err != nil {
		t.Fatalf("failed to create state manager: %v", err)
	}

	testConfig := config.Config{}

	args := []string{"sh", "-c", "echo $KUBECONFIG"}

	result := executeInContext(contexts[0], args, "", sm, testConfig)

	if result.err != nil {
		t.Fatalf("executeInContext failed: %v", result.err)
	}

	kubeconfigEnv := strings.TrimSpace(result.output)
	if kubeconfigEnv == "" {
		t.Fatal("KUBECONFIG environment variable not set")
	}

	if kubeconfigEnv == kubeconfigPath {
		t.Fatal("KUBECONFIG should point to temp file, not original")
	}

	if !strings.Contains(kubeconfigEnv, "kubert-") {
		t.Errorf("expected temp kubeconfig path to contain 'kubert-', got: %s", kubeconfigEnv)
	}
}

func TestExecuteInContextWithNamespace(t *testing.T) {
	tempDir := t.TempDir()

	kubeconfigPath := filepath.Join(tempDir, "test-config")
	cfg := createTestKubeconfig(t, kubeconfigPath, "test-context", "test-cluster", "test-user")

	contexts := []kubeconfig.Context{
		{
			Name: "test-context",
			WithPath: kubeconfig.WithPath{
				Config:   cfg,
				FilePath: kubeconfigPath,
			},
		},
	}

	sm, err := state.NewManager()
	if err != nil {
		t.Fatalf("failed to create state manager: %v", err)
	}

	testConfig := config.Config{}
	namespace := "custom-namespace"

	args := []string{"sh", "-c", "cat $KUBECONFIG | grep namespace"}

	result := executeInContext(contexts[0], args, namespace, sm, testConfig)

	if result.err != nil {
		t.Fatalf("executeInContext failed: %v", result.err)
	}

	if !strings.Contains(result.output, namespace) {
		t.Errorf("expected namespace %q in kubeconfig, output: %s", namespace, result.output)
	}
}

func TestExecuteParallelIsolation(t *testing.T) {
	tempDir := t.TempDir()

	contexts := []kubeconfig.Context{}
	for i := range 5 {
		contextName := fmt.Sprintf("context-%d", i)
		kubeconfigPath := filepath.Join(tempDir, fmt.Sprintf("config-%d", i))
		cfg := createTestKubeconfig(t, kubeconfigPath, contextName, "cluster-"+contextName, "user-"+contextName)

		contexts = append(contexts, kubeconfig.Context{
			Name: contextName,
			WithPath: kubeconfig.WithPath{
				Config:   cfg,
				FilePath: kubeconfigPath,
			},
		})
	}

	sm, err := state.NewManager()
	if err != nil {
		t.Fatalf("failed to create state manager: %v", err)
	}

	testConfig := config.Config{}

	var wg sync.WaitGroup
	resultsChan := make(chan contextExecResult, len(contexts))

	kubeconfigPaths := make(map[string]string)
	var mu sync.Mutex

	for _, ctx := range contexts {
		wg.Add(1)
		go func(ctx kubeconfig.Context) {
			defer wg.Done()

			args := []string{"sh", "-c", "echo $KUBECONFIG"}
			result := executeInContext(ctx, args, "", sm, testConfig)

			mu.Lock()
			kubeconfigPath := strings.TrimSpace(result.output)
			kubeconfigPaths[ctx.Name] = kubeconfigPath
			mu.Unlock()

			resultsChan <- result
		}(ctx)
	}

	wg.Wait()
	close(resultsChan)

	results := []contextExecResult{}
	for result := range resultsChan {
		results = append(results, result)
		if result.err != nil {
			t.Errorf("context %s failed: %v", result.contextName, result.err)
		}
	}

	if len(results) != len(contexts) {
		t.Fatalf("expected %d results, got %d", len(contexts), len(results))
	}

	seenPaths := make(map[string]bool)
	for contextName, path := range kubeconfigPaths {
		if path == "" {
			t.Errorf("context %s has empty KUBECONFIG path", contextName)
			continue
		}

		if seenPaths[path] {
			t.Errorf("duplicate KUBECONFIG path %s found - contexts are not isolated", path)
		}
		seenPaths[path] = true

		if !strings.Contains(path, "kubert-") {
			t.Errorf("context %s: KUBECONFIG path should contain 'kubert-', got: %s", contextName, path)
		}
	}

	if len(seenPaths) != len(contexts) {
		t.Errorf("expected %d unique KUBECONFIG paths, got %d", len(contexts), len(seenPaths))
	}
}

func TestRunCommandEnvironmentIsolation(t *testing.T) {
	tempDir := t.TempDir()

	config1 := filepath.Join(tempDir, "config1")
	config2 := filepath.Join(tempDir, "config2")

	if err := os.WriteFile(config1, []byte("config1-content"), 0o600); err != nil {
		t.Fatalf("failed to write config1: %v", err)
	}
	if err := os.WriteFile(config2, []byte("config2-content"), 0o600); err != nil {
		t.Fatalf("failed to write config2: %v", err)
	}

	args := []string{"sh", "-c", "echo $KUBECONFIG"}

	output1, err := runCommand(args, config1)
	if err != nil {
		t.Fatalf("runCommand failed for config1: %v", err)
	}

	output2, err := runCommand(args, config2)
	if err != nil {
		t.Fatalf("runCommand failed for config2: %v", err)
	}

	if !strings.Contains(output1, config1) {
		t.Errorf("expected output1 to contain %s, got: %s", config1, output1)
	}

	if !strings.Contains(output2, config2) {
		t.Errorf("expected output2 to contain %s, got: %s", config2, output2)
	}

	if strings.TrimSpace(output1) == strings.TrimSpace(output2) {
		t.Error("KUBECONFIG values should be different for different contexts")
	}
}

func createTestKubeconfig(t *testing.T, path, contextName, clusterName, userName string) *api.Config {
	t.Helper()

	cfg := api.NewConfig()
	cfg.Clusters[clusterName] = &api.Cluster{
		Server: "https://test-server:6443",
	}
	cfg.AuthInfos[userName] = &api.AuthInfo{
		Token: "test-token",
	}
	cfg.Contexts[contextName] = &api.Context{
		Cluster:  clusterName,
		AuthInfo: userName,
	}
	cfg.CurrentContext = contextName

	if err := clientcmd.WriteToFile(*cfg, path); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	return cfg
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
