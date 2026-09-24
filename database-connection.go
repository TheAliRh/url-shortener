package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var dbPool *pgxpool.Pool

func initDB() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	dbHost := os.Getenv("DBHost")
	dbPort := os.Getenv("DBPort")
	dbUser := os.Getenv("DBUser")
	dbPassword := os.Getenv("DBPassword")
	dbName := os.Getenv("DBName")

	connStr := "postgres://" + dbUser + ":" + dbPassword + "@" + dbHost + ":" + dbPort + "/" + dbName

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatalf("Unable to parse database configuration: %v", err)
	}

	dbPool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}

	log.Println("Database connection established")
}

func closeDB() {
	if dbPool != nil {
		dbPool.Close()
		log.Println("Database connection closed")
	}
}
