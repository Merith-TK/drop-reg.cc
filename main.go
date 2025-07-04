package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Load configuration
	config, err := LoadConfig("config.toml")
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Create server instance
	server, err := InitServer("drop-reg.db", config)
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}
	defer server.db.Close()

	// Get port from configuration
	port := config.GetPort()

	log.Printf("Starting drop-reg.cc server on :%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), server))
}
