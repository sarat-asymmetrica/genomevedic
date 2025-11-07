package ai

import "time"

// VariantInput represents a genetic variant to be explained
type VariantInput struct {
	Gene       string `json:"gene"`
	Variant    string `json:"variant"`     // e.g., "R175H"
	Chromosome string `json:"chromosome"`  // e.g., "17"
	Position   int64  `json:"position"`    // genomic position
	RefAllele  string `json:"ref_allele"`  // reference allele
	AltAllele  string `json:"alt_allele"`  // alternate allele
}

// VariantContext holds all external data about a variant
type VariantContext struct {
	Gene       string
	Variant    string
	Chromosome string
	Position   int64

	// ClinVar data
	ClinVar *ClinVarData

	// COSMIC data
	COSMIC *COSMICData

	// gnomAD data
	GnomAD *GnomADData

	// PubMed citations
	PubMed *PubMedData
}

// ClinVarData holds ClinVar API response
type ClinVarData struct {
	Pathogenicity string   `json:"pathogenicity"` // Pathogenic/Benign/VUS
	ReviewStatus  string   `json:"review_status"` // Practice guideline/Expert panel/etc
	Conditions    []string `json:"conditions"`    // Associated diseases
	LastEvaluated string   `json:"last_evaluated"`
	Found         bool     `json:"found"` // Whether data was found in ClinVar
}

// COSMICData holds COSMIC mutation data
type COSMICData struct {
	CancerAssociation string   `json:"cancer_association"` // Primary cancer types
	Frequency         int      `json:"frequency"`          // Number of samples with mutation
	CancerTypes       []string `json:"cancer_types"`       // List of cancer types
	IsHotspot         bool     `json:"is_hotspot"`         // Whether it's a known hotspot
	Found             bool     `json:"found"`              // Whether data was found in COSMIC
}

// GnomADData holds gnomAD population frequency data
type GnomADData struct {
	AlleleFrequency    float64 `json:"allele_frequency"`    // Overall AF
	PopulationMaxAF    float64 `json:"population_max_af"`   // Max AF in any population
	PopulationMaxName  string  `json:"population_max_name"` // Population with max AF
	HomozygoteCount    int     `json:"homozygote_count"`    // Number of homozygotes
	Found              bool    `json:"found"`               // Whether data was found in gnomAD
}

// PubMedData holds relevant PubMed citations
type PubMedData struct {
	TotalCount int             `json:"total_count"` // Total papers found
	Citations  []PubMedCitation `json:"citations"`   // Top 5 most relevant papers
	Found      bool            `json:"found"`       // Whether data was found
}

// PubMedCitation represents a single PubMed paper
type PubMedCitation struct {
	PMID     string `json:"pmid"`
	Title    string `json:"title"`
	Authors  string `json:"authors"`
	Journal  string `json:"journal"`
	Year     string `json:"year"`
}

// ExplanationRequest represents a request to explain a variant
type ExplanationRequest struct {
	VariantInput
	IncludeReferences bool `json:"include_references"` // Whether to include PubMed refs
}

// ExplanationResponse represents the GPT-4 explanation
type ExplanationResponse struct {
	Explanation   string          `json:"explanation"`    // The GPT-4 generated explanation
	Context       *VariantContext `json:"context"`        // The context data used
	Cached        bool            `json:"cached"`         // Whether response came from cache
	ResponseTime  time.Duration   `json:"response_time"`  // Time taken to generate
	TokensUsed    int             `json:"tokens_used"`    // OpenAI tokens used (0 if cached)
	CostUSD       float64         `json:"cost_usd"`       // Cost in USD (0 if cached)
	Quality       float64         `json:"quality"`        // Quality score (0-1)
	Error         string          `json:"error,omitempty"` // Error message if any
}

// CacheEntry represents a cached explanation
type CacheEntry struct {
	Explanation   string          `json:"explanation"`
	Context       *VariantContext `json:"context"`
	TokensUsed    int             `json:"tokens_used"`
	CachedAt      time.Time       `json:"cached_at"`
	ExpiresAt     time.Time       `json:"expires_at"`
}

// Config holds configuration for the AI service
type Config struct {
	OpenAIAPIKey     string
	OpenAIModel      string // default: "gpt-4-turbo-preview"
	RedisAddr        string
	RedisPassword    string
	RedisDB          int
	CacheTTLDays     int     // default: 30
	MaxTokens        int     // default: 500
	Temperature      float32 // default: 0.3 (more deterministic)
	TimeoutSeconds   int     // default: 30
	EnableCache      bool    // default: true
	EnableBatching   bool    // default: true (Williams Optimizer)
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		OpenAIModel:    "gpt-4-turbo-preview",
		RedisAddr:      "localhost:6379",
		RedisDB:        0,
		CacheTTLDays:   30,
		MaxTokens:      500,
		Temperature:    0.3,
		TimeoutSeconds: 30,
		EnableCache:    true,
		EnableBatching: true,
	}
}
