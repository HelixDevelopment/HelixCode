package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 5432, config.Port)
	assert.Equal(t, "testuser", config.User)
	assert.Equal(t, "testpass", config.Password)
	assert.Equal(t, "testdb", config.DBName)
	assert.Equal(t, "disable", config.SSLMode)
}

func TestNew_InvalidConfig(t *testing.T) {
	// Test with invalid host
	config := Config{
		Host:     "", // Invalid host
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	db, err := New(config)
	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to ping database")
}

func TestDatabase_Close(t *testing.T) {
	// Test Close on database with nil pool
	db := &Database{Pool: nil}
	// Should not panic
	db.Close()
}

func TestDatabase_GetDB(t *testing.T) {
	// Test GetDB on database with nil pool
	db := &Database{Pool: nil}
	sqlDB, err := db.GetDB()
	assert.Error(t, err)
	assert.Nil(t, sqlDB)
	assert.Contains(t, err.Error(), "database pool is not initialized")
}

func TestDatabase_HealthCheck(t *testing.T) {
	// Test HealthCheck on database with nil pool
	db := &Database{Pool: nil}
	err := db.HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database pool is not initialized")
}
