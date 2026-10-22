package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

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

	if err := initCacheTTL(); err != nil {
		log.Fatalf("[FATAL] cache: %v", err)
	}

	if err := initURLSafety(); err != nil {
		log.Fatalf("[FATAL] url safety: %v", err)
	}

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

		router.GET("/:code", rateLimit(ruleRedirectIP, byIP), redirectHandler)
	}

	router.Use(securityHeaders())

	api := router.Group("/api", noStore(), bodyLimit(maxAPIBody), rateLimit(ruleAPIIP, byIP))

	auth := api.Group("/auth", bodyLimit(maxAPIBody))
	{
		auth.POST("/signup", signupHandler)
		auth.POST("/login", loginHandler)
		auth.POST("/logout", authRequired(), logoutHandler)
	}

	url := api.Group("/url", authRequired(), rateLimit(ruleAPIUser, byUser))
	{
		url.POST("/shorten", rateLimit(ruleCreateIP, byIP), rateLimit(ruleCreateUser, byUser), func(c *gin.Context) { // Endpoint for creating short URL

			var req struct {
				LongURL string `json:"long_url" binding:"required"`
			}
			if !bindJSON(c, &req, "Invalid request") {
				return
			}

			userID := c.GetInt64(userIDKey)

			// Create the short URL using the provided long URL
			shortURL, err := createShortURL(c.Request.Context(), req.LongURL, userID)
			if err != nil {
				switch {
				case errors.Is(err, ErrInvalid):
					c.JSON(http.StatusBadRequest, gin.H{"error": "Only valid http and https URLs are allowed"})
				case errors.Is(err, ErrBlockedURL):
					c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "This link can't be shortened because it may be unsafe."})
				case errors.Is(err, ErrSafetyUnavailable):
					c.Header("Retry-After", "30")
					c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Couldn't verify this link right now. Try again shortly."})
				default:
					log.Printf("[ERROR] create: %v", err) // your original 500 handling, unchanged
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				}
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

	router.SetTrustedProxies(nil)

	log.Printf("Starting server on port %v...\n", port)

	addr := host + ":" + port

	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("[FATAL] server: %v", err)
	}

}
