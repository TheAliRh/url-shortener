package main

import (
	"fmt"
	"net/url"
	"os"
)

var baseURL string

// initBaseURL validates the public base URL once at startup.
func initBaseURL() error {
	raw := os.Getenv("BaseURL")
	if raw == "" {
		return fmt.Errorf("missing required environment variable BaseURL")
	}

	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("invalid BaseURL %q: must look like http://host:port or https://domain", raw)
	}

	baseURL = raw
	return nil
}

// shortLink builds the full, clickable link for a short code.
func shortLink(code string) (string, error) {
	return url.JoinPath(baseURL, "api", "redirect", code)
}
