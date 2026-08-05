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

func TestSolvePKCEChallenge(t *testing.T) {
	cases := []struct {
		challenge   string
		verifier    string
		should_pass bool
	}{
		{
			challenge:   "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			verifier:    "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			should_pass: true,
		},
		{
			challenge:   "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			verifier:    "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXm", // wrong last symbol
			should_pass: false,
		},
		{
			challenge:   "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			verifier:    "",
			should_pass: false,
		},
		{
			challenge:   "",
			verifier:    "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			should_pass: false,
		},
		{
			challenge:   "",
			verifier:    "",
			should_pass: false,
		},
	}

	for _, c := range cases {
		result := crypt.SolvePKCEChallenge(c.challenge, c.verifier)
		assert.Equal(t, c.should_pass, result)
	}
}
