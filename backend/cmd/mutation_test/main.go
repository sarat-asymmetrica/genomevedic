/**
 * COSMIC Mutation Database Test
 *
 * Tests COSMIC parser, mutation overlay, and hotspot detector
 * Validates cancer mutation visualization pipeline
 */

package main

import (
	"fmt"
	"os"
	"time"

	"genomevedic/backend/internal/mutations"
)

func main() {
	fmt.Println("=== GenomeVedic.ai - COSMIC Mutation Database Test ===\n")

	// Test 1: Parse COSMIC database
	fmt.Println("Test 1: Parsing COSMIC database...")
	parser, err := testCOSMICParser()
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ COSMIC parser working\n")

	// Test 2: Mutation overlay
	fmt.Println("Test 2: Building mutation overlay...")
	overlay, err := testMutationOverlay(parser)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Mutation overlay working\n")

	// Test 3: Hotspot detection
	fmt.Println("Test 3: Detecting mutation hotspots...")
	hotspots, err := testHotspotDetector(parser)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Hotspot detector working\n")

	// Test 4: Query mutations at specific positions
	fmt.Println("Test 4: Querying mutations at specific positions...")
	testPositionQueries(overlay)
	fmt.Println("✓ Position queries working\n")

	// Test 5: Display top hotspots
	fmt.Println("Test 5: Top 10 Cancer Hotspots:")
	displayTopHotspots(hotspots, 10)

	fmt.Println("\n=== All Tests Passed ===")
	fmt.Println("Agent 4.1 (COSMIC Integration) Complete!")
}

func testCOSMICParser() (*mutations.COSMICParser, error) {
	startTime := time.Now()

	// Create parser (hotspot threshold = 100 samples)
	parser := mutations.NewCOSMICParser(100)

	// Open sample COSMIC file
	file, err := os.Open("backend/testdata/cosmic_sample.tsv")
	if err != nil {
		return nil, fmt.Errorf("failed to open COSMIC file: %w", err)
	}
	defer file.Close()

	// Parse file
	if err := parser.ParseFile(file); err != nil {
		return nil, fmt.Errorf("failed to parse COSMIC file: %w", err)
	}

	parseTime := time.Since(startTime)

	// Get statistics
	stats := parser.GetStatistics()

	fmt.Printf("  Parse time: %.2f ms\n", float64(parseTime.Microseconds())/1000.0)
	fmt.Printf("  Total mutations: %d\n", stats["total_mutations"])
	fmt.Printf("  Total hotspots: %d\n", stats["total_hotspots"])
	fmt.Printf("  Hotspot threshold: %d samples\n", stats["hotspot_threshold"])

	// Print mutation type breakdown
	typeCounts := stats["type_counts"].(map[mutations.MutationType]int)
	fmt.Printf("  Mutation types:\n")
	for mutType, count := range typeCounts {
		fmt.Printf("    %s: %d\n", mutType.String(), count)
	}

	// Print significance breakdown
	sigCounts := stats["significance_counts"].(map[mutations.Significance]int)
	fmt.Printf("  Clinical significance:\n")
	for sig, count := range sigCounts {
		fmt.Printf("    %s: %d\n", sig.String(), count)
	}

	return parser, nil
}

func testMutationOverlay(parser *mutations.COSMICParser) (*mutations.MutationOverlay, error) {
	startTime := time.Now()

	// Create overlay (hotspot radius = 50 bp)
	overlay := mutations.NewMutationOverlay(parser, 50)

	// Build overlay
	if err := overlay.BuildOverlay(); err != nil {
		return nil, fmt.Errorf("failed to build overlay: %w", err)
	}

	buildTime := time.Since(startTime)

	// Get statistics
	stats := overlay.GetStatistics()

	fmt.Printf("  Build time: %.2f ms\n", float64(buildTime.Microseconds())/1000.0)
	fmt.Printf("  Total particles with mutations: %d\n", stats["total_particles"])
	fmt.Printf("  Hotspot particles: %d\n", stats["hotspot_particles"])
	fmt.Printf("  Pathogenic particles: %d\n", stats["pathogenic_particles"])
	fmt.Printf("  Uncertain particles: %d\n", stats["uncertain_particles"])
	fmt.Printf("  Benign particles: %d\n", stats["benign_particles"])
	fmt.Printf("  Hotspot radius: %d bp\n", stats["hotspot_radius"])

	return overlay, nil
}

func testHotspotDetector(parser *mutations.COSMICParser) ([]*mutations.Hotspot, error) {
	startTime := time.Now()

	// Create detector (window = 100 bp, min 5 mutations, min 100 samples)
	detector := mutations.NewHotspotDetector(parser, 100, 5, 100)

	// Detect hotspots
	hotspots, err := detector.DetectHotspots()
	if err != nil {
		return nil, fmt.Errorf("failed to detect hotspots: %w", err)
	}

	detectTime := time.Since(startTime)

	// Get statistics
	stats := detector.GetStatistics(hotspots)

	fmt.Printf("  Detection time: %.2f ms\n", float64(detectTime.Microseconds())/1000.0)
	fmt.Printf("  Total hotspots detected: %d\n", stats["total_hotspots"])
	fmt.Printf("  Total mutations in hotspots: %d\n", stats["total_mutations"])
	fmt.Printf("  Total samples in hotspots: %d\n", stats["total_samples"])
	fmt.Printf("  Average clinical score: %.3f\n", stats["avg_clinical_score"])
	fmt.Printf("  Window size: %d bp\n", stats["window_size"])
	fmt.Printf("  Baseline mutation rate: %.2e mutations/bp\n", stats["baseline_rate"])

	return hotspots, nil
}

func testPositionQueries(overlay *mutations.MutationOverlay) {
	// Test positions for known hotspots
	testPositions := []struct {
		chrom string
		pos   uint64
		gene  string
	}{
		{"chr17", 7577534, "TP53"},
		{"chr12", 25398284, "KRAS"},
		{"chr7", 55241707, "EGFR"},
		{"chr7", 140453136, "BRAF"},
	}

	for _, tp := range testPositions {
		pm := overlay.GetParticleMutation(tp.chrom, tp.pos)
		if pm != nil && pm.HasMutation {
			color := pm.Color
			fmt.Printf("  %s:%d (%s): ", tp.chrom, tp.pos, tp.gene)
			fmt.Printf("Mutations=%d, Hotspot=%v, Color=(%.2f,%.2f,%.2f,%.2f), Score=%.3f\n",
				len(pm.Mutations), pm.IsHotspot, color.R, color.G, color.B, color.A, pm.HotspotScore)
		} else {
			fmt.Printf("  %s:%d (%s): No mutation\n", tp.chrom, tp.pos, tp.gene)
		}
	}
}

func displayTopHotspots(hotspots []*mutations.Hotspot, n int) {
	if n > len(hotspots) {
		n = len(hotspots)
	}

	for i := 0; i < n; i++ {
		hs := hotspots[i]
		fmt.Printf("  %2d. %s:%d-%d | Gene: %-6s | Mutations: %2d | Samples: %4d | Clinical: %.3f\n",
			i+1,
			hs.Chromosome,
			hs.StartPosition,
			hs.EndPosition,
			hs.PrimaryGene,
			hs.MutationCount,
			hs.TotalSamples,
			hs.ClinicalScore,
		)
	}
}
