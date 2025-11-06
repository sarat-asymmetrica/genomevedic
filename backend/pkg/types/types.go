// Package types - Core data structures for GenomeVedic.ai
package types

import "image/color"

// Vector3D represents a 3D coordinate in space
type Vector3D struct {
	X float64
	Y float64
	Z float64
}

// Particle represents a single base pair as a 3D particle
type Particle struct {
	Position Vector3D  // 3D coordinate (from digital root hashing)
	Color    color.RGBA // Vedic color (GC content, mutation frequency, etc.)
	Size     float32   // Particle size (for LOD)
	Base     byte      // Original base (A, T, G, C, N)
	Quality  byte      // FASTQ quality score
}

// Voxel represents a spatial grid cell containing particles
type Voxel struct {
	ID       VoxelID    // Unique voxel identifier
	Bounds   AABB       // Axis-aligned bounding box
	Particles []Particle // Particles in this voxel
	Visible  bool       // Is voxel visible in frustum?
	LODLevel int        // Level of detail (0=full, 1=medium, 2=far, 3=culled)
}

// VoxelID represents a unique voxel identifier
type VoxelID struct {
	X int
	Y int
	Z int
}

// AABB represents an axis-aligned bounding box
type AABB struct {
	Min Vector3D
	Max Vector3D
}

// Camera represents the viewer's camera in 3D space
type Camera struct {
	Position Vector3D
	Target   Vector3D
	Up       Vector3D
	FOV      float64 // Field of view in degrees
	Near     float64 // Near clipping plane
	Far      float64 // Far clipping plane
}

// FrustumPlanes represents the 6 planes of a view frustum
type FrustumPlanes [6]Plane

// Plane represents a 3D plane (ax + by + cz + d = 0)
type Plane struct {
	A, B, C, D float64
}

// FASTQRead represents a single FASTQ read (4 lines)
type FASTQRead struct {
	Header   string // Line 1: @sequence_id
	Sequence string // Line 2: ATCGN...
	Plus     string // Line 3: + (optional description)
	Quality  string // Line 4: quality scores (ASCII encoded)
}

// BasePosition represents a base pair with its genomic position
type BasePosition struct {
	Position int    // Position in genome (0-indexed)
	Base     byte   // Base (A, T, G, C, N)
	Quality  byte   // Quality score (Phred+33 encoded)
	Coords   Vector3D // 3D coordinates (from digital root hashing)
}

// Batch represents a group of particles for GPU rendering
type Batch struct {
	VoxelID   VoxelID
	Particles []Particle
	Visible   bool
	LODLevel  int
}

// Mutation represents a genomic mutation
type Mutation struct {
	Position  int    // Genomic position
	Reference byte   // Reference base
	Alternate byte   // Mutated base
	Frequency float64 // Mutation frequency (local)
}

// GCContent represents GC content statistics
type GCContent struct {
	GCount     int
	CCount     int
	TotalBases int
	Percent    float64 // GC% (0-100)
}
