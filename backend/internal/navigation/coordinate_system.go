/**
 * Genomic Coordinate System
 *
 * Converts between genomic coordinates (chromosome, position) and 3D space
 * Uses golden spiral for spatial layout (from Wave 1)
 *
 * Coordinate mapping:
 * - Linear genomic position → 3D spatial position
 * - Golden spiral layout for aesthetics
 * - Chromosome boundaries preserved
 */

package navigation

import (
	"fmt"
	"math"
)

// Chromosome represents a human chromosome
type Chromosome struct {
	Name   string
	Length uint64
	Offset uint64 // Cumulative offset in genome
}

// Human chromosome lengths (GRCh38/hg38)
var HumanChromosomes = []Chromosome{
	{"chr1", 248956422, 0},
	{"chr2", 242193529, 248956422},
	{"chr3", 198295559, 491149951},
	{"chr4", 190214555, 689445510},
	{"chr5", 181538259, 879660065},
	{"chr6", 170805979, 1061198324},
	{"chr7", 159345973, 1232004303},
	{"chr8", 145138636, 1391350276},
	{"chr9", 138394717, 1536488912},
	{"chr10", 133797422, 1674883629},
	{"chr11", 135086622, 1808681051},
	{"chr12", 133275309, 1943767673},
	{"chr13", 114364328, 2077042982},
	{"chr14", 107043718, 2191407310},
	{"chr15", 101991189, 2298451028},
	{"chr16", 90338345, 2400442217},
	{"chr17", 83257441, 2490780562},
	{"chr18", 80373285, 2574038003},
	{"chr19", 58617616, 2654411288},
	{"chr20", 64444167, 2713028904},
	{"chr21", 46709983, 2777473071},
	{"chr22", 50818468, 2824183054},
	{"chrX", 156040895, 2875001522},
	{"chrY", 57227415, 3031042417},
}

// TotalGenomeLength is the total length of the human genome (hg38)
const TotalGenomeLength uint64 = 3088269832

// CoordinateSystem manages genomic-to-3D coordinate conversion
type CoordinateSystem struct {
	chromosomes      []Chromosome
	chromosomeMap    map[string]*Chromosome
	scaleFactor      float64 // Genomic bp → 3D units
	spiralRadius     float64 // Base radius for golden spiral
	spiralHeight     float64 // Height per spiral turn
	goldenAngle      float64 // 137.5° (golden angle)
}

// NewCoordinateSystem creates a new coordinate system
func NewCoordinateSystem(scaleFactor, spiralRadius, spiralHeight float64) *CoordinateSystem {
	// Build chromosome map
	chromMap := make(map[string]*Chromosome)
	for i := range HumanChromosomes {
		chromMap[HumanChromosomes[i].Name] = &HumanChromosomes[i]
	}

	return &CoordinateSystem{
		chromosomes:   HumanChromosomes,
		chromosomeMap: chromMap,
		scaleFactor:   scaleFactor,
		spiralRadius:  spiralRadius,
		spiralHeight:  spiralHeight,
		goldenAngle:   137.5 * math.Pi / 180.0, // 137.5° in radians
	}
}

// GenomicToLinear converts chromosome + position to linear genomic position
func (cs *CoordinateSystem) GenomicToLinear(chromosome string, position uint64) (uint64, error) {
	chrom, exists := cs.chromosomeMap[chromosome]
	if !exists {
		return 0, fmt.Errorf("unknown chromosome: %s", chromosome)
	}

	if position > chrom.Length {
		return 0, fmt.Errorf("position %d exceeds chromosome length %d", position, chrom.Length)
	}

	return chrom.Offset + position, nil
}

// LinearToGenomic converts linear genomic position to chromosome + position
func (cs *CoordinateSystem) LinearToGenomic(linearPos uint64) (string, uint64, error) {
	if linearPos >= TotalGenomeLength {
		return "", 0, fmt.Errorf("position %d exceeds genome length %d", linearPos, TotalGenomeLength)
	}

	// Binary search for chromosome
	for i := len(cs.chromosomes) - 1; i >= 0; i-- {
		chrom := cs.chromosomes[i]
		if linearPos >= chrom.Offset {
			chromPos := linearPos - chrom.Offset
			return chrom.Name, chromPos, nil
		}
	}

	return "", 0, fmt.Errorf("failed to find chromosome for position %d", linearPos)
}

// GenomicTo3D converts genomic position to 3D spatial position (golden spiral)
func (cs *CoordinateSystem) GenomicTo3D(chromosome string, position uint64) ([3]float32, error) {
	// Convert to linear position
	linearPos, err := cs.GenomicToLinear(chromosome, position)
	if err != nil {
		return [3]float32{}, err
	}

	return cs.LinearTo3D(linearPos), nil
}

// LinearTo3D converts linear genomic position to 3D spatial position
func (cs *CoordinateSystem) LinearTo3D(linearPos uint64) [3]float32 {
	// Normalize position to [0, 1]
	t := float64(linearPos) / float64(TotalGenomeLength)

	// Golden spiral formula
	radius := cs.spiralRadius * math.Sqrt(t)
	angle := float64(linearPos) * cs.goldenAngle
	height := (t - 0.5) * cs.spiralHeight

	// Convert to Cartesian coordinates
	x := float32(radius * math.Cos(angle))
	z := float32(radius * math.Sin(angle))
	y := float32(height)

	return [3]float32{x, y, z}
}

// ThreeDToLinear converts 3D position back to linear genomic position (approximate)
func (cs *CoordinateSystem) ThreeDToLinear(x, y, z float32) uint64 {
	// Convert back to cylindrical coordinates
	radius := math.Sqrt(float64(x*x + z*z))

	// Approximate t from radius (inverse of sqrt)
	t := (radius / cs.spiralRadius) * (radius / cs.spiralRadius)

	// Convert to linear position
	linearPos := uint64(t * float64(TotalGenomeLength))

	// Clamp to valid range
	if linearPos >= TotalGenomeLength {
		linearPos = TotalGenomeLength - 1
	}

	return linearPos
}

// ThreeDToGenomic converts 3D position to genomic coordinates (approximate)
func (cs *CoordinateSystem) ThreeDToGenomic(x, y, z float32) (string, uint64, error) {
	linearPos := cs.ThreeDToLinear(x, y, z)
	return cs.LinearToGenomic(linearPos)
}

// GetChromosome returns chromosome info by name
func (cs *CoordinateSystem) GetChromosome(name string) (*Chromosome, error) {
	chrom, exists := cs.chromosomeMap[name]
	if !exists {
		return nil, fmt.Errorf("unknown chromosome: %s", name)
	}
	return chrom, nil
}

// GetChromosomes returns all chromosomes
func (cs *CoordinateSystem) GetChromosomes() []Chromosome {
	return cs.chromosomes
}

// DistanceBetweenPositions calculates 3D distance between two genomic positions
func (cs *CoordinateSystem) DistanceBetweenPositions(chrom1 string, pos1 uint64, chrom2 string, pos2 uint64) (float32, error) {
	p1, err := cs.GenomicTo3D(chrom1, pos1)
	if err != nil {
		return 0, err
	}

	p2, err := cs.GenomicTo3D(chrom2, pos2)
	if err != nil {
		return 0, err
	}

	dx := p2[0] - p1[0]
	dy := p2[1] - p1[1]
	dz := p2[2] - p1[2]

	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz))), nil
}

// GetRegionBounds returns 3D bounding box for a genomic region
func (cs *CoordinateSystem) GetRegionBounds(chromosome string, startPos, endPos uint64) (min, max [3]float32, err error) {
	// Sample positions along the region
	samples := 100
	step := (endPos - startPos) / uint64(samples)

	min = [3]float32{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}
	max = [3]float32{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}

	for i := 0; i < samples; i++ {
		pos := startPos + uint64(i)*step
		if pos > endPos {
			pos = endPos
		}

		p3d, err := cs.GenomicTo3D(chromosome, pos)
		if err != nil {
			return min, max, err
		}

		// Update bounds
		if p3d[0] < min[0] {
			min[0] = p3d[0]
		}
		if p3d[1] < min[1] {
			min[1] = p3d[1]
		}
		if p3d[2] < min[2] {
			min[2] = p3d[2]
		}

		if p3d[0] > max[0] {
			max[0] = p3d[0]
		}
		if p3d[1] > max[1] {
			max[1] = p3d[1]
		}
		if p3d[2] > max[2] {
			max[2] = p3d[2]
		}
	}

	return min, max, nil
}
