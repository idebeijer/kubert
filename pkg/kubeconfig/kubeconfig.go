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
	files, err := findFiles(f.IncludePatterns)
	if err != nil {
		return nil, err
	}

	filteredFiles, err := filterFiles(files, f.ExcludePatterns)
	if err != nil {
		return nil, err
	}

	for _, file := range filteredFiles {
		kubeconfig, err := clientcmd.LoadFromFile(file)
		if err != nil {
			return nil, err
		}
		kubeconfigs = append(kubeconfigs, WithPath{Config: kubeconfig, FilePath: file})
	}

	return kubeconfigs, nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("failed to get home directory:", err)
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func findFiles(patterns []string) ([]string, error) {
	var files []string
	for _, pattern := range patterns {
		expandedPattern := expandPath(pattern)
		matches, err := filepath.Glob(expandedPattern)
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}
	return files, nil
}

func filterFiles(files []string, excludePatterns []string) ([]string, error) {
	var filteredFiles []string
	excludeMap := make(map[string]bool)
	for _, pattern := range excludePatterns {
		expandedPattern := expandPath(pattern)
		matches, err := filepath.Glob(expandedPattern)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			excludeMap[match] = true
		}
	}
	for _, file := range files {
		if !excludeMap[file] {
			filteredFiles = append(filteredFiles, file)
		}
	}
	return filteredFiles, nil
}

// Loader struct to handle multiple providers
type Loader struct {
	Providers []Provider
}

// NewLoader returns a new Loader with given providers
func NewLoader(providers ...Provider) *Loader {
	return &Loader{Providers: providers}
}

// LoadAll method to load all kubeconfigs from all providers
func (l *Loader) LoadAll() ([]WithPath, error) {
	var allKubeconfigs []WithPath
	for _, provider := range l.Providers {
		kubeconfigs, err := provider.Load()
		if err != nil {
			return nil, err
		}
		allKubeconfigs = append(allKubeconfigs, kubeconfigs...)
	}

	return allKubeconfigs, nil
}
