package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/icymoonray-ui/snxplore/internal/output"
	"github.com/icymoonray-ui/snxplore/internal/snclient"
	"github.com/spf13/cobra"
)

func newQueryCmd() *cobra.Command {
	var query, fields, display string
	var limit, offset, pageSize int
	var all bool
	c := &cobra.Command{
		Use:   "query <table>",
		Short: "Read records from any table via the generic Table API",
		Long:  "query is the raw substrate: a thin, generic read over /api/now/table/{table} with the standard sysparm_* options.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cc *cobra.Command, args []string) error {
			if err := validateTableName(args[0]); err != nil {
				return err
			}
			cl, err := clientForProfile(cc.Context())
			if err != nil {
				return err
			}
			var fl []string
			if fields != "" {
				fl = strings.Split(fields, ",")
			}
			opt := snclient.GetOptions{
				Query:        query,
				Fields:       fl,
				Limit:        limit,
				Offset:       offset,
				DisplayValue: display,
			}
			r := renderer()

			if all {
				recs, err := cl.GetAll(cc.Context(), args[0], opt, pageSize)
				if err != nil {
					return err
				}
				return r.Emit(recordsView{records: recs, columns: fl})
			}

			page, err := cl.GetPage(cc.Context(), args[0], opt)
			if err != nil {
				return err
			}
			if page.Total > len(page.Records) {
				r.Note(fmt.Sprintf("showing %d of %d matching records — use --all to fetch every page",
					len(page.Records), page.Total))
			}
			return r.Emit(recordsView{records: page.Records, columns: fl})
		},
	}
	c.Flags().StringVarP(&query, "query", "q", "", "encoded query (sysparm_query)")
	c.Flags().StringVar(&fields, "fields", "", "comma-separated fields (sysparm_fields)")
	c.Flags().IntVar(&limit, "limit", 0, "max records (sysparm_limit)")
	c.Flags().IntVar(&offset, "offset", 0, "record offset (sysparm_offset)")
	c.Flags().StringVar(&display, "display-value", "", "sysparm_display_value: false|true|all")
	c.Flags().BoolVar(&all, "all", false, "fetch every matching record by paging (ignores --limit)")
	c.Flags().IntVar(&pageSize, "page-size", 0, "page size for --all (0 = default 1000)")
	return c
}

// recordsView renders generic records: raw JSON in json mode, columns in table
// mode (the requested --fields, or the sorted union of keys when unspecified).
type recordsView struct {
	records []snclient.Record
	columns []string
}

func (v recordsView) MarshalJSON() ([]byte, error) { return json.Marshal(v.records) }

func (v recordsView) TableView() output.Table {
	cols := v.columns
	if len(cols) == 0 {
		cols = unionKeys(v.records)
	}
	t := output.Table{Headers: cols}
	for _, r := range v.records {
		row := make([]string, len(cols))
		for i, col := range cols {
			row[i] = r.Str(col)
		}
		t.Rows = append(t.Rows, row)
	}
	return t
}

func unionKeys(recs []snclient.Record) []string {
	set := map[string]bool{}
	for _, r := range recs {
		for k := range r {
			set[k] = true
		}
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
