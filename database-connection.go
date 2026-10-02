package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var dbPool *pgxpool.Pool

func initDB() error {

	// Load and validate environment variables for database connection
	env := make(map[string]string)
	for _, key := range []string{"DBHost", "DBPort", "DBUser", "DBPassword", "DBName"} {
		value := os.Getenv(key)
		if value == "" {
			return fmt.Errorf("environment variable %s is not set", key)
		}
		env[key] = value
	}

	sslMode := os.Getenv("DBSSLMode")
	if sslMode == "" {
		return errors.New("environment variable DBSSLMode is not set")
	}

	connURL := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(env["DBUser"], env["DBPassword"]),
		Host:     net.JoinHostPort(env["DBHost"], env["DBPort"]),
		Path:     "/" + env["DBName"],
		RawQuery: url.Values{"sslmode": {sslMode}}.Encode(),
	}

	config, err := pgxpool.ParseConfig(connURL.String())
	if err != nil {
		return fmt.Errorf("failed to parse database configuration: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = time.Minute
	config.ConnConfig.ConnectTimeout = 5 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		dbPool = nil
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		return fmt.Errorf("ping database: %w", err)
	}

	log.Println("Database connection established")
	return nil
}

func closeDB() {
	if dbPool != nil {
		dbPool.Close()
		dbPool = nil
		log.Println("Database connection closed")
	}
}
