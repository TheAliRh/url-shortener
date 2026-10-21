package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

var (
	baseURL  string
	baseHost string // normalized "host:port" of BaseURL
)

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
	baseHost = normalizeHost(u)
	return nil
}

func normalizeHost(u *url.URL) string {
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")

	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return net.JoinHostPort(host, port)
}

// shortLink builds the full, clickable link for a short code.
func shortLink(code string) (string, error) {
	return url.JoinPath(baseURL, code)
}
