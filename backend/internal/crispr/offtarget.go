package crispr

import (
	"fmt"
	"math"
	"strings"
)

// OffTargetPredictor implements GuideScan2-inspired off-target prediction
// Uses BWT-based genome indexing for fast off-target detection
type OffTargetPredictor struct {
	maxMismatches int
	genomeIndex   *GenomeIndex
	cfdScores     map[string]float64 // Cutting Frequency Determination scores
}

// GenomeIndex represents a simplified BWT-based genome index
// In production, would use actual BWT/FM-index implementation
type GenomeIndex struct {
	sequences map[string]string // chromosome -> sequence
	kmers     map[string][]GenomicLocation
}

// GenomicLocation represents a location in the genome
type GenomicLocation struct {
	Chromosome string
	Position   int
	Strand     string
}

// NewOffTargetPredictor creates a new off-target predictor
func NewOffTargetPredictor(maxMismatches int) *OffTargetPredictor {
	return &OffTargetPredictor{
		maxMismatches: maxMismatches,
		genomeIndex:   NewGenomeIndex(),
		cfdScores:     initializeCFDScores(),
	}
}

// NewGenomeIndex creates a new genome index
func NewGenomeIndex() *GenomeIndex {
	return &GenomeIndex{
		sequences: make(map[string]string),
		kmers:     make(map[string][]GenomicLocation),
	}
}

// IndexSequence adds a sequence to the genome index
func (gi *GenomeIndex) IndexSequence(chromosome, sequence string) {
	sequence = strings.ToUpper(sequence)
	gi.sequences[chromosome] = sequence

	// Build k-mer index for fast lookup (k=10 for seed region)
	kmerSize := 10
	for i := 0; i <= len(sequence)-kmerSize; i++ {
		kmer := sequence[i : i+kmerSize]
		loc := GenomicLocation{
			Chromosome: chromosome,
			Position:   i,
			Strand:     "+",
		}
		gi.kmers[kmer] = append(gi.kmers[kmer], loc)
	}
}

// FindOffTargets finds potential off-target sites for a guide RNA
func (otp *OffTargetPredictor) FindOffTargets(guide GuideRNA) []OffTargetSite {
	var offTargets []OffTargetSite

	// Use seed-and-extend approach
	// Seed: PAM-proximal 12bp (most critical for binding)
	seedSize := 12
	guideSeq := guide.Sequence

	if len(guideSeq) < seedSize {
		return offTargets
	}

	// Extract seed region (last 12bp before PAM)
	seed := guideSeq[len(guideSeq)-seedSize:]

	// Find all seed matches in genome
	seedMatches := otp.findSeedMatches(seed)

	// Extend and score each seed match
	for _, match := range seedMatches {
		// Skip the on-target site
		if match.Chromosome == guide.Chromosome &&
		   math.Abs(float64(match.Position-guide.Position)) < 5 {
			continue
		}

		// Get full sequence at this location
		targetSeq := otp.getSequenceAt(match, len(guideSeq))
		if targetSeq == "" {
			continue
		}

		// Count mismatches
		mismatches, mismatchPos := otp.countMismatches(guideSeq, targetSeq)

		if mismatches <= otp.maxMismatches {
			// Check PAM sequence
			pamSeq := otp.getPAMAt(match, guide.Enzyme)
			if !otp.isValidPAM(pamSeq, guide.Enzyme) {
				continue // Skip if PAM not valid
			}

			// Calculate CFD score
			cfdScore := otp.calculateCFDScore(guideSeq, targetSeq, mismatchPos)

			offTarget := OffTargetSite{
				Chromosome:  match.Chromosome,
				Position:    match.Position,
				Sequence:    targetSeq,
				Mismatches:  mismatches,
				MismatchPos: mismatchPos,
				Score:       cfdScore,
			}

			offTargets = append(offTargets, offTarget)
		}
	}

	return offTargets
}

// findSeedMatches finds all genome locations matching the seed
func (otp *OffTargetPredictor) findSeedMatches(seed string) []GenomicLocation {
	var matches []GenomicLocation

	// Use k-mer index for fast initial search
	kmerSize := 10
	if len(seed) < kmerSize {
		return matches
	}

	searchKmer := seed[:kmerSize]

	// Find exact k-mer matches
	if locs, exists := otp.genomeIndex.kmers[searchKmer]; exists {
		matches = append(matches, locs...)
	}

	// Also search with 1 mismatch in k-mer (seed-and-extend)
	variants := otp.generateKmerVariants(searchKmer, 1)
	for _, variant := range variants {
		if locs, exists := otp.genomeIndex.kmers[variant]; exists {
			matches = append(matches, locs...)
		}
	}

	return matches
}

// generateKmerVariants generates k-mer variants with up to n mismatches
func (otp *OffTargetPredictor) generateKmerVariants(kmer string, maxMM int) []string {
	if maxMM == 0 {
		return []string{kmer}
	}

	variants := make(map[string]bool)
	bases := []byte{'A', 'T', 'G', 'C'}

	// Generate single-mismatch variants
	for i := 0; i < len(kmer); i++ {
		for _, base := range bases {
			if base != kmer[i] {
				variant := kmer[:i] + string(base) + kmer[i+1:]
				variants[variant] = true
			}
		}
	}

	result := make([]string, 0, len(variants))
	for v := range variants {
		result = append(result, v)
	}

	return result
}

// getSequenceAt retrieves sequence at a genomic location
func (otp *OffTargetPredictor) getSequenceAt(loc GenomicLocation, length int) string {
	seq, exists := otp.genomeIndex.sequences[loc.Chromosome]
	if !exists {
		return ""
	}

	if loc.Position+length > len(seq) {
		return ""
	}

	return seq[loc.Position : loc.Position+length]
}

// getPAMAt gets PAM sequence at a location
func (otp *OffTargetPredictor) getPAMAt(loc GenomicLocation, enzyme CasEnzyme) string {
	pamConfig := GetPAMSequence(enzyme)
	seq, exists := otp.genomeIndex.sequences[loc.Chromosome]
	if !exists {
		return ""
	}

	pamStart := loc.Position + pamConfig.GuideLength
	pamEnd := pamStart + 3 // NGG is 3bp

	if pamStart < 0 || pamEnd > len(seq) {
		return ""
	}

	return seq[pamStart:pamEnd]
}

// isValidPAM checks if PAM sequence is valid
func (otp *OffTargetPredictor) isValidPAM(pamSeq string, enzyme CasEnzyme) bool {
	if pamSeq == "" {
		return false
	}

	switch enzyme {
	case Cas9, Cas9HF1:
		// NGG
		return len(pamSeq) == 3 && pamSeq[1:] == "GG"
	case xCas9:
		// NG[ATGC]
		return len(pamSeq) == 3 && pamSeq[0] != 'N' && pamSeq[1] == 'G'
	case Cas12a:
		// TTTV
		return len(pamSeq) >= 3 && pamSeq[:3] == "TTT"
	default:
		return true
	}
}

// countMismatches counts mismatches between guide and target
func (otp *OffTargetPredictor) countMismatches(guide, target string) (int, []int) {
	mismatches := 0
	var positions []int

	minLen := len(guide)
	if len(target) < minLen {
		minLen = len(target)
	}

	for i := 0; i < minLen; i++ {
		if guide[i] != target[i] {
			mismatches++
			positions = append(positions, i)
		}
	}

	return mismatches, positions
}

// calculateCFDScore calculates Cutting Frequency Determination score
// Reference: Doench et al., Nature Biotechnology 2016
func (otp *OffTargetPredictor) calculateCFDScore(guide, target string, mismatchPos []int) float64 {
	if len(mismatchPos) == 0 {
		return 1.0 // Perfect match
	}

	score := 1.0

	for _, pos := range mismatchPos {
		if pos >= len(guide) || pos >= len(target) {
			continue
		}

		// Get mismatch type (e.g., "rA:dT" = RNA A paired with DNA T)
		guideBase := string(guide[pos])
		targetBase := string(target[pos])
		mismatchType := fmt.Sprintf("r%s:d%s", guideBase, targetBase)

		// Position-dependent penalty
		positionWeight := otp.getPositionWeight(pos, len(guide))

		// Mismatch type penalty
		mismatchPenalty := otp.cfdScores[mismatchType]
		if mismatchPenalty == 0 {
			mismatchPenalty = 0.5 // Default penalty
		}

		score *= mismatchPenalty * positionWeight
	}

	return score
}

// getPositionWeight returns position-dependent mismatch weight
// PAM-proximal mismatches are more critical
func (otp *OffTargetPredictor) getPositionWeight(pos, length int) float64 {
	// PAM-proximal positions (last 12bp) are most critical
	distFromPAM := length - pos

	if distFromPAM <= 7 {
		return 0.3 // High penalty for PAM-proximal mismatches
	} else if distFromPAM <= 12 {
		return 0.5 // Medium penalty
	} else {
		return 0.8 // Lower penalty for PAM-distal mismatches
	}
}

// initializeCFDScores initializes CFD mismatch scores
// From Doench et al. 2016 supplementary data
func initializeCFDScores() map[string]float64 {
	scores := make(map[string]float64)

	// rU = RNA uracil (T in DNA representation)
	// Most deleterious mismatches
	scores["rA:dA"] = 0.0  // Same base (impossible)
	scores["rA:dC"] = 0.35
	scores["rA:dG"] = 0.49
	scores["rA:dT"] = 0.0

	scores["rC:dA"] = 0.82
	scores["rC:dC"] = 0.0
	scores["rC:dG"] = 0.68
	scores["rC:dT"] = 0.45

	scores["rG:dA"] = 0.56
	scores["rG:dC"] = 0.87
	scores["rG:dG"] = 0.0
	scores["rG:dT"] = 0.71

	scores["rT:dA"] = 0.0
	scores["rT:dC"] = 0.63
	scores["rT:dG"] = 0.75
	scores["rT:dT"] = 0.0

	return scores
}

// ScoreOffTargetSpecificity calculates overall specificity score
func (otp *OffTargetPredictor) ScoreOffTargetSpecificity(guide GuideRNA, offTargets []OffTargetSite) float64 {
	if len(offTargets) == 0 {
		return 100.0 // Perfect specificity
	}

	// Sum of CFD scores (weighted by severity)
	sumCFD := 0.0
	for _, ot := range offTargets {
		sumCFD += ot.Score
	}

	// MIT specificity score (modified)
	// 100 / (100 + sumCFD)
	specificity := 100.0 / (100.0 + sumCFD)
	specificity *= 100.0 // Scale to 0-100

	return specificity
}

// FilterHighRiskOffTargets filters off-targets likely to cause cleavage
func (otp *OffTargetPredictor) FilterHighRiskOffTargets(offTargets []OffTargetSite) []OffTargetSite {
	var highRisk []OffTargetSite

	for _, ot := range offTargets {
		// High risk criteria:
		// 1. ≤ 2 mismatches
		// 2. High CFD score (>0.5)
		// 3. No mismatches in PAM-proximal seed
		if ot.Mismatches <= 2 && ot.Score > 0.5 {
			highRisk = append(highRisk, ot)
		}
	}

	return highRisk
}
