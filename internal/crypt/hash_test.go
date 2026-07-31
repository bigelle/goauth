package crypt_test

import (
	"os"
	"testing"

	"github.com/bigelle/auth/internal/crypt"
	"github.com/stretchr/testify/assert"
)

func TestHashPassword_Success(t *testing.T) {
	os.Setenv("CRYPT_PEPPER", "top-secret-value")
	pass := "ziliboba42"

	hash, err := crypt.HashPassword(pass)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	isCorrect := crypt.VerifyPassword(string(pass), string(hash))
	assert.True(t, isCorrect)
}
