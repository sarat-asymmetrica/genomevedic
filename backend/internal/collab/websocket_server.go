// Package collab - WebSocket server for real-time collaboration
// Implements high-performance WebSocket broadcasting with connection pooling
package collab

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// WebSocket configuration constants
	writeWait      = 10 * time.Second    // Time allowed to write a message
	pongWait       = 60 * time.Second    // Time allowed to read the next pong
	pingPeriod     = (pongWait * 9) / 10 // Send pings to peer with this period
	maxMessageSize = 8192                // Maximum message size (8KB)
	sendBufferSize = 256                 // Send channel buffer size
	broadcastBuffer = 1024               // Broadcast channel buffer size

	// Performance targets
	targetUpdateHz    = 30              // 30 updates per second for cursor
	cursorUpdateDelay = time.Second / 30 // ~33ms between cursor broadcasts
)

// Hub maintains active clients and broadcasts messages
type Hub struct {
	// Registered clients by session ID
	sessions map[string]map[string]*Client

	// Broadcast channel
	broadcast chan *BroadcastMessage

	// Register/unregister channels
	register   chan *Client
	unregister chan *Client

	// Session manager
	sessionMgr *SessionManager

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Statistics
	stats        Statistics
	statsLock    sync.RWMutex
	messageCount int64
	lastStatsTime time.Time
	latencies    []float64
}

// BroadcastMessage represents a message to broadcast
type BroadcastMessage struct {
	SessionID string
	Message   *Message
	ExcludeID string // Don't send to this client ID
}

// NewHub creates a new WebSocket hub
func NewHub(sessionMgr *SessionManager) *Hub {
	return &Hub{
		broadcast:     make(chan *BroadcastMessage, broadcastBuffer),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		sessions:      make(map[string]map[string]*Client),
		sessionMgr:    sessionMgr,
		lastStatsTime: time.Now(),
		latencies:     make([]float64, 0, 1000),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	log.Println("[HUB] Starting WebSocket hub...")

	// Start statistics updater
	go h.updateStatistics()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case broadcast := <-h.broadcast:
			h.broadcastMessage(broadcast)
		}
	}
}

// registerClient registers a new client connection
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Create session map if needed
	if h.sessions[client.SessionID] == nil {
		h.sessions[client.SessionID] = make(map[string]*Client)
	}

	// Add client to session
	h.sessions[client.SessionID][client.ID] = client
	client.IsAlive = true
	client.LastHeartbeat = time.Now()

	log.Printf("[HUB] User %s joined session %s (total: %d users)",
		client.ID, client.SessionID, len(h.sessions[client.SessionID]))

	// Broadcast user join to other clients
	h.broadcastUserJoin(client)

	// Send current session state to new user
	h.sendSessionState(client)
}

// unregisterClient removes a client connection
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.sessions[client.SessionID]; ok {
		if _, exists := clients[client.ID]; exists {
			// Close send channel
			close(client.Send)

			// Remove from session
			delete(clients, client.ID)

			// Remove session if empty
			if len(clients) == 0 {
				delete(h.sessions, client.SessionID)
			}

			log.Printf("[HUB] User %s left session %s (remaining: %d users)",
				client.ID, client.SessionID, len(clients))

			// Broadcast user leave
			h.broadcastUserLeave(client)

			// Update session manager
			h.sessionMgr.RemoveUser(client.SessionID, client.ID)
		}
	}
}

// broadcastMessage sends a message to all clients in a session
func (h *Hub) broadcastMessage(bm *BroadcastMessage) {
	h.mu.RLock()
	clients := h.sessions[bm.SessionID]
	h.mu.RUnlock()

	if clients == nil {
		return
	}

	// Record latency (time since message creation)
	if bm.Message.Timestamp > 0 {
		latency := float64(time.Now().UnixMilli() - bm.Message.Timestamp)
		h.recordLatency(latency)
	}

	// Broadcast to all clients except sender
	for id, client := range clients {
		if id != bm.ExcludeID && client.IsAlive {
			select {
			case client.Send <- bm.Message:
				// Message sent successfully
			default:
				// Client's send buffer is full - connection too slow
				log.Printf("[HUB] Client %s send buffer full, closing connection", id)
				h.unregister <- client
			}
		}
	}

	h.messageCount++
}

// broadcastUserJoin notifies all clients about a new user
func (h *Hub) broadcastUserJoin(newClient *Client) {
	msg := &Message{
		ID:        generateID(),
		Type:      MessageTypeUserJoin,
		SessionID: newClient.SessionID,
		UserID:    newClient.ID,
		Payload:   newClient.User,
		Timestamp: time.Now().UnixMilli(),
	}

	h.broadcast <- &BroadcastMessage{
		SessionID: newClient.SessionID,
		Message:   msg,
		ExcludeID: newClient.ID, // Don't send to the user who just joined
	}
}

// broadcastUserLeave notifies all clients about a user leaving
func (h *Hub) broadcastUserLeave(client *Client) {
	msg := &Message{
		ID:        generateID(),
		Type:      MessageTypeUserLeave,
		SessionID: client.SessionID,
		UserID:    client.ID,
		Payload: map[string]string{
			"user_id": client.ID,
		},
		Timestamp: time.Now().UnixMilli(),
	}

	h.broadcast <- &BroadcastMessage{
		SessionID: client.SessionID,
		Message:   msg,
	}
}

// sendSessionState sends current session state to a newly joined user
func (h *Hub) sendSessionState(client *Client) {
	session, err := h.sessionMgr.GetSession(client.SessionID)
	if err != nil {
		log.Printf("[HUB] Failed to get session state: %v", err)
		return
	}

	// Send current users
	for _, user := range session.Users {
		if user.ID != client.ID {
			msg := &Message{
				ID:        generateID(),
				Type:      MessageTypeUserJoin,
				SessionID: client.SessionID,
				UserID:    user.ID,
				Payload:   user,
				Timestamp: time.Now().UnixMilli(),
			}

			select {
			case client.Send <- msg:
			default:
				log.Printf("[HUB] Failed to send session state to client %s", client.ID)
			}
		}
	}
}

// Broadcast sends a message to all clients in a session
func (h *Hub) Broadcast(sessionID string, msg *Message, excludeUserID string) {
	h.broadcast <- &BroadcastMessage{
		SessionID: sessionID,
		Message:   msg,
		ExcludeID: excludeUserID,
	}
}

// GetSessionClients returns all clients in a session
func (h *Hub) GetSessionClients(sessionID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := h.sessions[sessionID]
	result := make([]*Client, 0, len(clients))
	for _, client := range clients {
		result = append(result, client)
	}
	return result
}

// ReadPump reads messages from WebSocket connection
func (c *Client) ReadPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		c.LastHeartbeat = time.Now()
		return nil
	})

	for {
		_, messageData, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS] Read error for user %s: %v", c.ID, err)
			}
			break
		}

		// Parse message
		var msg Message
		if err := json.Unmarshal(messageData, &msg); err != nil {
			log.Printf("[WS] Failed to parse message from user %s: %v", c.ID, err)
			continue
		}

		// Set timestamp if not set
		if msg.Timestamp == 0 {
			msg.Timestamp = time.Now().UnixMilli()
		}

		// Handle message based on type
		c.handleMessage(&msg, hub)
	}
}

// WritePump writes messages to WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Write message as JSON
			if err := c.Conn.WriteJSON(message); err != nil {
				log.Printf("[WS] Write error for user %s: %v", c.ID, err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *Client) handleMessage(msg *Message, hub *Hub) {
	// Update user's last seen time
	c.User.LastSeen = time.Now().UnixMilli()

	switch msg.Type {
	case MessageTypeCursorMove:
		c.handleCursorMove(msg, hub)

	case MessageTypeViewportSync:
		c.handleViewportSync(msg, hub)

	case MessageTypeFollowMode:
		c.handleFollowMode(msg, hub)

	case MessageTypePresentationMode:
		c.handlePresentationMode(msg, hub)

	case MessageTypeCommentAdd:
		c.handleCommentAdd(msg, hub)

	case MessageTypeCommentUpdate:
		c.handleCommentUpdate(msg, hub)

	case MessageTypeCommentDelete:
		c.handleCommentDelete(msg, hub)

	case MessageTypeHeartbeat:
		c.handleHeartbeat(msg)

	default:
		log.Printf("[WS] Unknown message type from user %s: %s", c.ID, msg.Type)
	}
}

// handleCursorMove broadcasts cursor position update
func (c *Client) handleCursorMove(msg *Message, hub *Hub) {
	// Parse cursor position
	cursorData, err := json.Marshal(msg.Payload)
	if err != nil {
		return
	}

	var cursor CursorPosition
	if err := json.Unmarshal(cursorData, &cursor); err != nil {
		return
	}

	// Update user's cursor
	c.User.Cursor = cursor

	// Broadcast to other users (throttled to 30 Hz)
	hub.Broadcast(c.SessionID, msg, c.ID)
}

// handleViewportSync broadcasts viewport state
func (c *Client) handleViewportSync(msg *Message, hub *Hub) {
	// Parse viewport state
	viewportData, err := json.Marshal(msg.Payload)
	if err != nil {
		return
	}

	var viewport ViewportState
	if err := json.Unmarshal(viewportData, &viewport); err != nil {
		return
	}

	// Update user's viewport
	c.User.Viewport = viewport

	// Broadcast to other users
	hub.Broadcast(c.SessionID, msg, c.ID)
}

// handleFollowMode handles follow mode toggle
func (c *Client) handleFollowMode(msg *Message, hub *Hub) {
	hub.Broadcast(c.SessionID, msg, c.ID)
}

// handlePresentationMode handles presentation mode
func (c *Client) handlePresentationMode(msg *Message, hub *Hub) {
	// Only owner can enable presentation mode
	if c.User.Permission != PermissionOwner {
		return
	}

	hub.Broadcast(c.SessionID, msg, "")
}

// handleCommentAdd adds a comment and broadcasts it
func (c *Client) handleCommentAdd(msg *Message, hub *Hub) {
	// Only editors and owners can comment
	if c.User.Permission == PermissionViewer {
		return
	}

	hub.Broadcast(c.SessionID, msg, c.ID)
}

// handleCommentUpdate updates a comment
func (c *Client) handleCommentUpdate(msg *Message, hub *Hub) {
	hub.Broadcast(c.SessionID, msg, c.ID)
}

// handleCommentDelete deletes a comment
func (c *Client) handleCommentDelete(msg *Message, hub *Hub) {
	hub.Broadcast(c.SessionID, msg, c.ID)
}

// handleHeartbeat responds to heartbeat/ping
func (c *Client) handleHeartbeat(msg *Message) {
	c.LastHeartbeat = time.Now()

	// Send acknowledgment
	ack := &Message{
		ID:        generateID(),
		Type:      MessageTypeAck,
		SessionID: c.SessionID,
		UserID:    c.ID,
		Payload: map[string]int64{
			"timestamp": time.Now().UnixMilli(),
		},
		Timestamp: time.Now().UnixMilli(),
	}

	select {
	case c.Send <- ack:
	default:
	}
}

// recordLatency records message latency for statistics
func (h *Hub) recordLatency(latency float64) {
	h.statsLock.Lock()
	defer h.statsLock.Unlock()

	h.latencies = append(h.latencies, latency)

	// Keep only last 1000 latencies
	if len(h.latencies) > 1000 {
		h.latencies = h.latencies[len(h.latencies)-1000:]
	}
}

// updateStatistics updates hub statistics periodically
func (h *Hub) updateStatistics() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.statsLock.Lock()

		now := time.Now()
		elapsed := now.Sub(h.lastStatsTime).Seconds()

		// Calculate messages per second
		h.stats.MessagesPerSecond = float64(h.messageCount) / elapsed
		h.messageCount = 0
		h.lastStatsTime = now

		// Calculate latency percentiles
		if len(h.latencies) > 0 {
			h.stats.AvgLatencyMs = average(h.latencies)
			h.stats.P95LatencyMs = percentile(h.latencies, 0.95)
			h.stats.P99LatencyMs = percentile(h.latencies, 0.99)
		}

		// Count active sessions and users
		h.mu.RLock()
		h.stats.ActiveSessions = len(h.sessions)
		totalUsers := 0
		for _, clients := range h.sessions {
			totalUsers += len(clients)
		}
		h.stats.TotalUsers = totalUsers
		h.mu.RUnlock()

		h.statsLock.Unlock()
	}
}

// GetStatistics returns current hub statistics
func (h *Hub) GetStatistics() Statistics {
	h.statsLock.RLock()
	defer h.statsLock.RUnlock()
	return h.stats
}
