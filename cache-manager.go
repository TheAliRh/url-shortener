package main

import (
	"context"
	"encoding/json"
)

var ctx = context.Background()

func setOneToCache(key string, value *URL) error {

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	err = redisDB.Set(ctx, key, data, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

func getOneFromCache(key string) (*URL, error) {

	response, err := redisDB.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	url := &URL{}

	err = json.Unmarshal([]byte(response), url)
	if err != nil {
		return nil, err
	}

	return url, nil
}
