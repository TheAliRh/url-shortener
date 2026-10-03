package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("[FATAL] No .env file: %v", err)
	}
	port := os.Getenv("Port")
	host := os.Getenv("Host")

	// Initialize database connection
	err = initDB()
	if err != nil {
		log.Fatalf("[FATAL] database: %v", err)
	}
	defer closeDB()

	// Initialize cache connection
	err = initCache()
	if err != nil {
		log.Fatalf("[FATAL] cache: %v", err)
	}
	defer closeCache()

	// Routers
	router := gin.Default()

	api := router.Group("/api")

	redir := api.Group("/redirect")
	{
		redir.GET(":shortURL", func(c *gin.Context) { // Endpoint for redirection

			shortURL := c.Param("shortURL")

			originalURL, err := redirectURL(shortURL)
			if err != nil {
				c.JSON(400, gin.H{"error": "could not find the url"})
			}

			c.Redirect(http.StatusFound, originalURL)
		})
	}

	url := api.Group("/url")
	{
		url.POST("/shorten", func(c *gin.Context) { // Endpoint for creating short URL

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

		url.DELETE("/shortened/:shortURL", func(c *gin.Context) { // Endpoint for removing a single short url
			shortURL := c.Param("shortURL")

			resp, err := deleteURL(shortURL)
			if err != nil {
				c.JSON(404, gin.H{"error": "Short URL not found"})
				return
			}

			c.JSON(200, gin.H{"message": resp})
		})
		url.GET("/shortened/:shortURL", func(c *gin.Context) { // Endpoint for retrieving long URL
			shortURL := c.Param("shortURL")

			// Retrieve the long URL associated with the provided short URL
			URLdoc, err := getShortURL(shortURL)
			if err != nil {
				c.JSON(404, gin.H{"error": "Short URL not found"})
				return
			}

			c.JSON(200, gin.H{"url": URLdoc})
		})
		url.GET("/", func(c *gin.Context) { // Endpoint for getting all of the urls

			resp, err := getAllURLS()
			if err != nil {
				c.JSON(400, gin.H{"error": "Could not get the data"})
				return
			}

			c.JSON(200, gin.H{"urls": resp})
		})

	}

	log.Printf("Starting server on port %v...\n", port)
	router.Run(host + ":" + port)

}
