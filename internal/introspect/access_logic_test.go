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

func jsonResult(w http.ResponseWriter, rows []map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"result": rows})
}

func TestResolveLogic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/now/table/sys_script"+"_client"):
			jsonResult(w, []map[string]any{
				{"name": "Set priority", "type": "onChange", "field_name": "urgency", "order": "100", "active": "true"},
			})
		case strings.HasPrefix(r.URL.Path, "/api/now/table/sys_script"):
			jsonResult(w, []map[string]any{
				{"name": "Assign group", "when": "before", "order": "100", "active": "true", "condition": "", "action_insert": "true", "action_update": "true"},
			})
		default:
			jsonResult(w, nil)
		}
	}))
	defer srv.Close()

	l, err := ResolveLogic(context.Background(), snclient.New(srv.URL, srv.Client()), "incident")
	if err != nil {
		t.Fatalf("ResolveLogic: %v", err)
	}
	if len(l.BusinessRules) != 1 || l.BusinessRules[0].When != "before" {
		t.Fatalf("business rules = %+v", l.BusinessRules)
	}
	if got := strings.Join(l.BusinessRules[0].Actions, ","); got != "insert,update" {
		t.Errorf("actions = %q, want insert,update", got)
	}
	if len(l.ClientScripts) != 1 || l.ClientScripts[0].Type != "onChange" || l.ClientScripts[0].Field != "urgency" {
		t.Fatalf("client scripts = %+v", l.ClientScripts)
	}
}

func TestResolveAccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/now/table/sys_security_acl_role"):
			jsonResult(w, []map[string]any{
				{"sys_security_acl": "a1", "sys_user_role.name": "itil"},
				{"sys_security_acl": "a2", "sys_user_role.name": "admin"},
			})
		case strings.HasPrefix(r.URL.Path, "/api/now/table/sys_security_acl"):
			jsonResult(w, []map[string]any{
				{"sys_id": "a1", "operation": "read", "name": "incident", "active": "true", "admin_overrides": "true", "condition": "", "script": ""},
				{"sys_id": "a2", "operation": "write", "name": "incident.state", "active": "true", "condition": "current.active", "script": ""},
			})
		default:
			jsonResult(w, nil)
		}
	}))
	defer srv.Close()

	a, err := ResolveAccess(context.Background(), snclient.New(srv.URL, srv.Client()), "incident")
	if err != nil {
		t.Fatalf("ResolveAccess: %v", err)
	}
	if len(a.ACLs) != 2 {
		t.Fatalf("want 2 ACLs, got %d", len(a.ACLs))
	}
	byOp := map[string]ACL{}
	for _, acl := range a.ACLs {
		byOp[acl.Operation] = acl
	}
	if r := strings.Join(byOp["read"].Roles, ","); r != "itil" {
		t.Errorf("read roles = %q, want itil", r)
	}
	if w := byOp["write"]; strings.Join(w.Roles, ",") != "admin" || !w.HasCondition {
		t.Errorf("write acl = %+v", w)
	}
}

func TestResolveAccessDegrade(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate "ACL table not readable without security_admin".
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"Insufficient rights"},"status":"failure"}`))
	}))
	defer srv.Close()

	a, err := ResolveAccess(context.Background(), snclient.New(srv.URL, srv.Client()), "incident")
	if err != nil {
		t.Fatalf("expected graceful degrade, got error: %v", err)
	}
	if len(a.ACLs) != 0 || a.Note == "" {
		t.Fatalf("expected empty ACLs + a note, got %+v", a)
	}
	if !strings.Contains(a.Note, "security_admin") {
		t.Errorf("note should mention security_admin: %q", a.Note)
	}
}
