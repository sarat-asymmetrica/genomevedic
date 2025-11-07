/**
 * API Server for GenomeVedic
 *
 * Provides HTTP endpoints for natural language queries and variant data access
 */

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"genomevedic/backend/internal/ai"
)

// Server represents the API server
type Server struct {
	nlEngine         *ai.NLQueryEngine
	variantInterpreter *ai.ChatGPTInterpreter
	port             int
	mux              *http.ServeMux
}

// NewServer creates a new API server
func NewServer(port int) (*Server, error) {
	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	nlEngine := ai.NewNLQueryEngine(apiKey)

	// Create ChatGPT interpreter for variant explanations
	aiConfig := ai.DefaultConfig()
	aiConfig.OpenAIAPIKey = apiKey
	variantInterpreter, err := ai.NewChatGPTInterpreter(aiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create variant interpreter: %w", err)
	}

	server := &Server{
		nlEngine:           nlEngine,
		variantInterpreter: variantInterpreter,
		port:               port,
		mux:                http.NewServeMux(),
	}

	server.registerRoutes()

	return server, nil
}

// registerRoutes registers all HTTP routes
func (s *Server) registerRoutes() {
	// CORS middleware
	s.mux.HandleFunc("/api/v1/query/natural-language", s.corsMiddleware(s.handleNaturalLanguageQuery))
	s.mux.HandleFunc("/api/v1/query/examples", s.corsMiddleware(s.handleGetExamples))
	s.mux.HandleFunc("/api/v1/variants/explain", s.corsMiddleware(s.handleExplainVariant))
	s.mux.HandleFunc("/api/v1/variants/batch-explain", s.corsMiddleware(s.handleBatchExplainVariants))
	s.mux.HandleFunc("/api/v1/cache/stats", s.corsMiddleware(s.handleCacheStats))
	s.mux.HandleFunc("/api/v1/health", s.corsMiddleware(s.handleHealth))
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// NaturalLanguageQueryRequest represents the request body
type NaturalLanguageQueryRequest struct {
	Query  string `json:"query"`
	UserID string `json:"user_id,omitempty"`
}

// NaturalLanguageQueryResponse represents the response body
type NaturalLanguageQueryResponse struct {
	Success         bool                  `json:"success"`
	OriginalQuery   string                `json:"original_query"`
	GeneratedSQL    string                `json:"generated_sql"`
	IsValid         bool                  `json:"is_valid"`
	ValidationError string                `json:"validation_error,omitempty"`
	Explanation     string                `json:"explanation"`
	Results         []map[string]interface{} `json:"results,omitempty"`
	ResultCount     int                   `json:"result_count"`
	ExecutionTimeMs int64                 `json:"execution_time_ms"`
	Error           string                `json:"error,omitempty"`
}

// handleNaturalLanguageQuery handles POST /api/v1/query/natural-language
func (s *Server) handleNaturalLanguageQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req NaturalLanguageQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Query == "" {
		s.sendError(w, http.StatusBadRequest, "query is required")
		return
	}

	// Use IP as user ID if not provided
	userID := req.UserID
	if userID == "" {
		userID = r.RemoteAddr
	}

	// Convert natural language to SQL
	result, err := s.nlEngine.ConvertToSQL(userID, req.Query)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Note: In a production system, you would execute the SQL query here
	// against a real database and return results. For this implementation,
	// we return the SQL query for transparency and validation.
	response := NaturalLanguageQueryResponse{
		Success:         result.IsValid,
		OriginalQuery:   result.OriginalQuery,
		GeneratedSQL:    result.GeneratedSQL,
		IsValid:         result.IsValid,
		ValidationError: result.ValidationError,
		Explanation:     result.Explanation,
		ResultCount:     0, // Would be populated from actual query execution
		ExecutionTimeMs: result.ExecutionTimeMs,
	}

	if !result.IsValid {
		response.Error = result.ValidationError
	}

	s.sendJSON(w, http.StatusOK, response)
}

// handleGetExamples handles GET /api/v1/query/examples
func (s *Server) handleGetExamples(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	examples := s.nlEngine.GetExamples()

	response := map[string]interface{}{
		"success":  true,
		"examples": examples,
		"count":    len(examples),
	}

	s.sendJSON(w, http.StatusOK, response)
}

// handleExplainVariant handles POST /api/v1/variants/explain
func (s *Server) handleExplainVariant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req ai.ExplanationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Gene == "" {
		s.sendError(w, http.StatusBadRequest, "gene is required")
		return
	}
	if req.Variant == "" {
		s.sendError(w, http.StatusBadRequest, "variant is required")
		return
	}

	// Generate explanation
	ctx := r.Context()
	response, err := s.variantInterpreter.ExplainVariant(ctx, req)
	if err != nil {
		log.Printf("Error explaining variant: %v", err)
		s.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.sendJSON(w, http.StatusOK, response)
}

// handleBatchExplainVariants handles POST /api/v1/variants/batch-explain
func (s *Server) handleBatchExplainVariants(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var requests []ai.ExplanationRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(requests) == 0 {
		s.sendError(w, http.StatusBadRequest, "at least one variant required")
		return
	}

	// Generate explanations
	ctx := r.Context()
	responses, err := s.variantInterpreter.BatchExplainVariants(ctx, requests)
	if err != nil {
		log.Printf("Error batch explaining variants: %v", err)
		s.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"count":     len(responses),
		"responses": responses,
	})
}

// handleCacheStats handles GET /api/v1/cache/stats
func (s *Server) handleCacheStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	stats, err := s.variantInterpreter.GetCacheStats(ctx)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"success": true,
		"stats":   stats,
	}

	s.sendJSON(w, http.StatusOK, response)
}

// handleHealth handles GET /api/v1/health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	response := map[string]interface{}{
		"success": true,
		"status":  "healthy",
		"time":    time.Now().Unix(),
	}

	s.sendJSON(w, http.StatusOK, response)
}

// sendJSON sends a JSON response
func (s *Server) sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// sendError sends an error response
func (s *Server) sendError(w http.ResponseWriter, statusCode int, message string) {
	s.sendJSON(w, statusCode, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// Start starts the API server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Starting GenomeVedic API server on %s", addr)
	return http.ListenAndServe(addr, s.mux)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down API server...")

	// Close variant interpreter
	if s.variantInterpreter != nil {
		if err := s.variantInterpreter.Close(); err != nil {
			log.Printf("Error closing variant interpreter: %v", err)
		}
	}

	// In a real implementation with http.Server, we would use server.Shutdown(ctx)
	return nil
}
