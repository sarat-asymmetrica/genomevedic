/**
 * GTF/GFF3 Parser for Gene Annotations
 *
 * Parses Gene Transfer Format (GTF) and General Feature Format (GFF3)
 * Standard formats for genomic annotations (Ensembl, GENCODE, RefSeq)
 *
 * GTF Format (9 tab-separated columns):
 * 1. seqname   - Chromosome/contig
 * 2. source    - Database source (e.g., GENCODE, Ensembl)
 * 3. feature   - Feature type (gene, exon, CDS, etc.)
 * 4. start     - Start position (1-based)
 * 5. end       - End position (inclusive)
 * 6. score     - Confidence score (or ".")
 * 7. strand    - "+" or "-"
 * 8. frame     - Codon frame (0, 1, 2, or ".")
 * 9. attributes - Semicolon-separated key-value pairs
 */

package annotations

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// GTFParser parses GTF/GFF3 annotation files
type GTFParser struct {
	features       []*GenomicFeature
	featuresByPos  map[string][]*GenomicFeature // Key: "chr:position"
	genesByName    map[string][]*GenomicFeature
	genesByID      map[string]*GenomicFeature
	exonCount      int
	intronCount    int
	promoterRegion int // Base pairs upstream of gene for promoter
}

// NewGTFParser creates a new GTF parser
func NewGTFParser(promoterRegion int) *GTFParser {
	return &GTFParser{
		features:       make([]*GenomicFeature, 0, 100000),
		featuresByPos:  make(map[string][]*GenomicFeature),
		genesByName:    make(map[string][]*GenomicFeature),
		genesByID:      make(map[string]*GenomicFeature),
		promoterRegion: promoterRegion,
	}
}

// ParseFile parses a GTF/GFF3 file
func (gp *GTFParser) ParseFile(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		feature, err := gp.parseLine(line, lineNum)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNum, err)
		}

		if feature != nil {
			gp.addFeature(feature)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	// Post-processing: infer introns and promoters
	gp.inferIntrons()
	gp.inferPromoters()

	return nil
}

// parseLine parses a single GTF/GFF3 line
func (gp *GTFParser) parseLine(line string, lineNum int) (*GenomicFeature, error) {
	fields := strings.Split(line, "\t")
	if len(fields) < 9 {
		return nil, fmt.Errorf("expected 9 fields, got %d", len(fields))
	}

	// Parse chromosome
	chromosome := fields[0]

	// Parse feature type
	featureType := ParseFeatureType(fields[2])
	if featureType == FeatureUnknown {
		// Skip unknown feature types
		return nil, nil
	}

	// Parse start and end positions (GTF is 1-based, convert to 0-based)
	start, err := strconv.ParseUint(fields[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid start position: %w", err)
	}
	start-- // Convert to 0-based

	end, err := strconv.ParseUint(fields[4], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid end position: %w", err)
	}
	end-- // Convert to 0-based

	// Parse strand
	strand := fields[6]

	// Create feature
	feature := NewGenomicFeature(featureType, chromosome, start, end, strand)

	// Parse attributes (semicolon-separated key-value pairs)
	attributes := parseAttributes(fields[8])
	feature.Attributes = attributes

	// Extract common attributes
	if geneID, ok := attributes["gene_id"]; ok {
		feature.GeneID = geneID
	}
	if geneName, ok := attributes["gene_name"]; ok {
		feature.GeneName = geneName
	}
	if transcriptID, ok := attributes["transcript_id"]; ok {
		feature.TranscriptID = transcriptID
	}

	return feature, nil
}

// parseAttributes parses GTF attribute string
func parseAttributes(attrStr string) map[string]string {
	attrs := make(map[string]string)

	// Split by semicolons
	pairs := strings.Split(attrStr, ";")

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// Split by space or "="
		var key, value string
		if strings.Contains(pair, "=") {
			// GFF3 style: key=value
			parts := strings.SplitN(pair, "=", 2)
			key = strings.TrimSpace(parts[0])
			value = strings.Trim(strings.TrimSpace(parts[1]), "\"")
		} else {
			// GTF style: key "value"
			parts := strings.SplitN(pair, " ", 2)
			if len(parts) >= 2 {
				key = strings.TrimSpace(parts[0])
				value = strings.Trim(strings.TrimSpace(parts[1]), "\"")
			}
		}

		if key != "" {
			attrs[key] = value
		}
	}

	return attrs
}

// addFeature adds a feature to the parser
func (gp *GTFParser) addFeature(feature *GenomicFeature) {
	gp.features = append(gp.features, feature)

	// Add to gene index
	if feature.Type == FeatureGene {
		if feature.GeneID != "" {
			gp.genesByID[feature.GeneID] = feature
		}
		if feature.GeneName != "" {
			gp.genesByName[feature.GeneName] = append(gp.genesByName[feature.GeneName], feature)
		}
	}

	// Count exons
	if feature.Type == FeatureExon {
		gp.exonCount++
	}

	// Index by position (every 1000 bp)
	for pos := feature.Start; pos <= feature.End; pos += 1000 {
		key := fmt.Sprintf("%s:%d", feature.Chromosome, pos)
		gp.featuresByPos[key] = append(gp.featuresByPos[key], feature)
	}
}

// inferIntrons infers intron positions from exons
func (gp *GTFParser) inferIntrons() {
	// Group exons by transcript
	exonsByTranscript := make(map[string][]*GenomicFeature)

	for _, feature := range gp.features {
		if feature.Type == FeatureExon && feature.TranscriptID != "" {
			exonsByTranscript[feature.TranscriptID] = append(exonsByTranscript[feature.TranscriptID], feature)
		}
	}

	// For each transcript, find gaps between exons (introns)
	for transcriptID, exons := range exonsByTranscript {
		if len(exons) < 2 {
			continue // Need at least 2 exons for an intron
		}

		// Sort exons by start position
		// (Simple bubble sort for small arrays)
		for i := 0; i < len(exons); i++ {
			for j := i + 1; j < len(exons); j++ {
				if exons[j].Start < exons[i].Start {
					exons[i], exons[j] = exons[j], exons[i]
				}
			}
		}

		// Find introns (gaps between consecutive exons)
		for i := 0; i < len(exons)-1; i++ {
			intronStart := exons[i].End + 1
			intronEnd := exons[i+1].Start - 1

			if intronEnd > intronStart {
				intron := NewGenomicFeature(
					FeatureIntron,
					exons[i].Chromosome,
					intronStart,
					intronEnd,
					exons[i].Strand,
				)
				intron.GeneID = exons[i].GeneID
				intron.GeneName = exons[i].GeneName
				intron.TranscriptID = transcriptID
				gp.addFeature(intron)
				gp.intronCount++
			}
		}
	}
}

// inferPromoters infers promoter regions (upstream of genes)
func (gp *GTFParser) inferPromoters() {
	for _, feature := range gp.features {
		if feature.Type != FeatureGene {
			continue
		}

		var promoterStart, promoterEnd uint64

		if feature.Strand == "+" {
			// Positive strand: promoter is upstream (lower coordinates)
			if feature.Start >= uint64(gp.promoterRegion) {
				promoterStart = feature.Start - uint64(gp.promoterRegion)
			} else {
				promoterStart = 0
			}
			promoterEnd = feature.Start - 1
		} else {
			// Negative strand: promoter is downstream (higher coordinates)
			promoterStart = feature.End + 1
			promoterEnd = feature.End + uint64(gp.promoterRegion)
		}

		promoter := NewGenomicFeature(
			FeaturePromoter,
			feature.Chromosome,
			promoterStart,
			promoterEnd,
			feature.Strand,
		)
		promoter.GeneID = feature.GeneID
		promoter.GeneName = feature.GeneName

		gp.addFeature(promoter)
	}
}

// GetFeatures returns all parsed features
func (gp *GTFParser) GetFeatures() []*GenomicFeature {
	return gp.features
}

// GetFeaturesAtPosition returns features overlapping a specific position
func (gp *GTFParser) GetFeaturesAtPosition(chromosome string, position uint64) []*GenomicFeature {
	// Check position and nearby positions (within 1000 bp)
	results := make([]*GenomicFeature, 0, 10)
	seen := make(map[*GenomicFeature]bool)

	for offset := uint64(0); offset <= 1000; offset += 1000 {
		key := fmt.Sprintf("%s:%d", chromosome, position-position%1000+offset)
		features := gp.featuresByPos[key]

		for _, feature := range features {
			if !seen[feature] && feature.Contains(position) {
				results = append(results, feature)
				seen[feature] = true
			}
		}
	}

	return results
}

// GetGeneByName returns a gene by name
func (gp *GTFParser) GetGeneByName(name string) []*GenomicFeature {
	return gp.genesByName[name]
}

// GetStatistics returns parser statistics
func (gp *GTFParser) GetStatistics() map[string]interface{} {
	// Count by type
	typeCounts := make(map[FeatureType]int)
	for _, feature := range gp.features {
		typeCounts[feature.Type]++
	}

	return map[string]interface{}{
		"total_features":  len(gp.features),
		"total_genes":     len(gp.genesByID),
		"exon_count":      gp.exonCount,
		"intron_count":    gp.intronCount,
		"promoter_region": gp.promoterRegion,
		"type_counts":     typeCounts,
	}
}
