// Package loader - FASTQ streaming with spatial hashing
package loader

import (
	"fmt"
	"math"
	"strings"

	"genomevedic/backend/pkg/types"
)

// FASTQStreamer streams FASTQ files and converts to 3D particles
// Implements the streaming architecture: Disk → CPU → GPU
type FASTQStreamer struct {
	parser   *FASTQParser
	position int // Current position in genome
}

// NewFASTQStreamer creates a new FASTQ streamer
func NewFASTQStreamer(filepath string) (*FASTQStreamer, error) {
	parser, err := NewFASTQParser(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to create parser: %w", err)
	}

	return &FASTQStreamer{
		parser:   parser,
		position: 0,
	}, nil
}

// StreamWindow streams a window of base positions
// This is the core streaming function that implements the Disk → CPU → GPU pipeline
func (s *FASTQStreamer) StreamWindow(maxBases int) ([]types.BasePosition, error) {
	basePositions := make([]types.BasePosition, 0, maxBases)

	// Stream reads from FASTQ
	readChan := s.parser.StreamReads()

	for read := range readChan {
		// Convert each base in the read to a BasePosition
		for i, base := range read.Sequence {
			if len(basePositions) >= maxBases {
				return basePositions, nil // Window full
			}

			// Get quality score
			quality := byte(0)
			if i < len(read.Quality) {
				quality = read.Quality[i]
			}

			// Convert to 3D coordinates using digital root spatial hashing
			coords := SequenceTo3D(read.Sequence, i, s.position+i)

			basePosition := types.BasePosition{
				Position: s.position + i,
				Base:     byte(base),
				Quality:  quality,
				Coords:   coords,
			}

			basePositions = append(basePositions, basePosition)
		}

		s.position += len(read.Sequence)
	}

	return basePositions, nil
}

// Close closes the underlying parser
func (s *FASTQStreamer) Close() error {
	return s.parser.Close()
}

// GetPosition returns the current position in the genome
func (s *FASTQStreamer) GetPosition() int {
	return s.position
}

// =============================================================================
// DIGITAL ROOT SPATIAL MAPPING
// Implements the Vedic mathematics approach from MATHEMATICAL_FOUNDATIONS.md
// =============================================================================

// DigitalRoot computes the Vedic digital root of a number
// Formula: dr(n) = 1 + ((n - 1) mod 9)
func DigitalRoot(n int) int {
	if n == 0 {
		return 0
	}
	if n < 0 {
		n = -n
	}
	result := n % 9
	if result == 0 {
		return 9
	}
	return result
}

// SequenceTo3D maps DNA sequence to 3D coordinates using digital root hashing
// This is the core algorithm from MATHEMATICAL_FOUNDATIONS.md
//
// Algorithm:
// 1. Extract triplet codon (biological unit: 3 bases = 1 amino acid)
// 2. Compute digital root of each base + position
// 3. Map to golden spiral for natural clustering
func SequenceTo3D(sequence string, offset int, position int) types.Vector3D {
	// Ensure we have at least 3 bases for triplet
	if offset+3 > len(sequence) {
		// Fallback for incomplete triplet
		return fallbackTo3D(position)
	}

	// Extract triplet codon
	triplet := strings.ToUpper(sequence[offset : offset+3])

	// Digital root of each base + position (for uniqueness)
	rootX := DigitalRoot(EncodeBase(triplet[0]) + position)
	rootY := DigitalRoot(EncodeBase(triplet[1]) + position*2)
	rootZ := DigitalRoot(EncodeBase(triplet[2]) + position*3)

	// Map to golden spiral (phyllotaxis pattern)
	angle := float64(position) * types.GoldenAngleRad
	radius := math.Sqrt(float64(position))

	return types.Vector3D{
		X: radius * math.Cos(angle) * float64(rootX) / 9.0,
		Y: radius * math.Sin(angle) * float64(rootY) / 9.0,
		Z: float64(rootZ) * radius / 9.0,
	}
}

// fallbackTo3D provides coordinates when triplet is not available
func fallbackTo3D(position int) types.Vector3D {
	// Use golden spiral without digital root modulation
	angle := float64(position) * types.GoldenAngleRad
	radius := math.Sqrt(float64(position))

	return types.Vector3D{
		X: radius * math.Cos(angle),
		Y: radius * math.Sin(angle),
		Z: 0,
	}
}

// SpatialHash computes the voxel ID for a 3D position
// This enables O(1) spatial queries
func SpatialHash(pos types.Vector3D) types.VoxelID {
	return types.VoxelID{
		X: int(math.Floor(pos.X / types.VoxelSize)),
		Y: int(math.Floor(pos.Y / types.VoxelSize)),
		Z: int(math.Floor(pos.Z / types.VoxelSize)),
	}
}
