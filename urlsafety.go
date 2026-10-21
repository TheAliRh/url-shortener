package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/net/idna"
)

var (
	// ErrBlockedURL means the destination is on the blocklist or flagged by Safe Browsing.
	ErrBlockedURL = errors.New("blocked url")
	// ErrSafetyUnavailable is returned only when SafeBrowsingFailClosed=true and the check failed.
	ErrSafetyUnavailable = errors.New("safety check unavailable")
)

const blockedPage = `<!doctype html><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Link disabled</title><style>body{font:16px/1.5 system-ui,sans-serif;max-width:32rem;margin:20vh auto;padding:0 1rem;color:#18202e}h1{font-size:1.5rem}p{color:#667085}</style><h1>This link has been disabled</h1><p>It points to a page that may be unsafe, so the redirect was stopped.</p>`

var (
	blockedHosts atomic.Pointer[map[string]struct{}]
	safeBrowsing *sbClient
	sbFailClosed bool
)

// initURLSafety reads BlocklistFile, SafeBrowsingAPIKey and SafeBrowsingFailClosed.
// Call it once at startup, after initCache().
func initURLSafety() error {
	empty := map[string]struct{}{}
	blockedHosts.Store(&empty)

	if path := os.Getenv("BlocklistFile"); path != "" {
		if err := loadBlocklist(path); err != nil {
			return err
		}
		go reloadBlocklistOnSIGHUP(path)
	}

	if key := os.Getenv("SafeBrowsingAPIKey"); key != "" {
		safeBrowsing = &sbClient{apiKey: key, http: &http.Client{Timeout: 3 * time.Second}}
	} else {
		log.Printf("[WARN] SafeBrowsingAPIKey is not set: Safe Browsing checks are disabled")
	}
	sbFailClosed = os.Getenv("SafeBrowsingFailClosed") == "true"
	return nil
}

// ------------------------------------------------------------------ checks

// checkURLSafety runs the cheap local checks first, then Safe Browsing.
// rawURL must already have passed validateLongURL.
func checkURLSafety(ctx context.Context, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: unparsable url", ErrInvalid)
	}

	if reason := blocklistReason(u.Hostname()); reason != "" {
		log.Printf("[WARN] blocked url at creation: host=%q reason=%s", u.Hostname(), reason)
		return fmt.Errorf("%w: %s", ErrBlockedURL, reason)
	}

	if safeBrowsing != nil {
		return checkSafeBrowsing(ctx, rawURL)
	}
	return nil
}

// isBlockedDestination is the fast redirect-time check. It uses only the
// in-memory blocklist, so a domain added later stops existing links too.
func isBlockedDestination(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	return blocklistReason(u.Hostname()) != ""
}

// ---------------------------------------------------------------- blocklist

// asciiHost lowercases, strips a trailing dot and converts Unicode hosts to
// punycode, so "bücher.example" and "xn--bcher-kva.example" match the same entry.
func asciiHost(h string) (string, error) {
	h = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(h)), ".")
	if h == "" {
		return "", errors.New("empty host")
	}
	return idna.Lookup.ToASCII(h)
}

// blocklistReason returns "" if the host is allowed, or a short reason.
// A listed domain also blocks all of its subdomains.
func blocklistReason(host string) string {
	h, err := asciiHost(host)
	if err != nil {
		return "invalid host"
	}
	if _, err := netip.ParseAddr(h); err == nil {
		return "raw IP address"
	}

	p := blockedHosts.Load()
	if p == nil {
		return ""
	}
	set := *p
	for d := h; ; {
		if _, ok := set[d]; ok {
			return "listed domain"
		}
		i := strings.IndexByte(d, '.')
		if i < 0 {
			return ""
		}
		d = d[i+1:]
	}
}

// loadBlocklist reads one domain per line. '#' starts a comment. Hosts-file
// lines ("0.0.0.0 evil.com") are accepted, so public feeds work as-is.
// On error the previous list stays active.
func loadBlocklist(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open blocklist: %w", err)
	}
	defer f.Close()

	set := make(map[string]struct{})
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if h, err := asciiHost(fields[len(fields)-1]); err == nil {
			set[h] = struct{}{}
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("read blocklist: %w", err)
	}

	blockedHosts.Store(&set)
	log.Printf("[INFO] blocklist loaded: %d domains", len(set))
	return nil
}

// reloadBlocklistOnSIGHUP lets you update the file and run `kill -HUP <pid>`
// without restarting the server. (Linux/macOS only.)
func reloadBlocklistOnSIGHUP(path string) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP)
	for range ch {
		if err := loadBlocklist(path); err != nil {
			log.Printf("[ERROR] blocklist reload: %v", err)
		}
	}
}

// ------------------------------------------------------------ safe browsing

type sbClient struct {
	apiKey string
	http   *http.Client
}

type sbResponse struct {
	Threats []struct {
		URL         string   `json:"url"`
		ThreatTypes []string `json:"threatTypes"`
	} `json:"threats"`
	CacheDuration string `json:"cacheDuration"` // e.g. "300s"
}

// lookup calls Safe Browsing v5 urls:search. The key goes in a header, not the
// query string, so it can never end up in an error message or access log.
func (s *sbClient) lookup(ctx context.Context, rawURL string) (threats []string, cache time.Duration, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://safebrowsing.googleapis.com/v5/urls:search?urls="+url.QueryEscape(rawURL), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("X-Goog-Api-Key", s.apiKey)

	resp, err := s.http.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err // drop the URL being checked from the error text
		}
		return nil, 0, fmt.Errorf("safe browsing request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("safe browsing returned HTTP %d", resp.StatusCode)
	}

	var out sbResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return nil, 0, fmt.Errorf("decode safe browsing response: %w", err)
	}

	seen := map[string]bool{}
	for _, t := range out.Threats {
		for _, tt := range t.ThreatTypes {
			if !seen[tt] {
				seen[tt] = true
				threats = append(threats, tt)
			}
		}
	}
	// A match on any expression of the URL (host suffix, path prefix) still means a threat.
	if len(out.Threats) > 0 && len(threats) == 0 {
		threats = []string{"UNSPECIFIED"}
	}

	if secs, err := strconv.ParseFloat(strings.TrimSuffix(out.CacheDuration, "s"), 64); err == nil && secs > 0 {
		cache = time.Duration(secs * float64(time.Second))
	}
	return threats, cache, nil
}

func checkSafeBrowsing(ctx context.Context, rawURL string) error {
	sum := sha256.Sum256([]byte(rawURL))
	key := "sb:v5:" + hex.EncodeToString(sum[:])

	// Cached verdict: "clean" or a comma-separated list of threat types.
	switch v, err := redisDB.Get(ctx, key).Result(); {
	case err == nil && v == "clean":
		return nil
	case err == nil:
		return fmt.Errorf("%w: safe browsing (cached): %s", ErrBlockedURL, v)
	case !errors.Is(err, redis.Nil):
		log.Printf("[ERROR] safe browsing cache: %v", err)
	}

	threats, ttl, err := safeBrowsing.lookup(ctx, rawURL)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		if sbFailClosed {
			return ErrSafetyUnavailable
		}
		return nil // fail open: do not let a Google outage stop all link creation
	}

	if len(threats) > 0 {
		verdict := strings.Join(threats, ",")
		if ttl <= 0 {
			ttl = 5 * time.Minute
		}
		if err := redisDB.Set(ctx, key, verdict, ttl).Err(); err != nil {
			log.Printf("[ERROR] safe browsing cache: %v", err)
		}
		log.Printf("[WARN] blocked url at creation: safe browsing=%s", verdict)
		return fmt.Errorf("%w: safe browsing: %s", ErrBlockedURL, verdict)
	}

	// The API allows caching clean results for longer than it suggests (max 24h).
	ttl = min(max(ttl, 30*time.Minute), 24*time.Hour)
	if err := redisDB.Set(ctx, key, "clean", ttl).Err(); err != nil {
		log.Printf("[ERROR] safe browsing cache: %v", err)
	}
	return nil
}
