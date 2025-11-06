/**
 * Gene Annotation Overlay System
 *
 * Overlays gene annotations on genomic particles
 * Colors particles based on genomic features (exons, introns, promoters, etc.)
 *
 * Priority (highest to lowest):
 * 1. CDS (coding sequence) - Bright green
 * 2. Exon - Green-cyan
 * 3. Start/Stop codon - Bright green/red
 * 4. UTR - Yellow
 * 5. Promoter - Orange
 * 6. Intron - Dim blue
 * 7. Gene - Cyan
 */

package annotations

import (
	"fmt"
	"sync"
)

// ParticleAnnotation represents annotation data for a single particle
type ParticleAnnotation struct {
	Position       uint64
	Chromosome     string
	Features       []*GenomicFeature
	PrimaryFeature *GenomicFeature // Highest priority feature
	Color          FeatureColor
	InGene         bool
	InExon         bool
	InCDS          bool
	GeneNames      []string
}

// GeneOverlay overlays gene annotations on particles
type GeneOverlay struct {
	parser            *GTFParser
	particleAnnotations map[string]*ParticleAnnotation // Key: "chr:position"
	mu                sync.RWMutex
}

// NewGeneOverlay creates a new gene overlay
func NewGeneOverlay(parser *GTFParser) *GeneOverlay {
	return &GeneOverlay{
		parser:            parser,
		particleAnnotations: make(map[string]*ParticleAnnotation),
	}
}

// BuildOverlay builds the gene overlay for all particles
func (go_ *GeneOverlay) BuildOverlay() error {
	go_.mu.Lock()
	defer go_.mu.Unlock()

	// Clear existing overlay
	go_.particleAnnotations = make(map[string]*ParticleAnnotation)

	// For each feature, annotate all positions within it
	for _, feature := range go_.parser.GetFeatures() {
		// Sample every 10 bp within the feature (for performance)
		step := uint64(10)
		for pos := feature.Start; pos <= feature.End; pos += step {
			key := fmt.Sprintf("%s:%d", feature.Chromosome, pos)

			pa, exists := go_.particleAnnotations[key]
			if !exists {
				pa = &ParticleAnnotation{
					Position:   pos,
					Chromosome: feature.Chromosome,
					Features:   make([]*GenomicFeature, 0, 4),
					GeneNames:  make([]string, 0, 2),
				}
				go_.particleAnnotations[key] = pa
			}

			pa.Features = append(pa.Features, feature)

			// Track feature types
			switch feature.Type {
			case FeatureGene:
				pa.InGene = true
				if feature.GeneName != "" && !contains(pa.GeneNames, feature.GeneName) {
					pa.GeneNames = append(pa.GeneNames, feature.GeneName)
				}
			case FeatureExon:
				pa.InExon = true
			case FeatureCDS:
				pa.InCDS = true
			}
		}
	}

	// Compute primary features and colors
	for _, pa := range go_.particleAnnotations {
		go_.computePrimaryFeature(pa)
	}

	return nil
}

// GetParticleAnnotation returns annotation for a particle
func (go_ *GeneOverlay) GetParticleAnnotation(chromosome string, position uint64) *ParticleAnnotation {
	go_.mu.RLock()
	defer go_.mu.RUnlock()

	key := fmt.Sprintf("%s:%d", chromosome, position)
	return go_.particleAnnotations[key]
}

// GetParticleColor returns the color for a particle based on annotations
func (go_ *GeneOverlay) GetParticleColor(chromosome string, position uint64) (FeatureColor, bool) {
	pa := go_.GetParticleAnnotation(chromosome, position)
	if pa == nil || pa.PrimaryFeature == nil {
		return FeatureColor{}, false
	}
	return pa.Color, true
}

// computePrimaryFeature computes the primary feature for a particle (highest priority)
func (go_ *GeneOverlay) computePrimaryFeature(pa *ParticleAnnotation) {
	if len(pa.Features) == 0 {
		return
	}

	// Priority order
	priorityOrder := []FeatureType{
		FeatureStartCodon,
		FeatureStopCodon,
		FeatureCDS,
		FeatureExon,
		FeatureUTR5,
		FeatureUTR3,
		FeaturePromoter,
		FeatureIntron,
		FeatureGene,
	}

	// Find highest priority feature
	for _, priority := range priorityOrder {
		for _, feature := range pa.Features {
			if feature.Type == priority {
				pa.PrimaryFeature = feature
				pa.Color = GetFeatureColor(priority)
				return
			}
		}
	}

	// Default to first feature
	pa.PrimaryFeature = pa.Features[0]
	pa.Color = GetFeatureColor(pa.PrimaryFeature.Type)
}

// FilterByFeatureType filters annotations by feature type
func (go_ *GeneOverlay) FilterByFeatureType(featureType FeatureType) []*ParticleAnnotation {
	go_.mu.RLock()
	defer go_.mu.RUnlock()

	filtered := make([]*ParticleAnnotation, 0, len(go_.particleAnnotations))

	for _, pa := range go_.particleAnnotations {
		if pa.PrimaryFeature != nil && pa.PrimaryFeature.Type == featureType {
			filtered = append(filtered, pa)
		}
	}

	return filtered
}

// FilterByGene filters annotations by gene name
func (go_ *GeneOverlay) FilterByGene(geneName string) []*ParticleAnnotation {
	go_.mu.RLock()
	defer go_.mu.RUnlock()

	filtered := make([]*ParticleAnnotation, 0, 1000)

	for _, pa := range go_.particleAnnotations {
		if contains(pa.GeneNames, geneName) {
			filtered = append(filtered, pa)
		}
	}

	return filtered
}

// GetStatistics returns overlay statistics
func (go_ *GeneOverlay) GetStatistics() map[string]interface{} {
	go_.mu.RLock()
	defer go_.mu.RUnlock()

	totalParticles := len(go_.particleAnnotations)
	particlesInGenes := 0
	particlesInExons := 0
	particlesInCDS := 0

	typeCounts := make(map[FeatureType]int)

	for _, pa := range go_.particleAnnotations {
		if pa.InGene {
			particlesInGenes++
		}
		if pa.InExon {
			particlesInExons++
		}
		if pa.InCDS {
			particlesInCDS++
		}

		if pa.PrimaryFeature != nil {
			typeCounts[pa.PrimaryFeature.Type]++
		}
	}

	return map[string]interface{}{
		"total_particles":     totalParticles,
		"particles_in_genes":  particlesInGenes,
		"particles_in_exons":  particlesInExons,
		"particles_in_cds":    particlesInCDS,
		"primary_type_counts": typeCounts,
	}
}

// GetGeneList returns a list of all gene names
func (go_ *GeneOverlay) GetGeneList() []string {
	go_.mu.RLock()
	defer go_.mu.RUnlock()

	geneSet := make(map[string]bool)

	for _, pa := range go_.particleAnnotations {
		for _, geneName := range pa.GeneNames {
			geneSet[geneName] = true
		}
	}

	genes := make([]string, 0, len(geneSet))
	for gene := range geneSet {
		genes = append(genes, gene)
	}

	return genes
}

// contains checks if a string slice contains a value
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
