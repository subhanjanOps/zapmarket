package crypto

import (
	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost is the bcrypt cost factor
	BcryptCost = bcrypt.DefaultCost
)

// HashPassword generates a bcrypt hash of the password
func HashPassword(plaintext string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword compares a bcrypt hash with plaintext password
func VerifyPassword(hash, plaintext string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext))
	return err == nil
}
