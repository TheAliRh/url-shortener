package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {

	port := os.Getenv("Port")
	host := os.Getenv("Host")

	// Routers
	router := gin.Default()

	api := router.Group("/api")
	{
		api.POST("/shorten", func(c *gin.Context) {
			// Implementation for shortening URL
		})
		api.GET("/shortened/:shortURL", func(c *gin.Context) {
			// Implementation for retrieving original URL
		})
		api.DELETE("/shortened/:shortURL", func(c *gin.Context) {
			// Implementation for deleting short URL
		})
	}

	fmt.Printf("Starting server on port %v...\n", port)
	router.Run(host + ":" + port)

}
