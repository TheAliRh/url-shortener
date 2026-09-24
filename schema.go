package main

import (
	"time"
)

type URL struct {
	ID        int       `db:"id" json:"id"`
	ShortURL  string    `db:"short_url" json:"short_url"`
	LongURL   string    `db:"long_url" json:"long_url"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
