package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ChatGPTInterpreter handles GPT-4 API calls for variant interpretation
type ChatGPTInterpreter struct {
	config           *Config
	httpClient       *http.Client
	contextRetriever *ContextRetriever
	cacheManager     *CacheManager
}

// NewChatGPTInterpreter creates a new ChatGPT interpreter
func NewChatGPTInterpreter(config *Config) (*ChatGPTInterpreter, error) {
	if config.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	// Create cache store
	var cacheStore CacheStore
	var err error
	if config.EnableCache {
		if config.RedisAddr != "" {
			cacheStore, err = NewRedisCache(config.RedisAddr, config.RedisPassword, config.RedisDB)
			if err != nil {
				// Fall back to memory cache
				cacheStore = NewMemoryCache()
			}
		} else {
			cacheStore = NewMemoryCache()
		}
	} else {
		cacheStore = NewMemoryCache()
	}

	return &ChatGPTInterpreter{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
		},
		contextRetriever: NewContextRetriever(""), // NCBI API key can be added here
		cacheManager:     NewCacheManager(cacheStore, config.CacheTTLDays),
	}, nil
}

// ExplainVariant generates a GPT-4 explanation for a variant
func (ci *ChatGPTInterpreter) ExplainVariant(ctx context.Context, request ExplanationRequest) (*ExplanationResponse, error) {
	startTime := time.Now()

	// Generate cache key
	cacheKey := GenerateCacheKey(request.VariantInput)

	// Try cache first
	if ci.config.EnableCache {
		response, err := ci.cacheManager.GetOrCompute(ctx, cacheKey, func() (*ExplanationResponse, error) {
			return ci.generateExplanation(ctx, request, startTime)
		})
		if err == nil && response.Cached {
			response.ResponseTime = time.Since(startTime)
			return response, nil
		}
		if err == nil {
			return response, nil
		}
	}

	// Cache miss or disabled, generate fresh
	return ci.generateExplanation(ctx, request, startTime)
}

// generateExplanation generates a fresh explanation using GPT-4
func (ci *ChatGPTInterpreter) generateExplanation(ctx context.Context, request ExplanationRequest, startTime time.Time) (*ExplanationResponse, error) {
	// Fetch variant context
	variantContext, err := ci.contextRetriever.GetVariantContext(ctx, request.VariantInput)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve variant context: %w", err)
	}

	// Build prompt
	prompt := ci.buildPrompt(request.VariantInput, variantContext, request.IncludeReferences)

	// Call OpenAI API
	gptResponse, err := ci.callOpenAI(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}

	// Calculate cost (GPT-4 Turbo pricing as of 2024)
	// Input: $0.01 per 1K tokens, Output: $0.03 per 1K tokens
	inputCost := float64(gptResponse.Usage.PromptTokens) / 1000.0 * 0.01
	outputCost := float64(gptResponse.Usage.CompletionTokens) / 1000.0 * 0.03
	totalCost := inputCost + outputCost

	response := &ExplanationResponse{
		Explanation:  strings.TrimSpace(gptResponse.Choices[0].Message.Content),
		Context:      variantContext,
		Cached:       false,
		ResponseTime: time.Since(startTime),
		TokensUsed:   gptResponse.Usage.TotalTokens,
		CostUSD:      totalCost,
		Quality:      ci.evaluateQuality(gptResponse.Choices[0].Message.Content, variantContext),
	}

	return response, nil
}

// buildPrompt constructs the GPT-4 prompt with variant context
func (ci *ChatGPTInterpreter) buildPrompt(input VariantInput, context *VariantContext, includeRefs bool) string {
	var promptBuilder strings.Builder

	promptBuilder.WriteString("You are a genomics expert explaining variants to researchers.\n\n")

	// Variant information
	promptBuilder.WriteString(fmt.Sprintf("Gene: %s\n", input.Gene))
	promptBuilder.WriteString(fmt.Sprintf("Variant: %s\n", input.Variant))
	promptBuilder.WriteString(fmt.Sprintf("Position: %s:%d\n\n", input.Chromosome, input.Position))

	promptBuilder.WriteString("Context:\n")

	// ClinVar data
	if context.ClinVar != nil && context.ClinVar.Found {
		promptBuilder.WriteString(fmt.Sprintf("- ClinVar: %s (%s)\n",
			context.ClinVar.Pathogenicity,
			context.ClinVar.ReviewStatus))
		if len(context.ClinVar.Conditions) > 0 {
			promptBuilder.WriteString(fmt.Sprintf("  Associated with: %s\n",
				strings.Join(context.ClinVar.Conditions, ", ")))
		}
	} else {
		promptBuilder.WriteString("- ClinVar: No data available\n")
	}

	// COSMIC data
	if context.COSMIC != nil && context.COSMIC.Found {
		promptBuilder.WriteString(fmt.Sprintf("- COSMIC: %s (found in %d samples)\n",
			context.COSMIC.CancerAssociation,
			context.COSMIC.Frequency))
		if context.COSMIC.IsHotspot {
			promptBuilder.WriteString("  This is a known cancer hotspot mutation\n")
		}
	} else {
		promptBuilder.WriteString("- COSMIC: No cancer association data\n")
	}

	// gnomAD data
	if context.GnomAD != nil && context.GnomAD.Found {
		promptBuilder.WriteString(fmt.Sprintf("- gnomAD: Allele frequency = %.6f",
			context.GnomAD.AlleleFrequency))
		if context.GnomAD.AlleleFrequency > 0 {
			promptBuilder.WriteString(" (")
			if context.GnomAD.AlleleFrequency < 0.001 {
				promptBuilder.WriteString("very rare")
			} else if context.GnomAD.AlleleFrequency < 0.01 {
				promptBuilder.WriteString("rare")
			} else {
				promptBuilder.WriteString("common")
			}
			promptBuilder.WriteString(")\n")
		} else {
			promptBuilder.WriteString(" (not observed in general population)\n")
		}
	} else {
		promptBuilder.WriteString("- gnomAD: Population frequency unknown\n")
	}

	// PubMed data
	if includeRefs && context.PubMed != nil && context.PubMed.Found {
		promptBuilder.WriteString(fmt.Sprintf("- Recent papers: %d publications found\n",
			context.PubMed.TotalCount))
		if len(context.PubMed.Citations) > 0 {
			promptBuilder.WriteString("  Top citations:\n")
			for i, citation := range context.PubMed.Citations {
				if i >= 3 {
					break
				}
				promptBuilder.WriteString(fmt.Sprintf("  • PMID:%s - %s\n",
					citation.PMID, citation.Title))
			}
		}
	}

	promptBuilder.WriteString("\nProvide a concise explanation (200 words max) covering:\n")
	promptBuilder.WriteString("1. Pathogenicity (Pathogenic/Benign/VUS)\n")
	promptBuilder.WriteString("2. Molecular mechanism (how it affects protein function)\n")
	promptBuilder.WriteString("3. Clinical significance (disease associations)\n")
	promptBuilder.WriteString("4. Population context (rare/common in general population)\n")
	if includeRefs {
		promptBuilder.WriteString("5. Key references (PubMed IDs)\n")
	}

	promptBuilder.WriteString("\nMake it accessible to PhD-level researchers, focusing on actionable insights.")

	return promptBuilder.String()
}

// OpenAI API request/response structures
type openAIRequest struct {
	Model       string      `json:"model"`
	Messages    []message   `json:"messages"`
	MaxTokens   int         `json:"max_tokens"`
	Temperature float32     `json:"temperature"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int     `json:"index"`
		Message      message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// callOpenAI makes the actual API call to OpenAI
func (ci *ChatGPTInterpreter) callOpenAI(ctx context.Context, prompt string) (*openAIResponse, error) {
	requestBody := openAIRequest{
		Model: ci.config.OpenAIModel,
		Messages: []message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   ci.config.MaxTokens,
		Temperature: ci.config.Temperature,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ci.config.OpenAIAPIKey))

	resp, err := ci.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var gptResponse openAIResponse
	if err := json.Unmarshal(body, &gptResponse); err != nil {
		return nil, err
	}

	if len(gptResponse.Choices) == 0 {
		return nil, fmt.Errorf("no response from GPT-4")
	}

	return &gptResponse, nil
}

// evaluateQuality assigns a quality score to the explanation
func (ci *ChatGPTInterpreter) evaluateQuality(explanation string, context *VariantContext) float64 {
	score := 0.0
	checks := 0.0

	// Check 1: Length appropriate (50-250 words)
	wordCount := len(strings.Fields(explanation))
	if wordCount >= 50 && wordCount <= 250 {
		score += 0.2
	}
	checks += 0.2

	// Check 2: Contains pathogenicity assessment
	lowerExpl := strings.ToLower(explanation)
	if strings.Contains(lowerExpl, "pathogenic") || strings.Contains(lowerExpl, "benign") || strings.Contains(lowerExpl, "vus") {
		score += 0.2
	}
	checks += 0.2

	// Check 3: Discusses molecular mechanism
	if strings.Contains(lowerExpl, "protein") || strings.Contains(lowerExpl, "function") || strings.Contains(lowerExpl, "domain") {
		score += 0.2
	}
	checks += 0.2

	// Check 4: Mentions clinical significance
	if strings.Contains(lowerExpl, "cancer") || strings.Contains(lowerExpl, "disease") || strings.Contains(lowerExpl, "clinical") {
		score += 0.2
	}
	checks += 0.2

	// Check 5: Includes population context
	if strings.Contains(lowerExpl, "rare") || strings.Contains(lowerExpl, "common") || strings.Contains(lowerExpl, "frequency") {
		score += 0.2
	}
	checks += 0.2

	return score
}

// GetCacheStats returns cache statistics
func (ci *ChatGPTInterpreter) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	return ci.cacheManager.GetStats(ctx)
}

// Close closes the interpreter and releases resources
func (ci *ChatGPTInterpreter) Close() error {
	if ci.cacheManager != nil && ci.cacheManager.store != nil {
		return ci.cacheManager.store.Close()
	}
	return nil
}

// BatchExplainVariants explains multiple variants using Williams Optimizer batching
func (ci *ChatGPTInterpreter) BatchExplainVariants(ctx context.Context, requests []ExplanationRequest) ([]*ExplanationResponse, error) {
	if !ci.config.EnableBatching || len(requests) == 1 {
		// No batching, process individually
		responses := make([]*ExplanationResponse, len(requests))
		for i, req := range requests {
			resp, err := ci.ExplainVariant(ctx, req)
			if err != nil {
				return nil, err
			}
			responses[i] = resp
		}
		return responses, nil
	}

	// Williams Optimizer: BatchSize = √n × log₂(n)
	// For now, we'll process in parallel
	responses := make([]*ExplanationResponse, len(requests))
	errChan := make(chan error, len(requests))

	for i, req := range requests {
		go func(idx int, request ExplanationRequest) {
			resp, err := ci.ExplainVariant(ctx, request)
			if err != nil {
				errChan <- err
				return
			}
			responses[idx] = resp
			errChan <- nil
		}(i, req)
	}

	// Collect results
	for range requests {
		if err := <-errChan; err != nil {
			return nil, err
		}
	}

	return responses, nil
}
