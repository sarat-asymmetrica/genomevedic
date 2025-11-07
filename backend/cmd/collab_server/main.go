// GenomeVedic Collaboration Server
// Real-time multiplayer WebSocket server for genome visualization
// Supports cursor sharing, viewport sync, and comment threads
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"genomevedic/internal/collab"

	"github.com/gorilla/mux"
)

const (
	defaultPort        = 8080
	defaultRedisAddr   = "localhost:6379"
	defaultBaseURL     = "http://localhost:5173"
)

func main() {
	// Parse command-line flags
	port := flag.Int("port", defaultPort, "HTTP server port")
	redisAddr := flag.String("redis", defaultRedisAddr, "Redis address (empty to disable)")
	redisPassword := flag.String("redis-password", "", "Redis password")
	redisDB := flag.Int("redis-db", 0, "Redis database number")
	baseURL := flag.String("base-url", defaultBaseURL, "Base URL for session links")
	flag.Parse()

	log.Println("==============================================")
	log.Println("  GenomeVedic Collaboration Server")
	log.Println("  Real-Time Multiplayer Genome Visualization")
	log.Println("==============================================")
	log.Printf("Port: %d", *port)
	log.Printf("Redis: %s", *redisAddr)
	log.Printf("Base URL: %s", *baseURL)
	log.Println("==============================================")

	// Create collaboration server
	collabServer := collab.NewCollabServer(*redisAddr, *redisPassword, *redisDB, *baseURL)

	// Create HTTP router
	router := mux.NewRouter()

	// Register collaboration routes
	collabServer.RegisterRoutes(router)

	// Health check endpoint
	router.HandleFunc("/health", handleHealth).Methods("GET")

	// API info endpoint
	router.HandleFunc("/api/v1/info", handleInfo).Methods("GET")

	// Start HTTP server
	addr := fmt.Sprintf(":%d", *port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("[SERVER] Starting HTTP server on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[SERVER] Failed to start server: %v", err)
		}
	}()

	log.Println("[SERVER] Server started successfully")
	log.Printf("[SERVER] WebSocket endpoint: ws://localhost:%d/api/v1/collab/session/{id}", *port)
	log.Printf("[SERVER] REST API: http://localhost:%d/api/v1/collab/", *port)
	log.Println("[SERVER] Press Ctrl+C to stop")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("[SERVER] Shutting down server...")

	// Close collaboration server
	if err := collabServer.Close(); err != nil {
		log.Printf("[SERVER] Error closing server: %v", err)
	}

	log.Println("[SERVER] Server stopped")
}

// handleHealth returns server health status
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","timestamp":%d}`, time.Now().Unix())
}

// handleInfo returns API information
func handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	info := `{
		"name": "GenomeVedic Collaboration API",
		"version": "1.0.0",
		"description": "Real-time multiplayer collaboration for genome visualization",
		"endpoints": {
			"websocket": "WS /api/v1/collab/session/{id}",
			"create_session": "POST /api/v1/collab/sessions",
			"get_session": "GET /api/v1/collab/sessions/{id}",
			"list_sessions": "GET /api/v1/collab/sessions",
			"statistics": "GET /api/v1/collab/stats"
		},
		"features": [
			"Real-time cursor tracking (30 Hz)",
			"Viewport synchronization",
			"Follow mode",
			"Presentation mode",
			"Comment threads with markdown",
			"@mentions",
			"Connection pooling (10K users)",
			"Heartbeat/ping-pong"
		],
		"target_latency": "< 100ms (p95)"
	}`
	w.Write([]byte(info))
}
