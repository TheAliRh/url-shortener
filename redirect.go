package main

import (
	"log"
	"os"
)

func redirectURL(shortURL string) (string, error) {

	host := os.Getenv("Host")

	url := &URL{}

	qry := "short_url = '" + host + "/" + shortURL + "'"

	cachedURL, cacheError := getOneFromCache(shortURL)
	if cacheError == nil {
		url = cachedURL
	} else {
		_, err := getOneFromDatabase("urls", qry, url)
		if err != nil {

			log.Fatal("[Error]:", err)
		}

		err = setOneToCache(shortURL, url)
		if err != nil {
			return "", err
		}

		return url.LongURL, err
	}
	return url.LongURL, cacheError
}
