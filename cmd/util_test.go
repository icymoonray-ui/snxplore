package cmd

import (
	"errors"
	"testing"

	"github.com/icymoonray-ui/snxplore/internal/output"
)

func TestValidateTableName(t *testing.T) {
	valid := []string{"incident", "sys_db_object", "x_nuvo_eam_elocation", "u_custom1", "cmdb_ci"}
	for _, n := range valid {
		if err := validateTableName(n); err != nil {
			t.Errorf("validateTableName(%q) = %v, want nil", n, err)
		}
	}
	invalid := []string{"", "1bad", "incident^ORname=sys_user", "in cident", "inc;drop", "a.b", "тест"}
	for _, n := range invalid {
		err := validateTableName(n)
		if err == nil {
			t.Errorf("validateTableName(%q) = nil, want error", n)
			continue
		}
		var ce *output.Error
		if !errors.As(err, &ce) || ce.Exit != output.ExitUsage {
			t.Errorf("validateTableName(%q): want usage error (exit 2), got %v", n, err)
		}
	}
}

// TestCommandRejectsBadTableName ensures validation runs before any network
// activity: with no instance configured we still get the validation error
// (invalid_table), not a connection/instance error.
func TestCommandRejectsBadTableName(t *testing.T) {
	c := newSchemaCmd()
	c.SetArgs([]string{"incident^ORname=sys_user"})
	c.SilenceErrors = true
	c.SilenceUsage = true

	err := c.Execute()
	if err == nil {
		t.Fatal("expected error for bad table name")
	}
	var ce *output.Error
	if !errors.As(err, &ce) || ce.Code != "invalid_table" {
		t.Fatalf("want invalid_table error, got %v", err)
	}
}

func TestTimeoutFlagDefault(t *testing.T) {
	root := NewRootCmd()
	f := root.PersistentFlags().Lookup("timeout")
	if f == nil {
		t.Fatal("no --timeout flag registered")
	}
	if f.DefValue != "30s" {
		t.Errorf("--timeout default = %q, want 30s", f.DefValue)
	}
}
