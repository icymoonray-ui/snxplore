package introspect

import (
	"context"

	"github.com/icymoonray-ui/snxplore/internal/snclient"
)

// BusinessRule is a server-side rule from sys_script.
type BusinessRule struct {
	Name      string   `json:"name"`
	When      string   `json:"when"` // before | after | async | display
	Order     string   `json:"order"`
	Active    bool     `json:"active"`
	Condition string   `json:"condition,omitempty"`
	Actions   []string `json:"actions"` // insert | update | query | delete
}

// ClientScript is a client-side script from sys_script_client.
type ClientScript struct {
	Name   string `json:"name"`
	Type   string `json:"type"` // onLoad | onChange | onSubmit | onCellEdit
	Field  string `json:"field,omitempty"`
	Order  string `json:"order"`
	Active bool   `json:"active"`
}

// Logic is the server- and client-side logic that runs on a table.
type Logic struct {
	Table         string         `json:"table"`
	BusinessRules []BusinessRule `json:"business_rules"`
	ClientScripts []ClientScript `json:"client_scripts"`
}

// ResolveLogic lists business rules (sys_script) and client scripts
// (sys_script_client) bound to a table, in execution order (by when/type then
// order).
func ResolveLogic(ctx context.Context, c *snclient.Client, table string) (*Logic, error) {
	l := &Logic{Table: table}

	brs, err := c.Get(ctx, "sys_script", snclient.GetOptions{
		Query:  "collection=" + table + "^ORDERBYwhen^ORDERBYorder",
		Fields: []string{"name", "when", "order", "active", "condition", "action_insert", "action_update", "action_query", "action_delete"},
	})
	if err != nil {
		return nil, err
	}
	for _, r := range brs {
		l.BusinessRules = append(l.BusinessRules, BusinessRule{
			Name:      r.Str("name"),
			When:      r.Str("when"),
			Order:     r.Str("order"),
			Active:    r.Str("active") == "true",
			Condition: r.Str("condition"),
			Actions:   actionList(r),
		})
	}

	cs, err := c.Get(ctx, "sys_script_client", snclient.GetOptions{
		Query:  "table=" + table + "^ORDERBYtype^ORDERBYorder",
		Fields: []string{"name", "type", "field_name", "order", "active"},
	})
	if err != nil {
		return nil, err
	}
	for _, r := range cs {
		l.ClientScripts = append(l.ClientScripts, ClientScript{
			Name:   r.Str("name"),
			Type:   r.Str("type"),
			Field:  r.Str("field_name"),
			Order:  r.Str("order"),
			Active: r.Str("active") == "true",
		})
	}
	return l, nil
}

func actionList(r snclient.Record) []string {
	var a []string
	for _, op := range []struct{ field, name string }{
		{"action_insert", "insert"},
		{"action_update", "update"},
		{"action_query", "query"},
		{"action_delete", "delete"},
	} {
		if r.Str(op.field) == "true" {
			a = append(a, op.name)
		}
	}
	return a
}
