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

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()

	t.Run("successful registration", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		// Mock that user doesn't exist
		mockRepo.On("GetUserByUsername", ctx, "newuser").Return(nil, "", ErrUserNotFound)
		mockRepo.On("GetUserByEmail", ctx, "new@example.com").Return(nil, "", ErrUserNotFound)
		mockRepo.On("CreateUser", ctx, mock.AnythingOfType("*auth.User"), mock.AnythingOfType("string")).Return(nil)

		user, err := service.Register(ctx, "newuser", "new@example.com", "password123", "New User")
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "newuser", user.Username)
		assert.Equal(t, "new@example.com", user.Email)
		assert.Equal(t, "New User", user.DisplayName)
		assert.True(t, user.IsActive)
		assert.False(t, user.IsVerified)
		assert.False(t, user.MFAEnabled)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user already exists by username", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		existingUser := &User{ID: uuid.New(), Username: "existinguser"}
		mockRepo.On("GetUserByUsername", ctx, "existinguser").Return(existingUser, "hash", nil)

		user, err := service.Register(ctx, "existinguser", "new@example.com", "password123", "New User")
		assert.Error(t, err)
		assert.Equal(t, ErrUserExists, err)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user already exists by email", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		existingUser := &User{ID: uuid.New(), Email: "existing@example.com"}
		mockRepo.On("GetUserByUsername", ctx, "newuser").Return(nil, "", ErrUserNotFound)
		mockRepo.On("GetUserByEmail", ctx, "existing@example.com").Return(existingUser, "hash", nil)

		user, err := service.Register(ctx, "newuser", "existing@example.com", "password123", "New User")
		assert.Error(t, err)
		assert.Equal(t, ErrUserExists, err)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid username", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		user, err := service.Register(ctx, "ab", "new@example.com", "password123", "New User")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "username")
	})

	t.Run("invalid email", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		user, err := service.Register(ctx, "newuser", "invalid-email", "password123", "New User")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "email")
	})

	t.Run("weak password", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		user, err := service.Register(ctx, "newuser", "new@example.com", "123", "New User")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "password")
	})
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()

	t.Run("successful login by username", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		// Create test user and password hash
		userID := uuid.New()
		testUser := &User{
			ID:       userID,
			Username: "testuser",
			Email:    "test@example.com",
			IsActive: true,
		}
		passwordHash, _ := service.hashPassword("password123")

		// Set up mocks
		mockRepo.On("GetUserByUsername", ctx, "testuser").Return(testUser, passwordHash, nil)
		mockRepo.On("UpdateUserLastLogin", ctx, userID).Return(nil)
		mockRepo.On("CreateSession", ctx, mock.AnythingOfType("*auth.Session")).Return(nil)

		session, user, err := service.Login(ctx, "testuser", "password123", "web", "127.0.0.1", "Mozilla/5.0")
		require.NoError(t, err)
		assert.NotNil(t, session)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, "testuser", user.Username)
		assert.NotEmpty(t, session.SessionToken)
		assert.Equal(t, "web", session.ClientType)
		mockRepo.AssertExpectations(t)
	})

	t.Run("successful login by email", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		userID := uuid.New()
		testUser := &User{
			ID:       userID,
			Username: "testuser",
			Email:    "test@example.com",
			IsActive: true,
		}
		passwordHash, _ := service.hashPassword("password123")

		// Username lookup fails, email lookup succeeds
		mockRepo.On("GetUserByUsername", ctx, "test@example.com").Return(nil, "", ErrUserNotFound)
		mockRepo.On("GetUserByEmail", ctx, "test@example.com").Return(testUser, passwordHash, nil)
		mockRepo.On("UpdateUserLastLogin", ctx, userID).Return(nil)
		mockRepo.On("CreateSession", ctx, mock.AnythingOfType("*auth.Session")).Return(nil)

		session, user, err := service.Login(ctx, "test@example.com", "password123", "web", "127.0.0.1", "Mozilla/5.0")
		require.NoError(t, err)
		assert.NotNil(t, session)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		mockRepo.On("GetUserByUsername", ctx, "nonexistent").Return(nil, "", ErrUserNotFound)
		mockRepo.On("GetUserByEmail", ctx, "nonexistent").Return(nil, "", ErrUserNotFound)

		session, user, err := service.Login(ctx, "nonexistent", "password123", "web", "127.0.0.1", "Mozilla/5.0")
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCredentials, err)
		assert.Nil(t, session)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})

	t.Run("incorrect password", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		testUser := &User{
			ID:       uuid.New(),
			Username: "testuser",
			IsActive: true,
		}
		passwordHash, _ := service.hashPassword("correctpassword")

		mockRepo.On("GetUserByUsername", ctx, "testuser").Return(testUser, passwordHash, nil)

		session, user, err := service.Login(ctx, "testuser", "wrongpassword", "web", "127.0.0.1", "Mozilla/5.0")
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCredentials, err)
		assert.Nil(t, session)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})

	t.Run("inactive user", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		testUser := &User{
			ID:       uuid.New(),
			Username: "testuser",
			IsActive: false,
		}
		passwordHash, _ := service.hashPassword("password123")

		mockRepo.On("GetUserByUsername", ctx, "testuser").Return(testUser, passwordHash, nil)

		session, user, err := service.Login(ctx, "testuser", "password123", "web", "127.0.0.1", "Mozilla/5.0")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "deactivated")
		assert.Nil(t, session)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})
}

func TestAuthService_VerifySession(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()

	t.Run("valid session", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		userID := uuid.New()
		testSession := &Session{
			ID:           uuid.New(),
			UserID:       userID,
			SessionToken: "valid-token",
			ExpiresAt:    time.Now().Add(1 * time.Hour),
		}
		testUser := &User{
			ID:       userID,
			Username: "testuser",
			IsActive: true,
		}

		mockRepo.On("GetSession", ctx, "valid-token").Return(testSession, nil)
		mockRepo.On("GetUserByID", ctx, userID).Return(testUser, nil)

		user, err := service.VerifySession(ctx, "valid-token")
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, "testuser", user.Username)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid session token", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		mockRepo.On("GetSession", ctx, "invalid-token").Return(nil, ErrTokenInvalid)

		user, err := service.VerifySession(ctx, "invalid-token")
		assert.Error(t, err)
		assert.Equal(t, ErrTokenInvalid, err)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})

	t.Run("expired session", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		testSession := &Session{
			ID:           uuid.New(),
			UserID:       uuid.New(),
			SessionToken: "expired-token",
			ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
		}

		mockRepo.On("GetSession", ctx, "expired-token").Return(testSession, nil)
		mockRepo.On("DeleteSession", ctx, "expired-token").Return(nil)

		user, err := service.VerifySession(ctx, "expired-token")
		assert.Error(t, err)
		assert.Equal(t, ErrTokenExpired, err)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		userID := uuid.New()
		testSession := &Session{
			ID:           uuid.New(),
			UserID:       userID,
			SessionToken: "valid-token",
			ExpiresAt:    time.Now().Add(1 * time.Hour),
		}

		mockRepo.On("GetSession", ctx, "valid-token").Return(testSession, nil)
		mockRepo.On("GetUserByID", ctx, userID).Return(nil, ErrUserNotFound)

		user, err := service.VerifySession(ctx, "valid-token")
		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})

	t.Run("inactive user", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		userID := uuid.New()
		testSession := &Session{
			ID:           uuid.New(),
			UserID:       userID,
			SessionToken: "valid-token",
			ExpiresAt:    time.Now().Add(1 * time.Hour),
		}
		testUser := &User{
			ID:       userID,
			Username: "testuser",
			IsActive: false,
		}

		mockRepo.On("GetSession", ctx, "valid-token").Return(testSession, nil)
		mockRepo.On("GetUserByID", ctx, userID).Return(testUser, nil)

		user, err := service.VerifySession(ctx, "valid-token")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "deactivated")
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})
}

func TestAuthService_Logout(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()

	t.Run("successful logout", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		mockRepo.On("DeleteSession", ctx, "session-token").Return(nil)

		err := service.Logout(ctx, "session-token")
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("logout with error", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		mockRepo.On("DeleteSession", ctx, "session-token").Return(assert.AnError)

		err := service.Logout(ctx, "session-token")
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestAuthService_LogoutAll(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()

	t.Run("successful logout all", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		userID := uuid.New()
		mockRepo.On("DeleteUserSessions", ctx, userID).Return(nil)

		err := service.LogoutAll(ctx, userID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("logout all with error", func(t *testing.T) {
		mockRepo := &MockAuthRepository{}
		service := NewAuthService(config, mockRepo)

		userID := uuid.New()
		mockRepo.On("DeleteUserSessions", ctx, userID).Return(assert.AnError)

		err := service.LogoutAll(ctx, userID)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}
