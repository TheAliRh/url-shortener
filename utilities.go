package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/url"
	"strings"
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

const maxURLLength = 2048

func validateLongURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxURLLength {
		return "", fmt.Errorf("%w: empty or too long", ErrInvalid)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalid, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" { // url.Parse already lowercases the scheme
		return "", fmt.Errorf("%w: only http and https are allowed", ErrInvalid)
	}

	if u.Hostname() == "" {
		return "", fmt.Errorf("%w: missing host", ErrInvalid)
	}

	if normalizeHost(u) == baseHost {
		return "", fmt.Errorf("%w: URL points to this service", ErrInvalid)
	}

	if u.User != nil { // blocks https://trusted.com@evil.com tricks
		return "", fmt.Errorf("%w: credentials in URL are not allowed", ErrInvalid)
	}

	return u.String(), nil
}

func validCode(code string) bool {
	if len(code) != codeLength {
		return false
	}
	for i := 0; i < len(code); i++ {
		if strings.IndexByte(charset, code[i]) < 0 {
			return false
		}
	}
	return true
}
