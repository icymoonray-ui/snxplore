package snclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/icymoonray-ui/snxplore/internal/output"
)

func TestBuildURL(t *testing.T) {
	c := New("https://dev123.service-now.com", nil)
	raw := c.buildURL("sys_db_object", GetOptions{
		Fields: []string{"name", "label"},
		Query:  "ORDERBYname",
		Limit:  10,
		Offset: 20,
	})
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if u.Path != "/api/now/table/sys_db_object" {
		t.Errorf("path = %q", u.Path)
	}
	q := u.Query()
	if q.Get("sysparm_fields") != "name,label" {
		t.Errorf("fields = %q", q.Get("sysparm_fields"))
	}
	if q.Get("sysparm_query") != "ORDERBYname" {
		t.Errorf("query = %q", q.Get("sysparm_query"))
	}
	if q.Get("sysparm_limit") != "10" || q.Get("sysparm_offset") != "20" {
		t.Errorf("limit/offset = %q/%q", q.Get("sysparm_limit"), q.Get("sysparm_offset"))
	}
}

func TestGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/now/table/sys_db_object" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"result":[{"name":"incident","label":"Incident","sys_scope":{"value":"global","link":"x"}},{"name":"change_request","label":"Change Request"}]}`)
	}))
	defer srv.Close()

	c := New(srv.URL, srv.Client())
	recs, err := c.Get(context.Background(), "sys_db_object", GetOptions{Fields: []string{"name", "label"}})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("want 2 records, got %d", len(recs))
	}
	if recs[0].Str("name") != "incident" || recs[0].Str("label") != "Incident" {
		t.Errorf("record 0 = %+v", recs[0])
	}
	// Reference field decoded as {value,link} -> Str extracts value.
	if recs[0].Str("sys_scope") != "global" {
		t.Errorf("sys_scope = %q, want global", recs[0].Str("sys_scope"))
	}
}

func TestGetUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":{"message":"User Not Authenticated","detail":"Required to provide Auth information"},"status":"failure"}`)
	}))
	defer srv.Close()

	c := New(srv.URL, srv.Client())
	_, err := c.Get(context.Background(), "sys_db_object", GetOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	var ce *output.Error
	if !errors.As(err, &ce) {
		t.Fatalf("want *output.Error, got %T", err)
	}
	if ce.Exit != output.ExitAuth {
		t.Errorf("exit = %d, want %d", ce.Exit, output.ExitAuth)
	}
	if ce.Code != "auth_unauthorized" {
		t.Errorf("code = %q, want auth_unauthorized", ce.Code)
	}
}

// TestGetForbidden checks 403 maps to auth_forbidden, distinct from 401 — so
// callers can degrade on forbidden without masking bad credentials.
func TestGetForbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, `{"error":{"message":"Insufficient rights"},"status":"failure"}`)
	}))
	defer srv.Close()

	_, err := New(srv.URL, srv.Client()).Get(context.Background(), "sys_security_acl", GetOptions{})
	var ce *output.Error
	if !errors.As(err, &ce) {
		t.Fatalf("want *output.Error, got %T", err)
	}
	if ce.Code != "auth_forbidden" || ce.Exit != output.ExitAuth {
		t.Errorf("got code=%q exit=%d, want auth_forbidden / ExitAuth", ce.Code, ce.Exit)
	}
}
