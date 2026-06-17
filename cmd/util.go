package cmd

import (
	"regexp"

	"github.com/icymoonray-ui/snxplore/internal/output"
)

func yesno(b bool) string {
	if b {
		return "yes"
	}
	return ""
}

// tableNameRe matches a valid ServiceNow table name. Used to reject input that
// could break out of an encoded query (which uses `^` and `=` as operators)
// and read from unintended tables.
var tableNameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// validateTableName rejects table names that aren't a plain identifier, before
// any network activity. Returns a usage error (exit 2).
func validateTableName(name string) error {
	if !tableNameRe.MatchString(name) {
		return output.Errorf("invalid_table", output.ExitUsage,
			"invalid table name %q: must match [a-zA-Z][a-zA-Z0-9_]*", name)
	}
	return nil
}
