// Package collab - Session manager with Redis state management
// Handles session lifecycle, user management, and persistent state
package collab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// Redis key prefixes
	sessionKeyPrefix  = "genomevedic:session:"
	userKeyPrefix     = "genomevedic:user:"
	commentKeyPrefix  = "genomevedic:comment:"

	// Session defaults
	defaultMaxUsers       = 100
	defaultSessionExpiry  = 24 * time.Hour  // 24 hours
	sessionCleanupPeriod  = 5 * time.Minute // Cleanup check interval
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrUserNotFound    = errors.New("user not found")
	ErrSessionFull     = errors.New("session is full")
	ErrInvalidPermission = errors.New("invalid permission")
)

// SessionManager manages collaboration sessions with Redis persistence
type SessionManager struct {
	// Redis client for persistence
	redis *redis.Client

	// In-memory cache for fast access
	sessions map[string]*Session
	mu       sync.RWMutex

	// Context for operations
	ctx context.Context

	// Fallback to in-memory if Redis unavailable
	useRedis bool
}

// NewSessionManager creates a new session manager
func NewSessionManager(redisAddr string, redisPassword string, redisDB int) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
		ctx:      context.Background(),
	}

	// Try to connect to Redis
	if redisAddr != "" {
		sm.redis = redis.NewClient(&redis.Options{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       redisDB,
		})

		// Test connection
		if err := sm.redis.Ping(sm.ctx).Err(); err != nil {
			log.Printf("[SESSION] Redis connection failed: %v (falling back to in-memory)", err)
			sm.useRedis = false
		} else {
			log.Printf("[SESSION] Connected to Redis at %s", redisAddr)
			sm.useRedis = true

			// Start cleanup goroutine
			go sm.cleanupExpiredSessions()
		}
	} else {
		log.Println("[SESSION] Redis not configured, using in-memory storage")
		sm.useRedis = false
	}

	return sm
}

// CreateSession creates a new collaboration session
func (sm *SessionManager) CreateSession(name string, ownerName string, maxUsers int) (*Session, *User, error) {
	if maxUsers <= 0 {
		maxUsers = defaultMaxUsers
	}

	// Generate IDs
	sessionID := generateID()
	userID := generateID()

	// Create owner user
	owner := &User{
		ID:         userID,
		Name:       ownerName,
		Initials:   generateInitials(ownerName),
		Color:      generateColor(userID),
		Permission: PermissionOwner,
		JoinedAt:   time.Now().UnixMilli(),
		LastSeen:   time.Now().UnixMilli(),
	}

	// Create session
	session := &Session{
		ID:        sessionID,
		Name:      name,
		OwnerID:   userID,
		Users:     map[string]*User{userID: owner},
		Comments:  make(map[string]Comment),
		CreatedAt: time.Now().UnixMilli(),
		ExpiresAt: time.Now().Add(defaultSessionExpiry).UnixMilli(),
		MaxUsers:  maxUsers,
	}

	// Save to storage
	if err := sm.saveSession(session); err != nil {
		return nil, nil, fmt.Errorf("failed to save session: %w", err)
	}

	log.Printf("[SESSION] Created session %s (%s) with owner %s", sessionID, name, ownerName)

	return session, owner, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	// Try in-memory cache first
	sm.mu.RLock()
	if session, ok := sm.sessions[sessionID]; ok {
		sm.mu.RUnlock()
		return session, nil
	}
	sm.mu.RUnlock()

	// Try Redis if available
	if sm.useRedis {
		session, err := sm.loadSessionFromRedis(sessionID)
		if err != nil {
			return nil, ErrSessionNotFound
		}

		// Cache in memory
		sm.mu.Lock()
		sm.sessions[sessionID] = session
		sm.mu.Unlock()

		return session, nil
	}

	return nil, ErrSessionNotFound
}

// JoinSession adds a user to an existing session
func (sm *SessionManager) JoinSession(sessionID string, userName string, permission Permission) (*Session, *User, error) {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return nil, nil, err
	}

	// Check if session is full
	if len(session.Users) >= session.MaxUsers {
		return nil, nil, ErrSessionFull
	}

	// Create user
	userID := generateID()
	user := &User{
		ID:         userID,
		Name:       userName,
		Initials:   generateInitials(userName),
		Color:      generateColor(userID),
		Permission: permission,
		JoinedAt:   time.Now().UnixMilli(),
		LastSeen:   time.Now().UnixMilli(),
	}

	// Add user to session
	session.Users[userID] = user

	// Save session
	if err := sm.saveSession(session); err != nil {
		return nil, nil, fmt.Errorf("failed to save session: %w", err)
	}

	log.Printf("[SESSION] User %s joined session %s", userName, sessionID)

	return session, user, nil
}

// RemoveUser removes a user from a session
func (sm *SessionManager) RemoveUser(sessionID string, userID string) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	delete(session.Users, userID)

	// If no users left, mark session for cleanup
	if len(session.Users) == 0 {
		session.ExpiresAt = time.Now().Add(5 * time.Minute).UnixMilli()
	}

	return sm.saveSession(session)
}

// UpdateUser updates a user's information
func (sm *SessionManager) UpdateUser(sessionID string, userID string, updates map[string]interface{}) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	user, ok := session.Users[userID]
	if !ok {
		return ErrUserNotFound
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		user.Name = name
		user.Initials = generateInitials(name)
	}

	if permission, ok := updates["permission"].(Permission); ok {
		user.Permission = permission
	}

	user.LastSeen = time.Now().UnixMilli()

	return sm.saveSession(session)
}

// AddComment adds a comment to a session
func (sm *SessionManager) AddComment(sessionID string, comment Comment) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	comment.ID = generateID()
	comment.SessionID = sessionID
	comment.CreatedAt = time.Now().UnixMilli()
	comment.UpdatedAt = comment.CreatedAt

	session.Comments[comment.ID] = comment

	return sm.saveSession(session)
}

// UpdateComment updates an existing comment
func (sm *SessionManager) UpdateComment(sessionID string, commentID string, content string) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	comment, ok := session.Comments[commentID]
	if !ok {
		return errors.New("comment not found")
	}

	comment.Content = content
	comment.UpdatedAt = time.Now().UnixMilli()
	session.Comments[commentID] = comment

	return sm.saveSession(session)
}

// DeleteComment deletes a comment
func (sm *SessionManager) DeleteComment(sessionID string, commentID string) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	delete(session.Comments, commentID)

	return sm.saveSession(session)
}

// SetPresentationMode enables/disables presentation mode
func (sm *SessionManager) SetPresentationMode(sessionID string, enabled bool, presenterID string) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.IsPresenting = enabled
	session.PresenterID = presenterID

	return sm.saveSession(session)
}

// saveSession persists a session to storage
func (sm *SessionManager) saveSession(session *Session) error {
	// Update in-memory cache
	sm.mu.Lock()
	sm.sessions[session.ID] = session
	sm.mu.Unlock()

	// Save to Redis if available
	if sm.useRedis {
		return sm.saveSessionToRedis(session)
	}

	return nil
}

// saveSessionToRedis saves session to Redis
func (sm *SessionManager) saveSessionToRedis(session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := sessionKeyPrefix + session.ID
	expiry := time.Until(time.UnixMilli(session.ExpiresAt))

	if err := sm.redis.Set(sm.ctx, key, data, expiry).Err(); err != nil {
		return fmt.Errorf("failed to save to redis: %w", err)
	}

	return nil
}

// loadSessionFromRedis loads session from Redis
func (sm *SessionManager) loadSessionFromRedis(sessionID string) (*Session, error) {
	key := sessionKeyPrefix + sessionID

	data, err := sm.redis.Get(sm.ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// cleanupExpiredSessions removes expired sessions periodically
func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(sessionCleanupPeriod)
	defer ticker.Stop()

	for range ticker.C {
		sm.mu.Lock()

		now := time.Now().UnixMilli()
		for id, session := range sm.sessions {
			if session.ExpiresAt > 0 && session.ExpiresAt < now {
				delete(sm.sessions, id)
				log.Printf("[SESSION] Cleaned up expired session %s", id)

				// Also delete from Redis
				if sm.useRedis {
					key := sessionKeyPrefix + id
					sm.redis.Del(sm.ctx, key)
				}
			}
		}

		sm.mu.Unlock()
	}
}

// GetAllSessions returns all active sessions
func (sm *SessionManager) GetAllSessions() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// Close closes the session manager and Redis connection
func (sm *SessionManager) Close() error {
	if sm.useRedis && sm.redis != nil {
		return sm.redis.Close()
	}
	return nil
}
