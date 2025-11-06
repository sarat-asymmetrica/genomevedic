/**
 * COSMIC Mutation Database Parser
 *
 * Parses COSMIC mutation database (Cancer Genome Project)
 * Maps mutations to genomic positions with clinical significance
 *
 * Data Format: VCF-like simplified format
 * Columns: Chromosome, Position, Ref, Alt, MutationType, Gene, Samples, Significance
 */

package mutations

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// MutationType represents the type of genomic mutation
type MutationType int

const (
	MutationUnknown MutationType = iota
	MutationMissense
	MutationNonsense
	MutationFrameshift
	MutationSplice
	MutationInframe
	MutationSynonymous
)

func (mt MutationType) String() string {
	switch mt {
	case MutationMissense:
		return "Missense"
	case MutationNonsense:
		return "Nonsense"
	case MutationFrameshift:
		return "Frameshift"
	case MutationSplice:
		return "Splice"
	case MutationInframe:
		return "Inframe"
	case MutationSynonymous:
		return "Synonymous"
	default:
		return "Unknown"
	}
}

// Significance represents the clinical significance of a mutation
type Significance int

const (
	SignificanceUnknown Significance = iota
	SignificanceBenign
	SignificanceLikelyBenign
	SignificanceUncertain
	SignificanceLikelyPathogenic
	SignificancePathogenic
)

func (s Significance) String() string {
	switch s {
	case SignificanceBenign:
		return "Benign"
	case SignificanceLikelyBenign:
		return "Likely Benign"
	case SignificanceUncertain:
		return "Uncertain"
	case SignificanceLikelyPathogenic:
		return "Likely Pathogenic"
	case SignificancePathogenic:
		return "Pathogenic"
	default:
		return "Unknown"
	}
}

// Mutation represents a genomic mutation from COSMIC
type Mutation struct {
	Chromosome   string
	Position     uint64
	RefAllele    string
	AltAllele    string
	MutationType MutationType
	Gene         string
	SampleCount  int
	Significance Significance
	Frequency    float64 // Frequency in population (0.0-1.0)
}

// COSMICParser parses COSMIC mutation database files
type COSMICParser struct {
	mutations      []*Mutation
	mutationByPos  map[string][]*Mutation // Key: "chr:position"
	hotspots       []*Mutation
	hotspotThresh  int // Minimum sample count for hotspot
}

// NewCOSMICParser creates a new COSMIC parser
func NewCOSMICParser(hotspotThreshold int) *COSMICParser {
	return &COSMICParser{
		mutations:     make([]*Mutation, 0, 100000),
		mutationByPos: make(map[string][]*Mutation),
		hotspots:      make([]*Mutation, 0, 1000),
		hotspotThresh: hotspotThreshold,
	}
}

// ParseFile parses a COSMIC database file
func (cp *COSMICParser) ParseFile(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		mutation, err := cp.parseLine(line, lineNum)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNum, err)
		}

		if mutation != nil {
			cp.addMutation(mutation)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	cp.identifyHotspots()
	return nil
}

// parseLine parses a single line from COSMIC database
// Format: Chromosome\tPosition\tRef\tAlt\tType\tGene\tSamples\tSignificance\tFrequency
func (cp *COSMICParser) parseLine(line string, lineNum int) (*Mutation, error) {
	fields := strings.Split(line, "\t")
	if len(fields) < 8 {
		return nil, fmt.Errorf("expected at least 8 fields, got %d", len(fields))
	}

	// Parse position
	position, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid position: %w", err)
	}

	// Parse mutation type
	mutationType := parseMutationType(fields[4])

	// Parse sample count
	sampleCount, err := strconv.Atoi(fields[6])
	if err != nil {
		return nil, fmt.Errorf("invalid sample count: %w", err)
	}

	// Parse significance
	significance := parseSignificance(fields[7])

	// Parse frequency (if available)
	frequency := 0.0
	if len(fields) >= 9 && fields[8] != "" {
		frequency, err = strconv.ParseFloat(fields[8], 64)
		if err != nil {
			frequency = 0.0
		}
	}

	return &Mutation{
		Chromosome:   fields[0],
		Position:     position,
		RefAllele:    fields[2],
		AltAllele:    fields[3],
		MutationType: mutationType,
		Gene:         fields[5],
		SampleCount:  sampleCount,
		Significance: significance,
		Frequency:    frequency,
	}, nil
}

// addMutation adds a mutation to the parser
func (cp *COSMICParser) addMutation(mut *Mutation) {
	cp.mutations = append(cp.mutations, mut)

	// Add to position index
	key := fmt.Sprintf("%s:%d", mut.Chromosome, mut.Position)
	cp.mutationByPos[key] = append(cp.mutationByPos[key], mut)
}

// identifyHotspots identifies mutation hotspots (high sample count)
func (cp *COSMICParser) identifyHotspots() {
	cp.hotspots = make([]*Mutation, 0, 1000)

	for _, mut := range cp.mutations {
		// Hotspot criteria:
		// 1. Sample count >= threshold
		// 2. Pathogenic or likely pathogenic
		if mut.SampleCount >= cp.hotspotThresh &&
			(mut.Significance == SignificancePathogenic ||
				mut.Significance == SignificanceLikelyPathogenic) {
			cp.hotspots = append(cp.hotspots, mut)
		}
	}
}

// GetMutations returns all parsed mutations
func (cp *COSMICParser) GetMutations() []*Mutation {
	return cp.mutations
}

// GetHotspots returns mutation hotspots
func (cp *COSMICParser) GetHotspots() []*Mutation {
	return cp.hotspots
}

// GetMutationsAtPosition returns mutations at a specific position
func (cp *COSMICParser) GetMutationsAtPosition(chromosome string, position uint64) []*Mutation {
	key := fmt.Sprintf("%s:%d", chromosome, position)
	return cp.mutationByPos[key]
}

// GetStatistics returns mutation statistics
func (cp *COSMICParser) GetStatistics() map[string]interface{} {
	// Count by type
	typeCounts := make(map[MutationType]int)
	for _, mut := range cp.mutations {
		typeCounts[mut.MutationType]++
	}

	// Count by significance
	sigCounts := make(map[Significance]int)
	for _, mut := range cp.mutations {
		sigCounts[mut.Significance]++
	}

	// Average sample count
	totalSamples := 0
	for _, mut := range cp.mutations {
		totalSamples += mut.SampleCount
	}
	avgSamples := 0.0
	if len(cp.mutations) > 0 {
		avgSamples = float64(totalSamples) / float64(len(cp.mutations))
	}

	return map[string]interface{}{
		"total_mutations":     len(cp.mutations),
		"total_hotspots":      len(cp.hotspots),
		"hotspot_threshold":   cp.hotspotThresh,
		"type_counts":         typeCounts,
		"significance_counts": sigCounts,
		"avg_sample_count":    avgSamples,
	}
}

// parseMutationType parses mutation type string
func parseMutationType(typeStr string) MutationType {
	typeStr = strings.ToLower(strings.TrimSpace(typeStr))
	switch typeStr {
	case "missense", "missense_variant":
		return MutationMissense
	case "nonsense", "stop_gained":
		return MutationNonsense
	case "frameshift", "frameshift_variant":
		return MutationFrameshift
	case "splice", "splice_site":
		return MutationSplice
	case "inframe", "inframe_insertion", "inframe_deletion":
		return MutationInframe
	case "synonymous", "synonymous_variant":
		return MutationSynonymous
	default:
		return MutationUnknown
	}
}

// parseSignificance parses clinical significance string
func parseSignificance(sigStr string) Significance {
	sigStr = strings.ToLower(strings.TrimSpace(sigStr))
	switch sigStr {
	case "benign":
		return SignificanceBenign
	case "likely_benign", "likely benign":
		return SignificanceLikelyBenign
	case "uncertain", "uncertain_significance", "vus":
		return SignificanceUncertain
	case "likely_pathogenic", "likely pathogenic":
		return SignificanceLikelyPathogenic
	case "pathogenic":
		return SignificancePathogenic
	default:
		return SignificanceUnknown
	}
}
