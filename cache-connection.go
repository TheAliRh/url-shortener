package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisDB *redis.Client

func initCache() error {

	host := os.Getenv("CacheHost")
	port := os.Getenv("CachePort")
	if host == "" || port == "" {
		return errors.New("missing required environment variables: CacheHost and CachePort")
	}

	password := os.Getenv("CachePassword")

	options := redis.Options{

		Addr:         net.JoinHostPort(host, port),
		Password:     password,
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	}

	if os.Getenv("CacheTLS") == "true" {
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	redisClient := redis.NewClient(&options)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		redisClient.Close()
		return fmt.Errorf("ping cache: %w", err)
	}

	redisDB = redisClient
	log.Println("[INFO] Cache connection established successfully")
	return nil
}

func closeCache() {
	if redisDB != nil {
		err := redisDB.Close()
		if err != nil {
			log.Printf("[ERROR] Failed to close cache connection: %v", err)
		}
		redisDB = nil
		log.Println("[INFO] Cache connection closed successfully")
	}
}
