package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information — overridden at build time via -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "kubectl-fqdn version %s (commit %s, built %s)\n", Version, Commit, Date)
		},
	}
}
