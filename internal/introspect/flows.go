package introspect

import (
	"context"
	"errors"

	"github.com/icymoonray-ui/snxplore/internal/output"
	"github.com/icymoonray-ui/snxplore/internal/snclient"
)

// LegacyWorkflow is a record from the legacy Workflow engine (wf_workflow).
type LegacyWorkflow struct {
	Name   string `json:"name"`
	Table  string `json:"table"`
	Active bool   `json:"active"`
}

// Flows reports automation bound to a table.
//
// Legacy Workflow binding is reliable (wf_workflow.table). Flow Designer
// flow→table binding is NOT reliably queryable via the documented Table API
// (the trigger/table link is polymorphic and changed shape in Flow Engine V2,
// Washington DC+); that resolution is an open item to confirm against a live
// instance, so it is reported via Note rather than guessed.
type Flows struct {
	Table           string           `json:"table"`
	LegacyWorkflows []LegacyWorkflow `json:"legacy_workflows"`
	Note            string           `json:"note,omitempty"`
}

// ResolveFlows lists legacy workflows bound to a table. If the legacy Workflow
// table is absent (newer instances may not ship it) it degrades to a Note.
func ResolveFlows(ctx context.Context, c *snclient.Client, table string) (*Flows, error) {
	f := &Flows{Table: table}

	wf, err := c.Get(ctx, "wf_workflow", snclient.GetOptions{
		Query:  "table=" + table + "^ORDERBYname",
		Fields: []string{"name", "table", "active"},
	})
	if err != nil {
		// A missing wf_workflow table (NotFound) is not fatal — just report it.
		if isNotFound(err) {
			f.Note = "legacy Workflow engine (wf_workflow) not present on this instance."
		} else {
			return nil, err
		}
	}
	for _, r := range wf {
		f.LegacyWorkflows = append(f.LegacyWorkflows, LegacyWorkflow{
			Name:   r.Str("name"),
			Table:  r.Str("table"),
			Active: r.Str("active") == "true",
		})
	}

	flowNote := "Flow Designer flow→table binding is not reliably resolvable via the Table API (polymorphic trigger link; differs in Flow Engine V2 / Washington DC+) — pending live-instance verification. Legacy workflows are listed above."
	if f.Note == "" {
		f.Note = flowNote
	} else {
		f.Note += " " + flowNote
	}
	return f, nil
}

func isNotFound(err error) bool {
	var ce *output.Error
	return errors.As(err, &ce) && ce.Code == "not_found"
}
