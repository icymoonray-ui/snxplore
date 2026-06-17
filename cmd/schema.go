package cmd

import (
	"github.com/icymoonray-ui/snxplore/internal/introspect"
	"github.com/icymoonray-ui/snxplore/internal/output"
	"github.com/spf13/cobra"
)

func newSchemaCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "schema <table>",
		Short: "Show a table's fields, with inherited columns resolved",
		Long:  "schema lists every column on a table — including fields inherited from parent tables — by walking the sys_db_object super_class chain (which the Table API does not resolve on its own). Each field shows the table that defines it (ORIGIN).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cc *cobra.Command, args []string) error {
			if err := validateTableName(args[0]); err != nil {
				return err
			}
			cl, err := clientForProfile(cc.Context())
			if err != nil {
				return err
			}
			sc, err := introspect.ResolveSchema(cc.Context(), cl, args[0])
			if err != nil {
				return err
			}
			return renderer().Emit(schemaView{sc})
		},
	}
	return c
}

// schemaView renders a Schema: promoted JSON in json mode, a field table in
// table mode.
type schemaView struct{ *introspect.Schema }

func (v schemaView) TableView() output.Table {
	t := output.Table{Headers: []string{"FIELD", "LABEL", "TYPE", "MANDATORY", "REFERENCE", "ORIGIN"}}
	for _, f := range v.Fields {
		mandatory := ""
		if f.Mandatory {
			mandatory = "yes"
		}
		t.Rows = append(t.Rows, []string{f.Element, f.Label, f.Type, mandatory, f.Reference, f.Origin})
	}
	return t
}
