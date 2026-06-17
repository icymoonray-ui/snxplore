package cmd

import (
	"github.com/icymoonray-ui/snxplore/internal/introspect"
	"github.com/icymoonray-ui/snxplore/internal/output"
	"github.com/spf13/cobra"
)

func newFlowsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "flows <table>",
		Short: "Show automation (legacy workflows / flows) bound to a table",
		Long:  "flows lists legacy Workflow records (wf_workflow) bound to a table. Flow Designer flow→table binding is not reliably resolvable via the Table API and is reported via a note pending live-instance verification.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cc *cobra.Command, args []string) error {
			if err := validateTableName(args[0]); err != nil {
				return err
			}
			cl, err := clientForProfile(cc.Context())
			if err != nil {
				return err
			}
			f, err := introspect.ResolveFlows(cc.Context(), cl, args[0])
			if err != nil {
				return err
			}
			r := renderer()
			if err := r.Emit(flowsView{f}); err != nil {
				return err
			}
			r.Note(f.Note)
			return nil
		},
	}
	return c
}

type flowsView struct{ *introspect.Flows }

func (v flowsView) TableView() output.Table {
	t := output.Table{Headers: []string{"KIND", "NAME", "TABLE", "ACTIVE"}}
	for _, w := range v.LegacyWorkflows {
		t.Rows = append(t.Rows, []string{"legacy_workflow", w.Name, w.Table, yesno(w.Active)})
	}
	return t
}
