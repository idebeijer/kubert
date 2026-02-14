package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/kubeconfig"
	"github.com/idebeijer/kubert/internal/util"
)

type lintResult struct {
	FilePath string
	Errors   []string
	Warnings []string
}

func newLintCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "lint [file...]",
		Short:        "Lint kubeconfig files for errors and issues",
		SilenceUsage: true,
		Long: `Lint kubeconfig files to check for errors, warnings, and potential issues.

If no files are provided, all kubeconfig files from the configured include patterns will be linted.
If file paths are provided as arguments (including glob patterns), only those files will be linted.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var filesToLint []string

			if len(args) > 0 {
				var err error
				filesToLint, err = expandGlobs(args)
				if err != nil {
					return fmt.Errorf("failed to expand file patterns: %w", err)
				}
			} else {
				// Lint all included kubeconfig files
				cfg := config.Cfg
				fsProvider := kubeconfig.NewFileSystemProvider(cfg.KubeconfigPaths.Include, cfg.KubeconfigPaths.Exclude)
				loader := kubeconfig.NewLoader(kubeconfig.WithProvider(fsProvider))

				kubeconfigs, err := loader.LoadAll()
				if err != nil {
					return fmt.Errorf("failed to load kubeconfigs: %w", err)
				}

				for _, k := range kubeconfigs {
					filesToLint = append(filesToLint, k.FilePath)
				}
			}

			if len(filesToLint) == 0 {
				fmt.Println("No kubeconfig files to lint")
				return nil
			}

			results := lintFiles(filesToLint)

			return printLintResults(results)
		},
	}

	return cmd
}

// expandGlobs expands glob patterns and ~ in file paths
func expandGlobs(patterns []string) ([]string, error) {
	var files []string
	for _, pattern := range patterns {
		expandedPattern, err := util.ExpandPath(pattern)
		if err != nil {
			return nil, err
		}

		matches, err := filepath.Glob(expandedPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to glob pattern %s: %w", expandedPattern, err)
		}

		// If no matches, keep the original pattern (user might have specified a direct path)
		if len(matches) == 0 {
			files = append(files, expandedPattern)
		} else {
			files = append(files, matches...)
		}
	}
	return files, nil
}

func lintFiles(files []string) []lintResult {
	var results []lintResult
	contextNames := make(map[string][]string) // track context names and their files

	for _, file := range files {
		// Check if file exists and is not a directory
		fileInfo, err := os.Stat(file)
		if os.IsNotExist(err) {
			result := lintResult{
				FilePath: file,
				Errors:   []string{"file does not exist"},
				Warnings: []string{},
			}
			results = append(results, result)
			continue
		}

		// Skip directories silently
		if fileInfo.IsDir() {
			continue
		}

		result := lintResult{
			FilePath: file,
			Errors:   make([]string, 0),
			Warnings: make([]string, 0),
		}

		// Try to load the kubeconfig
		cfg, err := clientcmd.LoadFromFile(file)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to load kubeconfig: %v", err))
			results = append(results, result)
			continue
		}

		// Validate the kubeconfig structure
		validateKubeconfig(cfg, &result)

		// Track context names for duplicate detection
		if cfg.Contexts != nil {
			for contextName := range cfg.Contexts {
				contextNames[contextName] = append(contextNames[contextName], file)
			}
		}

		results = append(results, result)
	}

	// Check for duplicate context names across files
	for contextName, files := range contextNames {
		if len(files) > 1 {
			for _, file := range files {
				for i, result := range results {
					if result.FilePath == file {
						results[i].Warnings = append(results[i].Warnings,
							fmt.Sprintf("duplicate context name %q found in: %v", contextName, files))
						break
					}
				}
			}
		}
	}

	return results
}

func validateKubeconfig(cfg *api.Config, result *lintResult) {
	if len(cfg.Contexts) == 0 {
		result.Warnings = append(result.Warnings, "no contexts defined")
	}

	if len(cfg.Clusters) == 0 {
		result.Warnings = append(result.Warnings, "no clusters defined")
	}

	if len(cfg.AuthInfos) == 0 {
		result.Warnings = append(result.Warnings, "no auth infos defined")
	}

	validateContextRefs(cfg, result)
	validateClusters(cfg, result)
	validateCurrentContext(cfg, result)
}

func validateContextRefs(cfg *api.Config, result *lintResult) {
	for name, ctx := range cfg.Contexts {
		if ctx.Cluster == "" {
			result.Errors = append(result.Errors,
				fmt.Sprintf("context %q has no cluster set", name))
		} else if _, exists := cfg.Clusters[ctx.Cluster]; !exists {
			result.Errors = append(result.Errors,
				fmt.Sprintf("context %q references non-existent cluster %q", name, ctx.Cluster))
		}

		if ctx.AuthInfo == "" {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("context %q has no auth info set", name))
			continue
		}
		if _, exists := cfg.AuthInfos[ctx.AuthInfo]; !exists {
			result.Errors = append(result.Errors,
				fmt.Sprintf("context %q references non-existent auth info %q", name, ctx.AuthInfo))
		}
	}
}

func validateClusters(cfg *api.Config, result *lintResult) {
	for name, cluster := range cfg.Clusters {
		if cluster.Server == "" {
			result.Errors = append(result.Errors,
				fmt.Sprintf("cluster %q has no server URL set", name))
		}
	}
}

func validateCurrentContext(cfg *api.Config, result *lintResult) {
	if cfg.CurrentContext == "" {
		return
	}
	if _, exists := cfg.Contexts[cfg.CurrentContext]; !exists {
		result.Errors = append(result.Errors,
			fmt.Sprintf("current-context %q does not exist", cfg.CurrentContext))
	}
}

func printLintResults(results []lintResult) error {
	if len(results) == 0 {
		fmt.Println("No kubeconfig files found to lint")
		return nil
	}

	hasErrors := false
	hasWarnings := false

	for _, result := range results {
		absPath, err := filepath.Abs(result.FilePath)
		if err != nil {
			absPath = result.FilePath
		}

		if len(result.Errors) > 0 || len(result.Warnings) > 0 {
			fmt.Printf("\n%s:\n", absPath)

			for _, errMsg := range result.Errors {
				fmt.Printf("  ✗ ERROR: %s\n", errMsg)
				hasErrors = true
			}

			for _, warnMsg := range result.Warnings {
				fmt.Printf("  ⚠ WARNING: %s\n", warnMsg)
				hasWarnings = true
			}
		} else {
			fmt.Printf("✓ %s\n", absPath)
		}
	}

	fmt.Println()
	if hasErrors {
		return fmt.Errorf("linting completed with errors")
	} else if hasWarnings {
		fmt.Println("Linting completed with warnings")
	} else {
		fmt.Println("All kubeconfig files are valid!")
	}

	return nil
}
