// Package auth authenticates HTTP requests to a ServiceNow instance.
//
// Three methods are supported behind one pluggable interface (HTTPClient):
//
//   - basic               — HTTP Basic (username/password). Simplest; used for
//                           the POC since the dev instance has no SSO. The
//                           documented Table API accepts Basic auth directly.
//   - client_credentials  — OAuth 2.0 against /oauth_token.do (no user).
//   - password            — OAuth 2.0 ROPC against /oauth_token.do.
//
// OAuth is retained for production/SSO scenarios; switching methods is a config
// change, not a rewrite. Secrets are passed in via Credentials; this package
// never reads or persists them.
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Method selects how requests are authenticated.
type Method string

const (
	MethodBasic             Method = "basic"
	MethodClientCredentials Method = "client_credentials"
	MethodPassword          Method = "password"
)

// Credentials describes how to authenticate to an instance.
type Credentials struct {
	Method       Method
	ClientID     string // OAuth methods
	ClientSecret string // OAuth methods
	Username     string // basic + password
	Password     string // basic + password
}

// BaseURL normalizes an instance identifier into a base URL.
// "dev12345" -> "https://dev12345.service-now.com"; a full URL is trimmed.
func BaseURL(instance string) string {
	s := strings.TrimSpace(instance)
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return strings.TrimRight(s, "/")
	}
	return "https://" + s + ".service-now.com"
}

// TokenURL returns the OAuth token endpoint for a base URL.
func TokenURL(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/oauth_token.do"
}

// HTTPClient returns an *http.Client that authenticates every request per the
// chosen method.
func HTTPClient(ctx context.Context, baseURL string, c Credentials) (*http.Client, error) {
	switch c.Method {
	case MethodBasic:
		if c.Username == "" || c.Password == "" {
			return nil, fmt.Errorf("basic auth requires username and password")
		}
		return &http.Client{Transport: &basicTransport{user: c.Username, pass: c.Password, rt: http.DefaultTransport}}, nil

	case MethodClientCredentials, MethodPassword:
		ts, err := tokenSource(ctx, baseURL, c)
		if err != nil {
			return nil, err
		}
		return oauth2.NewClient(ctx, ts), nil

	default:
		return nil, fmt.Errorf("unsupported auth method %q (want %q, %q, or %q)",
			c.Method, MethodBasic, MethodClientCredentials, MethodPassword)
	}
}

// basicTransport injects an HTTP Basic Authorization header on each request.
type basicTransport struct {
	user, pass string
	rt         http.RoundTripper
}

func (t *basicTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.SetBasicAuth(t.user, t.pass)
	return t.rt.RoundTrip(r)
}

// tokenSource builds a refreshing OAuth token source. ServiceNow accepts client
// credentials in the request body, so AuthStyleInParams is used.
func tokenSource(ctx context.Context, baseURL string, c Credentials) (oauth2.TokenSource, error) {
	tokenURL := TokenURL(baseURL)
	switch c.Method {
	case MethodClientCredentials:
		if c.ClientID == "" || c.ClientSecret == "" {
			return nil, fmt.Errorf("client_credentials requires client id and secret")
		}
		cfg := &clientcredentials.Config{
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			TokenURL:     tokenURL,
			AuthStyle:    oauth2.AuthStyleInParams,
		}
		return cfg.TokenSource(ctx), nil

	case MethodPassword:
		if c.ClientID == "" || c.Username == "" || c.Password == "" {
			return nil, fmt.Errorf("password grant requires client id, username and password")
		}
		cfg := &oauth2.Config{
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			Endpoint:     oauth2.Endpoint{TokenURL: tokenURL, AuthStyle: oauth2.AuthStyleInParams},
		}
		tok, err := cfg.PasswordCredentialsToken(ctx, c.Username, c.Password)
		if err != nil {
			return nil, fmt.Errorf("password grant token request failed: %w", err)
		}
		return cfg.TokenSource(ctx, tok), nil

	default:
		return nil, fmt.Errorf("unsupported OAuth grant %q", c.Method)
	}
}
