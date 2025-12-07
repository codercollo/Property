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
	ErrReviewNotFound   = errors.New("review not found")
)

// Models wraps all model types
type Models struct {
	Properties  PropertyModel
	Users       UserModel
	Tokens      TokenModel
	Permissions PermissionModel
	Reviews     ReviewModel
	Payments    PaymentModel
	Agents      AgentModel
	Admin       AdminModel // NEW: Admin model for platform-wide operations
}

// NewModels initializes and returns a Models struct with the given DB connection
func NewModels(db *sql.DB) Models {
	return Models{
		Properties:  PropertyModel{DB: db},
		Users:       UserModel{DB: db},
		Tokens:      TokenModel{DB: db},
		Permissions: PermissionModel{DB: db},
		Reviews:     ReviewModel{DB: db},
		Payments:    PaymentModel{DB: db},
		Agents:      AgentModel{DB: db},
		Admin:       AdminModel{DB: db}, // NEW: Initialize Admin model
	}
}
