package cmd

import "github.com/spf13/cobra"

// version is overridden at build time via -ldflags "-X ...cmd.version=...".
var version = "0.0.0-dev"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the snxplore version",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			return renderer().Emit(map[string]string{"version": version})
		},
	}
}
