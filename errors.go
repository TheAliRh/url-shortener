package main

import "errors"

var (
	ErrInvalidURL   = errors.New("invalid URL")
	ErrNotFound     = errors.New("not found")
	ErrCacheMiss    = errors.New("cache miss")
	ErrDuplicate    = errors.New("duplicate")
	ErrInvalid      = errors.New("invalid input")
	ErrUnauthorized = errors.New("unauthorized")
)
