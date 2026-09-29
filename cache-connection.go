package main

import (
	"os"

	"github.com/redis/go-redis/v9"
)

var redisDB *redis.Client

func initCache() {

	host := os.Getenv("CacheHost")
	port := os.Getenv("CachePort")
	password := os.Getenv("CachePassword")

	redisDB = redis.NewClient(&redis.Options{

		Addr:     host + ":" + port,
		Password: password,
		DB:       0,
	})

}

func closeCache() {
	if redisDB != nil {
		redisDB.Close()
	}
}
