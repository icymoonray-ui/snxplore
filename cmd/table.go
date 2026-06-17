package cmd

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/icymoonray-ui/snxplore/internal/introspect"
	"github.com/icymoonray-ui/snxplore/internal/output"
	"github.com/icymoonray-ui/snxplore/internal/snclient"
	"github.com/spf13/cobra"
)

func newTableCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "table [name]",
		Short: "Explain a table (schema + logic + access + flows), or list tables",
		Long: "With a table name, prints the flagship report: schema (inheritance-resolved), business\n" +
			"rules & client scripts, access (ACLs/roles), and automation — and caches it for offline\n" +
			"search. Use the 'list' subcommand to list all tables.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cc *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cc.Help()
			}
			if err := validateTableName(args[0]); err != nil {
				return err
			}
			cl, err := clientForProfile(cc.Context())
			if err != nil {
				return err
			}
			rep, err := introspect.ResolveTable(cc.Context(), cl, args[0])
			if err != nil {
				return err
			}
			indexReport(rep) // best-effort cache for offline search

			r := renderer()
			if r.Format == output.FormatJSON {
				return r.Emit(rep)
			}
			printReport(r.Out, rep)
			return nil
		},
	}
	c.AddCommand(newTableListCmd())
	return c
}

func newTableListCmd() *cobra.Command {
	var limit int
	c := &cobra.Command{
		Use:   "list",
		Short: "List tables on the instance (sys_db_object)",
		Args:  cobra.NoArgs,
		RunE: func(cc *cobra.Command, args []string) error {
			cl, err := clientForProfile(cc.Context())
			if err != nil {
				return err
			}
			recs, err := cl.Get(cc.Context(), "sys_db_object", snclient.GetOptions{
				Fields: []string{"name", "label", "super_class", "sys_scope"},
				Query:  "ORDERBYname",
				Limit:  limit,
			})
			if err != nil {
				return err
			}
			return renderer().Emit(tableList(recs))
		},
	}
	c.Flags().IntVar(&limit, "limit", 0, "max tables to return (0 = server default)")
	return c
}

type tableList []snclient.Record

func (t tableList) TableView() output.Table {
	tbl := output.Table{Headers: []string{"NAME", "LABEL", "SCOPE"}}
	for _, r := range t {
		tbl.Rows = append(tbl.Rows, []string{r.Str("name"), r.Str("label"), r.Str("sys_scope")})
	}
	return tbl
}

// printReport renders the human-readable flagship report.
func printReport(w io.Writer, rep *introspect.TableReport) {
	fmt.Fprintf(w, "TABLE  %s\n", rep.Table)
	if rep.Schema != nil {
		fmt.Fprintf(w, "Hierarchy: %s\n", strings.Join(rep.Schema.Hierarchy, " -> "))
		fmt.Fprintf(w, "\nFIELDS (%d)\n", len(rep.Schema.Fields))
		tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
		fmt.Fprintln(tw, "  FIELD\tTYPE\tMANDATORY\tORIGIN")
		for _, f := range rep.Schema.Fields {
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", f.Element, f.Type, yesno(f.Mandatory), f.Origin)
		}
		_ = tw.Flush()
	}
	if rep.Logic != nil {
		fmt.Fprintf(w, "\nLOGIC — %d business rules, %d client scripts\n", len(rep.Logic.BusinessRules), len(rep.Logic.ClientScripts))
		for _, b := range rep.Logic.BusinessRules {
			fmt.Fprintf(w, "  [BR %-7s] %s (order %s)%s\n", b.When, b.Name, b.Order, activeMark(b.Active))
		}
		for _, s := range rep.Logic.ClientScripts {
			fmt.Fprintf(w, "  [CS %-8s] %s (order %s)%s\n", s.Type, s.Name, s.Order, activeMark(s.Active))
		}
	}
	if rep.Access != nil {
		fmt.Fprintf(w, "\nACCESS — %d ACLs\n", len(rep.Access.ACLs))
		for _, a := range rep.Access.ACLs {
			fmt.Fprintf(w, "  %-7s %s -> %s\n", a.Operation, a.Name, strings.Join(a.Roles, ","))
		}
	}
	if rep.Flows != nil && len(rep.Flows.LegacyWorkflows) > 0 {
		fmt.Fprintf(w, "\nLEGACY WORKFLOWS (%d)\n", len(rep.Flows.LegacyWorkflows))
		for _, wf := range rep.Flows.LegacyWorkflows {
			fmt.Fprintf(w, "  %s%s\n", wf.Name, activeMark(wf.Active))
		}
	}
	for _, n := range rep.Notes {
		fmt.Fprintf(w, "\nnote: %s\n", n)
	}
}

func activeMark(active bool) string {
	if active {
		return ""
	}
	return " (inactive)"
}
