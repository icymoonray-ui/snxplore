// Package cmd wires the snxplore cobra command tree. Commands stay thin and
// delegate to the internal packages (snclient, introspect, store).
package cmd

import (
	"os"

	"github.com/icymoonray-ui/snxplore/internal/output"
	"github.com/spf13/cobra"
)

var (
	flagOutput  string
	flagProfile string
	flagVerbose bool
)

// NewRootCmd builds the root command and registers subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "snxplore",
		Short: "Explore and understand any ServiceNow instance via the documented Table API",
		Long: "snxplore is a read-first, agent-native CLI for understanding an arbitrary\n" +
			"ServiceNow instance. It builds on the documented Now Platform Table API and\n" +
			"its self-describing metadata tables — no internal/UI APIs, portable across\n" +
			"releases.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&flagOutput, "output", "table", "output format: json|table")
	root.PersistentFlags().StringVar(&flagProfile, "profile", "default", "instance profile to use")
	root.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "verbose logging to stderr")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newTableCmd())
	root.AddCommand(newQueryCmd())
	root.AddCommand(newSchemaCmd())
	root.AddCommand(newLogicCmd())
	root.AddCommand(newAccessCmd())
	root.AddCommand(newFlowsCmd())
	root.AddCommand(newSearchCmd())
	return root
}

// renderer returns an output.Renderer configured from the global --output flag.
func renderer() *output.Renderer {
	return output.New(output.Format(flagOutput), os.Stdout, os.Stderr)
}

// Execute runs the command tree and returns a process exit code.
func Execute() int {
	root := NewRootCmd()
	if err := root.Execute(); err != nil {
		return renderer().EmitError(err)
	}
	return output.ExitOK
}
