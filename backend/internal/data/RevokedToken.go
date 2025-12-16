package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"time"
)

// RevokedTokenModel handles database operations for revoked tokens
type RevokedTokenModel struct {
	DB *sql.DB
}

// RevokedToken represents a revoked JWT token
type RevokedToken struct {
	ID        int64     `json:"id"`
	TokenHash []byte    `json:"-"`
	UserID    int64     `json:"user_id"`
	RevokedAt time.Time `json:"revoked_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Insert adds a revoked token to the database
func (m RevokedTokenModel) Insert(token string, userID int64, expiresAt time.Time) error {
	// Hash the token
	hash := sha256.Sum256([]byte(token))

	query := `
		INSERT INTO revoked_tokens (token_hash, user_id, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (token_hash) DO NOTHING`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, hash[:], userID, expiresAt)
	return err
}

// IsRevoked checks if a token has been revoked
func (m RevokedTokenModel) IsRevoked(token string) (bool, error) {
	// Hash the token
	hash := sha256.Sum256([]byte(token))

	query := `
		SELECT EXISTS(
			SELECT 1 FROM revoked_tokens 
			WHERE token_hash = $1 AND expires_at > NOW()
		)`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var exists bool
	err := m.DB.QueryRowContext(ctx, query, hash[:]).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// RevokeAllForUser revokes all tokens for a specific user
func (m RevokedTokenModel) RevokeAllForUser(userID int64) error {
	// This would typically be used when changing password or security breach
	// Since we don't store all active tokens, we can't revoke them all directly
	// Instead, we'd need to implement a user_version field and check it during auth
	query := `
		UPDATE users 
		SET version = version + 1 
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, userID)
	return err
}

// DeleteExpired removes expired revoked tokens (cleanup job)
func (m RevokedTokenModel) DeleteExpired() (int64, error) {
	query := `
		DELETE FROM revoked_tokens 
		WHERE expires_at < NOW()`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

// GetAllForUser retrieves all revoked tokens for a user (for debugging/admin)
func (m RevokedTokenModel) GetAllForUser(userID int64) ([]*RevokedToken, error) {
	query := `
		SELECT id, token_hash, user_id, revoked_at, expires_at
		FROM revoked_tokens
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY revoked_at DESC`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*RevokedToken

	for rows.Next() {
		var token RevokedToken
		err := rows.Scan(
			&token.ID,
			&token.TokenHash,
			&token.UserID,
			&token.RevokedAt,
			&token.ExpiresAt,
		)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, &token)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}
