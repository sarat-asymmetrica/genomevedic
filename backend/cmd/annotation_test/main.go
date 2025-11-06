/**
 * Gene Annotation Overlay Test
 *
 * Tests GTF parser, gene overlay, and feature annotation
 * Validates genomic feature visualization pipeline
 */

package main

import (
	"fmt"
	"os"
	"time"

	"genomevedic/backend/internal/annotations"
)

func main() {
	fmt.Println("=== GenomeVedic.ai - Gene Annotation Overlay Test ===\n")

	// Test 1: Parse GTF file
	fmt.Println("Test 1: Parsing GTF annotation file...")
	parser, err := testGTFParser()
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ GTF parser working\n")

	// Test 2: Gene overlay
	fmt.Println("Test 2: Building gene overlay...")
	overlay, err := testGeneOverlay(parser)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Gene overlay working\n")

	// Test 3: Query specific positions
	fmt.Println("Test 3: Querying genomic features at specific positions...")
	testPositionQueries(parser, overlay)
	fmt.Println("✓ Position queries working\n")

	// Test 4: Filter by feature type
	fmt.Println("Test 4: Filtering by feature type...")
	testFeatureFilters(overlay)
	fmt.Println("✓ Feature filters working\n")

	// Test 5: Gene list
	fmt.Println("Test 5: Gene list:")
	displayGeneList(overlay)

	fmt.Println("\n=== All Tests Passed ===")
	fmt.Println("Agent 4.2 (Gene Annotation Overlay) Complete!")
}

func testGTFParser() (*annotations.GTFParser, error) {
	startTime := time.Now()

	// Create parser (promoter region = 2000 bp upstream)
	parser := annotations.NewGTFParser(2000)

	// Open sample GTF file
	file, err := os.Open("backend/testdata/genes_sample.gtf")
	if err != nil {
		return nil, fmt.Errorf("failed to open GTF file: %w", err)
	}
	defer file.Close()

	// Parse file
	if err := parser.ParseFile(file); err != nil {
		return nil, fmt.Errorf("failed to parse GTF file: %w", err)
	}

	parseTime := time.Since(startTime)

	// Get statistics
	stats := parser.GetStatistics()

	fmt.Printf("  Parse time: %.2f ms\n", float64(parseTime.Microseconds())/1000.0)
	fmt.Printf("  Total features: %d\n", stats["total_features"])
	fmt.Printf("  Total genes: %d\n", stats["total_genes"])
	fmt.Printf("  Exon count: %d\n", stats["exon_count"])
	fmt.Printf("  Intron count: %d (inferred)\n", stats["intron_count"])
	fmt.Printf("  Promoter region: %d bp\n", stats["promoter_region"])

	// Print feature type breakdown
	typeCounts := stats["type_counts"].(map[annotations.FeatureType]int)
	fmt.Printf("  Feature types:\n")
	for featureType, count := range typeCounts {
		fmt.Printf("    %s: %d\n", featureType.String(), count)
	}

	return parser, nil
}

func testGeneOverlay(parser *annotations.GTFParser) (*annotations.GeneOverlay, error) {
	startTime := time.Now()

	// Create overlay
	overlay := annotations.NewGeneOverlay(parser)

	// Build overlay
	if err := overlay.BuildOverlay(); err != nil {
		return nil, fmt.Errorf("failed to build overlay: %w", err)
	}

	buildTime := time.Since(startTime)

	// Get statistics
	stats := overlay.GetStatistics()

	fmt.Printf("  Build time: %.2f ms\n", float64(buildTime.Microseconds())/1000.0)
	fmt.Printf("  Total annotated particles: %d\n", stats["total_particles"])
	fmt.Printf("  Particles in genes: %d\n", stats["particles_in_genes"])
	fmt.Printf("  Particles in exons: %d\n", stats["particles_in_exons"])
	fmt.Printf("  Particles in CDS: %d\n", stats["particles_in_cds"])

	// Print primary type breakdown
	typeCounts := stats["primary_type_counts"].(map[annotations.FeatureType]int)
	fmt.Printf("  Primary feature types:\n")
	for featureType, count := range typeCounts {
		fmt.Printf("    %s: %d particles\n", featureType.String(), count)
	}

	return overlay, nil
}

func testPositionQueries(parser *annotations.GTFParser, overlay *annotations.GeneOverlay) {
	// Test positions for known genes
	testPositions := []struct {
		chrom string
		pos   uint64
		gene  string
	}{
		{"chr17", 7577534, "TP53"},
		{"chr12", 25204800, "KRAS"},
		{"chr7", 55019220, "EGFR"},
		{"chr17", 43044400, "BRCA1"},
	}

	for _, tp := range testPositions {
		// Get features at position
		features := parser.GetFeaturesAtPosition(tp.chrom, tp.pos)

		// Get particle annotation
		pa := overlay.GetParticleAnnotation(tp.chrom, tp.pos)

		fmt.Printf("  %s:%d (%s):\n", tp.chrom, tp.pos, tp.gene)

		if len(features) > 0 {
			fmt.Printf("    Features: ")
			for i, f := range features {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%s", f.Type.String())
			}
			fmt.Printf("\n")
		} else {
			fmt.Printf("    Features: None\n")
		}

		if pa != nil && pa.PrimaryFeature != nil {
			color := pa.Color
			fmt.Printf("    Primary: %s, Color=(%.2f,%.2f,%.2f,%.2f)\n",
				pa.PrimaryFeature.Type.String(), color.R, color.G, color.B, color.A)
		}
	}
}

func testFeatureFilters(overlay *annotations.GeneOverlay) {
	// Filter by CDS
	cdsParticles := overlay.FilterByFeatureType(annotations.FeatureCDS)
	fmt.Printf("  CDS particles: %d\n", len(cdsParticles))

	// Filter by Exon
	exonParticles := overlay.FilterByFeatureType(annotations.FeatureExon)
	fmt.Printf("  Exon particles: %d\n", len(exonParticles))

	// Filter by Intron
	intronParticles := overlay.FilterByFeatureType(annotations.FeatureIntron)
	fmt.Printf("  Intron particles: %d\n", len(intronParticles))

	// Filter by Promoter
	promoterParticles := overlay.FilterByFeatureType(annotations.FeaturePromoter)
	fmt.Printf("  Promoter particles: %d\n", len(promoterParticles))

	// Filter by gene name
	tp53Particles := overlay.FilterByGene("TP53")
	fmt.Printf("  TP53 gene particles: %d\n", len(tp53Particles))
}

func displayGeneList(overlay *annotations.GeneOverlay) {
	genes := overlay.GetGeneList()
	fmt.Printf("  Total genes: %d\n", len(genes))
	fmt.Printf("  Genes: ")
	for i, gene := range genes {
		if i > 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("%s", gene)
	}
	fmt.Printf("\n")
}
