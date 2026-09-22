package main

import (
	"fmt"
	"os"
)

func main() {

	port := os.Getenv("Port")

	fmt.Printf("Starting server on port %v...", port)
}
