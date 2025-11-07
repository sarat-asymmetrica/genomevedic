package ai

import (
	"context"
	"testing"
	"time"
)

// TestVariantContext tests the context retrieval for known variants
func TestVariantContext(t *testing.T) {
	retriever := NewContextRetriever("")

	testCases := []struct {
		name     string
		input    VariantInput
		validate func(*testing.T, *VariantContext)
	}{
		{
			name: "TP53 R175H - Known cancer hotspot",
			input: VariantInput{
				Gene:       "TP53",
				Variant:    "R175H",
				Chromosome: "17",
				Position:   7577538,
				RefAllele:  "C",
				AltAllele:  "A",
			},
			validate: func(t *testing.T, ctx *VariantContext) {
				if ctx == nil {
					t.Fatal("Expected context, got nil")
				}
				if ctx.Gene != "TP53" {
					t.Errorf("Expected gene TP53, got %s", ctx.Gene)
				}
				// ClinVar should have data for TP53
				if ctx.ClinVar == nil {
					t.Error("Expected ClinVar data")
				}
				// COSMIC should recognize TP53 as cancer gene
				if ctx.COSMIC == nil {
					t.Error("Expected COSMIC data")
				}
			},
		},
		{
			name: "BRCA1 185delAG - Known pathogenic variant",
			input: VariantInput{
				Gene:       "BRCA1",
				Variant:    "185delAG",
				Chromosome: "17",
				Position:   43094464,
				RefAllele:  "AG",
				AltAllele:  "-",
			},
			validate: func(t *testing.T, ctx *VariantContext) {
				if ctx == nil {
					t.Fatal("Expected context, got nil")
				}
				if ctx.Gene != "BRCA1" {
					t.Errorf("Expected gene BRCA1, got %s", ctx.Gene)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			variantCtx, err := retriever.GetVariantContext(ctx, tc.input)

			if err != nil {
				t.Logf("Warning: Some data sources failed (this is OK): %v", err)
			}

			tc.validate(t, variantCtx)
		})
	}
}

// TestCacheOperations tests cache functionality
func TestCacheOperations(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	entry := &CacheEntry{
		Explanation: "Test explanation",
		Context: &VariantContext{
			Gene:    "TP53",
			Variant: "R175H",
		},
		TokensUsed: 150,
		CachedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
	}

	key := "variant:TP53:R175H:17:7577538"

	// Test Set
	err := cache.Set(ctx, key, entry, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Test Get
	retrieved, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get cache entry: %v", err)
	}

	if retrieved.Explanation != entry.Explanation {
		t.Errorf("Expected explanation %s, got %s", entry.Explanation, retrieved.Explanation)
	}

	// Test hit rate
	hitRate, err := cache.GetHitRate(ctx)
	if err != nil {
		t.Fatalf("Failed to get hit rate: %v", err)
	}

	if hitRate == 0 {
		t.Error("Expected non-zero hit rate")
	}

	t.Logf("Cache hit rate: %.2f%%", hitRate*100)

	// Test Delete
	err = cache.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed to delete cache entry: %v", err)
	}

	// Verify deletion
	_, err = cache.Get(ctx, key)
	if err == nil {
		t.Error("Expected cache miss after deletion")
	}
}

// TestCacheKeyGeneration tests cache key generation
func TestCacheKeyGeneration(t *testing.T) {
	testCases := []struct {
		name     string
		input    VariantInput
		expected string
	}{
		{
			name: "TP53 R175H",
			input: VariantInput{
				Gene:       "TP53",
				Variant:    "R175H",
				Chromosome: "17",
				Position:   7577538,
			},
			expected: "variant:TP53:R175H:17:7577538",
		},
		{
			name: "BRCA1 185delAG",
			input: VariantInput{
				Gene:       "BRCA1",
				Variant:    "185delAG",
				Chromosome: "17",
				Position:   43094464,
			},
			expected: "variant:BRCA1:185delAG:17:43094464",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := GenerateCacheKey(tc.input)
			if key != tc.expected {
				t.Errorf("Expected key %s, got %s", tc.expected, key)
			}
		})
	}
}

// TestQualityEvaluation tests the quality evaluation function
func TestQualityEvaluation(t *testing.T) {
	config := DefaultConfig()
	config.OpenAIAPIKey = "test-key"

	// We can't create a real interpreter without a valid API key,
	// so we'll just test the quality evaluation logic directly

	testCases := []struct {
		name        string
		explanation string
		minQuality  float64
	}{
		{
			name: "Good explanation",
			explanation: `This is a pathogenic variant in TP53. The R175H mutation affects
			the DNA-binding domain of the p53 protein, compromising its tumor suppressor
			function. This variant is associated with Li-Fraumeni syndrome and various
			cancer types. It is extremely rare in the general population (allele frequency
			< 0.0001) and is classified as pathogenic by ClinVar. Multiple clinical studies
			have confirmed its role in cancer predisposition.`,
			minQuality: 0.8,
		},
		{
			name: "Short explanation",
			explanation: `TP53 R175H is pathogenic.`,
			minQuality: 0.2,
		},
		{
			name: "No pathogenicity mentioned",
			explanation: `This variant affects the protein in some way and may be important
			for understanding disease mechanisms in various contexts.`,
			minQuality: 0.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a minimal interpreter just for quality testing
			interpreter := &ChatGPTInterpreter{}
			quality := interpreter.evaluateQuality(tc.explanation, &VariantContext{})

			if quality < tc.minQuality {
				t.Errorf("Expected quality >= %.2f, got %.2f", tc.minQuality, quality)
			}

			t.Logf("Quality score: %.2f", quality)
		})
	}
}

// BenchmarkCacheOperations benchmarks cache operations
func BenchmarkCacheOperations(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()

	entry := &CacheEntry{
		Explanation: "Benchmark explanation",
		Context: &VariantContext{
			Gene:    "TP53",
			Variant: "R175H",
		},
		TokensUsed: 150,
		CachedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
	}

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := GenerateCacheKey(VariantInput{
				Gene:       "TP53",
				Variant:    "R175H",
				Chromosome: "17",
				Position:   int64(i),
			})
			_ = cache.Set(ctx, key, entry, 30*24*time.Hour)
		}
	})

	// Prepopulate cache for Get benchmark
	for i := 0; i < 1000; i++ {
		key := GenerateCacheKey(VariantInput{
			Gene:       "TP53",
			Variant:    "R175H",
			Chromosome: "17",
			Position:   int64(i),
		})
		_ = cache.Set(ctx, key, entry, 30*24*time.Hour)
	}

	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := GenerateCacheKey(VariantInput{
				Gene:       "TP53",
				Variant:    "R175H",
				Chromosome: "17",
				Position:   int64(i % 1000),
			})
			_, _ = cache.Get(ctx, key)
		}
	})
}

// TestDefaultConfig tests default configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.OpenAIModel != "gpt-4-turbo-preview" {
		t.Errorf("Expected model gpt-4-turbo-preview, got %s", config.OpenAIModel)
	}

	if config.CacheTTLDays != 30 {
		t.Errorf("Expected TTL 30 days, got %d", config.CacheTTLDays)
	}

	if config.Temperature != 0.3 {
		t.Errorf("Expected temperature 0.3, got %f", config.Temperature)
	}

	if config.MaxTokens != 500 {
		t.Errorf("Expected max tokens 500, got %d", config.MaxTokens)
	}

	if !config.EnableCache {
		t.Error("Expected cache to be enabled by default")
	}

	if !config.EnableBatching {
		t.Error("Expected batching to be enabled by default")
	}
}
