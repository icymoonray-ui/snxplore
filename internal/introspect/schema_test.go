package introspect

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/icymoonray-ui/snxplore/internal/snclient"
)

// mockInstance serves just enough of sys_db_object and sys_dictionary to model
// incident -> task with fields split across the two levels.
func mockInstance(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("sysparm_query")
		var result []map[string]any
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/now/table/sys_db_object"):
			switch {
			case strings.Contains(q, "name=incident"):
				result = []map[string]any{{"name": "incident", "super_class.name": "task"}}
			case strings.Contains(q, "name=task"):
				result = []map[string]any{{"name": "task", "super_class.name": ""}}
			}
		case strings.HasPrefix(r.URL.Path, "/api/now/table/sys_dictionary"):
			switch {
			case strings.Contains(q, "name=incident^"):
				result = []map[string]any{
					{"element": "severity", "column_label": "Severity", "internal_type.name": "integer", "mandatory": "false"},
				}
			case strings.Contains(q, "name=task^"):
				result = []map[string]any{
					{"element": "short_description", "column_label": "Short description", "internal_type.name": "string", "mandatory": "true"},
					{"element": "assigned_to", "column_label": "Assigned to", "internal_type.name": "reference", "reference.name": "sys_user", "mandatory": "false"},
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": result})
	}))
}

func TestResolveSchemaInheritance(t *testing.T) {
	srv := mockInstance(t)
	defer srv.Close()
	c := snclient.New(srv.URL, srv.Client())

	sc, err := ResolveSchema(context.Background(), c, "incident")
	if err != nil {
		t.Fatalf("ResolveSchema: %v", err)
	}

	if got, want := strings.Join(sc.Hierarchy, ">"), "incident>task"; got != want {
		t.Errorf("hierarchy = %q, want %q", got, want)
	}
	// 1 own field (severity) + 2 inherited (short_description, assigned_to).
	if len(sc.Fields) != 3 {
		t.Fatalf("want 3 fields, got %d: %+v", len(sc.Fields), sc.Fields)
	}

	byName := map[string]Field{}
	for _, f := range sc.Fields {
		byName[f.Element] = f
	}
	if byName["severity"].Origin != "incident" {
		t.Errorf("severity origin = %q, want incident", byName["severity"].Origin)
	}
	if f := byName["short_description"]; f.Origin != "task" || !f.Mandatory {
		t.Errorf("short_description = %+v, want origin=task mandatory=true", f)
	}
	if f := byName["assigned_to"]; f.Type != "reference" || f.Reference != "sys_user" {
		t.Errorf("assigned_to = %+v, want type=reference reference=sys_user", f)
	}
}

func TestResolveSchemaNotFound(t *testing.T) {
	srv := mockInstance(t)
	defer srv.Close()
	c := snclient.New(srv.URL, srv.Client())

	if _, err := ResolveSchema(context.Background(), c, "does_not_exist"); err == nil {
		t.Fatal("expected not-found error for unknown table")
	}
}
