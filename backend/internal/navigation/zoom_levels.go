/**
 * Multi-Scale Zoom Levels for Genomic Navigation
 *
 * Defines 5 zoom levels for navigating the genome:
 * 1. Genome (3B bp) - Full human genome
 * 2. Chromosome (~250M bp) - Individual chromosome
 * 3. Gene (~100K bp) - Gene region with introns/exons
 * 4. Exon (~1K bp) - Individual exon detail
 * 5. Nucleotide (1-100 bp) - Base-level detail
 *
 * Each level has different LOD (Level of Detail) settings
 */

package navigation

import (
	"fmt"
	"math"
)

// ZoomLevel represents a zoom level in the genomic view
type ZoomLevel int

const (
	ZoomGenome ZoomLevel = iota
	ZoomChromosome
	ZoomGene
	ZoomExon
	ZoomNucleotide
)

func (zl ZoomLevel) String() string {
	switch zl {
	case ZoomGenome:
		return "Genome"
	case ZoomChromosome:
		return "Chromosome"
	case ZoomGene:
		return "Gene"
	case ZoomExon:
		return "Exon"
	case ZoomNucleotide:
		return "Nucleotide"
	default:
		return "Unknown"
	}
}

// ZoomLevelConfig contains configuration for a zoom level
type ZoomLevelConfig struct {
	Level            ZoomLevel
	MinBasePairs     uint64  // Minimum bp visible at this level
	MaxBasePairs     uint64  // Maximum bp visible at this level
	CameraDistance   float32 // Default camera distance
	ParticleDensity  float32 // Particle density (0.0-1.0)
	LODLevel         int     // LOD level (0-3)
	ShowLabels       bool    // Show gene/exon labels
	ShowSequence     bool    // Show nucleotide sequence
	ShowAnnotations  bool    // Show gene annotations
	ShowMutations    bool    // Show mutation markers
}

// Predefined zoom level configurations
var ZoomLevelConfigs = []ZoomLevelConfig{
	// Genome level (3B bp)
	{
		Level:           ZoomGenome,
		MinBasePairs:    1_000_000_000, // 1 Gbp
		MaxBasePairs:    3_200_000_000, // 3.2 Gbp (full genome)
		CameraDistance:  5000.0,
		ParticleDensity: 0.01, // 1% (sparse view)
		LODLevel:        0,    // Lowest detail
		ShowLabels:      false,
		ShowSequence:    false,
		ShowAnnotations: false,
		ShowMutations:   true, // Show hotspots only
	},
	// Chromosome level (250M bp)
	{
		Level:           ZoomChromosome,
		MinBasePairs:    10_000_000, // 10 Mbp
		MaxBasePairs:    250_000_000, // 250 Mbp
		CameraDistance:  2000.0,
		ParticleDensity: 0.1, // 10%
		LODLevel:        1,
		ShowLabels:      true, // Show major genes
		ShowSequence:    false,
		ShowAnnotations: false,
		ShowMutations:   true,
	},
	// Gene level (100K bp)
	{
		Level:           ZoomGene,
		MinBasePairs:    10_000,   // 10 Kbp
		MaxBasePairs:    1_000_000, // 1 Mbp
		CameraDistance:  800.0,
		ParticleDensity: 0.5, // 50%
		LODLevel:        2,
		ShowLabels:      true,
		ShowSequence:    false,
		ShowAnnotations: true, // Show exons/introns
		ShowMutations:   true,
	},
	// Exon level (1K bp)
	{
		Level:           ZoomExon,
		MinBasePairs:    100,    // 100 bp
		MaxBasePairs:    10_000, // 10 Kbp
		CameraDistance:  200.0,
		ParticleDensity: 0.9, // 90%
		LODLevel:        3,
		ShowLabels:      true,
		ShowSequence:    false, // Sequence at next level
		ShowAnnotations: true,
		ShowMutations:   true,
	},
	// Nucleotide level (1-100 bp)
	{
		Level:           ZoomNucleotide,
		MinBasePairs:    1,
		MaxBasePairs:    100,
		CameraDistance:  50.0,
		ParticleDensity: 1.0, // 100% (all bases visible)
		LODLevel:        3,
		ShowLabels:      true,
		ShowSequence:    true, // Show ACGT sequence
		ShowAnnotations: true,
		ShowMutations:   true,
	},
}

// GetZoomLevelConfig returns configuration for a zoom level
func GetZoomLevelConfig(level ZoomLevel) ZoomLevelConfig {
	if int(level) < len(ZoomLevelConfigs) {
		return ZoomLevelConfigs[level]
	}
	return ZoomLevelConfigs[ZoomGenome]
}

// GetZoomLevelFromDistance determines zoom level based on camera distance
func GetZoomLevelFromDistance(distance float32) ZoomLevel {
	// Find closest zoom level
	minDiff := float32(math.MaxFloat32)
	bestLevel := ZoomGenome

	for _, config := range ZoomLevelConfigs {
		diff := float32(math.Abs(float64(distance - config.CameraDistance)))
		if diff < minDiff {
			minDiff = diff
			bestLevel = config.Level
		}
	}

	return bestLevel
}

// GetZoomLevelFromViewport determines zoom level based on visible base pairs
func GetZoomLevelFromViewport(visibleBasePairs uint64) ZoomLevel {
	for i := len(ZoomLevelConfigs) - 1; i >= 0; i-- {
		config := ZoomLevelConfigs[i]
		if visibleBasePairs >= config.MinBasePairs {
			return config.Level
		}
	}
	return ZoomNucleotide
}

// GenomicViewport represents a viewport into the genome
type GenomicViewport struct {
	Chromosome string
	StartPos   uint64
	EndPos     uint64
	ZoomLevel  ZoomLevel
	Config     ZoomLevelConfig
}

// NewGenomicViewport creates a new genomic viewport
func NewGenomicViewport(chromosome string, startPos, endPos uint64) *GenomicViewport {
	visibleBp := endPos - startPos
	zoomLevel := GetZoomLevelFromViewport(visibleBp)
	config := GetZoomLevelConfig(zoomLevel)

	return &GenomicViewport{
		Chromosome: chromosome,
		StartPos:   startPos,
		EndPos:     endPos,
		ZoomLevel:  zoomLevel,
		Config:     config,
	}
}

// Length returns the length of the viewport in base pairs
func (gv *GenomicViewport) Length() uint64 {
	return gv.EndPos - gv.StartPos
}

// Contains checks if a position is within the viewport
func (gv *GenomicViewport) Contains(pos uint64) bool {
	return pos >= gv.StartPos && pos <= gv.EndPos
}

// SetZoomLevel updates the zoom level and config
func (gv *GenomicViewport) SetZoomLevel(level ZoomLevel) {
	gv.ZoomLevel = level
	gv.Config = GetZoomLevelConfig(level)
}

// ZoomIn zooms in to the next level (if possible)
func (gv *GenomicViewport) ZoomIn() bool {
	if gv.ZoomLevel < ZoomNucleotide {
		gv.SetZoomLevel(gv.ZoomLevel + 1)
		return true
	}
	return false
}

// ZoomOut zooms out to the previous level (if possible)
func (gv *GenomicViewport) ZoomOut() bool {
	if gv.ZoomLevel > ZoomGenome {
		gv.SetZoomLevel(gv.ZoomLevel - 1)
		return true
	}
	return false
}

// Center returns the center position of the viewport
func (gv *GenomicViewport) Center() uint64 {
	return (gv.StartPos + gv.EndPos) / 2
}

// String returns a human-readable string representation
func (gv *GenomicViewport) String() string {
	return fmt.Sprintf("%s:%d-%d [%s] (%.1fK bp)",
		gv.Chromosome, gv.StartPos, gv.EndPos, gv.ZoomLevel.String(), float64(gv.Length())/1000.0)
}

// NavigationBookmark represents a saved location
type NavigationBookmark struct {
	Name       string
	Chromosome string
	Position   uint64
	ZoomLevel  ZoomLevel
	Notes      string
}

// NewNavigationBookmark creates a new navigation bookmark
func NewNavigationBookmark(name, chromosome string, position uint64, zoomLevel ZoomLevel) *NavigationBookmark {
	return &NavigationBookmark{
		Name:       name,
		Chromosome: chromosome,
		Position:   position,
		ZoomLevel:  zoomLevel,
	}
}

// String returns a human-readable string representation
func (nb *NavigationBookmark) String() string {
	return fmt.Sprintf("%s: %s:%d [%s]", nb.Name, nb.Chromosome, nb.Position, nb.ZoomLevel.String())
}
