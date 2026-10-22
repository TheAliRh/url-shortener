package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

const (
	defaultCacheTTL = 24 * time.Hour
	cacheTTLJitter  = 0.10 // spread expiry by ±10% so popular links don't all expire together
)

var cacheTTL = defaultCacheTTL

func initCacheTTL() error {
	raw := os.Getenv("CacheTTL")
	if raw == "" {
		return nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < time.Minute {
		return fmt.Errorf("invalid CacheTTL %q: use a duration of at least 1m, for example 24h", raw)
	}
	cacheTTL = d
	return nil
}

func ttlWithJitter() time.Duration {
	spread := float64(cacheTTL) * cacheTTLJitter * (rand.Float64()*2 - 1)
	return cacheTTL + time.Duration(spread)
}

func setOneToCache(key string, value *URL) error {

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal %q: %w", key, err)
	}

	if err := redisDB.Set(ctx, key, data, ttlWithJitter()).Err(); err != nil {
		return fmt.Errorf("cache set %q: %w", key, err)
	}

	return nil
}

func getOneFromCache(key string) (*URL, error) {

	response, err := redisDB.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrCacheMiss
		}
		return nil, fmt.Errorf("cache get %q: %w", key, err)
	}

	url := &URL{}

	if err := json.Unmarshal([]byte(response), url); err != nil {
		return nil, fmt.Errorf("cache decode %q: %w", key, err)
	}

	return url, nil
}

func deleteOneFromCache(key string) error {
	if err := redisDB.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("cache delete %q: %w", key, err)
	}
	return nil
}
