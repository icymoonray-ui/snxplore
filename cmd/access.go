package cmd

import (
	"strings"

	"github.com/icymoonray-ui/snxplore/internal/introspect"
	"github.com/icymoonray-ui/snxplore/internal/output"
	"github.com/spf13/cobra"
)

func newAccessCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "access <table>",
		Short: "Show who can read/write/create/delete a table (ACLs and roles)",
		Long:  "access lists the ACL rules protecting a table and the roles each requires. Reading ACLs over the Table API often needs the security_admin role; when unavailable, the command degrades with a note rather than failing.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cc *cobra.Command, args []string) error {
			if err := validateTableName(args[0]); err != nil {
				return err
			}
			cl, err := clientForProfile(cc.Context())
			if err != nil {
				return err
			}
			a, err := introspect.ResolveAccess(cc.Context(), cl, args[0])
			if err != nil {
				return err
			}
			r := renderer()
			if err := r.Emit(accessView{a}); err != nil {
				return err
			}
			r.Note(a.Note)
			return nil
		},
	}
	return c
}

type accessView struct{ *introspect.Access }

func (v accessView) TableView() output.Table {
	t := output.Table{Headers: []string{"OPERATION", "NAME", "ACTIVE", "ROLES", "COND", "SCRIPT"}}
	for _, a := range v.ACLs {
		t.Rows = append(t.Rows, []string{
			a.Operation, a.Name, yesno(a.Active),
			strings.Join(a.Roles, ","), yesno(a.HasCondition), yesno(a.HasScript),
		})
	}
	return t
}
