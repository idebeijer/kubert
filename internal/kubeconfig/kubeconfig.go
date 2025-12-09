package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// WithPath struct to hold the kubeconfig and its file path
type WithPath struct {
	Config   *api.Config
	FilePath string
}

type Context struct {
	Name string
	WithPath
}

// Provider interface for different kubeconfig sources
type Provider interface {
	Load() ([]WithPath, error)
}

// FileSystemProvider struct to load kubeconfigs from filesystem
type FileSystemProvider struct {
	IncludePatterns []string
	ExcludePatterns []string
}

// NewFileSystemProvider returns a new FileSystemProvider
func NewFileSystemProvider(IncludePatterns, ExcludePatterns []string) *FileSystemProvider {
	return &FileSystemProvider{
		IncludePatterns: IncludePatterns,
		ExcludePatterns: ExcludePatterns,
	}
}

func (f *FileSystemProvider) Load() ([]WithPath, error) {
	var kubeconfigs []WithPath
	loadedPaths := make(map[string]bool)

	// Include default 'recommended' kubeconfig path
	defaultKubeconfigPath := clientcmd.RecommendedHomeFile
	if _, err := os.Stat(defaultKubeconfigPath); err == nil {
		defaultKubeconfig, err := clientcmd.LoadFromFile(defaultKubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load default kubeconfig: %w", err)
		}

		absPath, err := filepath.Abs(defaultKubeconfigPath)
		if err != nil {
			absPath = defaultKubeconfigPath
		}

		kubeconfigs = append(kubeconfigs, WithPath{Config: defaultKubeconfig, FilePath: defaultKubeconfigPath})
		loadedPaths[absPath] = true
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to stat default kubeconfig: %w", err)
	}

	files, err := findFiles(f.IncludePatterns)
	if err != nil {
		return nil, err
	}

	filteredFiles, err := filterFiles(files, f.ExcludePatterns)
	if err != nil {
		return nil, err
	}

	for _, file := range filteredFiles {
		absPath, err := filepath.Abs(file)
		if err != nil {
			absPath = file
		}

		if loadedPaths[absPath] {
			continue
		}

		kubeconfig, err := clientcmd.LoadFromFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", file, err)
		}
		kubeconfigs = append(kubeconfigs, WithPath{Config: kubeconfig, FilePath: file})
		loadedPaths[absPath] = true
	}

	return kubeconfigs, nil
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory for path %s: %w", path, err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func findFiles(patterns []string) ([]string, error) {
	var files []string
	for _, pattern := range patterns {
		expandedPattern, err := expandPath(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to expand pattern %s: %w", pattern, err)
		}
		matches, err := filepath.Glob(expandedPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to glob pattern %s: %w", expandedPattern, err)
		}
		files = append(files, matches...)
	}
	return files, nil
}

func filterFiles(files []string, excludePatterns []string) ([]string, error) {
	var filteredFiles []string
	excludeMap := make(map[string]bool)
	for _, pattern := range excludePatterns {
		expandedPattern, err := expandPath(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to expand exclude pattern %s: %w", pattern, err)
		}
		matches, err := filepath.Glob(expandedPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to glob exclude pattern %s: %w", expandedPattern, err)
		}
		for _, match := range matches {
			excludeMap[match] = true
		}
	}
	for _, file := range files {
		if !excludeMap[file] {
			info, err := os.Stat(file)
			if err != nil {
				return nil, fmt.Errorf("failed to stat file %s: %w", file, err)
			}
			if !info.IsDir() {
				filteredFiles = append(filteredFiles, file)
			}
		}
	}
	return filteredFiles, nil
}

// Loader struct to handle multiple providers
type Loader struct {
	Providers []Provider
}

// LoaderOption is a functional option for configuring a Loader
type LoaderOption func(*Loader)

// WithProvider adds a provider to the loader
func WithProvider(provider Provider) LoaderOption {
	return func(l *Loader) {
		l.Providers = append(l.Providers, provider)
	}
}

// NewLoader creates a new Loader with the given options
func NewLoader(options ...LoaderOption) *Loader {
	loader := &Loader{
		Providers: make([]Provider, 0),
	}

	for _, option := range options {
		option(loader)
	}

	return loader
}

// LoadAll method to load all kubeconfigs from all providers
func (l *Loader) LoadAll() ([]WithPath, error) {
	var allKubeconfigs []WithPath
	for _, provider := range l.Providers {
		kubeconfigs, err := provider.Load()
		if err != nil {
			return nil, fmt.Errorf("provider failed to load kubeconfigs: %w", err)
		}
		allKubeconfigs = append(allKubeconfigs, kubeconfigs...)
	}

	return allKubeconfigs, nil
}

func (l *Loader) LoadContexts() ([]Context, error) {
	allKubeconfigs, err := l.LoadAll()
	if err != nil {
		return nil, err
	}

	var contexts []Context
	contextSources := make(map[string]string)

	for _, kubeconfig := range allKubeconfigs {
		if kubeconfig.Config.Contexts == nil {
			continue
		}
		for contextName := range kubeconfig.Config.Contexts {
			if contextName == "" {
				continue
			}

			if existingSource, exists := contextSources[contextName]; exists {
				return nil, fmt.Errorf(
					"duplicate context name %q found:\n  - %s\n  - %s\n\n"+
						"Kubert requires unique context names across all kubeconfig files.\n"+
						"Please rename one of these contexts to avoid conflicts",
					contextName, existingSource, kubeconfig.FilePath)
			}

			contextSources[contextName] = kubeconfig.FilePath
			contexts = append(contexts, Context{
				Name:     contextName,
				WithPath: kubeconfig,
			})
		}
	}

	return contexts, nil
}
