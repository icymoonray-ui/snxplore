package introspect

import (
	"context"
	"errors"
	"strings"

	"github.com/icymoonray-ui/snxplore/internal/output"
	"github.com/icymoonray-ui/snxplore/internal/snclient"
)

// ACL is one access control rule from sys_security_acl plus its required roles.
type ACL struct {
	Operation      string   `json:"operation"` // read | write | create | delete | ...
	Name           string   `json:"name"`      // table | table.field | table.*
	Active         bool     `json:"active"`
	AdminOverrides bool     `json:"admin_overrides"`
	HasCondition   bool     `json:"has_condition"`
	HasScript      bool     `json:"has_script"`
	Roles          []string `json:"roles"` // explicitly-named roles required
}

// Access is the set of ACLs protecting a table.
//
// Note carries a human explanation when ACLs could not be read — reading
// sys_security_acl over REST commonly requires the security_admin role, which
// (unlike in the UI) does not "elevate" per-session for REST callers.
type Access struct {
	Table string `json:"table"`
	ACLs  []ACL  `json:"acls"`
	Note  string `json:"note,omitempty"`
}

// ResolveAccess reads the table-level and field-level ACLs for a table and the
// roles each requires. If the ACL tables are not readable with the current
// account (a permissions failure), it degrades gracefully: it returns an
// Access with an explanatory Note instead of an error.
func ResolveAccess(ctx context.Context, c *snclient.Client, table string) (*Access, error) {
	a := &Access{Table: table}

	acls, err := c.Get(ctx, "sys_security_acl", snclient.GetOptions{
		// table-level (name=table) and field/wildcard-level (name starts "table.")
		Query:  "name=" + table + "^ORnameSTARTSWITH" + table + ".^ORDERBYoperation^ORDERBYname",
		Fields: []string{"sys_id", "operation", "name", "active", "admin_overrides", "condition", "script"},
	})
	if err != nil {
		if isPermission(err) {
			a.Note = "could not read sys_security_acl with the current account — reading ACLs over the Table API typically requires the security_admin role granted persistently to the service account. Access details unavailable."
			return a, nil
		}
		return nil, err
	}

	rolesByACL, err := aclRoles(ctx, c, aclIDs(acls))
	if err != nil {
		// Roles unreadable but ACLs were: keep the ACLs, note the gap.
		if isPermission(err) {
			a.Note = "ACL rules listed, but sys_security_acl_role was not readable (security_admin may be required) — required roles omitted."
		} else {
			return nil, err
		}
	}

	for _, r := range acls {
		id := r.Str("sys_id")
		a.ACLs = append(a.ACLs, ACL{
			Operation:      r.Str("operation"),
			Name:           r.Str("name"),
			Active:         r.Str("active") == "true",
			AdminOverrides: r.Str("admin_overrides") == "true",
			HasCondition:   strings.TrimSpace(r.Str("condition")) != "",
			HasScript:      strings.TrimSpace(r.Str("script")) != "",
			Roles:          rolesByACL[id],
		})
	}
	return a, nil
}

func aclIDs(acls []snclient.Record) []string {
	ids := make([]string, 0, len(acls))
	for _, r := range acls {
		if id := r.Str("sys_id"); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// aclRoles maps each ACL sys_id to its required role names via the
// sys_security_acl_role junction.
func aclRoles(ctx context.Context, c *snclient.Client, ids []string) (map[string][]string, error) {
	out := map[string][]string{}
	if len(ids) == 0 {
		return out, nil
	}
	recs, err := c.Get(ctx, "sys_security_acl_role", snclient.GetOptions{
		Query:  "sys_security_aclIN" + strings.Join(ids, ","),
		Fields: []string{"sys_security_acl", "sys_user_role.name"},
	})
	if err != nil {
		return nil, err
	}
	for _, r := range recs {
		aclID := r.Str("sys_security_acl")
		role := r.Str("sys_user_role.name")
		if aclID != "" && role != "" {
			out[aclID] = append(out[aclID], role)
		}
	}
	return out, nil
}

// isPermission reports whether err is a 403 Forbidden (authenticated but
// lacking the role) — the case where ACL introspection should degrade
// gracefully. A 401 (not authenticated) is NOT treated as a permission issue;
// it propagates as a hard error so bad credentials aren't masked as
// "security_admin required".
func isPermission(err error) bool {
	var ce *output.Error
	return errors.As(err, &ce) && ce.Code == "auth_forbidden"
}
