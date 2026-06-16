package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBaseURL(t *testing.T) {
	cases := map[string]string{
		"dev12345":                      "https://dev12345.service-now.com",
		"  dev12345 ":                   "https://dev12345.service-now.com",
		"https://acme.service-now.com":  "https://acme.service-now.com",
		"https://acme.service-now.com/": "https://acme.service-now.com",
		"http://localhost:8080":         "http://localhost:8080",
	}
	for in, want := range cases {
		if got := BaseURL(in); got != want {
			t.Errorf("BaseURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTokenURL(t *testing.T) {
	if got, want := TokenURL("https://acme.service-now.com"), "https://acme.service-now.com/oauth_token.do"; got != want {
		t.Errorf("TokenURL = %q, want %q", got, want)
	}
}

func TestHTTPClientValidation(t *testing.T) {
	ctx := context.Background()
	if _, err := HTTPClient(ctx, "https://x.service-now.com", Credentials{Method: MethodBasic}); err == nil {
		t.Error("expected error for basic without username/password")
	}
	if _, err := HTTPClient(ctx, "https://x.service-now.com", Credentials{Method: MethodClientCredentials}); err == nil {
		t.Error("expected error for client_credentials without id/secret")
	}
	if _, err := HTTPClient(ctx, "https://x.service-now.com", Credentials{Method: "bogus"}); err == nil {
		t.Error("expected error for unsupported method")
	}
}

// TestBasicAuthHeader verifies the basic transport sets the Authorization header.
func TestBasicAuthHeader(t *testing.T) {
	var gotUser, gotPass string
	var ok bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, ok = r.BasicAuth()
	}))
	defer srv.Close()

	hc, err := HTTPClient(context.Background(), srv.URL, Credentials{
		Method: MethodBasic, Username: "admin", Password: "s3cret",
	})
	if err != nil {
		t.Fatalf("HTTPClient: %v", err)
	}
	resp, err := hc.Get(srv.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	resp.Body.Close()
	if !ok || gotUser != "admin" || gotPass != "s3cret" {
		t.Errorf("basic auth = (%q,%q,ok=%v)", gotUser, gotPass, ok)
	}
}
