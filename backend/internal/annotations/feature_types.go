/**
 * Genomic Feature Type Definitions
 *
 * Defines types and colors for genomic annotations
 * Based on GTF/GFF3 format specifications
 */

package annotations

import "fmt"

// FeatureType represents a type of genomic feature
type FeatureType int

const (
	FeatureUnknown FeatureType = iota
	FeatureGene
	FeatureTranscript
	FeatureExon
	FeatureIntron
	FeatureCDS
	FeatureUTR5
	FeatureUTR3
	FeaturePromoter
	FeatureEnhancer
	FeatureStartCodon
	FeatureStopCodon
)

func (ft FeatureType) String() string {
	switch ft {
	case FeatureGene:
		return "Gene"
	case FeatureTranscript:
		return "Transcript"
	case FeatureExon:
		return "Exon"
	case FeatureIntron:
		return "Intron"
	case FeatureCDS:
		return "CDS"
	case FeatureUTR5:
		return "5'UTR"
	case FeatureUTR3:
		return "3'UTR"
	case FeaturePromoter:
		return "Promoter"
	case FeatureEnhancer:
		return "Enhancer"
	case FeatureStartCodon:
		return "Start Codon"
	case FeatureStopCodon:
		return "Stop Codon"
	default:
		return "Unknown"
	}
}

// FeatureColor represents RGB color for feature visualization
type FeatureColor struct {
	R, G, B, A float32
}

// Predefined colors for genomic features
var (
	ColorGene       = FeatureColor{R: 0.2, G: 0.8, B: 1.0, A: 0.6}  // Cyan
	ColorExon       = FeatureColor{R: 0.0, G: 1.0, B: 0.5, A: 0.8}  // Green-cyan
	ColorIntron     = FeatureColor{R: 0.3, G: 0.3, B: 0.8, A: 0.4}  // Blue (dim)
	ColorCDS        = FeatureColor{R: 0.0, G: 1.0, B: 0.0, A: 0.9}  // Bright green (coding)
	ColorUTR        = FeatureColor{R: 1.0, G: 1.0, B: 0.5, A: 0.5}  // Yellow (untranslated)
	ColorPromoter   = FeatureColor{R: 1.0, G: 0.5, B: 0.0, A: 0.7}  // Orange (regulatory)
	ColorEnhancer   = FeatureColor{R: 1.0, G: 0.0, B: 1.0, A: 0.6}  // Magenta (regulatory)
	ColorStartCodon = FeatureColor{R: 0.0, G: 1.0, B: 0.0, A: 1.0}  // Bright green
	ColorStopCodon  = FeatureColor{R: 1.0, G: 0.0, B: 0.0, A: 1.0}  // Red
)

// GetFeatureColor returns the color for a feature type
func GetFeatureColor(ft FeatureType) FeatureColor {
	switch ft {
	case FeatureGene:
		return ColorGene
	case FeatureExon:
		return ColorExon
	case FeatureIntron:
		return ColorIntron
	case FeatureCDS:
		return ColorCDS
	case FeatureUTR5, FeatureUTR3:
		return ColorUTR
	case FeaturePromoter:
		return ColorPromoter
	case FeatureEnhancer:
		return ColorEnhancer
	case FeatureStartCodon:
		return ColorStartCodon
	case FeatureStopCodon:
		return ColorStopCodon
	default:
		return FeatureColor{R: 0.5, G: 0.5, B: 0.5, A: 0.5} // Gray
	}
}

// GenomicFeature represents a genomic annotation feature
type GenomicFeature struct {
	Type       FeatureType
	Chromosome string
	Start      uint64
	End        uint64
	Strand     string // "+" or "-"
	GeneID     string
	GeneName   string
	TranscriptID string
	Attributes map[string]string
}

// NewGenomicFeature creates a new genomic feature
func NewGenomicFeature(featureType FeatureType, chrom string, start, end uint64, strand string) *GenomicFeature {
	return &GenomicFeature{
		Type:       featureType,
		Chromosome: chrom,
		Start:      start,
		End:        end,
		Strand:     strand,
		Attributes: make(map[string]string),
	}
}

// Length returns the length of the feature in base pairs
func (gf *GenomicFeature) Length() uint64 {
	if gf.End >= gf.Start {
		return gf.End - gf.Start + 1
	}
	return 0
}

// Contains checks if a position is within this feature
func (gf *GenomicFeature) Contains(pos uint64) bool {
	return pos >= gf.Start && pos <= gf.End
}

// Overlaps checks if this feature overlaps with another
func (gf *GenomicFeature) Overlaps(other *GenomicFeature) bool {
	if gf.Chromosome != other.Chromosome {
		return false
	}
	return gf.Start <= other.End && gf.End >= other.Start
}

// String returns a human-readable string representation
func (gf *GenomicFeature) String() string {
	return fmt.Sprintf("%s:%d-%d (%s) [%s: %s]",
		gf.Chromosome, gf.Start, gf.End, gf.Strand, gf.Type.String(), gf.GeneName)
}

// ParseFeatureType parses a feature type string from GTF/GFF3
func ParseFeatureType(typeStr string) FeatureType {
	switch typeStr {
	case "gene":
		return FeatureGene
	case "transcript", "mRNA":
		return FeatureTranscript
	case "exon":
		return FeatureExon
	case "intron":
		return FeatureIntron
	case "CDS":
		return FeatureCDS
	case "five_prime_UTR", "5UTR":
		return FeatureUTR5
	case "three_prime_UTR", "3UTR":
		return FeatureUTR3
	case "promoter":
		return FeaturePromoter
	case "enhancer":
		return FeatureEnhancer
	case "start_codon":
		return FeatureStartCodon
	case "stop_codon":
		return FeatureStopCodon
	default:
		return FeatureUnknown
	}
}
