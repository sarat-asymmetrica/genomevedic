// Package collab - Real-time collaboration types and message protocol
// Supports WebSocket-based multiplayer genome viewing with <100ms latency
package collab

import (
	"time"

	"github.com/gorilla/websocket"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// Cursor and viewport messages
	MessageTypeCursorMove      MessageType = "cursor_move"
	MessageTypeViewportSync    MessageType = "viewport_sync"
	MessageTypeFollowMode      MessageType = "follow_mode"
	MessageTypePresentationMode MessageType = "presentation_mode"

	// User presence messages
	MessageTypeUserJoin   MessageType = "user_join"
	MessageTypeUserLeave  MessageType = "user_leave"
	MessageTypeUserUpdate MessageType = "user_update"

	// Comment messages
	MessageTypeCommentAdd    MessageType = "comment_add"
	MessageTypeCommentUpdate MessageType = "comment_update"
	MessageTypeCommentDelete MessageType = "comment_delete"
	MessageTypeMention       MessageType = "mention"

	// Session control messages
	MessageTypeHeartbeat MessageType = "heartbeat"
	MessageTypeError     MessageType = "error"
	MessageTypeAck       MessageType = "ack"
)

// Message represents a WebSocket message with timestamp
type Message struct {
	ID        string      `json:"id"`                  // Unique message ID
	Type      MessageType `json:"type"`                // Message type
	SessionID string      `json:"session_id"`          // Session ID
	UserID    string      `json:"user_id"`             // Sender user ID
	Payload   interface{} `json:"payload"`             // Message payload
	Timestamp int64       `json:"timestamp,omitempty"` // Unix milliseconds
}

// CursorPosition represents a collaborator's cursor position
type CursorPosition struct {
	X            float64 `json:"x"`                       // Screen X coordinate (0-1)
	Y            float64 `json:"y"`                       // Screen Y coordinate (0-1)
	Chromosome   string  `json:"chromosome,omitempty"`    // Current chromosome
	BPPosition   int64   `json:"bp_position,omitempty"`   // Base pair position
	IsPointing   bool    `json:"is_pointing,omitempty"`   // Is cursor pointing/clicking
}

// ViewportState represents the current view state
type ViewportState struct {
	Chromosome string  `json:"chromosome"`           // Current chromosome
	StartBP    int64   `json:"start_bp"`             // Start base pair
	EndBP      int64   `json:"end_bp"`               // End base pair
	ZoomLevel  float64 `json:"zoom_level"`           // Zoom level (1.0 = normal)
	CameraX    float64 `json:"camera_x,omitempty"`   // 3D camera X
	CameraY    float64 `json:"camera_y,omitempty"`   // 3D camera Y
	CameraZ    float64 `json:"camera_z,omitempty"`   // 3D camera Z
	TargetX    float64 `json:"target_x,omitempty"`   // Camera target X
	TargetY    float64 `json:"target_y,omitempty"`   // Camera target Y
	TargetZ    float64 `json:"target_z,omitempty"`   // Camera target Z
}

// User represents a collaborator in a session
type User struct {
	ID         string         `json:"id"`                   // User ID (UUID)
	Name       string         `json:"name"`                 // Display name
	Initials   string         `json:"initials"`             // Avatar initials (2 chars)
	Color      string         `json:"color"`                // Avatar color (hex)
	Permission Permission     `json:"permission"`           // User permission level
	Cursor     CursorPosition `json:"cursor,omitempty"`     // Current cursor position
	Viewport   ViewportState  `json:"viewport,omitempty"`   // Current viewport
	IsFollowing string        `json:"is_following,omitempty"` // Following user ID
	JoinedAt   int64          `json:"joined_at"`            // Join timestamp (Unix ms)
	LastSeen   int64          `json:"last_seen"`            // Last activity timestamp
}

// Permission represents user access level
type Permission string

const (
	PermissionOwner  Permission = "owner"  // Session owner (full control)
	PermissionEditor Permission = "editor" // Can edit and comment
	PermissionViewer Permission = "viewer" // Read-only access
)

// Comment represents a comment thread on a variant/position
type Comment struct {
	ID         string   `json:"id"`                   // Comment ID (UUID)
	SessionID  string   `json:"session_id"`           // Session ID
	UserID     string   `json:"user_id"`              // Author user ID
	UserName   string   `json:"user_name"`            // Author name
	Chromosome string   `json:"chromosome"`           // Chromosome location
	BPPosition int64    `json:"bp_position"`          // Base pair position
	Content    string   `json:"content"`              // Markdown content
	Mentions   []string `json:"mentions,omitempty"`   // Mentioned user IDs
	ParentID   string   `json:"parent_id,omitempty"`  // Parent comment (for replies)
	Resolved   bool     `json:"resolved,omitempty"`   // Is thread resolved
	CreatedAt  int64    `json:"created_at"`           // Creation timestamp
	UpdatedAt  int64    `json:"updated_at,omitempty"` // Last update timestamp
}

// Session represents a collaboration session
type Session struct {
	ID          string            `json:"id"`                    // Session ID (UUID)
	Name        string            `json:"name"`                  // Session name
	OwnerID     string            `json:"owner_id"`              // Owner user ID
	Users       map[string]*User  `json:"users"`                 // Active users
	Comments    map[string]Comment `json:"comments,omitempty"`    // Comments by ID
	CreatedAt   int64             `json:"created_at"`            // Creation timestamp
	ExpiresAt   int64             `json:"expires_at,omitempty"`  // Expiration timestamp
	IsPresenting bool            `json:"is_presenting,omitempty"` // Presentation mode active
	PresenterID  string          `json:"presenter_id,omitempty"` // Presenter user ID
	MaxUsers     int             `json:"max_users,omitempty"`   // Max concurrent users
}

// Client represents a WebSocket client connection
type Client struct {
	ID         string              // User ID
	SessionID  string              // Session ID
	Conn       *websocket.Conn     // WebSocket connection
	Send       chan *Message       // Outbound message channel
	Session    *Session            // Session reference
	User       *User               // User reference
	LastHeartbeat time.Time        // Last heartbeat received
	IsAlive    bool                // Connection alive status
}

// SessionCreateRequest represents a session creation request
type SessionCreateRequest struct {
	Name     string `json:"name"`                // Session name
	UserName string `json:"user_name"`           // Creator name
	MaxUsers int    `json:"max_users,omitempty"` // Max users (default: 100)
}

// SessionCreateResponse represents a session creation response
type SessionCreateResponse struct {
	SessionID string `json:"session_id"` // Created session ID
	UserID    string `json:"user_id"`    // Creator user ID
	UserToken string `json:"user_token"` // Authentication token
	URL       string `json:"url"`        // Session URL
}

// SessionJoinRequest represents a session join request
type SessionJoinRequest struct {
	UserName string `json:"user_name"`          // User display name
	Token    string `json:"token,omitempty"`    // Optional auth token
}

// SessionInfoResponse represents session information
type SessionInfoResponse struct {
	Session Session `json:"session"` // Session details
}

// ErrorResponse represents an error message
type ErrorResponse struct {
	Code    string `json:"code"`              // Error code
	Message string `json:"message"`           // Error message
	Details string `json:"details,omitempty"` // Additional details
}

// HeartbeatPayload represents a heartbeat/ping message
type HeartbeatPayload struct {
	Timestamp int64  `json:"timestamp"` // Client timestamp
	ClientID  string `json:"client_id"` // Client ID
}

// Statistics represents session statistics
type Statistics struct {
	ActiveSessions    int     `json:"active_sessions"`    // Number of active sessions
	TotalUsers        int     `json:"total_users"`        // Total connected users
	MessagesPerSecond float64 `json:"messages_per_second"` // Message throughput
	AvgLatencyMs      float64 `json:"avg_latency_ms"`     // Average message latency
	P95LatencyMs      float64 `json:"p95_latency_ms"`     // 95th percentile latency
	P99LatencyMs      float64 `json:"p99_latency_ms"`     // 99th percentile latency
}
