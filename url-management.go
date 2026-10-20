package main

import (
	"errors"
	"fmt"
	"log"
	"time"
)

const (
	codeLength      = 7
	maxCodeAttempts = 5
)

func createShortURL(longURL string) (string, error) { // Create a short URL for the given long URL

	longURL, err := validateLongURL(longURL)
	if err != nil {
		return "", err
	}

	for attempt := 1; attempt <= maxCodeAttempts; attempt++ {
		// Implementation for creating short URL
		code := generateRandomCode(7)

		link, err := shortLink(code)
		if err != nil {
			return "", fmt.Errorf("build short link: %w", err)
		}

		url := URL{
			ID:        0, // ID will be auto-incremented by the database
			ShortURL:  code,
			LongURL:   longURL,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Add the short URL and long URL to the database
		err = addOneToDatabase(url, "urls")
		if err == nil {
			log.Printf("Short URL created: %s for long URL: %s\n", link, longURL)
			return link, nil
		}

		if !errors.Is(err, ErrDuplicate) {
			return "", fmt.Errorf("create short url: %w", err) // a real failure, don't retry
		}

		log.Printf("[WARN] short code collision (attempt %d/%d), retrying", attempt, maxCodeAttempts)
	}

	return "", fmt.Errorf("create short url: no free code after %d attempts", maxCodeAttempts)
}

func deleteURL(shortURL string) (string, error) {

	table := "urls"

	response, err := deleteOneFromDatabase("short_url = $1", table, shortURL)
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

	url, err := getOneFromDatabase("urls", "short_url = $1", &URL{}, shortURL)
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
