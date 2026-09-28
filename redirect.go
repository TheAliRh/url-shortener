package main

import (
	"log"
	"os"
)

func redirectURL(shortURL string) (string, error) {

	host := os.Getenv("Host")

	url := &URL{}

	qry := "short_url = '" + host + "/" + shortURL + "'"

	_, err := getOneFromDatabase("urls", qry, url)
	if err != nil {

		log.Fatal("[Error]:", err)
	}

	return url.LongURL, err
}
