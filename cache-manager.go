package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

var ErrCacheMiss = errors.New("cache miss")

var ctx = context.Background()

func setOneToCache(key string, value *URL) error {

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal %q: %w", key, err)
	}

	err = redisDB.Set(ctx, key, data, 0).Err()
	if err != nil {
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
