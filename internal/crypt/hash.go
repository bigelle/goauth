package crypt

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"

	"golang.org/x/crypto/bcrypt"
)

var pepper = []byte(os.Getenv("CRYPT_PEPPER"))

func HashPassword(pass string) (string, error) {
	passWithPepper := encodeString(pass)

	bytes, err := bcrypt.GenerateFromPassword([]byte(passWithPepper), bcrypt.DefaultCost)
	return string(bytes), err
}

func VerifyPassword(pass string, hash string) bool {
	passWithPepper := encodeString(pass)

	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(passWithPepper)) == nil
}

func encodeString(str string) string {
	mac := hmac.New(sha256.New, pepper)
	mac.Write([]byte(str))
	return hex.EncodeToString(mac.Sum(nil))
}

func MakeAuthCode() string {
	buf := make([]byte, 32)
	rand.Read(buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}

func HashAuthCodeSHA256(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
