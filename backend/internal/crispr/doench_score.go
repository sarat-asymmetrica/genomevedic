package crispr

import (
	"math"
	"strings"
)

// DoenchScorer implements the Doench 2016 (Rule Set 2) on-target efficiency scoring
// Reference: Doench et al., Nature Biotechnology 2016
// Also known as "Azimuth" score
type DoenchScorer struct {
	// Pre-trained model weights (simplified version)
	// In production, would load full gradient boosting model
	weights map[string]float64
}

// NewDoenchScorer creates a new Doench scorer instance
func NewDoenchScorer() *DoenchScorer {
	return &DoenchScorer{
		weights: initializeDoenchWeights(),
	}
}

// Score calculates the Doench 2016 on-target efficiency score
// Input: 30bp context (4bp upstream + 20bp guide + 3bp PAM + 3bp downstream)
// Output: Score from 0-1 (higher = better predicted efficiency)
func (ds *DoenchScorer) Score(guide GuideRNA, context string) float64 {
	// Ensure we have 30bp context
	context = strings.ToUpper(context)
	if len(context) < 30 {
		// Use simplified scoring if context not available
		return ds.scoreSimplified(guide.Sequence)
	}

	// Extract features
	features := ds.extractFeatures(context)

	// Calculate weighted sum
	score := ds.predictScore(features)

	// Normalize to 0-1 range
	score = 1.0 / (1.0 + math.Exp(-score)) // Sigmoid

	return score
}

// scoreSimplified uses simplified scoring when full context not available
func (ds *DoenchScorer) scoreSimplified(guideSeq string) float64 {
	if len(guideSeq) != 20 {
		return 0.0
	}

	score := 0.5 // Base score

	// GC content penalty (optimal around 50%)
	gc := calculateGCContent(guideSeq)
	gcPenalty := math.Abs(gc-50.0) / 50.0
	score -= gcPenalty * 0.2

	// Position-specific nucleotide preferences (from Doench 2014/2016)
	// Position 1-5 (important for efficiency)
	if guideSeq[0] == 'G' {
		score += 0.05 // Prefer G at position 1
	}
	if guideSeq[19] == 'G' || guideSeq[19] == 'C' {
		score -= 0.05 // Penalize G/C at position 20
	}

	// Middle region (positions 10-15) - prefer purines
	middleScore := 0.0
	for i := 9; i < 15; i++ {
		if guideSeq[i] == 'A' || guideSeq[i] == 'G' {
			middleScore += 0.01
		}
	}
	score += middleScore

	// PAM-proximal region (positions 16-20) - critical for binding
	pamProxScore := 0.0
	for i := 15; i < 20; i++ {
		if guideSeq[i] == 'T' {
			pamProxScore += 0.015
		}
	}
	score += pamProxScore

	// Normalize to 0-1
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// extractFeatures extracts sequence features from 30bp context
func (ds *DoenchScorer) extractFeatures(context string) map[string]float64 {
	features := make(map[string]float64)

	// Position-specific nucleotide features
	for i := 0; i < 30; i++ {
		base := string(context[i])
		features[fmt.Sprintf("pos%d_%s", i, base)] = 1.0
	}

	// Dinucleotide features
	for i := 0; i < 29; i++ {
		dinuc := context[i : i+2]
		features[fmt.Sprintf("dinuc_%d_%s", i, dinuc)] = 1.0
	}

	// GC content in different regions
	features["gc_content_full"] = calculateGCContent(context)
	features["gc_content_guide"] = calculateGCContent(context[4:24]) // Guide region
	features["gc_content_pam_proximal"] = calculateGCContent(context[19:24])

	// Thermodynamic features (simplified)
	features["tm_estimate"] = ds.estimateTm(context[4:24])

	return features
}

// predictScore calculates weighted score from features
func (ds *DoenchScorer) predictScore(features map[string]float64) float64 {
	score := 0.0

	// Intercept
	score += 0.5

	// Position-specific contributions
	guideStart := 4
	for i := 0; i < 20; i++ {
		pos := guideStart + i
		if pos < 30 {
			base := features[fmt.Sprintf("pos%d_A", pos)]
			if base > 0 {
				score += ds.getPositionWeight(i, 'A')
			}
			base = features[fmt.Sprintf("pos%d_T", pos)]
			if base > 0 {
				score += ds.getPositionWeight(i, 'T')
			}
			base = features[fmt.Sprintf("pos%d_G", pos)]
			if base > 0 {
				score += ds.getPositionWeight(i, 'G')
			}
			base = features[fmt.Sprintf("pos%d_C", pos)]
			if base > 0 {
				score += ds.getPositionWeight(i, 'C')
			}
		}
	}

	// GC content contribution
	gcContent := features["gc_content_guide"]
	score += ds.getGCWeight(gcContent)

	return score
}

// getPositionWeight returns position-specific nucleotide weights
// Simplified from actual Doench model
func (ds *DoenchScorer) getPositionWeight(pos int, base byte) float64 {
	// Key positions from Doench et al. 2016
	switch pos {
	case 0: // Position 1
		if base == 'G' {
			return 0.15
		}
		return -0.05
	case 1, 2, 3: // Positions 2-4
		if base == 'A' || base == 'T' {
			return 0.05
		}
		return -0.02
	case 4, 5, 6, 7: // Positions 5-8
		if base == 'G' || base == 'C' {
			return 0.03
		}
		return 0.0
	case 15, 16, 17, 18, 19: // PAM-proximal (16-20)
		if base == 'T' {
			return 0.08
		}
		if base == 'G' || base == 'C' {
			return -0.06
		}
		return 0.0
	default:
		return 0.0
	}
}

// getGCWeight returns GC content weight
func (ds *DoenchScorer) getGCWeight(gcContent float64) float64 {
	// Optimal GC around 40-60%
	if gcContent < 30 || gcContent > 70 {
		return -0.3
	}
	if gcContent >= 40 && gcContent <= 60 {
		return 0.2
	}
	return 0.0
}

// estimateTm estimates melting temperature (simplified)
func (ds *DoenchScorer) estimateTm(seq string) float64 {
	// Simplified Tm calculation
	gc := 0
	at := 0
	for _, base := range seq {
		if base == 'G' || base == 'C' {
			gc++
		} else {
			at++
		}
	}

	// Basic Tm formula
	tm := float64(4*gc + 2*at)
	return tm
}

// initializeDoenchWeights initializes model weights
// In production, would load pre-trained gradient boosting model
func initializeDoenchWeights() map[string]float64 {
	weights := make(map[string]float64)

	// Simplified weights based on Doench et al. findings
	// Real model has ~2000 features from gradient boosting

	// Position-specific preferences
	weights["pos1_G"] = 0.15
	weights["pos20_T"] = 0.10
	weights["gc_optimal"] = 0.20

	return weights
}

// ScoreBatch scores multiple guides efficiently
func (ds *DoenchScorer) ScoreBatch(guides []GuideRNA, contexts []string) []float64 {
	scores := make([]float64, len(guides))

	for i, guide := range guides {
		var context string
		if i < len(contexts) {
			context = contexts[i]
		}
		scores[i] = ds.Score(guide, context)
	}

	return scores
}

// GetEfficiencyCategory categorizes guide by efficiency
func (ds *DoenchScorer) GetEfficiencyCategory(score float64) string {
	if score >= 0.7 {
		return "High"
	} else if score >= 0.4 {
		return "Medium"
	} else if score >= 0.2 {
		return "Low"
	} else {
		return "Very Low"
	}
}

// AdjustForContext adjusts score based on genomic context
func (ds *DoenchScorer) AdjustForContext(baseScore float64, guide GuideRNA) float64 {
	score := baseScore

	// Penalize guides in repetitive regions (simplified)
	if guide.OffTargetCount > 10 {
		score *= 0.5
	} else if guide.OffTargetCount > 5 {
		score *= 0.7
	}

	// Bonus for guides in exons (more likely to be functional)
	if guide.Exon > 0 {
		score *= 1.1
		if score > 1.0 {
			score = 1.0
		}
	}

	return score
}

// Add missing import
import "fmt"
