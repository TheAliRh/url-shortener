package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

const (
	sessionPrefix = "st_" // tokens issued by login/signup
	apiKeyPrefix  = "sk_" // long-lived keys (see newAPIKey)

	sessionTTL  = 7 * 24 * time.Hour
	maxAuthBody = 4 << 10 // 4 KiB is plenty for an email and a password

	signupPerIPPerHour = 10
	loginFailWindow    = 15 * time.Minute
	loginFailsPerIP    = 20 // all accounts, one IP
	loginFailsPerAcct  = 5  // one account, one IP
)

// dummyHash is verified against when an email is unknown, so "no such user"
// and "wrong password" take the same time.
var dummyHash string

func initAuth() error {
	h, err := hashPassword(context.Background(), "timing-equalizer-not-a-real-password")
	if err != nil {
		return fmt.Errorf("init auth: %w", err)
	}
	dummyHash = h
	return nil
}

type signupRequest struct {
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=10,max=128"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,max=128"`
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ---------------------------------------------------------------- handlers

func signupHandler(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAuthBody)
	ctx := c.Request.Context()

	if rateHit(ctx, "rl:signup:ip:"+c.ClientIP(), time.Hour) > signupPerIPPerHour {
		tooManyRequests(c, time.Hour)
		return
	}

	var req signupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide a valid email and a password of 10 to 128 characters"})
		return
	}
	email := normalizeEmail(req.Email)

	hash, err := hashPassword(ctx, req.Password)
	if err != nil {
		log.Printf("[ERROR] signup: hash password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	var userID int64
	err = dbPool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		email, hash).Scan(&userID)
	if err != nil {
		if isUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
			return
		}
		log.Printf("[ERROR] signup: insert user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	token, expires, err := createSession(ctx, userID)
	if err != nil {
		log.Printf("[ERROR] signup: create session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user":       gin.H{"id": userID, "email": email},
		"token":      token,
		"token_type": "Bearer",
		"expires_at": expires,
	})
}

func loginHandler(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAuthBody)
	ctx := c.Request.Context()

	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide an email and a password"})
		return
	}
	email := normalizeEmail(req.Email)

	ipKey := "rl:login:ip:" + c.ClientIP()
	acctKey := "rl:login:acct:" + c.ClientIP() + ":" + email

	if rateCount(ctx, ipKey) >= loginFailsPerIP || rateCount(ctx, acctKey) >= loginFailsPerAcct {
		tooManyRequests(c, loginFailWindow)
		return
	}

	fail := func() {
		rateHit(ctx, ipKey, loginFailWindow)
		rateHit(ctx, acctKey, loginFailWindow)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
	}

	var userID int64
	var hash *string // NULL for users created without a password
	err := dbPool.QueryRow(ctx,
		`SELECT id, password_hash FROM users WHERE lower(email) = $1`, email).Scan(&userID, &hash)

	switch {
	case errors.Is(err, pgx.ErrNoRows), err == nil && hash == nil:
		_, _ = verifyPassword(ctx, dummyHash, req.Password) // equalize timing
		fail()
		return
	case err != nil:
		log.Printf("[ERROR] login: find user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	ok, err := verifyPassword(ctx, *hash, req.Password)
	if err != nil {
		log.Printf("[ERROR] login: verify password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	if !ok {
		fail()
		return
	}

	token, expires, err := createSession(ctx, userID)
	if err != nil {
		log.Printf("[ERROR] login: create session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if err := redisDB.Del(ctx, acctKey).Err(); err != nil { // successful login resets this account's counter
		log.Printf("[ERROR] login: reset rate limit: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"user":       gin.H{"id": userID, "email": email},
		"token":      token,
		"token_type": "Bearer",
		"expires_at": expires,
	})
}

// logoutHandler ends the session behind the presented token. It must be
// mounted behind authRequired(). API keys are not affected.
func logoutHandler(c *gin.Context) {
	token, _ := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")

	if strings.HasPrefix(token, sessionPrefix) {
		if _, err := dbPool.Exec(c.Request.Context(),
			`DELETE FROM sessions WHERE token_hash = $1`, hashKey(token)); err != nil {
			log.Printf("[ERROR] logout: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
	}

	c.Status(http.StatusNoContent)
}

// ---------------------------------------------------------------- sessions

func newToken(prefix string) (plain string, hash []byte, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}
	plain = prefix + base64.RawURLEncoding.EncodeToString(b)
	return plain, hashKey(plain), nil
}

func createSession(ctx context.Context, userID int64) (string, time.Time, error) {
	plain, hash, err := newToken(sessionPrefix)
	if err != nil {
		return "", time.Time{}, err
	}

	expires := time.Now().UTC().Add(sessionTTL)
	if _, err := dbPool.Exec(ctx,
		`INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, hash, expires); err != nil {
		return "", time.Time{}, fmt.Errorf("insert session: %w", err)
	}

	return plain, expires, nil
}

func lookupSession(ctx context.Context, token string) (int64, error) {
	var userID int64
	err := dbPool.QueryRow(ctx,
		`SELECT user_id FROM sessions WHERE token_hash = $1 AND expires_at > now()`,
		hashKey(token)).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrUnauthorized
		}
		return 0, fmt.Errorf("lookup session: %w", err)
	}
	return userID, nil
}

// authenticate resolves a bearer token (session or API key) to a user ID.
func authenticate(ctx context.Context, token string) (int64, error) {
	switch {
	case strings.HasPrefix(token, sessionPrefix):
		return lookupSession(ctx, token)
	case strings.HasPrefix(token, apiKeyPrefix):
		return lookupAPIKey(ctx, token)
	default:
		return 0, ErrUnauthorized
	}
}

// ------------------------------------------------------------ rate limiting
// Both helpers fail open: if Redis is down, logins still work.

// rateHit increments the counter at key (creating it with a TTL if needed)
// and returns the new count.
func rateHit(ctx context.Context, key string, window time.Duration) int64 {
	pipe := redisDB.TxPipeline()
	pipe.SetNX(ctx, key, 0, window)
	incr := pipe.Incr(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("[ERROR] rate limit: %v", err)
		return 0
	}
	return incr.Val()
}

func rateCount(ctx context.Context, key string) int64 {
	n, err := redisDB.Get(ctx, key).Int64()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			log.Printf("[ERROR] rate limit: %v", err)
		}
		return 0
	}
	return n
}

func tooManyRequests(c *gin.Context, retryAfter time.Duration) {
	c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
	c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many attempts, try again later"})
}
