package main

//Roles represents user roles in the system
type Role string

const (
	RoleUser  Role = "user"
	RoleAgent Role = "agent"
	RoleAdmin Role = "admin"
)

//grantRolePermissions assigns permissions based on the user's role
func (app *application) grantRolePermissions(userID int64, role Role) error {
	var permissions []string

	switch role {
	//Regular users can only browse properties
	case RoleUser:
		permissions = []string{
			"properties:read",
			"reviews:read",
			"reviews:write",
		}
	//Agents can manage their own listings
	case RoleAgent:
		permissions = []string{
			"properties:read",
			"properties:write",
			"properties:feature",
			"reviews:read",
			"reviews:write",
		}

		// Admins have full control
	case RoleAdmin:
		permissions = []string{
			"properties:read",
			"properties:write",
			"properties:delete",
			"properties:feature",
			"agents:manage",
			"reviews:moderate",
			"users:manage",
			"reviews:read",
			"reviews:write",
			"reviews:moderate",
		}

	//Default to basic user permissions
	default:
		permissions = []string{
			"properties:read",
			"reviews:read",
			"reviews:write",
		}
	}

	return app.models.Permissions.AddForUser(userID, permissions...)
}
