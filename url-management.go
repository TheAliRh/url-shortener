package main

import (
	"errors"
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
		return "", err
	}

	return response, nil

}

func getShortURL(shortURL string) (any, error) { // Retrieve the URL data for the given short URL

	url, err := findURL(shortURL)
	if err != nil {
		return "", err
	}
	return url, nil
}

func findURL(shortURL string) (*URL, error) {

	cachedURL, err := getOneFromCache(shortURL)
	if err == nil {
		return cachedURL, nil
	}

	if !errors.Is(err, ErrCacheMiss) {
		log.Printf("[ERROR] cache: %v", err)

	}

	host := os.Getenv("Host")
	qry := "short_url = '" + host + "/" + shortURL + "'" // existing query, still injectable (see below)

	url, err := getOneFromDatabase("urls", qry, &URL{})
	if err != nil {
		return nil, err
	}

	// fill the cache for next time (key = the code only, same as getOneFromCache)
	if err := setOneToCache(shortURL, url); err != nil {
		log.Printf("[ERROR] cache: %v", err)
	}

	return url, nil
}

func getAllURLS() ([]URL, error) {

	tableName := "urls"

	docs, err := getAllFromDatabase(tableName)
	if err != nil {
		return nil, err
	}
	defer docs.Close()

	response := []URL{}

	for docs.Next() {

		doc := URL{}

		err := docs.Scan(
			&doc.ID,
			&doc.ShortURL,
			&doc.LongURL,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		response = append(response, doc)
	}

	return response, nil

}
