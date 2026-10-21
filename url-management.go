package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

const (
	codeLength      = 7
	maxCodeAttempts = 5
)

func createShortURL(ctx context.Context, longURL string, userID int64) (string, error) { // Create a short URL for the given long URL

	longURL, err := validateLongURL(longURL)
	if err != nil {
		return "", err
	}

	if err := checkURLSafety(ctx, longURL); err != nil {
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
			UserID:    userID,
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

func deleteURL(shortURL string, userID int64) (string, error) {
	resp, err := deleteOneFromDatabase("short_url = $1 AND user_id = $2", "urls", shortURL, userID)
	if err != nil {
		return "", err // ErrNotFound if not found OR not owned
	}
	if err := deleteOneFromCache(shortURL); err != nil {
		log.Printf("[ERROR] cache: %v", err)
	}
	return resp, nil
}

func getShortURL(shortURL string, userID int64) (*URL, error) {
	// DB only, never the cache, because the cached copy has no owner
	return getOneFromDatabase("urls", "short_url = $1 AND user_id = $2", &URL{}, shortURL, userID)
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

func getAllURLS(userID int64, limit, offset int) ([]URL, error) {
	tableName := "urls"

	rows, err := getManyFromDatabase(tableName, "user_id = $1 ORDER BY id DESC LIMIT $2 OFFSET $3", userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	response := []URL{}

	for rows.Next() {

		doc := URL{}

		err := rows.Scan(
			&doc.ID,
			&doc.ShortURL,
			&doc.LongURL,
			&doc.CreatedAt,
			&doc.UpdatedAt,
			&doc.UserID,
		)
		if err != nil {
			return nil, err
		}

		response = append(response, doc)
	}

	return response, nil

}
