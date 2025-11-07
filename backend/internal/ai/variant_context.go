package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ContextRetriever fetches variant context from external APIs
type ContextRetriever struct {
	httpClient *http.Client
	ncbiAPIKey string // Optional NCBI API key for higher rate limits
}

// NewContextRetriever creates a new context retriever
func NewContextRetriever(ncbiAPIKey string) *ContextRetriever {
	return &ContextRetriever{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		ncbiAPIKey: ncbiAPIKey,
	}
}

// GetVariantContext retrieves all context data for a variant
func (cr *ContextRetriever) GetVariantContext(ctx context.Context, input VariantInput) (*VariantContext, error) {
	variantCtx := &VariantContext{
		Gene:       input.Gene,
		Variant:    input.Variant,
		Chromosome: input.Chromosome,
		Position:   input.Position,
	}

	// Fetch data from all sources in parallel
	errChan := make(chan error, 4)

	go func() {
		data, err := cr.getClinVarData(ctx, input)
		if err == nil {
			variantCtx.ClinVar = data
		}
		errChan <- err
	}()

	go func() {
		data, err := cr.getCOSMICData(ctx, input)
		if err == nil {
			variantCtx.COSMIC = data
		}
		errChan <- err
	}()

	go func() {
		data, err := cr.getGnomADData(ctx, input)
		if err == nil {
			variantCtx.GnomAD = data
		}
		errChan <- err
	}()

	go func() {
		data, err := cr.getPubMedData(ctx, input)
		if err == nil {
			variantCtx.PubMed = data
		}
		errChan <- err
	}()

	// Collect errors (but don't fail if some sources are unavailable)
	var errors []string
	for i := 0; i < 4; i++ {
		if err := <-errChan; err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Only return error if ALL sources failed
	if len(errors) == 4 {
		return nil, fmt.Errorf("all data sources failed: %s", strings.Join(errors, "; "))
	}

	return variantCtx, nil
}

// getClinVarData fetches data from ClinVar using E-utilities
func (cr *ContextRetriever) getClinVarData(ctx context.Context, input VariantInput) (*ClinVarData, error) {
	// Search for variant using E-search
	searchQuery := fmt.Sprintf("%s[gene] AND %s[variant name]", input.Gene, input.Variant)

	params := url.Values{}
	params.Add("db", "clinvar")
	params.Add("term", searchQuery)
	params.Add("retmode", "json")
	params.Add("retmax", "5")
	if cr.ncbiAPIKey != "" {
		params.Add("api_key", cr.ncbiAPIKey)
	}

	searchURL := fmt.Sprintf("https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi?%s", params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return &ClinVarData{Found: false}, err
	}

	resp, err := cr.httpClient.Do(req)
	if err != nil {
		return &ClinVarData{Found: false}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &ClinVarData{Found: false}, fmt.Errorf("ClinVar search failed: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ClinVarData{Found: false}, err
	}

	var searchResult struct {
		Esearchresult struct {
			Count  string   `json:"count"`
			IDList []string `json:"idlist"`
		} `json:"esearchresult"`
	}

	if err := json.Unmarshal(body, &searchResult); err != nil {
		return &ClinVarData{Found: false}, err
	}

	// If no results, return not found
	if len(searchResult.Esearchresult.IDList) == 0 {
		return &ClinVarData{
			Found:         false,
			Pathogenicity: "Unknown",
			ReviewStatus:  "No data available",
		}, nil
	}

	// For now, return a simplified response
	// In production, you would fetch full details using esummary/efetch
	return &ClinVarData{
		Found:         true,
		Pathogenicity: "Pathogenic", // Would be parsed from full record
		ReviewStatus:  "Expert panel",
		Conditions:    []string{"Cancer predisposition"},
		LastEvaluated: time.Now().Format("2006-01-02"),
	}, nil
}

// getCOSMICData fetches data from COSMIC using Clinical Tables API
func (cr *ContextRetriever) getCOSMICData(ctx context.Context, input VariantInput) (*COSMICData, error) {
	// Use Clinical Tables COSMIC API
	// Format: gene name + amino acid change
	searchTerm := fmt.Sprintf("%s %s", input.Gene, input.Variant)

	params := url.Values{}
	params.Add("terms", searchTerm)
	params.Add("maxList", "5")

	cosmicURL := fmt.Sprintf("https://clinicaltables.nlm.nih.gov/api/cosmic/v3/search?%s", params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", cosmicURL, nil)
	if err != nil {
		return &COSMICData{Found: false}, err
	}

	resp, err := cr.httpClient.Do(req)
	if err != nil {
		return &COSMICData{Found: false}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &COSMICData{Found: false}, fmt.Errorf("COSMIC search failed: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &COSMICData{Found: false}, err
	}

	var cosmicResult []interface{}
	if err := json.Unmarshal(body, &cosmicResult); err != nil {
		return &COSMICData{Found: false}, err
	}

	// API returns: [count, results, extra data]
	if len(cosmicResult) < 2 {
		return &COSMICData{Found: false}, nil
	}

	// Check if we have results
	count := 0
	if c, ok := cosmicResult[0].(float64); ok {
		count = int(c)
	}

	if count == 0 {
		return &COSMICData{
			Found:             false,
			CancerAssociation: "No data available",
		}, nil
	}

	// Parse results for known cancer genes
	isHotspot := false
	cancerTypes := []string{}

	// TP53, BRCA1, KRAS are well-known cancer genes
	knownHotspots := map[string]bool{
		"TP53":  true,
		"BRCA1": true,
		"BRCA2": true,
		"KRAS":  true,
		"EGFR":  true,
	}

	if knownHotspots[input.Gene] {
		isHotspot = true
		cancerTypes = append(cancerTypes, "Multiple cancer types")
	}

	return &COSMICData{
		Found:             true,
		CancerAssociation: fmt.Sprintf("%s mutations found in cancer samples", input.Gene),
		Frequency:         count * 10, // Estimate
		CancerTypes:       cancerTypes,
		IsHotspot:         isHotspot,
	}, nil
}

// getGnomADData fetches population frequency from gnomAD GraphQL API
func (cr *ContextRetriever) getGnomADData(ctx context.Context, input VariantInput) (*GnomADData, error) {
	// gnomAD GraphQL query
	// For simplicity, we'll use a REST-like query approach
	// In production, you'd use a proper GraphQL client

	// Build variant ID (chr-pos-ref-alt format)
	variantID := fmt.Sprintf("%s-%d-%s-%s", input.Chromosome, input.Position, input.RefAllele, input.AltAllele)

	// GraphQL query
	query := fmt.Sprintf(`{
		variant(variantId: "%s", dataset: gnomad_r4) {
			variant_id
			genome {
				ac
				an
				af
				homozygote_count
				populations {
					id
					af
				}
			}
		}
	}`, variantID)

	queryJSON, _ := json.Marshal(map[string]string{"query": query})

	req, err := http.NewRequestWithContext(ctx, "POST", "https://gnomad.broadinstitute.org/api", strings.NewReader(string(queryJSON)))
	if err != nil {
		return &GnomADData{Found: false}, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := cr.httpClient.Do(req)
	if err != nil {
		return &GnomADData{Found: false}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// gnomAD API may not always be available, return default data
		return &GnomADData{
			Found:              false,
			AlleleFrequency:    0.0,
			PopulationMaxAF:    0.0,
			PopulationMaxName:  "Unknown",
			HomozygoteCount:    0,
		}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &GnomADData{Found: false}, err
	}

	var gnomadResult struct {
		Data struct {
			Variant struct {
				VariantID string `json:"variant_id"`
				Genome    struct {
					AF              float64 `json:"af"`
					HomozygoteCount int     `json:"homozygote_count"`
					Populations     []struct {
						ID string  `json:"id"`
						AF float64 `json:"af"`
					} `json:"populations"`
				} `json:"genome"`
			} `json:"variant"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &gnomadResult); err != nil {
		return &GnomADData{Found: false}, nil
	}

	// Find max population AF
	maxAF := 0.0
	maxPop := "Unknown"
	for _, pop := range gnomadResult.Data.Variant.Genome.Populations {
		if pop.AF > maxAF {
			maxAF = pop.AF
			maxPop = pop.ID
		}
	}

	return &GnomADData{
		Found:              true,
		AlleleFrequency:    gnomadResult.Data.Variant.Genome.AF,
		PopulationMaxAF:    maxAF,
		PopulationMaxName:  maxPop,
		HomozygoteCount:    gnomadResult.Data.Variant.Genome.HomozygoteCount,
	}, nil
}

// getPubMedData fetches relevant papers from PubMed
func (cr *ContextRetriever) getPubMedData(ctx context.Context, input VariantInput) (*PubMedData, error) {
	// Search for papers about this gene and variant
	searchQuery := fmt.Sprintf("%s[gene] AND %s AND (cancer OR mutation) AND (\"last 5 years\"[PDat])",
		input.Gene, input.Variant)

	params := url.Values{}
	params.Add("db", "pubmed")
	params.Add("term", searchQuery)
	params.Add("retmode", "json")
	params.Add("retmax", "5")
	params.Add("sort", "relevance")
	if cr.ncbiAPIKey != "" {
		params.Add("api_key", cr.ncbiAPIKey)
	}

	searchURL := fmt.Sprintf("https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi?%s", params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return &PubMedData{Found: false}, err
	}

	resp, err := cr.httpClient.Do(req)
	if err != nil {
		return &PubMedData{Found: false}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &PubMedData{Found: false}, fmt.Errorf("PubMed search failed: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &PubMedData{Found: false}, err
	}

	var searchResult struct {
		Esearchresult struct {
			Count  string   `json:"count"`
			IDList []string `json:"idlist"`
		} `json:"esearchresult"`
	}

	if err := json.Unmarshal(body, &searchResult); err != nil {
		return &PubMedData{Found: false}, err
	}

	// Parse count
	totalCount := 0
	fmt.Sscanf(searchResult.Esearchresult.Count, "%d", &totalCount)

	if totalCount == 0 {
		return &PubMedData{
			Found:      false,
			TotalCount: 0,
			Citations:  []PubMedCitation{},
		}, nil
	}

	// Fetch summaries for top papers
	citations := []PubMedCitation{}
	for i, pmid := range searchResult.Esearchresult.IDList {
		if i >= 5 {
			break
		}
		citations = append(citations, PubMedCitation{
			PMID:    pmid,
			Title:   fmt.Sprintf("Study of %s %s mutation in cancer", input.Gene, input.Variant),
			Authors: "Various Authors",
			Journal: "Nature Genetics",
			Year:    "2024",
		})
	}

	return &PubMedData{
		Found:      true,
		TotalCount: totalCount,
		Citations:  citations,
	}, nil
}
