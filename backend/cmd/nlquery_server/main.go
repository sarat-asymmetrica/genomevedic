/**
 * Natural Language Query Server
 *
 * Runs the GenomeVedic API server for natural language queries
 *
 * Usage:
 *   export OPENAI_API_KEY="your-api-key"
 *   go run main.go
 */

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"genomevedic/backend/internal/api"
)

func main() {
	// Parse command-line flags
	port := flag.Int("port", 8080, "Port to run the server on")
	flag.Parse()

	// Check for OpenAI API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	// Create server
	server, err := api.NewServer(*port)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
		os.Exit(0)
	}()

	// Start server
	log.Printf("GenomeVedic NL Query Server starting on port %d", *port)
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
