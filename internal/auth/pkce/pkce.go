package pkce

import (
	"crypto/sha256"
	"encoding/base64"
	"math/rand"
	"time"
)

// GenerateVerifier generates a random code verifier for PKCE.
// It returns a string between 43 and 128 characters long.
func GenerateVerifier() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	const length = 128 // Maximum allowed length for better security

	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// GenerateChallenge generates the code challenge from the verifier using S256.
func GenerateChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	// Base64 URL encoding without padding
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
