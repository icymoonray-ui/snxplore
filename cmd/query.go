package cmd

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/icymoonray-ui/snxplore/internal/output"
	"github.com/icymoonray-ui/snxplore/internal/snclient"
	"github.com/spf13/cobra"
)

func newQueryCmd() *cobra.Command {
	var query, fields, display string
	var limit, offset int
	c := &cobra.Command{
		Use:   "query <table>",
		Short: "Read records from any table via the generic Table API",
		Long:  "query is the raw substrate: a thin, generic read over /api/now/table/{table} with the standard sysparm_* options.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cc *cobra.Command, args []string) error {
			cl, err := clientForProfile(cc.Context())
			if err != nil {
				return err
			}
			var fl []string
			if fields != "" {
				fl = strings.Split(fields, ",")
			}
			recs, err := cl.Get(cc.Context(), args[0], snclient.GetOptions{
				Query:        query,
				Fields:       fl,
				Limit:        limit,
				Offset:       offset,
				DisplayValue: display,
			})
			if err != nil {
				return err
			}
			return renderer().Emit(recordsView{records: recs, columns: fl})
		},
	}
	c.Flags().StringVarP(&query, "query", "q", "", "encoded query (sysparm_query)")
	c.Flags().StringVar(&fields, "fields", "", "comma-separated fields (sysparm_fields)")
	c.Flags().IntVar(&limit, "limit", 0, "max records (sysparm_limit)")
	c.Flags().IntVar(&offset, "offset", 0, "record offset (sysparm_offset)")
	c.Flags().StringVar(&display, "display-value", "", "sysparm_display_value: false|true|all")
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
