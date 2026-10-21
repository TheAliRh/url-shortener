package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

const userIDKey = "userID"

// newAPIKey returns the plaintext key (show it once), its hash, and a display prefix.
func newAPIKey() (plain string, hash []byte, prefix string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", nil, "", fmt.Errorf("generate api key: %w", err)
	}
	plain = "sk_" + base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(plain))
	return plain, sum[:], plain[:10], nil
}

func hashKey(plain string) []byte {
	sum := sha256.Sum256([]byte(plain))
	return sum[:]
}

func lookupAPIKey(ctx context.Context, plain string) (int64, error) {
	var userID int64
	err := dbPool.QueryRow(ctx,
		`UPDATE api_keys SET last_used_at = now()
		 WHERE key_hash = $1 AND revoked_at IS NULL
		 RETURNING user_id`, hashKey(plain)).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrUnauthorized
		}
		return 0, fmt.Errorf("lookup api key: %w", err)
	}
	return userID, nil
}

func authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || token == "" {
			c.Header("WWW-Authenticate", `Bearer realm="api"`)
			c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
			return
		}

		userID, err := authenticate(c.Request.Context(), token)
		if err != nil {
			if errors.Is(err, ErrUnauthorized) {
				c.Header("WWW-Authenticate", `Bearer realm="api"`)
				c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
				return
			}
			log.Printf("[ERROR] auth: %v", err)
			c.AbortWithStatusJSON(500, gin.H{"error": "Internal server error"})
			return
		}

		c.Set(userIDKey, userID)
		c.Next()
	}
}
