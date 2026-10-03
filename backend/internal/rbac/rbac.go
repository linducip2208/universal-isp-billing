package rbac

import "context"

// Permission is a coarse-grained action string, e.g. "customers:write".
type Permission string

const (
	CustomersRead  Permission = "customers:read"
	CustomersWrite Permission = "customers:write"
	BillingRead    Permission = "billing:read"
	BillingWrite   Permission = "billing:write"
	NetworkRead    Permission = "network:read"
	NetworkWrite   Permission = "network:write"
	ProvisionExec  Permission = "provisioning:execute"
	SystemAdmin    Permission = "system:admin"
)

// Role maps to a permission set.
type Role struct {
	Name        string
	Permissions []Permission
}

var defaults = map[string]Role{
	"superadmin": {Name: "superadmin", Permissions: []Permission{CustomersRead, CustomersWrite, BillingRead, BillingWrite, NetworkRead, NetworkWrite, ProvisionExec, SystemAdmin}},
	"admin":      {Name: "admin", Permissions: []Permission{CustomersRead, CustomersWrite, BillingRead, BillingWrite, NetworkRead, NetworkWrite, ProvisionExec}},
	"noc":        {Name: "noc", Permissions: []Permission{CustomersRead, BillingRead, NetworkRead, ProvisionExec}},
	"billing":    {Name: "billing", Permissions: []Permission{CustomersRead, BillingRead, BillingWrite}},
	"support":    {Name: "support", Permissions: []Permission{CustomersRead, BillingRead, NetworkRead}},
	"viewer":     {Name: "viewer", Permissions: []Permission{CustomersRead, BillingRead, NetworkRead}},
}

type ctxKey struct{}
type orgKey struct{}

func WithRoles(ctx context.Context, roles []string) context.Context {
	return context.WithValue(ctx, ctxKey{}, roles)
}

// WithOrg carries the tenant. All store access must use OrgOf.
func WithOrg(ctx context.Context, org string) context.Context {
	return context.WithValue(ctx, orgKey{}, org)
}

func OrgOf(ctx context.Context) string {
	v, _ := ctx.Value(orgKey{}).(string)
	return v
}

type scopeKey struct{}

// WithScopes carries API-key scopes (permission strings like "billing:read").
func WithScopes(ctx context.Context, scopes []string) context.Context {
	return context.WithValue(ctx, scopeKey{}, scopes)
}

func rolesOf(ctx context.Context) []string {
	v, _ := ctx.Value(ctxKey{}).([]string)
	return v
}

// RolesOf exposes caller roles for services (e.g. copilot permission checks).
func RolesOf(ctx context.Context) []string { return rolesOf(ctx) }

// Can reports whether any role in ctx grants p (roles or API-key scopes).
func Can(ctx context.Context, p Permission) bool {
	if scopes, _ := ctx.Value(scopeKey{}).([]string); scopes != nil {
		for _, sc := range scopes {
			if sc == string(p) || sc == string(SystemAdmin) || sc == "*" {
				return true
			}
		}
	}
	for _, r := range rolesOf(ctx) {
		role, ok := defaults[r]
		if !ok {
			continue
		}
		for _, perm := range role.Permissions {
			if perm == p || perm == SystemAdmin {
				return true
			}
		}
		// superadmin wildcard
		if r == "superadmin" {
			return true
		}
		_ = role
	}
	return false
}

func DefaultRoles() map[string]Role { return defaults }
