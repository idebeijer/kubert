package cmd

import (
	"fmt"
	"sort"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/fzf"
	"github.com/idebeijer/kubert/internal/kubeconfig"
	"github.com/spf13/cobra"
)

func NewEncryptedCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "encrypted",
		Short: "Manage encrypted kubeconfigs using macOS Keychain",
		Long: `Manage encrypted kubeconfigs that are stored encrypted at rest using macOS Keychain.
This provides enhanced security for sensitive cluster credentials while maintaining
the same user experience for context switching.`,
	}

	cmd.AddCommand(NewEncryptedAddCommand())
	cmd.AddCommand(NewEncryptedRemoveCommand())
	cmd.AddCommand(NewEncryptedListCommand())

	return cmd
}

func NewEncryptedAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [kubeconfig-path] [context-name]",
		Short: "Encrypt and store a kubeconfig context",
		Long: `Encrypt a kubeconfig context and store it securely using macOS Keychain.
The context will be available for use with 'kubert ctx' once encrypted.

Examples:
  kubert encrypted add ~/.kube/config production
  kubert encrypted add ./my-cluster.yaml my-cluster-context`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfigPath := args[0]
			contextName := args[1]

			cfg := config.Cfg
			if !cfg.KubeconfigProviders.MacOSEncrypted.Enabled {
				return fmt.Errorf("macOS encrypted provider is not enabled in configuration")
			}

			expandedStorageDir, err := expandPath(cfg.KubeconfigProviders.MacOSEncrypted.StorageDir)
			if err != nil {
				return fmt.Errorf("failed to expand storage directory path: %w", err)
			}

			provider, err := kubeconfig.NewMacOSEncryptedProvider(expandedStorageDir)
			if err != nil {
				return fmt.Errorf("failed to create encrypted provider: %w", err)
			}

			if err := provider.EncryptKubeconfig(kubeconfigPath, contextName); err != nil {
				return fmt.Errorf("failed to encrypt kubeconfig: %w", err)
			}

			fmt.Printf("Successfully encrypted context '%s' from '%s'\n", contextName, kubeconfigPath)
			return nil
		},
	}

	return cmd
}

func NewEncryptedRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [context-name]",
		Short: "Remove an encrypted kubeconfig context",
		Long: `Remove an encrypted kubeconfig context and its associated encryption key.
This will permanently delete the encrypted kubeconfig data.`,
		Aliases:           []string{"rm", "delete"},
		ValidArgsFunction: validEncryptedContextArgsFunction,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Cfg
			if !cfg.KubeconfigProviders.MacOSEncrypted.Enabled {
				return fmt.Errorf("macOS encrypted provider is not enabled in configuration")
			}

			expandedStorageDir, err := expandPath(cfg.KubeconfigProviders.MacOSEncrypted.StorageDir)
			if err != nil {
				return fmt.Errorf("failed to expand storage directory path: %w", err)
			}

			provider, err := kubeconfig.NewMacOSEncryptedProvider(expandedStorageDir)
			if err != nil {
				return fmt.Errorf("failed to create encrypted provider: %w", err)
			}

			var contextName string
			if len(args) > 0 {
				contextName = args[0]
			} else {
				// Interactive selection
				contexts, err := provider.ListEncryptedContexts()
				if err != nil {
					return fmt.Errorf("failed to list encrypted contexts: %w", err)
				}

				if len(contexts) == 0 {
					fmt.Println("No encrypted contexts found")
					return nil
				}

				sort.Strings(contexts)
				selectedContext, err := fzf.Select(contexts)
				if err != nil {
					return fmt.Errorf("failed to select context: %w", err)
				}
				if selectedContext == "" {
					return nil
				}
				contextName = selectedContext
			}

			if err := provider.RemoveEncryptedKubeconfig(contextName); err != nil {
				return fmt.Errorf("failed to remove encrypted context: %w", err)
			}

			fmt.Printf("Successfully removed encrypted context '%s'\n", contextName)
			return nil
		},
	}

	return cmd
}

func NewEncryptedListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all encrypted kubeconfig contexts",
		Long:    `List all encrypted kubeconfig contexts that are stored using macOS Keychain.`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Cfg
			if !cfg.KubeconfigProviders.MacOSEncrypted.Enabled {
				return fmt.Errorf("macOS encrypted provider is not enabled in configuration")
			}

			expandedStorageDir, err := expandPath(cfg.KubeconfigProviders.MacOSEncrypted.StorageDir)
			if err != nil {
				return fmt.Errorf("failed to expand storage directory path: %w", err)
			}

			provider, err := kubeconfig.NewMacOSEncryptedProvider(expandedStorageDir)
			if err != nil {
				return fmt.Errorf("failed to create encrypted provider: %w", err)
			}

			contexts, err := provider.ListEncryptedContexts()
			if err != nil {
				return fmt.Errorf("failed to list encrypted contexts: %w", err)
			}

			if len(contexts) == 0 {
				fmt.Println("No encrypted contexts found")
				return nil
			}

			sort.Strings(contexts)
			for _, context := range contexts {
				fmt.Println(context)
			}

			return nil
		},
	}

	return cmd
}

func validEncryptedContextArgsFunction(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg := config.Cfg
	if !cfg.KubeconfigProviders.MacOSEncrypted.Enabled {
		return nil, cobra.ShellCompDirectiveError
	}

	expandedStorageDir, err := expandPath(cfg.KubeconfigProviders.MacOSEncrypted.StorageDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	provider, err := kubeconfig.NewMacOSEncryptedProvider(expandedStorageDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	contexts, err := provider.ListEncryptedContexts()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	sort.Strings(contexts)
	return contexts, cobra.ShellCompDirectiveNoFileComp
}
