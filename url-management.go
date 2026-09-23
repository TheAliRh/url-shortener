package main

import (
	"log"
	"os"
)

func createShortURL(longURL string) string {

	host := os.Getenv("Host")

	// Implementation for creating short URL
	shortURL := host + "/" + generateRandomCode(7)

	log.Printf("Short URL created: %s for long URL: %s\n", shortURL, longURL)

	return shortURL
}
