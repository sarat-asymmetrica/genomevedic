// Package integrations - Galaxy OAuth authentication and API key management
package integrations

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// GalaxyOAuthConfig holds OAuth configuration for Galaxy integration
type GalaxyOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	GalaxyURL    string // Base Galaxy server URL
	Scopes       []string
}

// GalaxyOAuthClient manages OAuth flow with Galaxy servers
type GalaxyOAuthClient struct {
	config       *GalaxyOAuthConfig
	mu           sync.RWMutex
	sessions     map[string]*OAuthSession // state -> session
	apiKeys      map[string]*APIKeyInfo   // apiKey -> info
	httpClient   *http.Client
	stateTimeout time.Duration
}

// OAuthSession tracks an OAuth authentication session
type OAuthSession struct {
	State        string
	CodeVerifier string
	UserID       string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	RedirectTo   string
}

// APIKeyInfo stores information about a Galaxy API key
type APIKeyInfo struct {
	Key         string
	GalaxyURL   string
	UserID      string
	Username    string
	Email       string
	CreatedAt   time.Time
	LastUsed    time.Time
	IsValid     bool
	Permissions []string
}

// NewGalaxyOAuthClient creates a new OAuth client for Galaxy
func NewGalaxyOAuthClient(config *GalaxyOAuthConfig) *GalaxyOAuthClient {
	return &GalaxyOAuthClient{
		config:       config,
		sessions:     make(map[string]*OAuthSession),
		apiKeys:      make(map[string]*APIKeyInfo),
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		stateTimeout: 10 * time.Minute,
	}
}

// GenerateAuthURL generates the OAuth authorization URL for Galaxy
func (c *GalaxyOAuthClient) GenerateAuthURL(userID string, redirectTo string) (string, error) {
	// Generate random state for CSRF protection
	state, err := generateRandomString(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Generate code verifier and challenge for PKCE
	codeVerifier, err := generateRandomString(43)
	if err != nil {
		return "", fmt.Errorf("failed to generate code verifier: %w", err)
	}

	// Store session
	session := &OAuthSession{
		State:        state,
		CodeVerifier: codeVerifier,
		UserID:       userID,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(c.stateTimeout),
		RedirectTo:   redirectTo,
	}

	c.mu.Lock()
	c.sessions[state] = session
	c.mu.Unlock()

	// Build authorization URL
	codeChallenge := generateCodeChallenge(codeVerifier)
	authURL := fmt.Sprintf("%s/api/oauth2/authorize", c.config.GalaxyURL)

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", c.config.ClientID)
	params.Set("redirect_uri", c.config.RedirectURL)
	params.Set("state", state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")
	params.Set("scope", "read write")

	return fmt.Sprintf("%s?%s", authURL, params.Encode()), nil
}

// HandleCallback handles the OAuth callback from Galaxy
func (c *GalaxyOAuthClient) HandleCallback(code, state string) (*APIKeyInfo, error) {
	// Retrieve and validate session
	c.mu.Lock()
	session, exists := c.sessions[state]
	if exists {
		delete(c.sessions, state)
	}
	c.mu.Unlock()

	if !exists {
		return nil, fmt.Errorf("invalid or expired state")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	// Exchange authorization code for access token
	token, err := c.exchangeCode(code, session.CodeVerifier)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Fetch user info from Galaxy
	userInfo, err := c.fetchUserInfo(token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	// Create API key info
	apiKeyInfo := &APIKeyInfo{
		Key:         token,
		GalaxyURL:   c.config.GalaxyURL,
		UserID:      session.UserID,
		Username:    userInfo.Username,
		Email:       userInfo.Email,
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		IsValid:     true,
		Permissions: []string{"read", "write", "execute"},
	}

	// Store API key
	c.mu.Lock()
	c.apiKeys[token] = apiKeyInfo
	c.mu.Unlock()

	return apiKeyInfo, nil
}

// exchangeCode exchanges authorization code for access token
func (c *GalaxyOAuthClient) exchangeCode(code, codeVerifier string) (string, error) {
	tokenURL := fmt.Sprintf("%s/api/oauth2/token", c.config.GalaxyURL)

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("redirect_uri", c.config.RedirectURL)
	params.Set("client_id", c.config.ClientID)
	params.Set("code_verifier", codeVerifier)

	resp, err := c.httpClient.PostForm(tokenURL, params)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, body)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	return tokenResp.AccessToken, nil
}

// fetchUserInfo fetches user information from Galaxy
func (c *GalaxyOAuthClient) fetchUserInfo(token string) (*GalaxyUserInfo, error) {
	userURL := fmt.Sprintf("%s/api/users/current", c.config.GalaxyURL)

	req, err := http.NewRequest("GET", userURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("user info request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed with status %d", resp.StatusCode)
	}

	var userInfo GalaxyUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}

// GalaxyUserInfo represents user information from Galaxy
type GalaxyUserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// ValidateAPIKey validates a Galaxy API key
func (c *GalaxyOAuthClient) ValidateAPIKey(apiKey string) (*APIKeyInfo, error) {
	// Check cache first
	c.mu.RLock()
	info, exists := c.apiKeys[apiKey]
	c.mu.RUnlock()

	if exists && info.IsValid {
		// Update last used
		c.mu.Lock()
		info.LastUsed = time.Now()
		c.mu.Unlock()
		return info, nil
	}

	// Validate with Galaxy server
	validated, err := c.validateWithGalaxy(apiKey)
	if err != nil {
		return nil, err
	}

	// Cache the result
	c.mu.Lock()
	c.apiKeys[apiKey] = validated
	c.mu.Unlock()

	return validated, nil
}

// validateWithGalaxy validates an API key with the Galaxy server
func (c *GalaxyOAuthClient) validateWithGalaxy(apiKey string) (*APIKeyInfo, error) {
	url := fmt.Sprintf("%s/api/users/current", c.config.GalaxyURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-API-KEY", apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API key validation failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid API key")
	}

	var userInfo GalaxyUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &APIKeyInfo{
		Key:         apiKey,
		GalaxyURL:   c.config.GalaxyURL,
		UserID:      userInfo.ID,
		Username:    userInfo.Username,
		Email:       userInfo.Email,
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		IsValid:     true,
		Permissions: []string{"read", "write", "execute"},
	}, nil
}

// RevokeAPIKey revokes an API key
func (c *GalaxyOAuthClient) RevokeAPIKey(apiKey string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if info, exists := c.apiKeys[apiKey]; exists {
		info.IsValid = false
		delete(c.apiKeys, apiKey)
	}

	return nil
}

// CleanupExpiredSessions removes expired OAuth sessions
func (c *GalaxyOAuthClient) CleanupExpiredSessions() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for state, session := range c.sessions {
		if now.After(session.ExpiresAt) {
			delete(c.sessions, state)
		}
	}
}

// GetGalaxyAPIClient creates an authenticated HTTP client for Galaxy API calls
func (c *GalaxyOAuthClient) GetGalaxyAPIClient(apiKey string) (*GalaxyAPIClient, error) {
	info, err := c.ValidateAPIKey(apiKey)
	if err != nil {
		return nil, err
	}

	return &GalaxyAPIClient{
		baseURL:    info.GalaxyURL,
		apiKey:     apiKey,
		httpClient: c.httpClient,
	}, nil
}

// GalaxyAPIClient provides methods to interact with Galaxy API
type GalaxyAPIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// GetHistory retrieves a Galaxy history
func (g *GalaxyAPIClient) GetHistory(historyID string) (*GalaxyHistory, error) {
	url := fmt.Sprintf("%s/api/histories/%s", g.baseURL, historyID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-API-KEY", g.apiKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get history: status %d", resp.StatusCode)
	}

	var history GalaxyHistory
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		return nil, err
	}

	return &history, nil
}

// GalaxyHistory represents a Galaxy workflow history
type GalaxyHistory struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"create_time"`
	UpdatedAt time.Time `json:"update_time"`
	UserID    string    `json:"user_id"`
}

// generateRandomString generates a cryptographically secure random string
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// generateCodeChallenge generates PKCE code challenge from verifier
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.URLEncoding.EncodeToString(hash[:])
}

// StartPeriodicCleanup starts a goroutine to periodically cleanup expired sessions
func (c *GalaxyOAuthClient) StartPeriodicCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			c.CleanupExpiredSessions()
		}
	}()
}
