package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func redirectURL(shortURL string) (string, error) {

	url, err := findURL(shortURL)
	if err != nil {
		return "", err
	}
	return url.LongURL, nil
}

func redirectHandler(c *gin.Context) {
	code := c.Param("code")

	if !validCode(code) { // skip the DB for junk like /wp-login.php
		c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
		return
	}

	originalURL, err := redirectURL(code)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
			return
		}
		log.Printf("[ERROR] redirect: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if isBlockedDestination(originalURL) {
		c.Data(http.StatusGone, "text/html; charset=utf-8", []byte(blockedPage))
		return
	}

	c.Redirect(http.StatusFound, originalURL)
}
