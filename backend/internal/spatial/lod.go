// Package spatial - Level of Detail (LOD) system
package spatial

import (
	"math"
	"genomevedic/backend/pkg/types"
)

// LODManager manages level-of-detail for particles
// Implements the 10× reduction strategy from RED_TEAM_FINDINGS.md
type LODManager struct {
	camera types.Camera
}

// NewLODManager creates a new LOD manager
func NewLODManager(camera types.Camera) *LODManager {
	return &LODManager{
		camera: camera,
	}
}

// ApplyLOD applies level-of-detail to visible voxels
// Returns particles for GPU upload (50K voxels → 5K effective batches)
func (lm *LODManager) ApplyLOD(voxels []*types.Voxel) []types.Particle {
	particles := make([]types.Particle, 0)

	for _, voxel := range voxels {
		// Compute distance from camera to voxel center
		distance := lm.distanceToVoxel(voxel)

		// Determine LOD level based on distance
		lodLevel := lm.getLODLevel(distance)
		voxel.LODLevel = lodLevel

		// Add particles based on LOD level
		particles = append(particles, lm.selectParticles(voxel, lodLevel)...)
	}

	return particles
}

// getLODLevel determines LOD level based on distance from camera
func (lm *LODManager) getLODLevel(distance float64) int {
	if distance < types.LODClose {
		return 0 // Full detail (100% particles)
	} else if distance < types.LODMedium {
		return 0 // Close (100% particles)
	} else if distance < types.LODFar {
		return 1 // Medium (50% particles)
	} else if distance < types.LODCulled {
		return 2 // Far (10% particles)
	} else {
		return 3 // Culled (0% particles)
	}
}

// selectParticles selects particles based on LOD level
func (lm *LODManager) selectParticles(voxel *types.Voxel, lodLevel int) []types.Particle {
	switch lodLevel {
	case 0:
		// Full detail - all particles
		return voxel.Particles

	case 1:
		// Medium detail - every other particle (50%)
		return selectEveryNth(voxel.Particles, 2)

	case 2:
		// Far detail - every 10th particle (10%)
		return selectEveryNth(voxel.Particles, 10)

	case 3:
		// Culled - no particles
		return []types.Particle{}

	default:
		return voxel.Particles
	}
}

// selectEveryNth selects every Nth particle from a slice
func selectEveryNth(particles []types.Particle, n int) []types.Particle {
	if n <= 1 {
		return particles
	}

	selected := make([]types.Particle, 0, len(particles)/n)
	for i := 0; i < len(particles); i += n {
		selected = append(selected, particles[i])
	}

	return selected
}

// distanceToVoxel computes distance from camera to voxel center
func (lm *LODManager) distanceToVoxel(voxel *types.Voxel) float64 {
	// Compute voxel center
	center := types.Vector3D{
		X: (voxel.Bounds.Min.X + voxel.Bounds.Max.X) / 2.0,
		Y: (voxel.Bounds.Min.Y + voxel.Bounds.Max.Y) / 2.0,
		Z: (voxel.Bounds.Min.Z + voxel.Bounds.Max.Z) / 2.0,
	}

	// Compute Euclidean distance
	dx := center.X - lm.camera.Position.X
	dy := center.Y - lm.camera.Position.Y
	dz := center.Z - lm.camera.Position.Z

	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// UpdateCamera updates the camera used for LOD calculations
func (lm *LODManager) UpdateCamera(camera types.Camera) {
	lm.camera = camera
}

// GetLODStats returns statistics about LOD distribution
func (lm *LODManager) GetLODStats(voxels []*types.Voxel) LODStats {
	stats := LODStats{
		TotalVoxels: len(voxels),
		Levels:      make(map[int]int),
	}

	for _, voxel := range voxels {
		stats.Levels[voxel.LODLevel]++
	}

	return stats
}

// LODStats contains statistics about LOD distribution
type LODStats struct {
	TotalVoxels int
	Levels      map[int]int // LOD level → count
}

// GetParticleReduction returns the reduction factor for a LOD level
func GetParticleReduction(lodLevel int) float64 {
	switch lodLevel {
	case 0:
		return 1.0 // 100%
	case 1:
		return 0.5 // 50%
	case 2:
		return 0.1 // 10%
	case 3:
		return 0.0 // 0%
	default:
		return 1.0
	}
}
