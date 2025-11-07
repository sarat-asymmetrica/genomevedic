package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"genomevedic/internal/ai"
)

func main() {
	fmt.Println("=== GenomeVedic AI Variant Interpreter Demo ===")
	fmt.Println()

	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	// Create configuration
	config := ai.DefaultConfig()
	config.OpenAIAPIKey = apiKey

	// Create interpreter
	interpreter, err := ai.NewChatGPTInterpreter(config)
	if err != nil {
		log.Fatalf("Failed to create interpreter: %v", err)
	}
	defer interpreter.Close()

	// Test cases: Real known variants
	testVariants := []ai.ExplanationRequest{
		{
			VariantInput: ai.VariantInput{
				Gene:       "TP53",
				Variant:    "R175H",
				Chromosome: "17",
				Position:   7577538,
				RefAllele:  "C",
				AltAllele:  "A",
			},
			IncludeReferences: true,
		},
		{
			VariantInput: ai.VariantInput{
				Gene:       "BRCA1",
				Variant:    "185delAG",
				Chromosome: "17",
				Position:   43094464,
				RefAllele:  "AG",
				AltAllele:  "-",
			},
			IncludeReferences: true,
		},
		{
			VariantInput: ai.VariantInput{
				Gene:       "KRAS",
				Variant:    "G12D",
				Chromosome: "12",
				Position:   25398284,
				RefAllele:  "C",
				AltAllele:  "T",
			},
			IncludeReferences: true,
		},
	}

	// Track performance metrics
	var totalUncachedTime time.Duration
	var totalCachedTime time.Duration
	var uncachedCount int
	var cachedCount int
	var totalCost float64
	var totalTokens int
	var qualityScores []float64

	fmt.Println("Running tests with real variants...")
	fmt.Println()

	for i, request := range testVariants {
		fmt.Printf("--- Test %d: %s %s ---\n", i+1, request.Gene, request.Variant)

		// First call (uncached)
		ctx := context.Background()
		startTime := time.Now()

		response, err := interpreter.ExplainVariant(ctx, request)
		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}

		elapsed := time.Since(startTime)

		// Print results
		fmt.Printf("Gene: %s\n", request.Gene)
		fmt.Printf("Variant: %s\n", request.Variant)
		fmt.Printf("Position: %s:%d\n", request.Chromosome, request.Position)
		fmt.Println()
		fmt.Printf("Explanation:\n%s\n", response.Explanation)
		fmt.Println()

		// Print context data
		if response.Context != nil {
			fmt.Println("Data Sources:")
			if response.Context.ClinVar != nil {
				fmt.Printf("  ClinVar: %s (Found: %v)\n",
					response.Context.ClinVar.Pathogenicity,
					response.Context.ClinVar.Found)
			}
			if response.Context.COSMIC != nil {
				fmt.Printf("  COSMIC: %s (Found: %v)\n",
					response.Context.COSMIC.CancerAssociation,
					response.Context.COSMIC.Found)
			}
			if response.Context.GnomAD != nil {
				fmt.Printf("  gnomAD: AF=%.2e (Found: %v)\n",
					response.Context.GnomAD.AlleleFrequency,
					response.Context.GnomAD.Found)
			}
			if response.Context.PubMed != nil {
				fmt.Printf("  PubMed: %d publications (Found: %v)\n",
					response.Context.PubMed.TotalCount,
					response.Context.PubMed.Found)
			}
			fmt.Println()
		}

		// Print metrics
		fmt.Println("Metrics:")
		fmt.Printf("  Response Time: %v\n", elapsed)
		fmt.Printf("  Cached: %v\n", response.Cached)
		fmt.Printf("  Tokens Used: %d\n", response.TokensUsed)
		fmt.Printf("  Cost: $%.4f\n", response.CostUSD)
		fmt.Printf("  Quality Score: %.2f (%.0f%%)\n", response.Quality, response.Quality*100)
		fmt.Println()

		// Track metrics
		if response.Cached {
			totalCachedTime += elapsed
			cachedCount++
		} else {
			totalUncachedTime += elapsed
			uncachedCount++
			totalCost += response.CostUSD
			totalTokens += response.TokensUsed
		}
		qualityScores = append(qualityScores, response.Quality)

		// Second call (should be cached)
		fmt.Println("Testing cache (second call)...")
		startTime = time.Now()

		response2, err := interpreter.ExplainVariant(ctx, request)
		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}

		elapsed2 := time.Since(startTime)

		fmt.Printf("  Response Time: %v\n", elapsed2)
		fmt.Printf("  Cached: %v\n", response2.Cached)

		if response2.Cached {
			totalCachedTime += elapsed2
			cachedCount++
			fmt.Printf("  Cache speedup: %.1fx faster\n", float64(elapsed)/float64(elapsed2))
		}

		fmt.Println()
		fmt.Println("========================================")
		fmt.Println()

		// Small delay between requests to respect API rate limits
		if i < len(testVariants)-1 {
			time.Sleep(2 * time.Second)
		}
	}

	// Print summary statistics
	fmt.Println()
	fmt.Println("=== SUMMARY STATISTICS ===")
	fmt.Println()

	if uncachedCount > 0 {
		avgUncachedTime := totalUncachedTime / time.Duration(uncachedCount)
		fmt.Printf("Average Uncached Response Time: %v\n", avgUncachedTime)
		fmt.Printf("Target: <5s ✓ %v\n", avgUncachedTime < 5*time.Second)
	}

	if cachedCount > 0 {
		avgCachedTime := totalCachedTime / time.Duration(cachedCount)
		fmt.Printf("Average Cached Response Time: %v\n", avgCachedTime)
		fmt.Printf("Target: <100ms ✓ %v\n", avgCachedTime < 100*time.Millisecond)
	}

	if uncachedCount > 0 {
		avgCost := totalCost / float64(uncachedCount)
		fmt.Printf("Average Cost per Explanation: $%.4f\n", avgCost)
		fmt.Printf("Target: <$0.01 ✓ %v\n", avgCost < 0.01)
	}

	if len(qualityScores) > 0 {
		avgQuality := 0.0
		for _, q := range qualityScores {
			avgQuality += q
		}
		avgQuality /= float64(len(qualityScores))
		fmt.Printf("Average Quality Score: %.2f (%.0f%%)\n", avgQuality, avgQuality*100)
		fmt.Printf("Target: ≥0.85 ✓ %v\n", avgQuality >= 0.85)
	}

	// Get cache statistics
	stats, err := interpreter.GetCacheStats(context.Background())
	if err == nil {
		fmt.Println()
		fmt.Println("Cache Statistics:")
		if hitRate, ok := stats["hit_rate"].(float64); ok {
			fmt.Printf("  Hit Rate: %.1f%%\n", hitRate*100)
		}
	}

	fmt.Println()
	fmt.Printf("Total Tests: %d\n", len(testVariants))
	fmt.Printf("Uncached Calls: %d\n", uncachedCount)
	fmt.Printf("Cached Calls: %d\n", cachedCount)
	fmt.Printf("Total Tokens Used: %d\n", totalTokens)
	fmt.Printf("Total Cost: $%.4f\n", totalCost)

	fmt.Println()

	// Export detailed results to JSON
	results := map[string]interface{}{
		"timestamp":          time.Now().Unix(),
		"test_count":         len(testVariants),
		"uncached_count":     uncachedCount,
		"cached_count":       cachedCount,
		"avg_uncached_ms":    totalUncachedTime.Milliseconds() / int64(max(uncachedCount, 1)),
		"avg_cached_ms":      totalCachedTime.Milliseconds() / int64(max(cachedCount, 1)),
		"total_tokens":       totalTokens,
		"total_cost_usd":     totalCost,
		"avg_quality_score":  average(qualityScores),
		"quality_scores":     qualityScores,
		"passed_5s_target":   totalUncachedTime/time.Duration(max(uncachedCount, 1)) < 5*time.Second,
		"passed_100ms_target": totalCachedTime/time.Duration(max(cachedCount, 1)) < 100*time.Millisecond,
		"passed_cost_target": totalCost/float64(max(uncachedCount, 1)) < 0.01,
		"passed_quality_target": average(qualityScores) >= 0.85,
	}

	jsonData, _ := json.MarshalIndent(results, "", "  ")
	fmt.Println("Detailed Results (JSON):")
	fmt.Println(string(jsonData))
	fmt.Println()

	// Save to file
	outputFile := "ai_demo_results.json"
	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		log.Printf("Failed to write results to file: %v", err)
	} else {
		fmt.Printf("Results saved to: %s\n", outputFile)
	}

	fmt.Println()
	fmt.Println("=== Demo Complete ===")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
