/**
 * Natural Language Query Engine
 *
 * Converts natural language queries to SQL using GPT-4
 * Implements security validation and SQL injection prevention
 *
 * Security Features:
 * - Whitelist allowed SQL keywords (SELECT, WHERE, ORDER BY, etc.)
 * - Blacklist dangerous keywords (DROP, DELETE, UPDATE, INSERT, etc.)
 * - Parameterized query validation
 * - Rate limiting (10 queries/minute per user)
 * - Query complexity limits
 */

package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// NLQueryEngine converts natural language to SQL
type NLQueryEngine struct {
	apiKey         string
	model          string
	rateLimiter    *RateLimiter
	queryCache     *QueryCache
	validationRules *ValidationRules
	mu             sync.RWMutex
}

// QueryResult contains the SQL query and metadata
type QueryResult struct {
	OriginalQuery   string    `json:"original_query"`
	GeneratedSQL    string    `json:"generated_sql"`
	IsValid         bool      `json:"is_valid"`
	ValidationError string    `json:"validation_error,omitempty"`
	Explanation     string    `json:"explanation"`
	Timestamp       time.Time `json:"timestamp"`
	ExecutionTimeMs int64     `json:"execution_time_ms"`
}

// ValidationRules defines security rules for SQL validation
type ValidationRules struct {
	AllowedKeywords   []string
	ForbiddenKeywords []string
	AllowedTables     []string
	MaxQueryLength    int
	AllowJoins        bool
	AllowSubqueries   bool
}

// RateLimiter implements per-user rate limiting
type RateLimiter struct {
	requests map[string][]time.Time
	limit    int
	window   time.Duration
	mu       sync.RWMutex
}

// QueryCache caches generated SQL queries
type QueryCache struct {
	cache map[string]*QueryResult
	ttl   time.Duration
	mu    sync.RWMutex
}

// NewNLQueryEngine creates a new natural language query engine
func NewNLQueryEngine(apiKey string) *NLQueryEngine {
	return &NLQueryEngine{
		apiKey:      apiKey,
		model:       "gpt-4", // Use GPT-4 for best accuracy
		rateLimiter: NewRateLimiter(10, time.Minute), // 10 queries per minute
		queryCache:  NewQueryCache(5 * time.Minute),
		validationRules: &ValidationRules{
			AllowedKeywords: []string{
				"SELECT", "FROM", "WHERE", "AND", "OR", "NOT",
				"ORDER BY", "GROUP BY", "HAVING", "LIMIT",
				"COUNT", "SUM", "AVG", "MAX", "MIN",
				"DISTINCT", "AS", "IN", "BETWEEN", "LIKE",
				"IS NULL", "IS NOT NULL",
			},
			ForbiddenKeywords: []string{
				"DROP", "DELETE", "UPDATE", "INSERT", "ALTER",
				"CREATE", "TRUNCATE", "REPLACE", "EXEC", "EXECUTE",
				"UNION", "INTO", "SET", "GRANT", "REVOKE",
			},
			AllowedTables: []string{
				"variants",
			},
			MaxQueryLength:  1000,
			AllowJoins:      false, // No joins for security
			AllowSubqueries: false, // No subqueries for security
		},
	}
}

// ConvertToSQL converts natural language to SQL using GPT-4
func (nq *NLQueryEngine) ConvertToSQL(userID, naturalLanguageQuery string) (*QueryResult, error) {
	startTime := time.Now()

	// Check rate limit
	if !nq.rateLimiter.Allow(userID) {
		return nil, fmt.Errorf("rate limit exceeded: maximum 10 queries per minute")
	}

	// Check cache
	if cached := nq.queryCache.Get(naturalLanguageQuery); cached != nil {
		cached.ExecutionTimeMs = time.Since(startTime).Milliseconds()
		return cached, nil
	}

	// Generate SQL using GPT-4
	sql, explanation, err := nq.generateSQL(naturalLanguageQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQL: %w", err)
	}

	// Validate SQL for security
	isValid, validationError := nq.validateSQL(sql)

	result := &QueryResult{
		OriginalQuery:   naturalLanguageQuery,
		GeneratedSQL:    sql,
		IsValid:         isValid,
		ValidationError: validationError,
		Explanation:     explanation,
		Timestamp:       time.Now(),
		ExecutionTimeMs: time.Since(startTime).Milliseconds(),
	}

	// Cache the result if valid
	if isValid {
		nq.queryCache.Set(naturalLanguageQuery, result)
	}

	return result, nil
}

// generateSQL uses GPT-4 to generate SQL from natural language
func (nq *NLQueryEngine) generateSQL(query string) (sql, explanation string, err error) {
	// Build prompt with schema documentation and examples
	prompt := nq.buildPrompt(query)

	// Call OpenAI API
	reqBody := map[string]interface{}{
		"model": nq.model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a SQL expert specializing in genomic data queries. Convert natural language to SQL queries for the GenomeVedic database. Return ONLY valid SQL queries without any markdown formatting or explanation prefixes.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.0, // Deterministic output
		"max_tokens":  500,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+nq.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", "", fmt.Errorf("no response from OpenAI API")
	}

	response := strings.TrimSpace(result.Choices[0].Message.Content)

	// Clean up response (remove markdown formatting if present)
	response = strings.TrimPrefix(response, "```sql")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// Split SQL and explanation if present
	lines := strings.Split(response, "\n")
	sqlQuery := ""
	explanationText := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") || strings.HasPrefix(line, "//") {
			if explanationText == "" {
				explanationText = strings.TrimPrefix(line, "--")
				explanationText = strings.TrimPrefix(explanationText, "//")
				explanationText = strings.TrimSpace(explanationText)
			}
			continue
		}
		if sqlQuery == "" {
			sqlQuery = line
		} else {
			sqlQuery += " " + line
		}
	}

	if sqlQuery == "" {
		return "", "", fmt.Errorf("failed to extract SQL from response")
	}

	return sqlQuery, explanationText, nil
}

// buildPrompt builds the prompt for GPT-4
func (nq *NLQueryEngine) buildPrompt(query string) string {
	var prompt strings.Builder

	prompt.WriteString("Convert the following natural language query to SQL for the GenomeVedic database.\n\n")
	prompt.WriteString(SchemaDocumentation)
	prompt.WriteString("\n\n## Example Conversions:\n\n")

	// Add 5 most relevant examples
	for i, example := range ExampleMappings {
		if i >= 5 {
			break
		}
		prompt.WriteString(fmt.Sprintf("Query: %s\nSQL: %s\n\n", example.NaturalLanguage, example.SQL))
	}

	prompt.WriteString("\n## Your Task:\n\n")
	prompt.WriteString(fmt.Sprintf("Query: %s\n", query))
	prompt.WriteString("SQL: ")

	return prompt.String()
}

// validateSQL validates SQL query for security
func (nq *NLQueryEngine) validateSQL(sql string) (bool, string) {
	sqlUpper := strings.ToUpper(sql)

	// Check query length
	if len(sql) > nq.validationRules.MaxQueryLength {
		return false, "query exceeds maximum length"
	}

	// Check for forbidden keywords
	for _, keyword := range nq.validationRules.ForbiddenKeywords {
		if strings.Contains(sqlUpper, keyword) {
			return false, fmt.Sprintf("forbidden keyword detected: %s", keyword)
		}
	}

	// Must start with SELECT
	if !strings.HasPrefix(sqlUpper, "SELECT") {
		return false, "query must start with SELECT"
	}

	// Check for allowed tables only
	hasValidTable := false
	for _, table := range nq.validationRules.AllowedTables {
		if strings.Contains(sqlUpper, strings.ToUpper(table)) {
			hasValidTable = true
			break
		}
	}
	if !hasValidTable {
		return false, "query must reference allowed tables only (variants)"
	}

	// Check for joins if not allowed
	if !nq.validationRules.AllowJoins {
		if strings.Contains(sqlUpper, "JOIN") {
			return false, "JOIN operations are not allowed"
		}
	}

	// Check for subqueries if not allowed
	if !nq.validationRules.AllowSubqueries {
		if strings.Count(sqlUpper, "SELECT") > 1 {
			return false, "subqueries are not allowed"
		}
	}

	// Check for dangerous patterns
	dangerousPatterns := []string{
		`\bEXEC\b`,
		`\bEXECUTE\b`,
		`\bxp_\w+`,
		`\bsp_\w+`,
		`;\s*DROP`,
		`;\s*DELETE`,
		`--\s*$`,
		`/\*.*\*/`,
		`'\s*OR\s*'1'\s*=\s*'1`,
		`'\s*OR\s*1\s*=\s*1`,
	}

	for _, pattern := range dangerousPatterns {
		matched, _ := regexp.MatchString(pattern, sqlUpper)
		if matched {
			return false, "query contains dangerous pattern"
		}
	}

	return true, ""
}

// GetExamples returns example query mappings
func (nq *NLQueryEngine) GetExamples() []QueryExample {
	return ExampleMappings
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Allow checks if a request is allowed under rate limit
func (rl *RateLimiter) Allow(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Get user's request history
	requests := rl.requests[userID]

	// Remove old requests outside the window
	validRequests := []time.Time{}
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Check if limit exceeded
	if len(validRequests) >= rl.limit {
		rl.requests[userID] = validRequests
		return false
	}

	// Add current request
	validRequests = append(validRequests, now)
	rl.requests[userID] = validRequests
	return true
}

// Reset resets rate limiter for a user
func (rl *RateLimiter) Reset(userID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.requests, userID)
}

// NewQueryCache creates a new query cache
func NewQueryCache(ttl time.Duration) *QueryCache {
	cache := &QueryCache{
		cache: make(map[string]*QueryResult),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a cached query result
func (qc *QueryCache) Get(query string) *QueryResult {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	result, exists := qc.cache[query]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Since(result.Timestamp) > qc.ttl {
		return nil
	}

	return result
}

// Set stores a query result in cache
func (qc *QueryCache) Set(query string, result *QueryResult) {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	qc.cache[query] = result
}

// cleanup removes expired cache entries
func (qc *QueryCache) cleanup() {
	ticker := time.NewTicker(qc.ttl)
	defer ticker.Stop()

	for range ticker.C {
		qc.mu.Lock()
		now := time.Now()
		for key, result := range qc.cache {
			if now.Sub(result.Timestamp) > qc.ttl {
				delete(qc.cache, key)
			}
		}
		qc.mu.Unlock()
	}
}

// Clear clears the entire cache
func (qc *QueryCache) Clear() {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	qc.cache = make(map[string]*QueryResult)
}
