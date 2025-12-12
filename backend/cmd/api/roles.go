package main

// Roles represents user roles in the system
type Role string

const (
	RoleUser  Role = "user"
	RoleAgent Role = "agent"
	RoleAdmin Role = "admin"
)

// grantRolePermissions assigns permissions based on the user's role
func (app *application) grantRolePermissions(userID int64, role Role) error {
	var permissions []string

	switch role {
	case RoleUser:
		permissions = []string{
			"properties:read",
			"reviews:read",
			"reviews:write",
			"inquiries:create",
			"inquiries:read",
		}
	case RoleAgent:
		permissions = []string{
			"properties:read",
			"properties:write",
			"properties:feature",
			"reviews:read",
			"reviews:write",
			"inquiries:create",
			"inquiries:read",
			"inquiries:manage",
		}
	case RoleAdmin:
		permissions = []string{
			"properties:read",
			"properties:write",
			"properties:delete",
			"properties:feature",
			"agents:manage",
			"reviews:read",
			"reviews:write",
			"reviews:moderate",
			"users:manage",
			"inquiries:create",
			"inquiries:manage",
			"inquiries:delete",
		}
	default:
		permissions = []string{
			"properties:read",
			"reviews:read",
			"reviews:write",
			"inquiries:create",
			"inquiries:read",
		}
	}

	return app.models.Permissions.AddForUser(userID, permissions...)
}
