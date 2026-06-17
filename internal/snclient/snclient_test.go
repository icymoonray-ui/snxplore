package snclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
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

// TestGetPageTotal verifies GetPage surfaces X-Total-Count so callers can
// detect truncation (more rows match than were returned in this page).
func TestGetPageTotal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Total-Count", "4200")
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"result":[{"name":"a"},{"name":"b"}]}`)
	}))
	defer srv.Close()

	p, err := New(srv.URL, srv.Client()).GetPage(context.Background(), "sys_db_object", GetOptions{})
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	if p.Total != 4200 {
		t.Errorf("Total = %d, want 4200", p.Total)
	}
	if len(p.Records) != 2 {
		t.Errorf("Records = %d, want 2", len(p.Records))
	}
}

// TestGetPageNoTotalHeader checks Total is -1 when the header is absent, so a
// missing count is distinguishable from a real zero.
func TestGetPageNoTotalHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":[]}`)
	}))
	defer srv.Close()

	p, err := New(srv.URL, srv.Client()).GetPage(context.Background(), "x", GetOptions{})
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	if p.Total != -1 {
		t.Errorf("Total = %d, want -1 (header absent)", p.Total)
	}
}

// TestGetAllPaging verifies GetAll walks the offset until a short page and
// concatenates every record across pages.
func TestGetAllPaging(t *testing.T) {
	const total, pageSize = 23, 10
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		offset, _ := strconv.Atoi(r.URL.Query().Get("sysparm_offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("sysparm_limit"))
		if limit != pageSize {
			t.Errorf("page %d: limit = %d, want %d", requests, limit, pageSize)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":[`))
		for i := offset; i < offset+limit && i < total; i++ {
			if i > offset {
				w.Write([]byte(","))
			}
			fmt.Fprintf(w, `{"name":"rec%d"}`, i)
		}
		w.Write([]byte(`]}`))
	}))
	defer srv.Close()

	recs, err := New(srv.URL, srv.Client()).GetAll(context.Background(), "x", GetOptions{}, pageSize)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(recs) != total {
		t.Fatalf("got %d records, want %d", len(recs), total)
	}
	// 23 rows / 10 per page = pages of 10,10,3 → 3 requests (last is short).
	if requests != 3 {
		t.Errorf("made %d requests, want 3", requests)
	}
	if recs[0].Str("name") != "rec0" || recs[total-1].Str("name") != "rec22" {
		t.Errorf("boundary records wrong: first=%q last=%q", recs[0].Str("name"), recs[total-1].Str("name"))
	}
}

// TestGetAllExactMultiple checks the terminating extra request when the total is
// an exact multiple of the page size (final page is empty, not short-by-content).
func TestGetAllExactMultiple(t *testing.T) {
	const total, pageSize = 20, 10
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		offset, _ := strconv.Atoi(r.URL.Query().Get("sysparm_offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("sysparm_limit"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":[`))
		for i := offset; i < offset+limit && i < total; i++ {
			if i > offset {
				w.Write([]byte(","))
			}
			fmt.Fprintf(w, `{"name":"rec%d"}`, i)
		}
		w.Write([]byte(`]}`))
	}))
	defer srv.Close()

	recs, err := New(srv.URL, srv.Client()).GetAll(context.Background(), "x", GetOptions{}, pageSize)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(recs) != total {
		t.Fatalf("got %d records, want %d", len(recs), total)
	}
	// pages of 10,10, then an empty page signals the end → 3 requests.
	if requests != 3 {
		t.Errorf("made %d requests, want 3", requests)
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
