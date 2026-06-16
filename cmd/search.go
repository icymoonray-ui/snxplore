package cmd

import (
	"github.com/icymoonray-ui/snxplore/internal/output"
	"github.com/icymoonray-ui/snxplore/internal/store"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "search <term>",
		Short: "Search locally cached metadata (offline, full-text)",
		Long:  "Searches metadata cached by previous `table <name>` explorations using SQLite FTS5. Offline — makes no instance call.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cc *cobra.Command, args []string) error {
			st, err := openCache()
			if err != nil {
				return output.Errorf("cache", output.ExitError, "open cache: %v", err)
			}
			defer st.Close()
			hits, err := st.Search(args[0])
			if err != nil {
				return output.Errorf("search", output.ExitError, "%v", err)
			}
			return renderer().Emit(searchView(hits))
		},
	}
	return c
}

type searchView []store.SearchHit

func (v searchView) TableView() output.Table {
	t := output.Table{Headers: []string{"TABLE", "KIND", "NAME"}}
	for _, h := range v {
		t.Rows = append(t.Rows, []string{h.TableName, h.Kind, h.Name})
	}
	return t
}
