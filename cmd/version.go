package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/idebeijer/kubert/pkg/versions"
)

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display version and build information",
		Long:  `Display version and build information for kubert.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("kubert %s\n", versions.Version)
			fmt.Printf("  commit: %s\n", versions.Commit)
			fmt.Printf("  built: %s\n", versions.Date)
			fmt.Printf("  built by: %s\n", versions.BuiltBy)
			return nil
		},
	}
}
