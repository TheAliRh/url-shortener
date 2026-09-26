package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	initDB()
	defer closeDB()

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	port := os.Getenv("Port")
	host := os.Getenv("Host")

	// Routers
	router := gin.Default()

	api := router.Group("/api")
	{
		api.POST("/shorten", func(c *gin.Context) { // Endpoint for creating short URL

			var req UserRequest

			// Bind the incoming JSON to the UserRequest struct and validate it
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			// Create the short URL using the provided long URL
			shortURL := createShortURL(req.LongURL)
			c.JSON(200, gin.H{"short_url": shortURL})
		})

		api.DELETE("/shortened/:shortURL", func(c *gin.Context) { // Endpoint for removing a single short url
			shortURL := c.Param("shortURL")

			resp, err := deleteURL(shortURL)
			if err != nil {
				c.JSON(404, gin.H{"error": "Short URL not found"})
				return
			}

			c.JSON(200, gin.H{"message": resp})
		})

	}

	log.Printf("Starting server on port %v...\n", port)
	router.Run(host + ":" + port)

}
