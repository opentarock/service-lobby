package util

import (
	"crypto/rand"
	"encoding/hex"
	"log"
)

// RandomToken generates a random token of specified byte length n with result
// being hex encoded.
// Token is generated using cryptographically secure random generator.
func RandomToken(n uint) string {
	token := make([]byte, n)
	_, err := rand.Read(token)
	if err != nil {
		log.Panicf("Error generating random token: %s", err)
	}
	return hex.EncodeToString(token)
}
