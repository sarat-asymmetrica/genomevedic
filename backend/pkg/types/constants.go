// Package types - Shared constants and types for GenomeVedic.ai
//
// Mathematical constants from Vedic mathematics and genomics
package types

import "math"

// Mathematical Constants
const (
	// Golden Ratio (φ = phi)
	Phi          = 1.618033988749894848204586834  // (1 + √5) / 2
	PhiConjugate = 0.618033988749894848204586834  // 1 / φ = φ - 1

	// Golden Angle (360° × (2 - φ))
	GoldenAngle    = 137.50776405003785080934328    // degrees
	GoldenAngleRad = 2.3999632297286531134242335    // radians
)

// Genomic Constants
const (
	// FASTQ format constants
	FastqLinesPerRead = 4 // FASTQ has 4 lines per sequence read

	// Streaming constants
	DefaultChunkSize  = 10 * 1024 * 1024 // 10 MB chunks for streaming
	DefaultBufferSize = 32 * 1024        // 32 KB buffer for reading

	// Memory budget (from Red Team Findings)
	MaxCPUMemoryGB  = 2.0   // 2 GB CPU RAM budget
	MaxGPUMemoryGB  = 0.25  // 250 MB GPU VRAM budget

	// Performance targets
	TargetFPS           = 60        // 60 fps minimum
	TargetLoadTimeMs    = 5000      // 5 seconds max load time
	TargetFrameBudgetMs = 16.67     // 16.67ms per frame at 60fps
)

// Voxel Grid Constants
const (
	// Spatial voxel grid (from corrected architecture)
	TotalVoxels   = 5_000_000 // 5M voxels for 3B particles
	VoxelSize     = 10.0      // Voxel size in 3D space

	// Frustum culling
	VisibilityRatio = 0.01  // 1% of voxels visible at once
	VisibleVoxels   = 50_000 // 50K visible voxels (1% of 5M)

	// LOD (Level of Detail) thresholds
	LODClose   = 0.0    // Full detail (100% particles)
	LODMedium  = 100.0  // Medium detail (50% particles)
	LODFar     = 500.0  // Far detail (10% particles)
	LODCulled  = 2000.0 // Too far, don't render
)

// Color Constants
const (
	// HSL color space
	MaxHue        = 360.0
	MaxSaturation = 1.0
	MaxLightness  = 1.0
)

// Base Encoding (ATCG → numeric values)
const (
	BaseA = 1
	BaseT = 2
	BaseG = 3
	BaseC = 4
	BaseN = 0 // Unknown/ambiguous base
)

// Utility functions
func DegreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}

func RadiansToDegrees(radians float64) float64 {
	return radians * 180.0 / math.Pi
}
