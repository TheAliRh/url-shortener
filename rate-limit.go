package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/netip"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// rateRule describes one limit: up to `burst` requests may arrive at once,
// and one more request is allowed every `every` after that.
type rateRule struct {
	name  string // key namespace, so rules never share counters
	burst int64
	every time.Duration
}

// Tune these. "Sustained" is what a client can keep up indefinitely.
var (
	ruleAPIIP      = rateRule{"api_ip", 60, 500 * time.Millisecond}       // 120/min sustained, per IP, all /api routes
	ruleAPIUser    = rateRule{"api_user", 60, 500 * time.Millisecond}     // 120/min sustained, per user, authenticated routes
	ruleCreateIP   = rateRule{"create_ip", 30, 4 * time.Second}           // 15/min sustained, per IP, across all accounts
	ruleCreateUser = rateRule{"create_user", 10, 10 * time.Second}        // 6/min sustained, per user
	ruleRedirectIP = rateRule{"redirect_ip", 100, 200 * time.Millisecond} // 300/min sustained, per IP
)

// GCRA (generic cell rate algorithm): one key per subject holds the time the
// bucket "would be empty". It is atomic, needs a single key, and has no
// window-boundary bursts like a fixed-window counter.
// Returns {allowed (1/0), remaining, retry_after_ms}.
var gcra = redis.NewScript(`
local now      = tonumber(ARGV[1])
local interval = tonumber(ARGV[2])
local burst    = tonumber(ARGV[3])

local tat = tonumber(redis.call('GET', KEYS[1]) or now)
if tat < now then tat = now end

local new_tat = tat + interval
local over = new_tat - now - interval * burst
if over > 0 then
  return {0, 0, math.ceil(over)}
end

redis.call('SET', KEYS[1], string.format('%.0f', new_tat), 'PX', string.format('%.0f', new_tat - now))
return {1, math.floor((interval * burst - (new_tat - now)) / interval), 0}
`)

func allow(ctx context.Context, key string, r rateRule) (ok bool, remaining int64, retry time.Duration, err error) {
	res, err := gcra.Run(ctx, redisDB, []string{key},
		time.Now().UnixMilli(), r.every.Milliseconds(), r.burst).Int64Slice()
	if err != nil {
		return false, 0, 0, err
	}
	return res[0] == 1, res[1], time.Duration(res[2]) * time.Millisecond, nil
}

// rateLimit returns a middleware enforcing rule per subject (IP, user, ...).
// It fails open: if Redis is unavailable the request is allowed and logged.
func rateLimit(rule rateRule, subject func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "rl:gcra:" + rule.name + ":" + subject(c)

		ok, remaining, retry, err := allow(c.Request.Context(), key, rule)
		if err != nil {
			log.Printf("[ERROR] rate limit %s: %v", rule.name, err)
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.FormatInt(rule.burst, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

		if !ok {
			secs := int(math.Ceil(retry.Seconds()))
			c.Header("Retry-After", strconv.Itoa(secs))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": fmt.Sprintf("Too many requests. Try again in %d seconds.", secs),
			})
			return
		}
		c.Next()
	}
}

// byIP limits per client address. IPv6 clients are grouped by /64, because a
// single machine usually controls a whole /64 and could otherwise rotate addresses.
func byIP(c *gin.Context) string {
	raw := c.ClientIP()
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		return raw
	}
	addr = addr.Unmap()
	if addr.Is6() {
		if p, err := addr.Prefix(64); err == nil {
			return p.String()
		}
	}
	return addr.String()
}

// byUser limits per account, so rotating API keys or logging in again does not
// reset the counter. Requires authRequired() to have run first; if it has not,
// it falls back to the IP so a misplaced middleware never means "unlimited".
func byUser(c *gin.Context) string {
	if id := c.GetInt64(userIDKey); id != 0 {
		return "u" + strconv.FormatInt(id, 10)
	}
	return "ip:" + byIP(c)
}
