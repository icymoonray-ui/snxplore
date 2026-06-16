// Package introspect builds higher-level views of an instance on top of the
// generic Table client: schema (with inheritance), forms, logic, access, flows.
package introspect

import (
	"context"
	"sort"

	"github.com/icymoonray-ui/snxplore/internal/output"
	"github.com/icymoonray-ui/snxplore/internal/snclient"
)

// maxHierarchyDepth bounds the super_class walk against pathological cycles.
const maxHierarchyDepth = 50

// Field is one resolved column on a table.
type Field struct {
	Element   string `json:"element"`
	Label     string `json:"label"`
	Type      string `json:"type"`
	MaxLength string `json:"max_length,omitempty"`
	Mandatory bool   `json:"mandatory"`
	Reference string `json:"reference,omitempty"` // referenced table, for reference fields
	Origin    string `json:"origin"`              // table in the hierarchy that defines it
}

// Schema is a table's full column set with its extension hierarchy.
type Schema struct {
	Table     string   `json:"table"`
	Hierarchy []string `json:"hierarchy"` // [table, parent, grandparent, ...]
	Fields    []Field  `json:"fields"`
}

// TableHierarchy walks sys_db_object.super_class from table up to the root,
// returning [table, parent, ...]. This is the core inheritance engine: the
// Table API does not resolve inherited fields, so callers must walk the chain.
func TableHierarchy(ctx context.Context, c *snclient.Client, table string) ([]string, error) {
	var chain []string
	cur := table
	for cur != "" && len(chain) < maxHierarchyDepth {
		recs, err := c.Get(ctx, "sys_db_object", snclient.GetOptions{
			Query:  "name=" + cur,
			Fields: []string{"name", "super_class.name"},
			Limit:  1,
		})
		if err != nil {
			return nil, err
		}
		if len(recs) == 0 {
			if len(chain) == 0 {
				return nil, output.Errorf("not_found", output.ExitNotFound, "table %q not found in sys_db_object", table)
			}
			break // parent not a real table row; stop cleanly
		}
		chain = append(chain, cur)
		cur = recs[0].Str("super_class.name")
	}
	return chain, nil
}

// ResolveSchema returns every column for a table, including inherited ones,
// by walking the hierarchy child→root and merging dictionary entries. The
// most-derived definition wins (child overrides), and each field records the
// table that defines it (Origin).
func ResolveSchema(ctx context.Context, c *snclient.Client, table string) (*Schema, error) {
	chain, err := TableHierarchy(ctx, c, table)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var fields []Field
	for _, t := range chain {
		recs, err := c.Get(ctx, "sys_dictionary", snclient.GetOptions{
			// elementISNOTEMPTY skips the table's own "collection" placeholder row.
			Query: "name=" + t + "^elementISNOTEMPTY",
			Fields: []string{
				"element", "column_label",
				"internal_type", "internal_type.name",
				"max_length", "mandatory",
				"reference", "reference.name",
			},
		})
		if err != nil {
			return nil, err
		}
		for _, r := range recs {
			el := r.Str("element")
			if el == "" || seen[el] {
				continue
			}
			seen[el] = true
			fields = append(fields, Field{
				Element:   el,
				Label:     r.Str("column_label"),
				Type:      firstNonEmpty(r, "internal_type.name", "internal_type"),
				MaxLength: r.Str("max_length"),
				Mandatory: r.Str("mandatory") == "true",
				Reference: firstNonEmpty(r, "reference.name", "reference"),
				Origin:    t,
			})
		}
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Element < fields[j].Element })
	return &Schema{Table: table, Hierarchy: chain, Fields: fields}, nil
}

// firstNonEmpty returns the first non-empty record field among keys. Used to
// tolerate internal_type/reference being either a plain value or a reference
// (dot-walked .name) depending on the instance.
func firstNonEmpty(r snclient.Record, keys ...string) string {
	for _, k := range keys {
		if v := r.Str(k); v != "" {
			return v
		}
	}
	return ""
}
