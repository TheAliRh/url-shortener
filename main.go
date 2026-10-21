package main

import (
	"errors"
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

	if err := initBaseURL(); err != nil {
		log.Fatalf("[FATAL] config: %v", err)
	}

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

	if err := initAuth(); err != nil {
		log.Fatalf("[FATAL] auth: %v", err)
	}

	// Routers
	router := gin.Default()

	{
		router.GET("/health", func(c *gin.Context) { // Endpoint for health check
			c.JSON(200, gin.H{"status": "ok"})
		})

		router.GET("/favicon.ico", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		router.GET("/:code", redirectHandler)
	}

	api := router.Group("/api")

	auth := api.Group("/auth")
	{
		auth.POST("/signup", signupHandler)
		auth.POST("/login", loginHandler)
		auth.POST("/logout", authRequired(), logoutHandler)
	}

	url := api.Group("/url", authRequired())
	{
		url.POST("/shorten", func(c *gin.Context) { // Endpoint for creating short URL

			var req UserRequest

			// Bind the incoming JSON to the UserRequest struct and validate it
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
				return
			}
			userID := c.GetInt64(userIDKey)

			// Create the short URL using the provided long URL
			shortURL, err := createShortURL(req.LongURL, userID)
			if err != nil {
				log.Printf("[ERROR] create: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"short_url": shortURL})
		})

		url.DELETE("/shortened/:shortURL", func(c *gin.Context) { // Endpoint for removing a single short url
			shortURL := c.Param("shortURL")

			userID := c.GetInt64(userIDKey)
			resp, err := deleteURL(shortURL, userID)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
					return
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
					return
				}
			}
			c.JSON(http.StatusOK, gin.H{"message": resp})
		})
		url.GET("/shortened/:shortURL", func(c *gin.Context) { // Endpoint for retrieving long URL
			shortURL := c.Param("shortURL")

			// Retrieve the long URL associated with the provided short URL
			userID := c.GetInt64(userIDKey)
			URLdoc, err := getShortURL(shortURL, userID)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
				} else {
					log.Printf("[ERROR] get: %v", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				}
				return
			}

			c.JSON(http.StatusOK, gin.H{"url": URLdoc})
		})
		url.GET("/", func(c *gin.Context) { // Endpoint for getting all of the urls

			userID := c.GetInt64(userIDKey)
			limit := 10 // Default limit
			offset := 0 // Default offset

			resp, err := getAllURLS(userID, limit, offset)
			if err != nil {
				log.Printf("[ERROR] list: %v", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Could not get the data"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"urls": resp})
		})

	}

	log.Printf("Starting server on port %v...\n", port)
	router.Run(host + ":" + port)

}
