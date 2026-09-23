package main

import (
	"crypto/rand"
	"math/big"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Generate a random code of specified length
// Using crypto/rand for better randomness and less predictability compared to math/rand
func generateRandomCode(length int) string {

	code := make([]byte, length)

	for i := range code {
		number, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(err)
		}
		code[i] = charset[number.Int64()]
	}

	return string(code)
}
