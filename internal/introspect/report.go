package introspect

import (
	"context"

	"github.com/icymoonray-ui/snxplore/internal/snclient"
)

// TableReport is the flagship "explain this table" view: schema, logic, access,
// and automation in one structure. Schema is mandatory (it also validates the
// table exists); the other sections are best-effort and any gaps are recorded
// in Notes so the report never fails wholesale.
type TableReport struct {
	Table   string   `json:"table"`
	Schema  *Schema  `json:"schema"`
	Logic   *Logic   `json:"logic,omitempty"`
	Access  *Access  `json:"access,omitempty"`
	Flows   *Flows   `json:"flows,omitempty"`
	Notes   []string `json:"notes,omitempty"`
}

// ResolveTable assembles a full report for a table.
func ResolveTable(ctx context.Context, c *snclient.Client, table string) (*TableReport, error) {
	rep := &TableReport{Table: table}

	sc, err := ResolveSchema(ctx, c, table)
	if err != nil {
		return nil, err // mandatory: also the existence check
	}
	rep.Schema = sc

	if l, err := ResolveLogic(ctx, c, table); err != nil {
		rep.Notes = append(rep.Notes, "logic unavailable: "+err.Error())
	} else {
		rep.Logic = l
	}

	if a, err := ResolveAccess(ctx, c, table); err != nil {
		rep.Notes = append(rep.Notes, "access unavailable: "+err.Error())
	} else {
		rep.Access = a
		if a.Note != "" {
			rep.Notes = append(rep.Notes, a.Note)
		}
	}

	if f, err := ResolveFlows(ctx, c, table); err != nil {
		rep.Notes = append(rep.Notes, "flows unavailable: "+err.Error())
	} else {
		rep.Flows = f
		if f.Note != "" {
			rep.Notes = append(rep.Notes, f.Note)
		}
	}

	return rep, nil
}
