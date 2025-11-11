package auth

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"time"

	"dev.helix.code/internal/database"
	"github.com/google/uuid"
)

// AuthDB implements AuthRepository using PostgreSQL
type AuthDB struct {
	db database.DatabaseInterface
}

// NewAuthDB creates a new AuthDB instance
func NewAuthDB(db database.DatabaseInterface) *AuthDB {
	return &AuthDB{db: db}
}

// CreateUser creates a new user in the database
func (a *AuthDB) CreateUser(ctx context.Context, user *User, passwordHash string) error {
	query := `
		INSERT INTO users (id, username, email, password_hash, display_name, is_active, is_verified, mfa_enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := a.db.Exec(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		passwordHash,
		user.DisplayName,
		user.IsActive,
		user.IsVerified,
		user.MFAEnabled,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}

	return nil
}

// GetUserByUsername retrieves a user by username
func (a *AuthDB) GetUserByUsername(ctx context.Context, username string) (*User, string, error) {
	query := `
		SELECT id, username, email, password_hash, display_name, is_active, is_verified, mfa_enabled, last_login, created_at, updated_at
		FROM users
		WHERE username = $1`

	var user User
	var passwordHash string
	var lastLogin sql.NullTime

	err := a.db.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&passwordHash,
		&user.DisplayName,
		&user.IsActive,
		&user.IsVerified,
		&user.MFAEnabled,
		&lastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", ErrUserNotFound
		}
		return nil, "", fmt.Errorf("failed to get user by username: %v", err)
	}

	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time
	}

	return &user, passwordHash, nil
}

// GetUserByEmail retrieves a user by email
func (a *AuthDB) GetUserByEmail(ctx context.Context, email string) (*User, string, error) {
	query := `
		SELECT id, username, email, password_hash, display_name, is_active, is_verified, mfa_enabled, last_login, created_at, updated_at
		FROM users
		WHERE email = $1`

	var user User
	var passwordHash string
	var lastLogin sql.NullTime

	err := a.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&passwordHash,
		&user.DisplayName,
		&user.IsActive,
		&user.IsVerified,
		&user.MFAEnabled,
		&lastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", ErrUserNotFound
		}
		return nil, "", fmt.Errorf("failed to get user by email: %v", err)
	}

	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time
	}

	return &user, passwordHash, nil
}

// GetUserByID retrieves a user by ID
func (a *AuthDB) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, username, email, display_name, is_active, is_verified, mfa_enabled, last_login, created_at, updated_at
		FROM users
		WHERE id = $1`

	var user User
	var lastLogin sql.NullTime

	err := a.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.DisplayName,
		&user.IsActive,
		&user.IsVerified,
		&user.MFAEnabled,
		&lastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %v", err)
	}

	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time
	}

	return &user, nil
}

// UpdateUserLastLogin updates the last login time for a user
func (a *AuthDB) UpdateUserLastLogin(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET last_login = $1, updated_at = $1 WHERE id = $2`

	_, err := a.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update user last login: %v", err)
	}

	return nil
}

// CreateSession creates a new session in the database
func (a *AuthDB) CreateSession(ctx context.Context, session *Session) error {
	query := `
		INSERT INTO user_sessions (id, user_id, session_token, client_type, ip_address, user_agent, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	var ipAddr interface{}
	if session.IPAddress != nil {
		ipAddr = session.IPAddress.String()
	}

	_, err := a.db.Exec(ctx, query,
		session.ID,
		session.UserID,
		session.SessionToken,
		session.ClientType,
		ipAddr,
		session.UserAgent,
		session.ExpiresAt,
		session.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}

	return nil
}

// GetSession retrieves a session by token
func (a *AuthDB) GetSession(ctx context.Context, token string) (*Session, error) {
	query := `
		SELECT id, user_id, session_token, client_type, ip_address, user_agent, expires_at, created_at
		FROM user_sessions
		WHERE session_token = $1`

	var session Session
	var ipAddr sql.NullString

	err := a.db.QueryRow(ctx, query, token).Scan(
		&session.ID,
		&session.UserID,
		&session.SessionToken,
		&session.ClientType,
		&ipAddr,
		&session.UserAgent,
		&session.ExpiresAt,
		&session.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %v", err)
	}

	if ipAddr.Valid {
		session.IPAddress = net.ParseIP(ipAddr.String)
	}

	return &session, nil
}

// DeleteSession deletes a session by token
func (a *AuthDB) DeleteSession(ctx context.Context, token string) error {
	query := `DELETE FROM user_sessions WHERE session_token = $1`

	result, err := a.db.Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

// DeleteUserSessions deletes all sessions for a user
func (a *AuthDB) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM user_sessions WHERE user_id = $1`

	_, err := a.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %v", err)
	}

	return nil
}
