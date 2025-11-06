/**
 * Mutation Hotspot Detector
 *
 * Identifies and ranks mutation hotspots
 * Uses statistical methods to find significant clustering
 *
 * Algorithm:
 * 1. Window-based clustering (sliding window)
 * 2. Statistical significance (Poisson distribution)
 * 3. Rank by clinical impact score
 */

package mutations

import (
	"fmt"
	"math"
	"sort"
)

// Hotspot represents a mutation hotspot region
type Hotspot struct {
	Chromosome      string
	StartPosition   uint64
	EndPosition     uint64
	MutationCount   int
	TotalSamples    int
	PrimaryGene     string
	Mutations       []*Mutation
	SignificanceScore float64 // Statistical significance (p-value)
	ClinicalScore   float64 // Clinical impact score (0.0-1.0)
}

// HotspotDetector detects mutation hotspots
type HotspotDetector struct {
	parser        *COSMICParser
	windowSize    uint64  // Window size for clustering (base pairs)
	minMutations  int     // Minimum mutations for hotspot
	minSamples    int     // Minimum total samples for hotspot
	baselineRate  float64 // Baseline mutation rate (mutations per bp)
}

// NewHotspotDetector creates a new hotspot detector
func NewHotspotDetector(parser *COSMICParser, windowSize uint64, minMutations int, minSamples int) *HotspotDetector {
	return &HotspotDetector{
		parser:       parser,
		windowSize:   windowSize,
		minMutations: minMutations,
		minSamples:   minSamples,
		baselineRate: 0.0,
	}
}

// DetectHotspots detects mutation hotspots
func (hd *HotspotDetector) DetectHotspots() ([]*Hotspot, error) {
	// Calculate baseline mutation rate
	hd.computeBaselineRate()

	// Group mutations by chromosome
	mutsByChrom := hd.groupMutationsByChromosome()

	// Detect hotspots in each chromosome
	allHotspots := make([]*Hotspot, 0, 100)

	for chrom, mutations := range mutsByChrom {
		// Sort by position
		sort.Slice(mutations, func(i, j int) bool {
			return mutations[i].Position < mutations[j].Position
		})

		// Sliding window approach
		hotspots := hd.detectInChromosome(chrom, mutations)
		allHotspots = append(allHotspots, hotspots...)
	}

	// Compute statistical significance and clinical scores
	for _, hs := range allHotspots {
		hd.computeSignificance(hs)
		hd.computeClinicalScore(hs)
	}

	// Sort by clinical score (most significant first)
	sort.Slice(allHotspots, func(i, j int) bool {
		return allHotspots[i].ClinicalScore > allHotspots[j].ClinicalScore
	})

	return allHotspots, nil
}

// computeBaselineRate computes baseline mutation rate
func (hd *HotspotDetector) computeBaselineRate() {
	mutations := hd.parser.GetMutations()
	if len(mutations) == 0 {
		hd.baselineRate = 0.0
		return
	}

	// Estimate genome size (assume mutations span ~3B bp)
	genomeSize := 3000000000.0

	// Baseline rate = total mutations / genome size
	hd.baselineRate = float64(len(mutations)) / genomeSize
}

// groupMutationsByChromosome groups mutations by chromosome
func (hd *HotspotDetector) groupMutationsByChromosome() map[string][]*Mutation {
	mutsByChrom := make(map[string][]*Mutation)

	for _, mut := range hd.parser.GetMutations() {
		mutsByChrom[mut.Chromosome] = append(mutsByChrom[mut.Chromosome], mut)
	}

	return mutsByChrom
}

// detectInChromosome detects hotspots in a single chromosome
func (hd *HotspotDetector) detectInChromosome(chrom string, mutations []*Mutation) []*Hotspot {
	hotspots := make([]*Hotspot, 0, 10)

	if len(mutations) == 0 {
		return hotspots
	}

	// Sliding window
	for i := 0; i < len(mutations); i++ {
		windowStart := mutations[i].Position
		windowEnd := windowStart + hd.windowSize

		// Collect mutations in window
		windowMutations := make([]*Mutation, 0, 20)
		totalSamples := 0

		for j := i; j < len(mutations) && mutations[j].Position <= windowEnd; j++ {
			windowMutations = append(windowMutations, mutations[j])
			totalSamples += mutations[j].SampleCount
		}

		// Check if this is a hotspot
		if len(windowMutations) >= hd.minMutations && totalSamples >= hd.minSamples {
			// Find primary gene (most frequent)
			primaryGene := hd.findPrimaryGene(windowMutations)

			hotspot := &Hotspot{
				Chromosome:    chrom,
				StartPosition: windowStart,
				EndPosition:   windowEnd,
				MutationCount: len(windowMutations),
				TotalSamples:  totalSamples,
				PrimaryGene:   primaryGene,
				Mutations:     windowMutations,
			}

			hotspots = append(hotspots, hotspot)

			// Skip ahead to avoid overlapping windows
			i += len(windowMutations) / 2
		}
	}

	return hotspots
}

// findPrimaryGene finds the most frequent gene in mutations
func (hd *HotspotDetector) findPrimaryGene(mutations []*Mutation) string {
	geneCounts := make(map[string]int)

	for _, mut := range mutations {
		if mut.Gene != "" {
			geneCounts[mut.Gene]++
		}
	}

	maxGene := ""
	maxCount := 0

	for gene, count := range geneCounts {
		if count > maxCount {
			maxGene = gene
			maxCount = count
		}
	}

	return maxGene
}

// computeSignificance computes statistical significance using Poisson
func (hd *HotspotDetector) computeSignificance(hs *Hotspot) {
	// Expected mutations in window based on baseline rate
	windowLength := float64(hs.EndPosition - hs.StartPosition)
	expectedMutations := hd.baselineRate * windowLength

	if expectedMutations == 0 {
		hs.SignificanceScore = 0.0
		return
	}

	// Poisson probability: P(X >= k) where k = observed mutations
	// Use normal approximation for large lambda
	observedMutations := float64(hs.MutationCount)

	// Z-score
	zscore := (observedMutations - expectedMutations) / math.Sqrt(expectedMutations)

	// Convert to p-value (one-tailed)
	pvalue := 0.5 * math.Erfc(zscore/math.Sqrt(2.0))

	// Significance score = -log10(p-value)
	if pvalue > 0 {
		hs.SignificanceScore = -math.Log10(pvalue)
	} else {
		hs.SignificanceScore = 10.0 // Cap at 10
	}
}

// computeClinicalScore computes clinical impact score
func (hd *HotspotDetector) computeClinicalScore(hs *Hotspot) {
	// Clinical score based on:
	// 1. Number of pathogenic mutations (40%)
	// 2. Total sample count (30%)
	// 3. Statistical significance (30%)

	// 1. Pathogenic mutations
	pathogenicCount := 0
	for _, mut := range hs.Mutations {
		if mut.Significance == SignificancePathogenic ||
			mut.Significance == SignificanceLikelyPathogenic {
			pathogenicCount++
		}
	}
	pathogenicScore := math.Min(float64(pathogenicCount)/float64(hs.MutationCount), 1.0)

	// 2. Sample count (normalize to 0-1)
	sampleScore := math.Min(float64(hs.TotalSamples)/1000.0, 1.0)

	// 3. Statistical significance (normalize to 0-1)
	sigScore := math.Min(hs.SignificanceScore/10.0, 1.0)

	// Weighted combination
	hs.ClinicalScore = 0.4*pathogenicScore + 0.3*sampleScore + 0.3*sigScore
}

// GetTopHotspots returns top N hotspots by clinical score
func (hd *HotspotDetector) GetTopHotspots(hotspots []*Hotspot, n int) []*Hotspot {
	if n > len(hotspots) {
		n = len(hotspots)
	}
	return hotspots[:n]
}

// FormatHotspot returns a human-readable string for a hotspot
func (hd *HotspotDetector) FormatHotspot(hs *Hotspot) string {
	return fmt.Sprintf(
		"Hotspot: %s:%d-%d | Gene: %s | Mutations: %d | Samples: %d | Clinical Score: %.3f | p-value: %.2e",
		hs.Chromosome,
		hs.StartPosition,
		hs.EndPosition,
		hs.PrimaryGene,
		hs.MutationCount,
		hs.TotalSamples,
		hs.ClinicalScore,
		math.Pow(10, -hs.SignificanceScore),
	)
}

// GetStatistics returns hotspot detection statistics
func (hd *HotspotDetector) GetStatistics(hotspots []*Hotspot) map[string]interface{} {
	if len(hotspots) == 0 {
		return map[string]interface{}{
			"total_hotspots": 0,
		}
	}

	// Compute statistics
	totalMutations := 0
	totalSamples := 0
	avgClinicalScore := 0.0

	for _, hs := range hotspots {
		totalMutations += hs.MutationCount
		totalSamples += hs.TotalSamples
		avgClinicalScore += hs.ClinicalScore
	}

	avgClinicalScore /= float64(len(hotspots))

	return map[string]interface{}{
		"total_hotspots":      len(hotspots),
		"total_mutations":     totalMutations,
		"total_samples":       totalSamples,
		"avg_clinical_score":  avgClinicalScore,
		"window_size":         hd.windowSize,
		"min_mutations":       hd.minMutations,
		"min_samples":         hd.minSamples,
		"baseline_rate":       hd.baselineRate,
	}
}
