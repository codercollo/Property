package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
)

// Permissions holds permission codes for a user
type Permissions []string

// Include checks if a code exists in the Permissions slice
func (p Permissions) Include(code string) bool {
	for i := range p {
		if code == p[i] {
			return true
		}
	}
	return false
}

// PermissionModel wraps the DB connection
type PermissionModel struct {
	DB *sql.DB
}

// GetAllForUser returns all permission codes for the given user
func (m PermissionModel) GetAllForUser(userID int64) (Permissions, error) {
	query := `
		SELECT permission
		FROM user_permissions
		WHERE user_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions Permissions

	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}

// AddForUser assigns one or more permission codes to a user
func (m PermissionModel) AddForUser(userID int64, codes ...string) error {
	query := `
		INSERT INTO user_permissions (user_id, permission)
		SELECT $1, unnest($2::text[])
		ON CONFLICT (user_id, permission) DO NOTHING`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, userID, pq.Array(codes))
	return err
}

// RemoveAllForUser removes all permissions for a given user
func (m PermissionModel) RemoveAllForUser(userID int64) error {
	query := `DELETE FROM user_permissions WHERE user_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, userID)
	return err
}
