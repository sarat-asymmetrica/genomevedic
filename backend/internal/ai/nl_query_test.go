/**
 * Natural Language Query Engine Tests
 *
 * Unit tests for NL query engine components
 * Tests validation, rate limiting, and caching without OpenAI API
 */

package ai

import (
	"strings"
	"testing"
	"time"
)

// TestValidateSQL tests SQL validation logic
func TestValidateSQL(t *testing.T) {
	engine := &NLQueryEngine{
		validationRules: &ValidationRules{
			AllowedKeywords: []string{
				"SELECT", "FROM", "WHERE", "AND", "OR",
				"ORDER BY", "GROUP BY", "LIMIT",
			},
			ForbiddenKeywords: []string{
				"DROP", "DELETE", "UPDATE", "INSERT",
			},
			AllowedTables:   []string{"variants"},
			MaxQueryLength:  1000,
			AllowJoins:      false,
			AllowSubqueries: false,
		},
	}

	tests := []struct {
		name          string
		sql           string
		shouldBeValid bool
		expectedError string
	}{
		{
			name:          "Valid basic SELECT",
			sql:           "SELECT * FROM variants WHERE gene = 'TP53'",
			shouldBeValid: true,
		},
		{
			name:          "Valid with ORDER BY",
			sql:           "SELECT * FROM variants WHERE af > 0.01 ORDER BY af DESC",
			shouldBeValid: true,
		},
		{
			name:          "Valid COUNT query",
			sql:           "SELECT gene, COUNT(*) FROM variants GROUP BY gene",
			shouldBeValid: true,
		},
		{
			name:          "Invalid - DROP TABLE",
			sql:           "DROP TABLE variants",
			shouldBeValid: false,
			expectedError: "forbidden keyword detected",
		},
		{
			name:          "Invalid - DELETE",
			sql:           "DELETE FROM variants WHERE gene = 'TP53'",
			shouldBeValid: false,
			expectedError: "forbidden keyword detected",
		},
		{
			name:          "Invalid - UPDATE",
			sql:           "UPDATE variants SET gene = 'HACKED'",
			shouldBeValid: false,
			expectedError: "forbidden keyword detected",
		},
		{
			name:          "Invalid - INSERT",
			sql:           "INSERT INTO variants VALUES (...)",
			shouldBeValid: false,
			expectedError: "forbidden keyword detected",
		},
		{
			name:          "Invalid - Not starting with SELECT",
			sql:           "SHOW TABLES",
			shouldBeValid: false,
			expectedError: "query must start with SELECT",
		},
		{
			name:          "Invalid - JOIN not allowed",
			sql:           "SELECT * FROM variants JOIN genes ON variants.gene_id = genes.id",
			shouldBeValid: false,
			expectedError: "JOIN operations are not allowed",
		},
		{
			name:          "Invalid - Subquery not allowed",
			sql:           "SELECT * FROM variants WHERE gene IN (SELECT gene FROM top_genes)",
			shouldBeValid: false,
			expectedError: "subqueries are not allowed",
		},
		{
			name:          "Invalid - No allowed table",
			sql:           "SELECT * FROM users",
			shouldBeValid: false,
			expectedError: "query must reference allowed tables only",
		},
		{
			name:          "Invalid - SQL injection attempt",
			sql:           "SELECT * FROM variants WHERE gene = 'TP53' OR '1'='1'",
			shouldBeValid: false,
			expectedError: "query contains dangerous pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid, errorMsg := engine.validateSQL(tt.sql)

			if isValid != tt.shouldBeValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.shouldBeValid, isValid)
			}

			if !tt.shouldBeValid && tt.expectedError != "" {
				if !strings.Contains(errorMsg, tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, errorMsg)
				}
			}
		})
	}
}

// TestRateLimiter tests rate limiting logic
func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(3, time.Second)

	userID := "test_user"

	// Should allow first 3 requests
	for i := 0; i < 3; i++ {
		if !limiter.Allow(userID) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should block 4th request
	if limiter.Allow(userID) {
		t.Error("4th request should be blocked")
	}

	// Wait for window to expire
	time.Sleep(1100 * time.Millisecond)

	// Should allow request after window expires
	if !limiter.Allow(userID) {
		t.Error("Request should be allowed after window expires")
	}
}

// TestRateLimiterMultipleUsers tests rate limiting with multiple users
func TestRateLimiterMultipleUsers(t *testing.T) {
	limiter := NewRateLimiter(2, time.Second)

	user1 := "user1"
	user2 := "user2"

	// User 1 makes 2 requests (at limit)
	limiter.Allow(user1)
	limiter.Allow(user1)

	// User 2 should still be able to make requests
	if !limiter.Allow(user2) {
		t.Error("User 2 should be allowed (independent rate limit)")
	}

	// User 1 should be blocked
	if limiter.Allow(user1) {
		t.Error("User 1 should be blocked")
	}
}

// TestRateLimiterReset tests rate limiter reset
func TestRateLimiterReset(t *testing.T) {
	limiter := NewRateLimiter(1, time.Second)

	userID := "test_user"

	// Make request (at limit)
	limiter.Allow(userID)

	// Should be blocked
	if limiter.Allow(userID) {
		t.Error("Should be blocked")
	}

	// Reset
	limiter.Reset(userID)

	// Should be allowed after reset
	if !limiter.Allow(userID) {
		t.Error("Should be allowed after reset")
	}
}

// TestQueryCache tests query caching
func TestQueryCache(t *testing.T) {
	cache := NewQueryCache(time.Second)

	query := "Show me all TP53 mutations"
	result := &QueryResult{
		OriginalQuery: query,
		GeneratedSQL:  "SELECT * FROM variants WHERE gene = 'TP53'",
		IsValid:       true,
		Timestamp:     time.Now(),
	}

	// Cache miss
	if cached := cache.Get(query); cached != nil {
		t.Error("Should be cache miss")
	}

	// Set cache
	cache.Set(query, result)

	// Cache hit
	if cached := cache.Get(query); cached == nil {
		t.Error("Should be cache hit")
	}

	// Wait for TTL expiration
	time.Sleep(1100 * time.Millisecond)

	// Should be expired
	if cached := cache.Get(query); cached != nil {
		t.Error("Should be expired")
	}
}

// TestQueryCacheClear tests cache clearing
func TestQueryCacheClear(t *testing.T) {
	cache := NewQueryCache(time.Minute)

	query := "Show me all TP53 mutations"
	result := &QueryResult{
		OriginalQuery: query,
		GeneratedSQL:  "SELECT * FROM variants WHERE gene = 'TP53'",
		IsValid:       true,
		Timestamp:     time.Now(),
	}

	cache.Set(query, result)

	// Should be in cache
	if cached := cache.Get(query); cached == nil {
		t.Error("Should be in cache")
	}

	// Clear cache
	cache.Clear()

	// Should be gone
	if cached := cache.Get(query); cached != nil {
		t.Error("Should be cleared")
	}
}

// TestBuildPrompt tests prompt building
func TestBuildPrompt(t *testing.T) {
	engine := &NLQueryEngine{}

	query := "Show me all TP53 mutations"
	prompt := engine.buildPrompt(query)

	// Check that prompt contains essential elements
	if !strings.Contains(prompt, "GenomeVedic database") {
		t.Error("Prompt should mention GenomeVedic database")
	}

	if !strings.Contains(prompt, "variants") {
		t.Error("Prompt should mention variants table")
	}

	if !strings.Contains(prompt, query) {
		t.Error("Prompt should contain the user query")
	}

	if !strings.Contains(prompt, "Example Conversions") {
		t.Error("Prompt should include examples")
	}
}

// BenchmarkValidateSQL benchmarks SQL validation performance
func BenchmarkValidateSQL(b *testing.B) {
	engine := &NLQueryEngine{
		validationRules: &ValidationRules{
			AllowedKeywords: []string{
				"SELECT", "FROM", "WHERE", "AND", "OR",
			},
			ForbiddenKeywords: []string{
				"DROP", "DELETE", "UPDATE",
			},
			AllowedTables:  []string{"variants"},
			MaxQueryLength: 1000,
		},
	}

	sql := "SELECT * FROM variants WHERE gene = 'TP53' AND af > 0.01 ORDER BY af DESC"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.validateSQL(sql)
	}
}

// BenchmarkRateLimiter benchmarks rate limiter performance
func BenchmarkRateLimiter(b *testing.B) {
	limiter := NewRateLimiter(1000, time.Minute)
	userID := "test_user"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(userID)
	}
}

// BenchmarkQueryCache benchmarks cache performance
func BenchmarkQueryCache(b *testing.B) {
	cache := NewQueryCache(time.Minute)
	query := "Show me all TP53 mutations"
	result := &QueryResult{
		OriginalQuery: query,
		GeneratedSQL:  "SELECT * FROM variants WHERE gene = 'TP53'",
		IsValid:       true,
		Timestamp:     time.Now(),
	}

	cache.Set(query, result)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(query)
	}
}
