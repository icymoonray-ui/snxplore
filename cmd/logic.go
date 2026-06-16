package cmd

import (
	"strings"

	"github.com/icymoonray-ui/snxplore/internal/introspect"
	"github.com/icymoonray-ui/snxplore/internal/output"
	"github.com/spf13/cobra"
)

func newLogicCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "logic <table>",
		Short: "Show business rules and client scripts that run on a table",
		Long:  "logic lists the server-side business rules (sys_script) and client scripts (sys_script_client) bound to a table, in execution order (by when/type, then order).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cc *cobra.Command, args []string) error {
			cl, err := clientForProfile(cc.Context())
			if err != nil {
				return err
			}
			l, err := introspect.ResolveLogic(cc.Context(), cl, args[0])
			if err != nil {
				return err
			}
			return renderer().Emit(logicView{l})
		},
	}
	return c
}

type logicView struct{ *introspect.Logic }

func (v logicView) TableView() output.Table {
	t := output.Table{Headers: []string{"KIND", "NAME", "WHEN/TYPE", "ORDER", "ACTIVE", "DETAIL"}}
	for _, b := range v.BusinessRules {
		t.Rows = append(t.Rows, []string{"business_rule", b.Name, b.When, b.Order, yesno(b.Active), strings.Join(b.Actions, ",")})
	}
	for _, s := range v.ClientScripts {
		t.Rows = append(t.Rows, []string{"client_script", s.Name, s.Type, s.Order, yesno(s.Active), s.Field})
	}
	return t
}
