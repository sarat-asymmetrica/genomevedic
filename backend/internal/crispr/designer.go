package crispr

import (
	"fmt"
	"sort"
	"time"
)

// Designer is the main CRISPR guide RNA designer
// Integrates CHOPCHOP, Doench scoring, and off-target prediction
type Designer struct {
	chopchop      *CHOPCHOPDesigner
	doenchScorer  *DoenchScorer
	offTargetPred *OffTargetPredictor
}

// NewDesigner creates a new CRISPR designer
func NewDesigner(enzyme CasEnzyme) *Designer {
	return &Designer{
		chopchop:      NewCHOPCHOPDesigner(enzyme),
		doenchScorer:  NewDoenchScorer(),
		offTargetPred: NewOffTargetPredictor(3), // Allow up to 3 mismatches
	}
}

// Design designs CRISPR guides for a given request
func (d *Designer) Design(req DesignRequest) (*DesignResponse, error) {
	startTime := time.Now()

	// Validate request
	if err := d.validateRequest(req); err != nil {
		return nil, err
	}

	// Get target sequence
	sequence, chromosome, startPos, err := d.getTargetSequence(req)
	if err != nil {
		return nil, err
	}

	// Find all potential guides using CHOPCHOP
	guides, err := d.chopchop.FindGuides(sequence, chromosome, startPos)
	if err != nil {
		return nil, fmt.Errorf("failed to find guides: %w", err)
	}

	// Apply quality filters
	guides = d.chopchop.FilterGuides(guides, req)

	// Score guides with Doench 2016
	guides = d.scoreGuides(guides, sequence, startPos)

	// Find off-targets
	guides = d.findOffTargets(guides)

	// Calculate final rank scores
	guides = d.rankGuides(guides)

	// Filter by minimum thresholds
	guides = d.applyThresholds(guides, req)

	// Sort by rank score (descending)
	sort.Slice(guides, func(i, j int) bool {
		return guides[i].RankScore > guides[j].RankScore
	})

	// Limit to requested number
	maxGuides := req.MaxGuides
	if maxGuides == 0 {
		maxGuides = 10
	}
	if len(guides) > maxGuides {
		guides = guides[:maxGuides]
	}

	processingTime := time.Since(startTime).Milliseconds()

	response := &DesignResponse{
		Guides:         guides,
		TotalFound:     len(guides),
		Region:         fmt.Sprintf("%s:%d-%d", chromosome, startPos, startPos+len(sequence)),
		ProcessingTime: float64(processingTime),
	}

	// Add warnings
	response.Warnings = d.generateWarnings(guides, req)

	return response, nil
}

// validateRequest validates the design request
func (d *Designer) validateRequest(req DesignRequest) error {
	// Must specify either gene name or coordinates or sequence
	if req.GeneName == "" && req.Sequence == "" && (req.Chromosome == "" || req.Start == 0 || req.End == 0) {
		return fmt.Errorf("must specify gene_name, coordinates (chromosome/start/end), or sequence")
	}

	// Validate enzyme
	if req.Enzyme == "" {
		return fmt.Errorf("must specify enzyme (e.g., SpCas9)")
	}

	return nil
}

// getTargetSequence retrieves the target sequence
func (d *Designer) getTargetSequence(req DesignRequest) (string, string, int, error) {
	// If sequence provided directly
	if req.Sequence != "" {
		return req.Sequence, "custom", 0, nil
	}

	// If coordinates provided
	if req.Chromosome != "" && req.Start > 0 && req.End > 0 {
		// In production, would fetch from genome database
		// For now, return placeholder
		return "PLACEHOLDER_SEQUENCE", req.Chromosome, req.Start, nil
	}

	// If gene name provided
	if req.GeneName != "" {
		// In production, would look up gene coordinates and fetch sequence
		// For now, return placeholder
		return "PLACEHOLDER_SEQUENCE", "chr17", 7676154, nil
	}

	return "", "", 0, fmt.Errorf("could not determine target sequence")
}

// scoreGuides scores guides with Doench 2016
func (d *Designer) scoreGuides(guides []GuideRNA, fullSeq string, seqStart int) []GuideRNA {
	for i := range guides {
		guide := &guides[i]

		// Get 30bp context (4bp upstream + 20bp guide + 3bp PAM + 3bp downstream)
		context := d.getContext(guide, fullSeq, seqStart)

		// Calculate Doench score
		guide.DoenchScore = d.doenchScorer.Score(*guide, context)
	}

	return guides
}

// getContext gets 30bp context around guide for Doench scoring
func (d *Designer) getContext(guide *GuideRNA, fullSeq string, seqStart int) string {
	// Calculate position in sequence
	relPos := guide.Position - seqStart

	contextStart := relPos - 4
	contextEnd := relPos + guide.PAMSequence + 3

	if contextStart < 0 || contextEnd > len(fullSeq) {
		return "" // Not enough context
	}

	return fullSeq[contextStart:contextEnd]
}

// findOffTargets finds off-targets for all guides
func (d *Designer) findOffTargets(guides []GuideRNA) []GuideRNA {
	for i := range guides {
		guide := &guides[i]

		// Find off-targets
		offTargets := d.offTargetPred.FindOffTargets(*guide)
		guide.OffTargetCount = len(offTargets)

		// Calculate specificity score
		guide.OffTargetScore = d.offTargetPred.ScoreOffTargetSpecificity(*guide, offTargets)
	}

	return guides
}

// rankGuides calculates final ranking scores
func (d *Designer) rankGuides(guides []GuideRNA) []GuideRNA {
	for i := range guides {
		guide := &guides[i]

		// Composite score:
		// RankScore = (Doench * 0.5) + (OffTargetScore * 0.01 * 0.4) + (GC_penalty * 0.1)

		doenchContribution := guide.DoenchScore * 0.5

		// Off-target contribution (0-100 scale, so divide by 100)
		offTargetContribution := (guide.OffTargetScore / 100.0) * 0.4

		// GC content penalty (optimal 40-60%)
		gcPenalty := 0.1
		if guide.GCContent < 40 || guide.GCContent > 60 {
			gcPenalty = 0.05
		}

		guide.RankScore = doenchContribution + offTargetContribution + gcPenalty
	}

	return guides
}

// applyThresholds filters guides by minimum thresholds
func (d *Designer) applyThresholds(guides []GuideRNA, req DesignRequest) []GuideRNA {
	var filtered []GuideRNA

	minDoench := req.MinDoench
	if minDoench == 0 {
		minDoench = 0.2 // Default minimum
	}

	maxOffTarget := req.MaxOffTarget
	if maxOffTarget == 0 {
		maxOffTarget = 5 // Default maximum
	}

	for _, guide := range guides {
		if guide.DoenchScore >= minDoench && guide.OffTargetCount <= maxOffTarget {
			filtered = append(filtered, guide)
		}
	}

	return filtered
}

// generateWarnings generates warnings for the user
func (d *Designer) generateWarnings(guides []GuideRNA, req DesignRequest) []string {
	var warnings []string

	if len(guides) == 0 {
		warnings = append(warnings, "No guides found matching criteria. Try relaxing filters.")
		return warnings
	}

	// Check if many guides have low Doench scores
	lowDoenchCount := 0
	for _, g := range guides {
		if g.DoenchScore < 0.3 {
			lowDoenchCount++
		}
	}
	if lowDoenchCount > len(guides)/2 {
		warnings = append(warnings, "Many guides have low predicted efficiency (Doench score < 0.3)")
	}

	// Check if many guides have high off-targets
	highOffTargetCount := 0
	for _, g := range guides {
		if g.OffTargetCount > 3 {
			highOffTargetCount++
		}
	}
	if highOffTargetCount > len(guides)/2 {
		warnings = append(warnings, "Many guides have >3 off-target sites. Consider redesigning.")
	}

	// Check GC content
	extremeGC := 0
	for _, g := range guides {
		if g.GCContent < 30 || g.GCContent > 70 {
			extremeGC++
		}
	}
	if extremeGC > len(guides)/3 {
		warnings = append(warnings, "Several guides have extreme GC content (<30% or >70%)")
	}

	return warnings
}

// DesignBatch designs guides for multiple targets
func (d *Designer) DesignBatch(requests []DesignRequest) ([]*DesignResponse, error) {
	responses := make([]*DesignResponse, len(requests))

	for i, req := range requests {
		resp, err := d.Design(req)
		if err != nil {
			// Continue with other requests even if one fails
			resp = &DesignResponse{
				Warnings: []string{fmt.Sprintf("Design failed: %v", err)},
			}
		}
		responses[i] = resp
	}

	return responses, nil
}

// GetTopGuides returns the top N guides by rank score
func (d *Designer) GetTopGuides(guides []GuideRNA, n int) []GuideRNA {
	// Already sorted by rankGuides
	if len(guides) <= n {
		return guides
	}
	return guides[:n]
}

// CompareGuides compares two guides
func (d *Designer) CompareGuides(g1, g2 GuideRNA) string {
	comparison := fmt.Sprintf("Guide 1 vs Guide 2:\n")
	comparison += fmt.Sprintf("  Doench Score: %.3f vs %.3f\n", g1.DoenchScore, g2.DoenchScore)
	comparison += fmt.Sprintf("  Off-targets: %d vs %d\n", g1.OffTargetCount, g2.OffTargetCount)
	comparison += fmt.Sprintf("  Specificity: %.1f vs %.1f\n", g1.OffTargetScore, g2.OffTargetScore)
	comparison += fmt.Sprintf("  GC%%: %.1f vs %.1f\n", g1.GCContent, g2.GCContent)
	comparison += fmt.Sprintf("  Rank Score: %.3f vs %.3f\n", g1.RankScore, g2.RankScore)

	if g1.RankScore > g2.RankScore {
		comparison += "  Winner: Guide 1\n"
	} else {
		comparison += "  Winner: Guide 2\n"
	}

	return comparison
}
