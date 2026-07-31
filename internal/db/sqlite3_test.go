package db_test

import (
	"context"
	"testing"

	"github.com/bigelle/auth/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserByEmail(t *testing.T) {
	conn, err := db.NewDatabase(db.SQLITE3_driver, ":memory:")
	require.NoError(t, err, "failed to open a sqlite3 connection")
	defer conn.Close()

	initialUser := db.User{
		Name:         "Mike Wazowski",
		Email:        "mike_wazowski@example.com",
		PasswordHash: "hashed_password_AABBCC",
	}

	err = conn.CreateUser(context.Background(), initialUser.Name, initialUser.Email, initialUser.PasswordHash)
	require.NoError(t, err, "failed to insert new record")

	requestedUser, err := conn.GetUserByEmail(context.Background(), initialUser.Email)
	require.NoError(t, err, "failed to request a user")

	require.NotNil(t, initialUser, "user is nil")
	assert.Equal(t, initialUser.Name, requestedUser.Name)
	assert.Equal(t, initialUser.Email, requestedUser.Email)
	assert.Equal(t, initialUser.PasswordHash, requestedUser.PasswordHash)
}
