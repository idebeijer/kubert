//nolint:errcheck
package cmd

import (
	"fmt"
	"os"
	"runtime"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/idebeijer/kubert/pkg/versions"
)

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display version and build information",
		Long:  `Display version and build information for kubert.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			defer w.Flush()

			version := fmt.Sprintf("v%s", versions.Version)

			fmt.Fprintf(w, "kubert:\t%s\n", version)
			fmt.Fprintf(w, "commit:\t%s\n", versions.Commit)
			fmt.Fprintf(w, "build date:\t%s\n", versions.Date)
			fmt.Fprintf(w, "built by:\t%s\n", versions.BuiltBy)
			fmt.Fprintf(w, "go version:\t%s\n", runtime.Version())
			fmt.Fprintf(w, "os/arch:\t%s/%s\n", runtime.GOOS, runtime.GOARCH)

			return nil
		},
	}
}
