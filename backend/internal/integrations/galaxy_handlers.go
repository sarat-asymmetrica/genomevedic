// Package integrations - HTTP handlers for Galaxy integration API
package integrations

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// GalaxyHandlers provides HTTP handlers for Galaxy integration endpoints
type GalaxyHandlers struct {
	importer     *BAMImporter
	exporter     *GalaxyExporter
	oauthClient  *GalaxyOAuthClient
}

// NewGalaxyHandlers creates new Galaxy integration handlers
func NewGalaxyHandlers(maxParticles int64, oauthConfig *GalaxyOAuthConfig) *GalaxyHandlers {
	importer := NewBAMImporter(maxParticles)
	oauthClient := NewGalaxyOAuthClient(oauthConfig)
	exporter := NewGalaxyExporter(oauthClient)

	// Start periodic cleanup of expired OAuth sessions
	oauthClient.StartPeriodicCleanup(5 * time.Minute)

	return &GalaxyHandlers{
		importer:     importer,
		exporter:     exporter,
		oauthClient:  oauthClient,
	}
}

// HandleGalaxyImport handles POST /api/v1/import/galaxy
func (h *GalaxyHandlers) HandleGalaxyImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse request
	var req GalaxyImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if req.SessionID == "" {
		h.sendError(w, http.StatusBadRequest, "session_id is required")
		return
	}
	if req.BAMPath == "" {
		h.sendError(w, http.StatusBadRequest, "bam_path is required")
		return
	}
	if req.GenomeBuild == "" {
		h.sendError(w, http.StatusBadRequest, "genome_build is required")
		return
	}

	// Set default quality threshold if not specified
	if req.QualityThreshold == 0 {
		req.QualityThreshold = 20
	}

	log.Printf("Galaxy import request: session=%s, bam=%s, build=%s",
		req.SessionID, req.BAMPath, req.GenomeBuild)

	// Process BAM import
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	response, err := h.importer.ImportBAM(ctx, req)
	if err != nil {
		log.Printf("Galaxy import failed: %v", err)
		h.sendJSON(w, http.StatusInternalServerError, response)
		return
	}

	h.sendJSON(w, http.StatusOK, response)
}

// HandleGalaxyExport handles POST /api/v1/export/galaxy
func (h *GalaxyHandlers) HandleGalaxyExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Get API key from header
	apiKey := r.Header.Get("X-API-KEY")
	if apiKey == "" {
		apiKey = r.Header.Get("Authorization")
		if apiKey != "" && len(apiKey) > 7 && apiKey[:7] == "Bearer " {
			apiKey = apiKey[7:]
		}
	}

	if apiKey == "" {
		h.sendError(w, http.StatusUnauthorized, "API key required")
		return
	}

	// Parse request
	var req GalaxyExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if req.SessionID == "" {
		h.sendError(w, http.StatusBadRequest, "session_id is required")
		return
	}
	if req.HistoryID == "" {
		h.sendError(w, http.StatusBadRequest, "history_id is required")
		return
	}
	if len(req.Annotations) == 0 {
		h.sendError(w, http.StatusBadRequest, "annotations array is required")
		return
	}

	// Set defaults
	if req.Format == "" {
		req.Format = FormatBED
	}
	if req.DatasetName == "" {
		req.DatasetName = "GenomeVedic_Annotations_" + time.Now().Format("20060102_150405")
	}

	log.Printf("Galaxy export request: session=%s, history=%s, format=%s, features=%d",
		req.SessionID, req.HistoryID, req.Format, len(req.Annotations))

	// Export to Galaxy
	response, err := h.exporter.ExportToGalaxy(req, apiKey)
	if err != nil {
		log.Printf("Galaxy export failed: %v", err)
		h.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.sendJSON(w, http.StatusOK, response)
}

// HandleGalaxyOAuthInit handles GET /api/v1/galaxy/oauth/init
func (h *GalaxyHandlers) HandleGalaxyOAuthInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Get user ID from query parameter
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "anonymous"
	}

	redirectTo := r.URL.Query().Get("redirect_to")
	if redirectTo == "" {
		redirectTo = "/"
	}

	// Generate authorization URL
	authURL, err := h.oauthClient.GenerateAuthURL(userID, redirectTo)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "failed to generate auth URL: "+err.Error())
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"auth_url": authURL,
	})
}

// HandleGalaxyOAuthCallback handles GET /api/v1/galaxy/oauth/callback
func (h *GalaxyHandlers) HandleGalaxyOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Get authorization code and state from query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		h.sendError(w, http.StatusBadRequest, "code and state are required")
		return
	}

	// Handle OAuth callback
	apiKeyInfo, err := h.oauthClient.HandleCallback(code, state)
	if err != nil {
		h.sendError(w, http.StatusUnauthorized, "OAuth authentication failed: "+err.Error())
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"api_key":  apiKeyInfo.Key,
		"username": apiKeyInfo.Username,
		"email":    apiKeyInfo.Email,
		"message":  "Authentication successful. Save your API key securely.",
	})
}

// HandleGalaxyValidateKey handles POST /api/v1/galaxy/validate-key
func (h *GalaxyHandlers) HandleGalaxyValidateKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		APIKey string `json:"api_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.APIKey == "" {
		h.sendError(w, http.StatusBadRequest, "api_key is required")
		return
	}

	// Validate API key
	info, err := h.oauthClient.ValidateAPIKey(req.APIKey)
	if err != nil {
		h.sendJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"valid":   false,
			"error":   err.Error(),
		})
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"valid":       true,
		"username":    info.Username,
		"email":       info.Email,
		"galaxy_url":  info.GalaxyURL,
		"permissions": info.Permissions,
	})
}

// HandleGalaxyStatus handles GET /api/v1/galaxy/status
func (h *GalaxyHandlers) HandleGalaxyStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	status := map[string]interface{}{
		"success":            true,
		"service":            "GenomeVedic Galaxy Integration",
		"version":            "1.0.0",
		"supported_formats": []string{"bam"},
		"export_formats":    []string{"bed", "gtf", "gff3", "vcf"},
		"features": map[string]bool{
			"oauth_authentication": true,
			"bam_import":          true,
			"annotation_export":   true,
			"streaming_import":    true,
			"quality_filtering":   true,
			"region_selection":    true,
		},
		"limits": map[string]interface{}{
			"max_particles":        h.importer.maxParticles,
			"max_file_size_gb":    10,
			"import_timeout_min":  5,
			"export_timeout_min":  2,
		},
	}

	h.sendJSON(w, http.StatusOK, status)
}

// HandleImportProgress handles GET /api/v1/galaxy/import/progress/:session_id
func (h *GalaxyHandlers) HandleImportProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract session ID from URL path
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		h.sendError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	// Get progress
	progress, exists := h.importer.GetSessionProgress(sessionID)

	if !exists {
		h.sendJSON(w, http.StatusOK, map[string]interface{}{
			"success":    true,
			"in_progress": false,
			"progress":    0,
			"message":     "Session not found or completed",
		})
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"in_progress": true,
		"progress":    progress,
		"message":     "Import in progress",
	})
}

// sendJSON sends a JSON response
func (h *GalaxyHandlers) sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-KEY")

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// sendError sends an error response
func (h *GalaxyHandlers) sendError(w http.ResponseWriter, statusCode int, message string) {
	h.sendJSON(w, statusCode, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// RegisterRoutes registers all Galaxy integration routes with a ServeMux
func (h *GalaxyHandlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/import/galaxy", h.HandleGalaxyImport)
	mux.HandleFunc("/api/v1/export/galaxy", h.HandleGalaxyExport)
	mux.HandleFunc("/api/v1/galaxy/oauth/init", h.HandleGalaxyOAuthInit)
	mux.HandleFunc("/api/v1/galaxy/oauth/callback", h.HandleGalaxyOAuthCallback)
	mux.HandleFunc("/api/v1/galaxy/validate-key", h.HandleGalaxyValidateKey)
	mux.HandleFunc("/api/v1/galaxy/status", h.HandleGalaxyStatus)
	mux.HandleFunc("/api/v1/galaxy/import/progress", h.HandleImportProgress)

	log.Println("Galaxy integration routes registered")
}
