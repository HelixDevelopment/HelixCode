package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAuthRepository is a mock implementation of AuthRepository
type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) CreateUser(ctx context.Context, user *User, passwordHash string) error {
	args := m.Called(ctx, user, passwordHash)
	return args.Error(0)
}

func (m *MockAuthRepository) GetUserByUsername(ctx context.Context, username string) (*User, string, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, "", args.Error(2)
	}
	return args.Get(0).(*User), args.String(1), args.Error(2)
}

func (m *MockAuthRepository) GetUserByEmail(ctx context.Context, email string) (*User, string, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, "", args.Error(2)
	}
	return args.Get(0).(*User), args.String(1), args.Error(2)
}

func (m *MockAuthRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockAuthRepository) UpdateUserLastLogin(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAuthRepository) CreateSession(ctx context.Context, session *Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockAuthRepository) GetSession(ctx context.Context, token string) (*Session, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Session), args.Error(1)
}

func (m *MockAuthRepository) DeleteSession(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockAuthRepository) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.NotEmpty(t, config.JWTSecret)
	assert.Equal(t, 24*time.Hour, config.TokenExpiry)
	assert.Equal(t, 7*24*time.Hour, config.SessionExpiry)
	assert.Equal(t, 10, config.BcryptCost) // bcrypt.DefaultCost
}

func TestNewAuthService(t *testing.T) {
	config := DefaultConfig()
	mockRepo := &MockAuthRepository{}
	service := NewAuthService(config, mockRepo)
	assert.NotNil(t, service)
	assert.Equal(t, config, service.config)
	assert.Equal(t, mockRepo, service.db)
}

func TestAuthService_validateRegistration(t *testing.T) {
	service := &AuthService{config: DefaultConfig()}

	tests := []struct {
		name     string
		username string
		email    string
		password string
		wantErr  bool
	}{
		{
			name:     "valid input",
			username: "testuser",
			email:    "test@example.com",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "empty username",
			username: "",
			email:    "test@example.com",
			password: "password123",
			wantErr:  true,
		},
		{
			name:     "invalid email",
			username: "testuser",
			email:    "invalid-email",
			password: "password123",
			wantErr:  true,
		},
		{
			name:     "short password",
			username: "testuser",
			email:    "test@example.com",
			password: "123",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateRegistration(tt.username, tt.email, tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthService_hashPassword(t *testing.T) {
	service := &AuthService{config: DefaultConfig()}

	password := "testpassword"
	hash, err := service.hashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Test password verification
	valid := service.verifyPassword(password, hash)
	assert.True(t, valid)

	// Test wrong password
	valid = service.verifyPassword("wrongpassword", hash)
	assert.False(t, valid)
}

func TestAuthService_GenerateJWT(t *testing.T) {
	service := &AuthService{config: DefaultConfig()}
	user := &User{
		ID:       uuid.New(),
		Username: "testuser",
		Email:    "test@example.com",
	}

	token, err := service.GenerateJWT(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token
	verifiedUser, err := service.VerifyJWT(token)
	require.NoError(t, err)
	assert.Equal(t, user.ID, verifiedUser.ID)
	assert.Equal(t, user.Username, verifiedUser.Username)
	assert.Equal(t, user.Email, verifiedUser.Email)
}

func TestAuthService_VerifyJWT(t *testing.T) {
	service := &AuthService{config: DefaultConfig()}

	// Test invalid token
	_, err := service.VerifyJWT("invalid-token")
	assert.Error(t, err)

	// Test expired token (simulate by creating token with past expiry)
	// This is harder to test without mocking time, so we'll skip for now
}

func TestAuthService_generateSessionToken(t *testing.T) {
	service := &AuthService{config: DefaultConfig()}

	token, err := service.generateSessionToken()
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Len(t, token, 44) // Should be 32 bytes base64 encoded (32 * 4/3 = 42.67, rounded up to 44)
}
