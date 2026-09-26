package main

import (
	"log"
	"os"
	"time"
)

func createShortURL(longURL string) string { // Create a short URL for the given long URL

	host := os.Getenv("Host")

	// Implementation for creating short URL
	shortURL := host + "/" + generateRandomCode(7)

	log.Printf("Short URL created: %s for long URL: %s\n", shortURL, longURL)

	url := URL{
		ID:        0, // ID will be auto-incremented by the database
		ShortURL:  shortURL,
		LongURL:   longURL,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add the short URL and long URL to the database
	err := addOneToDatabase(url, "urls")
	if err != nil {
		log.Printf("Error adding URL to database: %v\n", err)
	}

	return shortURL
}

func deleteURL(shortURL string) (string, error) {

	host := os.Getenv("Host")

	url := "short_url = '" + host + "/" + shortURL + "'"

	table := "urls"

	response, err := deleteOneFromDatabase(url, table)
	if err != nil {
		log.Fatal("Error:", err)
		return "", err
	}

	return response, nil

}
