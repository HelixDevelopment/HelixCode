package auth

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"testing"
	"time"

	"dev.helix.code/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ========================================
// AuthDB Tests with MockDatabase
// ========================================

func TestNewAuthDB(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	assert.NotNil(t, authDB)
	assert.NotNil(t, authDB.db)
}

func TestAuthDB_CreateUserSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	user := &User{
		ID:          uuid.New(),
		Username:    "testuser",
		Email:       "test@example.com",
		DisplayName: "Test User",
		IsActive:    true,
		IsVerified:  false,
		MFAEnabled:  false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	passwordHash := "$2a$10$hashedpassword"

	// Mock successful insert
	mockDB.MockExecSuccess(1)

	err := authDB.CreateUser(context.Background(), user, passwordHash)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestAuthDB_CreateUserDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	user := &User{
		ID:       uuid.New(),
		Username: "testuser",
	}
	passwordHash := "$2a$10$hashedpassword"

	dbError := errors.New("database connection failed")
	mockDB.MockExecError(dbError)

	err := authDB.CreateUser(context.Background(), user, passwordHash)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create user")
	mockDB.AssertExpectations(t)
}

func TestAuthDB_GetUserByUsernameSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	userID := uuid.New()
	now := time.Now()
	lastLogin := sql.NullTime{Time: now, Valid: true}
	passwordHash := "$2a$10$hashedpassword"

	mockRow := database.NewMockRowWithValues(
		userID,
		"testuser",
		"test@example.com",
		passwordHash,
		"Test User",
		true,  // is_active
		false, // is_verified
		false, // mfa_enabled
		lastLogin,
		now, // created_at
		now, // updated_at
	)

	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	user, hash, err := authDB.GetUserByUsername(context.Background(), "testuser")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, passwordHash, hash)
	assert.Equal(t, now.Unix(), user.LastLogin.Unix())
	mockDB.AssertExpectations(t)
}

func TestAuthDB_GetUserByUsernameNotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	mockRow := database.NewMockRowWithError(sql.ErrNoRows)
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	user, hash, err := authDB.GetUserByUsername(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
	assert.Nil(t, user)
	assert.Empty(t, hash)
	mockDB.AssertExpectations(t)
}

func TestAuthDB_GetUserByUsernameNoLastLogin(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	userID := uuid.New()
	now := time.Now()
	noLastLogin := sql.NullTime{Valid: false}
	passwordHash := "$2a$10$hashedpassword"

	mockRow := database.NewMockRowWithValues(
		userID,
		"testuser",
		"test@example.com",
		passwordHash,
		"Test User",
		true,
		false,
		false,
		noLastLogin,
		now,
		now,
	)

	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	user, hash, err := authDB.GetUserByUsername(context.Background(), "testuser")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, passwordHash, hash)
	assert.True(t, user.LastLogin.IsZero())
	mockDB.AssertExpectations(t)
}

func TestAuthDB_GetUserByEmailSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	userID := uuid.New()
	now := time.Now()
	lastLogin := sql.NullTime{Time: now, Valid: true}
	passwordHash := "$2a$10$hashedpassword"

	mockRow := database.NewMockRowWithValues(
		userID,
		"testuser",
		"test@example.com",
		passwordHash,
		"Test User",
		true,
		true,
		false,
		lastLogin,
		now,
		now,
	)

	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	user, hash, err := authDB.GetUserByEmail(context.Background(), "test@example.com")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, passwordHash, hash)
	mockDB.AssertExpectations(t)
}

func TestAuthDB_GetUserByEmailNotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	mockRow := database.NewMockRowWithError(sql.ErrNoRows)
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	user, hash, err := authDB.GetUserByEmail(context.Background(), "notfound@example.com")

	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
	assert.Nil(t, user)
	assert.Empty(t, hash)
	mockDB.AssertExpectations(t)
}

func TestAuthDB_GetUserByIDSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	userID := uuid.New()
	now := time.Now()
	lastLogin := sql.NullTime{Time: now, Valid: true}

	mockRow := database.NewMockRowWithValues(
		userID,
		"testuser",
		"test@example.com",
		"Test User",
		true,
		true,
		true, // mfa_enabled
		lastLogin,
		now,
		now,
	)

	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	user, err := authDB.GetUserByID(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "testuser", user.Username)
	assert.True(t, user.MFAEnabled)
	assert.Equal(t, now.Unix(), user.LastLogin.Unix())
	mockDB.AssertExpectations(t)
}

func TestAuthDB_GetUserByIDNotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	userID := uuid.New()

	mockRow := database.NewMockRowWithError(sql.ErrNoRows)
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	user, err := authDB.GetUserByID(context.Background(), userID)

	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
	assert.Nil(t, user)
	mockDB.AssertExpectations(t)
}

func TestAuthDB_GetUserByIDDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	userID := uuid.New()
	dbError := errors.New("connection lost")

	mockRow := database.NewMockRowWithError(dbError)
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	user, err := authDB.GetUserByID(context.Background(), userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user by ID")
	assert.Nil(t, user)
	mockDB.AssertExpectations(t)
}

func TestAuthDB_UpdateUserLastLoginSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	userID := uuid.New()

	mockDB.MockExecSuccess(1)

	err := authDB.UpdateUserLastLogin(context.Background(), userID)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestAuthDB_UpdateUserLastLoginDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	userID := uuid.New()
	dbError := errors.New("update failed")

	mockDB.MockExecError(dbError)

	err := authDB.UpdateUserLastLogin(context.Background(), userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update user last login")
	mockDB.AssertExpectations(t)
}

func TestAuthDB_CreateSessionSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	session := &Session{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		SessionToken: "token123",
		ClientType:   "web",
		IPAddress:    net.ParseIP("192.168.1.1"),
		UserAgent:    "Mozilla/5.0",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
	}

	mockDB.MockExecSuccess(1)

	err := authDB.CreateSession(context.Background(), session)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestAuthDB_CreateSessionNoIPAddress(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	session := &Session{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		SessionToken: "token123",
		ClientType:   "mobile",
		IPAddress:    nil,
		UserAgent:    "MobileApp/1.0",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
	}

	mockDB.MockExecSuccess(1)

	err := authDB.CreateSession(context.Background(), session)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestAuthDB_CreateSessionDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	session := &Session{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		SessionToken: "token123",
	}

	dbError := errors.New("insert failed")
	mockDB.MockExecError(dbError)

	err := authDB.CreateSession(context.Background(), session)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create session")
	mockDB.AssertExpectations(t)
}

func TestAuthDB_GetSessionSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	sessionID := uuid.New()
	userID := uuid.New()
	now := time.Now()
	ipAddr := sql.NullString{String: "192.168.1.1", Valid: true}

	mockRow := database.NewMockRowWithValues(
		sessionID,
		userID,
		"token123",
		"web",
		ipAddr,
		"Mozilla/5.0",
		now.Add(24*time.Hour),
		now,
	)

	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	session, err := authDB.GetSession(context.Background(), "token123")

	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, sessionID, session.ID)
	assert.Equal(t, userID, session.UserID)
	assert.Equal(t, "token123", session.SessionToken)
	assert.NotNil(t, session.IPAddress)
	mockDB.AssertExpectations(t)
}

func TestAuthDB_GetSessionNoIPAddress(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	sessionID := uuid.New()
	userID := uuid.New()
	now := time.Now()
	noIP := sql.NullString{Valid: false}

	mockRow := database.NewMockRowWithValues(
		sessionID,
		userID,
		"token456",
		"mobile",
		noIP,
		"MobileApp/1.0",
		now.Add(24*time.Hour),
		now,
	)

	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	session, err := authDB.GetSession(context.Background(), "token456")

	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Nil(t, session.IPAddress)
	mockDB.AssertExpectations(t)
}

func TestAuthDB_GetSessionNotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	mockRow := database.NewMockRowWithError(sql.ErrNoRows)
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	session, err := authDB.GetSession(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
	assert.Nil(t, session)
	mockDB.AssertExpectations(t)
}

func TestAuthDB_GetSessionDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	dbError := errors.New("query failed")
	mockRow := database.NewMockRowWithError(dbError)
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	session, err := authDB.GetSession(context.Background(), "token")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get session")
	assert.Nil(t, session)
	mockDB.AssertExpectations(t)
}

func TestAuthDB_DeleteSessionSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	mockDB.MockExecSuccess(1)

	err := authDB.DeleteSession(context.Background(), "token123")

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestAuthDB_DeleteSessionNotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	mockDB.MockExecSuccess(0)

	err := authDB.DeleteSession(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
	mockDB.AssertExpectations(t)
}

func TestAuthDB_DeleteSessionDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	dbError := errors.New("delete failed")
	mockDB.MockExecError(dbError)

	err := authDB.DeleteSession(context.Background(), "token")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete session")
	mockDB.AssertExpectations(t)
}

func TestAuthDB_DeleteUserSessionsSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	userID := uuid.New()

	mockDB.MockExecSuccess(3)

	err := authDB.DeleteUserSessions(context.Background(), userID)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestAuthDB_DeleteUserSessionsNoSessions(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	userID := uuid.New()

	mockDB.MockExecSuccess(0)

	err := authDB.DeleteUserSessions(context.Background(), userID)

	assert.NoError(t, err) // Function doesn't check rows affected
	mockDB.AssertExpectations(t)
}

func TestAuthDB_DeleteUserSessionsDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	authDB := NewAuthDB(mockDB)

	userID := uuid.New()
	dbError := errors.New("delete failed")

	mockDB.MockExecError(dbError)

	err := authDB.DeleteUserSessions(context.Background(), userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete user sessions")
	mockDB.AssertExpectations(t)
}
