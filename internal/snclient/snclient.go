// Package snclient is the generic ServiceNow Table API client — the spine of
// snxplore. One introspective interface reads any table (OOB or custom) via
// /api/now/table/{table}; everything else (schema, forms, logic, ACLs, flows)
// is built on top of it by querying metadata tables.
package snclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/icymoonray-ui/snxplore/internal/output"
)

// Client talks to one instance's Table API. HTTP carries the OAuth token
// (e.g. via oauth2.NewClient); for tests it can be any *http.Client.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New builds a client for the given base URL.
func New(baseURL string, hc *http.Client) *Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), HTTP: hc}
}

// Record is a single Table API row. With sysparm_display_value=false (default)
// values are JSON strings; reference fields without dot-walking come back as
// {"value","link"} objects, so values are kept as raw any.
type Record map[string]any

// GetOptions are the supported Table API query parameters.
type GetOptions struct {
	Query        string   // sysparm_query (encoded query string)
	Fields       []string // sysparm_fields
	Limit        int      // sysparm_limit (0 = unset)
	Offset       int      // sysparm_offset
	DisplayValue string   // sysparm_display_value: "" | "false" | "true" | "all"
}

// buildURL constructs the request URL for a table read. Kept separate so query
// construction is unit-testable without a live instance.
func (c *Client) buildURL(table string, opt GetOptions) string {
	q := url.Values{}
	if opt.Query != "" {
		q.Set("sysparm_query", opt.Query)
	}
	if len(opt.Fields) > 0 {
		q.Set("sysparm_fields", strings.Join(opt.Fields, ","))
	}
	if opt.Limit > 0 {
		q.Set("sysparm_limit", strconv.Itoa(opt.Limit))
	}
	if opt.Offset > 0 {
		q.Set("sysparm_offset", strconv.Itoa(opt.Offset))
	}
	if opt.DisplayValue != "" {
		q.Set("sysparm_display_value", opt.DisplayValue)
	}
	u := c.BaseURL + "/api/now/table/" + url.PathEscape(table)
	if enc := q.Encode(); enc != "" {
		u += "?" + enc
	}
	return u
}

// tableResponse is the envelope returned by the Table API.
type tableResponse struct {
	Result []Record `json:"result"`
}

// errorResponse is ServiceNow's error envelope.
type errorResponse struct {
	Error struct {
		Message string `json:"message"`
		Detail  string `json:"detail"`
	} `json:"error"`
	Status string `json:"status"`
}

// Get reads records from a table.
func (c *Client) Get(ctx context.Context, table string, opt GetOptions) ([]Record, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildURL(table, opt), nil)
	if err != nil {
		return nil, output.Errorf("request_build", output.ExitError, "build request: %v", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, output.Errorf("network", output.ExitError, "request to %s failed: %v", c.BaseURL, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(table, resp.StatusCode, body)
	}

	var tr tableResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, output.Errorf("decode", output.ExitAPI, "decode response from %s: %v", table, err)
	}
	return tr.Result, nil
}

// apiError maps a non-200 response to a coded output.Error.
func apiError(table string, status int, body []byte) error {
	msg := strings.TrimSpace(string(body))
	var er errorResponse
	if json.Unmarshal(body, &er) == nil && er.Error.Message != "" {
		msg = er.Error.Message
		if er.Error.Detail != "" {
			msg += ": " + er.Error.Detail
		}
	}
	switch status {
	case http.StatusUnauthorized:
		// 401 = not authenticated (bad/missing credentials) — a hard error.
		return output.Errorf("auth_unauthorized", output.ExitAuth, "not authenticated (table %q): %s", table, msg)
	case http.StatusForbidden:
		// 403 = authenticated but lacking the role — callers may degrade on this.
		return output.Errorf("auth_forbidden", output.ExitAuth, "forbidden (table %q): %s", table, msg)
	case http.StatusNotFound:
		return output.Errorf("not_found", output.ExitNotFound, "table %q not found: %s", table, msg)
	default:
		return output.Errorf("api_error", output.ExitAPI, "table %q returned %d: %s", table, status, msg)
	}
}

// Str returns a record field as a string. Table API string fields decode as
// strings; reference fields (without dot-walk) decode as {"value","link"}
// objects, from which the value is extracted.
func (r Record) Str(field string) string {
	switch v := r[field].(type) {
	case string:
		return v
	case map[string]any:
		if s, ok := v["value"].(string); ok {
			return s
		}
	case nil:
		return ""
	}
	return fmt.Sprintf("%v", r[field])
}
