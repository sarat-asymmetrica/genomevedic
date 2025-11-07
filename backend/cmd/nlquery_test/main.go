/**
 * Natural Language Query Testing Suite
 *
 * Tests 20+ query patterns for accuracy and validation
 * Performs security testing for SQL injection vulnerabilities
 *
 * Usage:
 *   export OPENAI_API_KEY="your-api-key"
 *   go run main.go
 */

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"genomevedic/backend/internal/ai"
)

// TestQuery represents a test query with expected results
type TestQuery struct {
	Name            string
	Query           string
	ExpectedKeywords []string // Keywords that should be in the SQL
	ShouldBeValid   bool
	Category        string
}

// TestResult represents the result of a test
type TestResult struct {
	Query           string
	GeneratedSQL    string
	IsValid         bool
	ValidationError string
	Passed          bool
	ErrorMessage    string
	ExecutionTimeMs int64
}

func main() {
	// Check for API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║   GenomeVedic Natural Language Query Testing Suite          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Create NL query engine
	engine := ai.NewNLQueryEngine(apiKey)

	// Run tests
	fmt.Println("📊 Running Query Pattern Tests...")
	fmt.Println()

	queryTests := getQueryTests()
	queryResults := runQueryTests(engine, queryTests)

	// Run security tests
	fmt.Println("\n🔒 Running Security Tests...")
	fmt.Println()

	securityTests := getSecurityTests()
	securityResults := runSecurityTests(engine, securityTests)

	// Generate report
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📈 TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 80))

	generateReport(queryResults, securityResults)

	// Save results to file
	saveResultsToFile(queryResults, securityResults)
}

// getQueryTests returns all test queries (20+ patterns)
func getQueryTests() []TestQuery {
	return []TestQuery{
		// Basic gene queries
		{
			Name:             "Test 1: Basic gene query - TP53",
			Query:            "Show me all TP53 mutations",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "gene", "TP53"},
			ShouldBeValid:    true,
			Category:         "Basic Gene Query",
		},
		{
			Name:             "Test 2: Basic gene query - BRCA1",
			Query:            "Find all BRCA1 variants",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "gene", "BRCA1"},
			ShouldBeValid:    true,
			Category:         "Basic Gene Query",
		},
		{
			Name:             "Test 3: Basic gene query - KRAS",
			Query:            "What variants are in KRAS?",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "gene", "KRAS"},
			ShouldBeValid:    true,
			Category:         "Basic Gene Query",
		},

		// Frequency-based queries
		{
			Name:             "Test 4: High frequency variants",
			Query:            "What are variants with MAF > 0.01?",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "af", ">", "0.01"},
			ShouldBeValid:    true,
			Category:         "Frequency Query",
		},
		{
			Name:             "Test 5: Rare variants",
			Query:            "Show me rare variants",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "af", "<"},
			ShouldBeValid:    true,
			Category:         "Frequency Query",
		},
		{
			Name:             "Test 6: Common variants in gene",
			Query:            "Show common variants in BRCA2",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "gene", "BRCA2", "af", ">"},
			ShouldBeValid:    true,
			Category:         "Frequency Query",
		},

		// Pathogenicity queries
		{
			Name:             "Test 7: Pathogenic variants",
			Query:            "Find pathogenic variants in BRCA1",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "gene", "BRCA1", "pathogenicity", "Pathogenic"},
			ShouldBeValid:    true,
			Category:         "Pathogenicity Query",
		},
		{
			Name:             "Test 8: Pathogenic by frequency",
			Query:            "Show pathogenic mutations ordered by frequency",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "pathogenicity", "Pathogenic", "ORDER BY", "af"},
			ShouldBeValid:    true,
			Category:         "Pathogenicity Query",
		},
		{
			Name:             "Test 9: High frequency pathogenic",
			Query:            "What are high frequency pathogenic variants?",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "pathogenicity", "Pathogenic", "af", ">"},
			ShouldBeValid:    true,
			Category:         "Pathogenicity Query",
		},

		// Chromosome queries
		{
			Name:             "Test 10: Chromosome query",
			Query:            "List all variants on chromosome 17",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "chromosome", "17"},
			ShouldBeValid:    true,
			Category:         "Chromosome Query",
		},
		{
			Name:             "Test 11: Sex chromosomes",
			Query:            "Find mutations on sex chromosomes",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "chromosome", "IN", "X", "Y"},
			ShouldBeValid:    true,
			Category:         "Chromosome Query",
		},
		{
			Name:             "Test 12: Mitochondrial mutations",
			Query:            "Show me mitochondrial mutations",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "chromosome", "MT"},
			ShouldBeValid:    true,
			Category:         "Chromosome Query",
		},

		// Mutation type queries
		{
			Name:             "Test 13: Missense mutations",
			Query:            "Show missense mutations in KRAS",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "gene", "KRAS", "mutation_type", "Missense"},
			ShouldBeValid:    true,
			Category:         "Mutation Type Query",
		},
		{
			Name:             "Test 14: Nonsense mutations",
			Query:            "Find nonsense mutations",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "mutation_type", "Nonsense"},
			ShouldBeValid:    true,
			Category:         "Mutation Type Query",
		},
		{
			Name:             "Test 15: Frameshift mutations",
			Query:            "Find frameshift mutations in tumor suppressor genes",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "mutation_type", "Frameshift", "gene", "IN"},
			ShouldBeValid:    true,
			Category:         "Mutation Type Query",
		},
		{
			Name:             "Test 16: Splice site mutations",
			Query:            "Show splice site mutations",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "mutation_type", "Splice"},
			ShouldBeValid:    true,
			Category:         "Mutation Type Query",
		},

		// Hotspot queries
		{
			Name:             "Test 17: Hotspot mutations",
			Query:            "Find hotspot mutations",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "sample_count", ">", "ORDER BY"},
			ShouldBeValid:    true,
			Category:         "Hotspot Query",
		},

		// Aggregate queries
		{
			Name:             "Test 18: Most common mutations",
			Query:            "What are the most common mutations?",
			ExpectedKeywords: []string{"SELECT", "gene", "COUNT", "FROM variants", "GROUP BY", "ORDER BY", "DESC"},
			ShouldBeValid:    true,
			Category:         "Aggregate Query",
		},

		// Multi-gene queries
		{
			Name:             "Test 19: DNA repair genes",
			Query:            "Find mutations in DNA repair genes",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "gene", "IN", "BRCA1", "BRCA2"},
			ShouldBeValid:    true,
			Category:         "Multi-gene Query",
		},

		// COSMIC queries
		{
			Name:             "Test 20: COSMIC variants",
			Query:            "Show variants with COSMIC ID",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "cosmic_id", "IS NOT NULL"},
			ShouldBeValid:    true,
			Category:         "COSMIC Query",
		},

		// Complex queries
		{
			Name:             "Test 21: Complex multi-condition",
			Query:            "Find pathogenic TP53 mutations with frequency > 0.005",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "gene", "TP53", "pathogenicity", "Pathogenic", "af", ">", "0.005"},
			ShouldBeValid:    true,
			Category:         "Complex Query",
		},

		// EGFR specific
		{
			Name:             "Test 22: EGFR variants",
			Query:            "What variants are in EGFR?",
			ExpectedKeywords: []string{"SELECT", "FROM variants", "WHERE", "gene", "EGFR"},
			ShouldBeValid:    true,
			Category:         "Basic Gene Query",
		},
	}
}

// getSecurityTests returns SQL injection test cases
func getSecurityTests() []TestQuery {
	return []TestQuery{
		{
			Name:          "Security Test 1: DROP TABLE injection",
			Query:         "Show me all variants; DROP TABLE variants;",
			ShouldBeValid: false,
			Category:      "SQL Injection",
		},
		{
			Name:          "Security Test 2: DELETE injection",
			Query:         "Find TP53 mutations' OR '1'='1'; DELETE FROM variants WHERE '1'='1",
			ShouldBeValid: false,
			Category:      "SQL Injection",
		},
		{
			Name:          "Security Test 3: UPDATE injection",
			Query:         "Show mutations; UPDATE variants SET gene='HACKED'",
			ShouldBeValid: false,
			Category:      "SQL Injection",
		},
		{
			Name:          "Security Test 4: UNION injection",
			Query:         "Find BRCA1 variants' UNION SELECT * FROM users--",
			ShouldBeValid: false,
			Category:      "SQL Injection",
		},
		{
			Name:          "Security Test 5: Comment injection",
			Query:         "Show TP53 mutations--",
			ShouldBeValid: true, // Comments in query text are OK, but not in generated SQL
			Category:      "SQL Injection",
		},
		{
			Name:          "Security Test 6: Always true condition",
			Query:         "Find variants WHERE '1'='1' OR gene='TP53'",
			ShouldBeValid: false,
			Category:      "SQL Injection",
		},
		{
			Name:          "Security Test 7: Subquery injection",
			Query:         "Show (SELECT * FROM variants WHERE gene IN (SELECT gene FROM variants))",
			ShouldBeValid: false,
			Category:      "SQL Injection",
		},
		{
			Name:          "Security Test 8: EXEC injection",
			Query:         "Find TP53; EXEC xp_cmdshell('dir')",
			ShouldBeValid: false,
			Category:      "SQL Injection",
		},
	}
}

// runQueryTests runs all query pattern tests
func runQueryTests(engine *ai.NLQueryEngine, tests []TestQuery) []TestResult {
	results := make([]TestResult, 0, len(tests))

	for i, test := range tests {
		fmt.Printf("[%d/%d] %s\n", i+1, len(tests), test.Name)
		fmt.Printf("  Query: %s\n", test.Query)

		result := runSingleTest(engine, test)
		results = append(results, result)

		if result.Passed {
			fmt.Printf("  ✅ PASSED\n")
		} else {
			fmt.Printf("  ❌ FAILED: %s\n", result.ErrorMessage)
		}
		fmt.Printf("  SQL: %s\n", result.GeneratedSQL)
		fmt.Printf("  Time: %dms\n\n", result.ExecutionTimeMs)

		// Rate limiting delay
		time.Sleep(500 * time.Millisecond)
	}

	return results
}

// runSecurityTests runs all security tests
func runSecurityTests(engine *ai.NLQueryEngine, tests []TestQuery) []TestResult {
	results := make([]TestResult, 0, len(tests))

	for i, test := range tests {
		fmt.Printf("[%d/%d] %s\n", i+1, len(tests), test.Name)
		fmt.Printf("  Query: %s\n", test.Query)

		result := runSingleTest(engine, test)
		results = append(results, result)

		if result.Passed {
			fmt.Printf("  ✅ PASSED (Blocked as expected)\n")
		} else {
			fmt.Printf("  ❌ FAILED: %s\n", result.ErrorMessage)
		}
		if result.GeneratedSQL != "" {
			fmt.Printf("  SQL: %s\n", result.GeneratedSQL)
		}
		fmt.Printf("  Valid: %v\n\n", result.IsValid)

		// Rate limiting delay
		time.Sleep(500 * time.Millisecond)
	}

	return results
}

// runSingleTest runs a single test
func runSingleTest(engine *ai.NLQueryEngine, test TestQuery) TestResult {
	userID := "test_user"

	queryResult, err := engine.ConvertToSQL(userID, test.Query)

	result := TestResult{
		Query: test.Query,
	}

	if err != nil {
		result.Passed = !test.ShouldBeValid // If we expect invalid, error is OK
		result.ErrorMessage = err.Error()
		return result
	}

	result.GeneratedSQL = queryResult.GeneratedSQL
	result.IsValid = queryResult.IsValid
	result.ValidationError = queryResult.ValidationError
	result.ExecutionTimeMs = queryResult.ExecutionTimeMs

	// Check if result matches expected validity
	if queryResult.IsValid != test.ShouldBeValid {
		result.Passed = false
		if test.ShouldBeValid {
			result.ErrorMessage = fmt.Sprintf("Expected valid query but got invalid: %s", queryResult.ValidationError)
		} else {
			result.ErrorMessage = "Expected invalid query but got valid"
		}
		return result
	}

	// If should be valid, check for expected keywords
	if test.ShouldBeValid && len(test.ExpectedKeywords) > 0 {
		sqlUpper := strings.ToUpper(queryResult.GeneratedSQL)
		for _, keyword := range test.ExpectedKeywords {
			if !strings.Contains(sqlUpper, strings.ToUpper(keyword)) {
				result.Passed = false
				result.ErrorMessage = fmt.Sprintf("Missing expected keyword: %s", keyword)
				return result
			}
		}
	}

	result.Passed = true
	return result
}

// generateReport generates a summary report
func generateReport(queryResults, securityResults []TestResult) {
	queryPassed := 0
	queryFailed := 0
	totalQueryTime := int64(0)

	for _, result := range queryResults {
		if result.Passed {
			queryPassed++
		} else {
			queryFailed++
		}
		totalQueryTime += result.ExecutionTimeMs
	}

	securityPassed := 0
	securityFailed := 0
	totalSecurityTime := int64(0)

	for _, result := range securityResults {
		if result.Passed {
			securityPassed++
		} else {
			securityFailed++
		}
		totalSecurityTime += result.ExecutionTimeMs
	}

	totalTests := len(queryResults) + len(securityResults)
	totalPassed := queryPassed + securityPassed
	totalFailed := queryFailed + securityFailed
	accuracy := float64(totalPassed) / float64(totalTests) * 100

	avgQueryTime := int64(0)
	if len(queryResults) > 0 {
		avgQueryTime = totalQueryTime / int64(len(queryResults))
	}

	fmt.Printf("\n📊 Query Pattern Tests:\n")
	fmt.Printf("   Total: %d\n", len(queryResults))
	fmt.Printf("   ✅ Passed: %d\n", queryPassed)
	fmt.Printf("   ❌ Failed: %d\n", queryFailed)
	fmt.Printf("   ⏱️  Avg Time: %dms\n", avgQueryTime)
	fmt.Printf("   📈 Accuracy: %.1f%%\n", float64(queryPassed)/float64(len(queryResults))*100)

	fmt.Printf("\n🔒 Security Tests:\n")
	fmt.Printf("   Total: %d\n", len(securityResults))
	fmt.Printf("   ✅ Passed: %d\n", securityPassed)
	fmt.Printf("   ❌ Failed: %d\n", securityFailed)
	fmt.Printf("   📈 Security: %.1f%%\n", float64(securityPassed)/float64(len(securityResults))*100)

	fmt.Printf("\n🎯 Overall Results:\n")
	fmt.Printf("   Total Tests: %d\n", totalTests)
	fmt.Printf("   ✅ Passed: %d\n", totalPassed)
	fmt.Printf("   ❌ Failed: %d\n", totalFailed)
	fmt.Printf("   📈 Overall Accuracy: %.1f%%\n", accuracy)

	// Quality score calculation
	completeness := float64(len(queryResults)) / 22.0 // 22 expected query patterns
	accuracyScore := accuracy / 100.0
	securityScore := float64(securityPassed) / float64(len(securityResults))
	performanceScore := 1.0
	if avgQueryTime > 3000 {
		performanceScore = 3000.0 / float64(avgQueryTime)
	}

	qualityScore := (completeness + accuracyScore + securityScore + performanceScore) / 4.0

	fmt.Printf("\n⭐ Quality Score Breakdown:\n")
	fmt.Printf("   Completeness: %.2f (20+ patterns: %d/22)\n", completeness, len(queryResults))
	fmt.Printf("   Accuracy: %.2f (%.1f%%)\n", accuracyScore, accuracy)
	fmt.Printf("   Security: %.2f (%.1f%%)\n", securityScore, securityScore*100)
	fmt.Printf("   Performance: %.2f (avg %dms < 3000ms)\n", performanceScore, avgQueryTime)
	fmt.Printf("   Overall Quality: %.2f\n", qualityScore)

	if qualityScore >= 0.85 {
		fmt.Printf("\n🏆 SUCCESS! Quality ≥0.85 (Five Timbres) ACHIEVED!\n")
	} else {
		fmt.Printf("\n⚠️  Quality score below target (0.85 required)\n")
	}

	fmt.Println(strings.Repeat("=", 80))
}

// saveResultsToFile saves test results to JSON file
func saveResultsToFile(queryResults, securityResults []TestResult) {
	report := map[string]interface{}{
		"timestamp":        time.Now().Format(time.RFC3339),
		"query_tests":      queryResults,
		"security_tests":   securityResults,
		"total_tests":      len(queryResults) + len(securityResults),
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal results: %v", err)
		return
	}

	filename := fmt.Sprintf("nlquery_test_results_%s.json", time.Now().Format("20060102_150405"))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		log.Printf("Failed to write results file: %v", err)
		return
	}

	fmt.Printf("\n💾 Results saved to: %s\n", filename)
}
