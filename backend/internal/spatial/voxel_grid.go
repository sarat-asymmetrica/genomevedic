// Package spatial - Spatial indexing and voxel grid management
package spatial

import (
	"math"
	"genomevedic/backend/pkg/types"
)

// VoxelGrid represents a 3D spatial grid for O(1) particle lookups
// Implements the corrected architecture from RED_TEAM_FINDINGS.md
type VoxelGrid struct {
	voxels    map[types.VoxelID]*types.Voxel
	voxelSize float64
	bounds    types.AABB
	totalVoxels int
}

// NewVoxelGrid creates a new voxel grid
func NewVoxelGrid(voxelSize float64) *VoxelGrid {
	return &VoxelGrid{
		voxels:      make(map[types.VoxelID]*types.Voxel),
		voxelSize:   voxelSize,
		bounds:      types.AABB{}, // Will be computed as particles are added
		totalVoxels: 0,
	}
}

// Insert adds a particle to the voxel grid
func (vg *VoxelGrid) Insert(particle types.Particle) {
	// Compute voxel ID using spatial hash
	voxelID := SpatialHash(particle.Position, vg.voxelSize)

	// Get or create voxel
	voxel, exists := vg.voxels[voxelID]
	if !exists {
		// Create new voxel
		voxel = &types.Voxel{
			ID:       voxelID,
			Bounds:   computeVoxelBounds(voxelID, vg.voxelSize),
			Particles: make([]types.Particle, 0, 100), // Pre-allocate for ~100 particles
			Visible:  false,
			LODLevel: 0,
		}
		vg.voxels[voxelID] = voxel
		vg.totalVoxels++
	}

	// Add particle to voxel
	voxel.Particles = append(voxel.Particles, particle)

	// Update grid bounds
	vg.updateBounds(particle.Position)
}

// Query retrieves all particles in a specific voxel
func (vg *VoxelGrid) Query(voxelID types.VoxelID) []types.Particle {
	voxel, exists := vg.voxels[voxelID]
	if !exists {
		return []types.Particle{}
	}
	return voxel.Particles
}

// RangeQuery retrieves all particles in a bounding box
func (vg *VoxelGrid) RangeQuery(min, max types.Vector3D) []types.Particle {
	particles := make([]types.Particle, 0)

	// Compute voxel range
	minVoxel := SpatialHash(min, vg.voxelSize)
	maxVoxel := SpatialHash(max, vg.voxelSize)

	// Iterate through voxels in range
	for x := minVoxel.X; x <= maxVoxel.X; x++ {
		for y := minVoxel.Y; y <= maxVoxel.Y; y++ {
			for z := minVoxel.Z; z <= maxVoxel.Z; z++ {
				voxelID := types.VoxelID{X: x, Y: y, Z: z}
				voxelParticles := vg.Query(voxelID)
				particles = append(particles, voxelParticles...)
			}
		}
	}

	return particles
}

// GetAllVoxels returns all voxels in the grid
func (vg *VoxelGrid) GetAllVoxels() []*types.Voxel {
	voxels := make([]*types.Voxel, 0, len(vg.voxels))
	for _, voxel := range vg.voxels {
		voxels = append(voxels, voxel)
	}
	return voxels
}

// GetTotalVoxels returns the total number of voxels
func (vg *VoxelGrid) GetTotalVoxels() int {
	return vg.totalVoxels
}

// GetBounds returns the bounding box of all particles
func (vg *VoxelGrid) GetBounds() types.AABB {
	return vg.bounds
}

// updateBounds expands the grid bounds to include a new position
func (vg *VoxelGrid) updateBounds(pos types.Vector3D) {
	if vg.totalVoxels == 1 {
		// First particle - initialize bounds
		vg.bounds = types.AABB{
			Min: pos,
			Max: pos,
		}
		return
	}

	// Expand bounds
	vg.bounds.Min.X = math.Min(vg.bounds.Min.X, pos.X)
	vg.bounds.Min.Y = math.Min(vg.bounds.Min.Y, pos.Y)
	vg.bounds.Min.Z = math.Min(vg.bounds.Min.Z, pos.Z)

	vg.bounds.Max.X = math.Max(vg.bounds.Max.X, pos.X)
	vg.bounds.Max.Y = math.Max(vg.bounds.Max.Y, pos.Y)
	vg.bounds.Max.Z = math.Max(vg.bounds.Max.Z, pos.Z)
}

// SpatialHash computes the voxel ID for a 3D position
// This is O(1) and deterministic
func SpatialHash(pos types.Vector3D, voxelSize float64) types.VoxelID {
	return types.VoxelID{
		X: int(math.Floor(pos.X / voxelSize)),
		Y: int(math.Floor(pos.Y / voxelSize)),
		Z: int(math.Floor(pos.Z / voxelSize)),
	}
}

// computeVoxelBounds computes the AABB for a voxel
func computeVoxelBounds(voxelID types.VoxelID, voxelSize float64) types.AABB {
	min := types.Vector3D{
		X: float64(voxelID.X) * voxelSize,
		Y: float64(voxelID.Y) * voxelSize,
		Z: float64(voxelID.Z) * voxelSize,
	}

	max := types.Vector3D{
		X: min.X + voxelSize,
		Y: min.Y + voxelSize,
		Z: min.Z + voxelSize,
	}

	return types.AABB{
		Min: min,
		Max: max,
	}
}

// GetVoxelSize returns the voxel size
func (vg *VoxelGrid) GetVoxelSize() float64 {
	return vg.voxelSize
}
