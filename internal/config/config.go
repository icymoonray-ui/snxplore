// Package config loads instance profiles. Non-secret settings live in a JSON
// config file (default ~/.config/snxplore/config.json); OAuth secrets/tokens
// are kept in the OS keyring (see internal/auth), never in this file.
package config

import (
	"os"
	"path/filepath"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Profile is a single named ServiceNow instance configuration.
type Profile struct {
	// Instance is the instance identifier or full base URL,
	// e.g. "dev12345" or "https://dev12345.service-now.com".
	Instance string `koanf:"instance"`
	// Auth is the auth method: "basic" (default, POC), "client_credentials",
	// or "password".
	Auth string `koanf:"auth"`
	// ClientID is the OAuth client ID (OAuth methods only).
	ClientID string `koanf:"client_id"`
	// Username is used for basic and password auth.
	Username string `koanf:"username"`
}

// Config is the full on-disk configuration: a set of named profiles.
type Config struct {
	Profiles map[string]Profile `koanf:"profiles"`
}

// DefaultPath returns the default config file location.
func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, "snxplore", "config.json")
}

// Load reads the config from path (or DefaultPath when empty). A missing file
// is not an error — it yields an empty config.
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultPath()
	}
	cfg := &Config{Profiles: map[string]Profile{}}
	if _, err := os.Stat(path); err != nil {
		return cfg, nil
	}
	k := koanf.New(".")
	if err := k.Load(file.Provider(path), json.Parser()); err != nil {
		return nil, err
	}
	if err := k.Unmarshal("", cfg); err != nil {
		return nil, err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return cfg, nil
}
