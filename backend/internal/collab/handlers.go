// Package collab - HTTP handlers for collaboration API
// Provides REST endpoints for session management and WebSocket upgrades
package collab

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// WebSocket upgrader configuration
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// TODO: Restrict in production
		return true
	},
}

// CollabServer handles collaboration HTTP endpoints
type CollabServer struct {
	hub        *Hub
	sessionMgr *SessionManager
	baseURL    string
}

// NewCollabServer creates a new collaboration server
func NewCollabServer(redisAddr string, redisPassword string, redisDB int, baseURL string) *CollabServer {
	sessionMgr := NewSessionManager(redisAddr, redisPassword, redisDB)
	hub := NewHub(sessionMgr)

	// Start hub
	go hub.Run()

	return &CollabServer{
		hub:        hub,
		sessionMgr: sessionMgr,
		baseURL:    baseURL,
	}
}

// RegisterRoutes registers HTTP routes on a router
func (s *CollabServer) RegisterRoutes(router *mux.Router) {
	// REST API endpoints
	router.HandleFunc("/api/v1/collab/sessions", s.HandleCreateSession).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/collab/sessions/{id}", s.HandleGetSession).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/v1/collab/sessions", s.HandleListSessions).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/v1/collab/stats", s.HandleGetStats).Methods("GET", "OPTIONS")

	// WebSocket endpoint
	router.HandleFunc("/api/v1/collab/session/{id}", s.HandleWebSocket)

	log.Println("[API] Collaboration routes registered")
}

// HandleCreateSession creates a new collaboration session
func (s *CollabServer) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	s.setCORSHeaders(w)

	// Parse request
	var req SessionCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err.Error())
		return
	}

	// Validate input
	if req.Name == "" {
		req.Name = "Genome Session"
	}
	if req.UserName == "" {
		req.UserName = "Anonymous"
	}
	if req.MaxUsers <= 0 {
		req.MaxUsers = 100
	}

	// Create session
	session, owner, err := s.sessionMgr.CreateSession(req.Name, req.UserName, req.MaxUsers)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create session", err.Error())
		return
	}

	// Generate response
	response := SessionCreateResponse{
		SessionID: session.ID,
		UserID:    owner.ID,
		UserToken: owner.ID, // Simple token = user ID (improve in production)
		URL:       GenerateSessionURL(s.baseURL, session.ID),
	}

	s.sendJSON(w, http.StatusCreated, response)
	log.Printf("[API] Session created: %s by %s", session.ID, req.UserName)
}

// HandleGetSession retrieves session information
func (s *CollabServer) HandleGetSession(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	s.setCORSHeaders(w)

	// Get session ID from URL
	vars := mux.Vars(r)
	sessionID := vars["id"]

	// Retrieve session
	session, err := s.sessionMgr.GetSession(sessionID)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found", err.Error())
		return
	}

	// Return session info
	response := SessionInfoResponse{
		Session: *session,
	}

	s.sendJSON(w, http.StatusOK, response)
}

// HandleListSessions lists all active sessions
func (s *CollabServer) HandleListSessions(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	s.setCORSHeaders(w)

	sessions := s.sessionMgr.GetAllSessions()

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// HandleGetStats returns server statistics
func (s *CollabServer) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	s.setCORSHeaders(w)

	stats := s.hub.GetStatistics()
	s.sendJSON(w, http.StatusOK, stats)
}

// HandleWebSocket upgrades HTTP connection to WebSocket
func (s *CollabServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get session ID from URL
	vars := mux.Vars(r)
	sessionID := vars["id"]

	// Get user info from query parameters
	userName := r.URL.Query().Get("user_name")
	if userName == "" {
		userName = "Anonymous"
	}

	permission := Permission(r.URL.Query().Get("permission"))
	if !ValidatePermission(permission) {
		permission = PermissionViewer
	}

	// Check if session exists, or join existing
	session, err := s.sessionMgr.GetSession(sessionID)
	if err != nil {
		// Try to get the session, if it doesn't exist, return error
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Join session or create user
	var user *User
	_, user, err = s.sessionMgr.JoinSession(sessionID, userName, permission)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Failed to upgrade connection: %v", err)
		return
	}

	// Create client
	client := &Client{
		ID:            user.ID,
		SessionID:     sessionID,
		Conn:          conn,
		Send:          make(chan *Message, sendBufferSize),
		Session:       session,
		User:          user,
		LastHeartbeat: time.Now(),
		IsAlive:       true,
	}

	// Register client with hub
	s.hub.register <- client

	// Start client goroutines
	go client.WritePump()
	go client.ReadPump(s.hub)

	log.Printf("[WS] User %s connected to session %s", userName, sessionID)
}

// Helper methods

// setCORSHeaders sets CORS headers for API responses
func (s *CollabServer) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// sendJSON sends a JSON response
func (s *CollabServer) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// sendError sends an error response
func (s *CollabServer) sendError(w http.ResponseWriter, status int, code string, message string, details string) {
	response := ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// Close shuts down the collaboration server
func (s *CollabServer) Close() error {
	return s.sessionMgr.Close()
}
