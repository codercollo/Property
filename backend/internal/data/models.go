package data

import (
	"database/sql"
	"errors"
)

// Common errors used across the data package
var (
	ErrPropertyNotFound = errors.New("property not found")
	ErrEditConflict     = errors.New("edit conflict")
	ErrUserNotFound     = errors.New("user not found")
	ErrReviewNotFound   = errors.New("review not found")
	ErrDuplicateEmail   = errors.New("duplicate email")
	ErrPaymentNotFound  = errors.New("payment not found")
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
	Admin       AdminModel
	Media       MediaModel
	Inquiries   InquiryModel
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
		Admin:       AdminModel{DB: db},
		Media:       MediaModel{DB: db},
		Inquiries:   InquiryModel{DB: db},
	}
}
