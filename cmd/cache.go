package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/icymoonray-ui/snxplore/internal/introspect"
	"github.com/icymoonray-ui/snxplore/internal/store"
)

// cacheDBPath returns the local cache database path, creating its directory.
func cacheDBPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		dir = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	d := filepath.Join(dir, "snxplore")
	if err := os.MkdirAll(d, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(d, "cache.db"), nil
}

func openCache() (*store.Store, error) {
	p, err := cacheDBPath()
	if err != nil {
		return nil, err
	}
	return store.Open(p)
}

// indexReport caches a table report for offline search. Best-effort: failures
// never break the command (the cache is an optimization, not the result).
func indexReport(rep *introspect.TableReport) {
	st, err := openCache()
	if err != nil {
		return
	}
	defer st.Close()

	tbl := rep.Table
	_ = st.DeleteTable(tbl) // replace prior rows for this table

	if rep.Schema != nil {
		for _, f := range rep.Schema.Fields {
			_ = st.Index(tbl, "field", f.Element, strings.Join([]string{f.Element, f.Label, f.Type, f.Reference}, " "))
		}
	}
	if rep.Logic != nil {
		for _, b := range rep.Logic.BusinessRules {
			_ = st.Index(tbl, "business_rule", b.Name, strings.Join([]string{b.Name, b.When, b.Condition}, " "))
		}
		for _, s := range rep.Logic.ClientScripts {
			_ = st.Index(tbl, "client_script", s.Name, strings.Join([]string{s.Name, s.Type, s.Field}, " "))
		}
	}
	if rep.Access != nil {
		for _, a := range rep.Access.ACLs {
			_ = st.Index(tbl, "acl", a.Operation+" "+a.Name, a.Name+" "+strings.Join(a.Roles, " "))
		}
	}
	if rep.Flows != nil {
		for _, w := range rep.Flows.LegacyWorkflows {
			_ = st.Index(tbl, "legacy_workflow", w.Name, w.Name)
		}
	}
}
