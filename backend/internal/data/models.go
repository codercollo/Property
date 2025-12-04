package data

import (
	"database/sql"
	"errors"
)

// ErrPropertyNotFound is returned when a Property cannot be found in the database
var (
	ErrPropertyNotFound = errors.New("property not found")
	ErrEditConflict     = errors.New("edit conflict")
	ErrUserNotFound     = errors.New("user not found")
)

// Models wraps all model types
type Models struct {
	Properties  PropertyModel
	Users       UserModel
	Tokens      TokenModel
	Permissions PermissionModel
}

// NewModels initializes and returns a Models struct with the given DB connection
func NewModels(db *sql.DB) Models {
	return Models{
		Properties:  PropertyModel{DB: db},
		Users:       UserModel{DB: db},
		Tokens:      TokenModel{DB: db},
		Permissions: PermissionModel{DB: db},
	}
}
