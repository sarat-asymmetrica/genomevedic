package crispr

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// CHOPCHOPDesigner implements the CHOPCHOP algorithm for CRISPR guide design
type CHOPCHOPDesigner struct {
	enzyme CasEnzyme
	pam    PAMSequence
}

// NewCHOPCHOPDesigner creates a new CHOPCHOP designer instance
func NewCHOPCHOPDesigner(enzyme CasEnzyme) *CHOPCHOPDesigner {
	return &CHOPCHOPDesigner{
		enzyme: enzyme,
		pam:    GetPAMSequence(enzyme),
	}
}

// FindGuides finds all potential guide RNAs in a sequence
func (d *CHOPCHOPDesigner) FindGuides(sequence, chromosome string, startPos int) ([]GuideRNA, error) {
	sequence = strings.ToUpper(sequence)
	var guides []GuideRNA

	// Find PAM sites on both strands
	forwardGuides := d.findPAMSites(sequence, chromosome, startPos, "+")
	reverseGuides := d.findPAMSites(reverseComplement(sequence), chromosome, startPos, "-")

	guides = append(guides, forwardGuides...)
	guides = append(guides, reverseGuides...)

	return guides, nil
}

// findPAMSites finds PAM sites and extracts guide sequences
func (d *CHOPCHOPDesigner) findPAMSites(sequence, chromosome string, startPos int, strand string) []GuideRNA {
	var guides []GuideRNA

	// Convert PAM pattern to regex
	pamRegex := regexp.MustCompile(d.pam.Pattern)

	// Find all PAM sites
	matches := pamRegex.FindAllStringIndex(sequence, -1)

	for _, match := range matches {
		pamStart := match[0]
		pamEnd := match[1]
		pamSeq := sequence[pamStart:pamEnd]

		var guideSeq string
		var guideStart int

		// Extract guide sequence based on PAM orientation
		if d.pam.Orientation == "3prime" {
			// PAM is 3' to guide (e.g., Cas9: guide-NGG)
			guideStart = pamStart + d.pam.Offset
			if guideStart >= 0 && guideStart+d.pam.GuideLength <= len(sequence) {
				guideSeq = sequence[guideStart : guideStart+d.pam.GuideLength]
			}
		} else {
			// PAM is 5' to guide (e.g., Cas12a: TTTV-guide)
			guideStart = pamEnd + d.pam.Offset
			if guideStart >= 0 && guideStart+d.pam.GuideLength <= len(sequence) {
				guideSeq = sequence[guideStart : guideStart+d.pam.GuideLength]
			}
		}

		// Skip if guide sequence is invalid
		if guideSeq == "" || len(guideSeq) != d.pam.GuideLength {
			continue
		}

		// Skip if contains invalid characters
		if !isValidDNA(guideSeq) {
			continue
		}

		// Calculate genomic position
		genomicPos := startPos + guideStart
		if strand == "-" {
			// For reverse strand, calculate position from end
			genomicPos = startPos + (len(sequence) - guideStart - d.pam.GuideLength)
		}

		guide := GuideRNA{
			ID:          fmt.Sprintf("%s_%s_%d_%s", chromosome, strand, genomicPos, d.enzyme),
			Sequence:    guideSeq,
			Chromosome:  chromosome,
			Position:    genomicPos,
			Strand:      strand,
			PAMSequence: pamSeq,
			Enzyme:      d.enzyme,
			GCContent:   calculateGCContent(guideSeq),
			SelfCompScore: calculateSelfComplementarity(guideSeq),
			CreatedAt:   time.Now(),
		}

		guides = append(guides, guide)
	}

	return guides
}

// FilterGuides applies quality filters to guides
func (d *CHOPCHOPDesigner) FilterGuides(guides []GuideRNA, req DesignRequest) []GuideRNA {
	var filtered []GuideRNA

	// Set default parameters
	gcMin := req.GCMin
	if gcMin == 0 {
		gcMin = 40.0
	}
	gcMax := req.GCMax
	if gcMax == 0 {
		gcMax = 60.0
	}

	for _, guide := range guides {
		// GC content filter
		if guide.GCContent < gcMin || guide.GCContent > gcMax {
			continue
		}

		// Poly-T filter (problematic for U6 transcription termination)
		if req.ExcludePolyT && strings.Contains(guide.Sequence, "TTTT") {
			continue
		}

		// Avoid homopolymer runs (AAAA, GGGG, CCCC)
		if hasHomopolymerRun(guide.Sequence, 5) {
			continue
		}

		// Avoid high self-complementarity (can form hairpins)
		if guide.SelfCompScore > 10.0 {
			continue
		}

		filtered = append(filtered, guide)
	}

	return filtered
}

// calculateGCContent calculates GC percentage
func calculateGCContent(seq string) float64 {
	if len(seq) == 0 {
		return 0
	}

	gc := 0
	for _, base := range seq {
		if base == 'G' || base == 'C' {
			gc++
		}
	}

	return float64(gc) / float64(len(seq)) * 100.0
}

// calculateSelfComplementarity calculates self-complementarity score
// Higher score = more likely to form secondary structures
func calculateSelfComplementarity(seq string) float64 {
	score := 0.0
	revComp := reverseComplement(seq)

	// Simple sliding window approach
	windowSize := 4
	for i := 0; i <= len(seq)-windowSize; i++ {
		window := seq[i : i+windowSize]
		for j := 0; j <= len(revComp)-windowSize; j++ {
			if window == revComp[j:j+windowSize] {
				score += float64(windowSize)
			}
		}
	}

	return score
}

// reverseComplement returns the reverse complement of a DNA sequence
func reverseComplement(seq string) string {
	complement := map[rune]rune{
		'A': 'T',
		'T': 'A',
		'G': 'C',
		'C': 'G',
		'N': 'N',
	}

	runes := []rune(seq)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = complement[runes[j]], complement[runes[i]]
	}
	// Handle middle character if odd length
	if len(runes)%2 == 1 {
		mid := len(runes) / 2
		runes[mid] = complement[runes[mid]]
	}

	return string(runes)
}

// isValidDNA checks if sequence contains only valid DNA bases
func isValidDNA(seq string) bool {
	for _, base := range seq {
		if base != 'A' && base != 'T' && base != 'G' && base != 'C' {
			return false
		}
	}
	return true
}

// hasHomopolymerRun checks for homopolymer runs
func hasHomopolymerRun(seq string, minLength int) bool {
	if len(seq) < minLength {
		return false
	}

	count := 1
	for i := 1; i < len(seq); i++ {
		if seq[i] == seq[i-1] {
			count++
			if count >= minLength {
				return true
			}
		} else {
			count = 1
		}
	}

	return false
}

// OptimizeForRNA optimizes guide sequence for RNA transcription
func (d *CHOPCHOPDesigner) OptimizeForRNA(guides []GuideRNA) []GuideRNA {
	var optimized []GuideRNA

	for _, guide := range guides {
		// Skip guides starting with G (U6 promoter preference)
		if d.enzyme == Cas9 || d.enzyme == Cas9HF1 {
			if guide.Sequence[0] != 'G' {
				// Could add G to beginning for better transcription
				// but this changes the targeting, so we skip instead
				continue
			}
		}

		// Check RNA secondary structure prediction (simplified)
		// In production, would use RNAfold or similar
		if guide.SelfCompScore < 8.0 { // Lower is better
			optimized = append(optimized, guide)
		}
	}

	return optimized
}

// BatchDesign designs guides for multiple targets
func (d *CHOPCHOPDesigner) BatchDesign(requests []DesignRequest) ([]DesignResponse, error) {
	var responses []DesignResponse

	for _, req := range requests {
		// For each request, find and score guides
		// This would be implemented in the main designer
		// Placeholder for batch processing
		responses = append(responses, DesignResponse{
			Region: fmt.Sprintf("%s:%d-%d", req.Chromosome, req.Start, req.End),
		})
	}

	return responses, nil
}
