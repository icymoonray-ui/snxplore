package cmd

import (
	"context"
	"os"

	"github.com/icymoonray-ui/snxplore/internal/auth"
	"github.com/icymoonray-ui/snxplore/internal/config"
	"github.com/icymoonray-ui/snxplore/internal/output"
	"github.com/icymoonray-ui/snxplore/internal/snclient"
)

func orDefault(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// clientForProfile builds a Table API client for the active --profile, taking
// non-secret settings from the config file and secrets from the environment.
//
// Auth defaults to "basic" (the POC path): set SNXPLORE_INSTANCE,
// SNXPLORE_USERNAME, SNXPLORE_PASSWORD. For OAuth, set SNXPLORE_AUTH to
// client_credentials or password and provide SNXPLORE_CLIENT_ID/SECRET.
// Secrets are read from env for now; OS-keyring storage is a later milestone.
func clientForProfile(ctx context.Context) (*snclient.Client, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, output.Errorf("config", output.ExitError, "load config: %v", err)
	}
	prof := cfg.Profiles[flagProfile]

	instance := orDefault(prof.Instance, os.Getenv("SNXPLORE_INSTANCE"))
	if instance == "" {
		return nil, output.Errorf("no_instance", output.ExitUsage,
			"no instance for profile %q; set SNXPLORE_INSTANCE (+ SNXPLORE_USERNAME/PASSWORD) or add a profile", flagProfile)
	}
	base := auth.BaseURL(instance)

	creds := auth.Credentials{
		Method:       auth.Method(orDefault(prof.Auth, orDefault(os.Getenv("SNXPLORE_AUTH"), string(auth.MethodBasic)))),
		ClientID:     orDefault(prof.ClientID, os.Getenv("SNXPLORE_CLIENT_ID")),
		ClientSecret: os.Getenv("SNXPLORE_CLIENT_SECRET"),
		Username:     orDefault(prof.Username, os.Getenv("SNXPLORE_USERNAME")),
		Password:     os.Getenv("SNXPLORE_PASSWORD"),
	}
	hc, err := auth.HTTPClient(ctx, base, creds)
	if err != nil {
		return nil, output.Errorf("auth", output.ExitAuth, "%v", err)
	}
	hc.Timeout = flagTimeout // bounds the full round-trip on both Basic and OAuth clients (0 = no timeout)
	return snclient.New(base, hc), nil
}
