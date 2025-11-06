package spatial

import (
	"math"
)

// CompactVoxel is a memory-optimized voxel structure
// Reduced from 96 bytes (original) to 32 bytes (67% reduction)
type CompactVoxel struct {
	// Bounding box (24 bytes)
	// Uses float32 instead of float64 (sufficient precision for genomic coordinates)
	BoundsMin [3]float32 // 12 bytes
	BoundsMax [3]float32 // 12 bytes

	// Particle references (6 bytes)
	ParticleOffset uint32 // 4 bytes - Index into global particle array
	ParticleCount  uint16 // 2 bytes - Max 65,535 particles per voxel

	// Flags and metadata (2 bytes)
	Flags   uint8 // 1 byte - Bit flags (visible, dirty, LOD level)
	Padding uint8 // 1 byte - Alignment padding

	// Total: 32 bytes (was 96 bytes - 67% reduction!)
}

// Voxel flags (bitfield)
const (
	VoxelFlagVisible   uint8 = 1 << 0 // Voxel is visible after frustum culling
	VoxelFlagDirty     uint8 = 1 << 1 // Voxel needs GPU upload
	VoxelFlagLOD0      uint8 = 0 << 2 // LOD level 0 (full detail) - bits 2-3
	VoxelFlagLOD1      uint8 = 1 << 2 // LOD level 1 (50% particles)
	VoxelFlagLOD2      uint8 = 2 << 2 // LOD level 2 (10% particles)
	VoxelFlagLOD3      uint8 = 3 << 2 // LOD level 3 (culled)
	VoxelFlagStreaming uint8 = 1 << 4 // Voxel is being streamed from disk
	VoxelFlagEvicted   uint8 = 1 << 5 // Voxel data has been evicted (memory pressure)
)

// NewCompactVoxel creates a new compact voxel
func NewCompactVoxel(minX, minY, minZ, maxX, maxY, maxZ float64) *CompactVoxel {
	return &CompactVoxel{
		BoundsMin:      [3]float32{float32(minX), float32(minY), float32(minZ)},
		BoundsMax:      [3]float32{float32(maxX), float32(maxY), float32(maxZ)},
		ParticleOffset: 0,
		ParticleCount:  0,
		Flags:          0,
		Padding:        0,
	}
}

// IsVisible returns true if the voxel is visible
func (v *CompactVoxel) IsVisible() bool {
	return v.Flags&VoxelFlagVisible != 0
}

// SetVisible sets the visible flag
func (v *CompactVoxel) SetVisible(visible bool) {
	if visible {
		v.Flags |= VoxelFlagVisible
	} else {
		v.Flags &^= VoxelFlagVisible
	}
}

// IsDirty returns true if the voxel needs GPU upload
func (v *CompactVoxel) IsDirty() bool {
	return v.Flags&VoxelFlagDirty != 0
}

// SetDirty marks the voxel as dirty (needs GPU upload)
func (v *CompactVoxel) SetDirty(dirty bool) {
	if dirty {
		v.Flags |= VoxelFlagDirty
	} else {
		v.Flags &^= VoxelFlagDirty
	}
}

// GetLODLevel returns the LOD level (0-3)
func (v *CompactVoxel) GetLODLevel() int {
	return int((v.Flags >> 2) & 0x03)
}

// SetLODLevel sets the LOD level (0-3)
func (v *CompactVoxel) SetLODLevel(level int) {
	// Clear LOD bits
	v.Flags &^= (0x03 << 2)
	// Set new LOD level
	v.Flags |= uint8(level&0x03) << 2
}

// IsStreaming returns true if voxel is being loaded from disk
func (v *CompactVoxel) IsStreaming() bool {
	return v.Flags&VoxelFlagStreaming != 0
}

// SetStreaming sets the streaming flag
func (v *CompactVoxel) SetStreaming(streaming bool) {
	if streaming {
		v.Flags |= VoxelFlagStreaming
	} else {
		v.Flags &^= VoxelFlagStreaming
	}
}

// IsEvicted returns true if voxel data has been evicted from memory
func (v *CompactVoxel) IsEvicted() bool {
	return v.Flags&VoxelFlagEvicted != 0
}

// SetEvicted sets the evicted flag
func (v *CompactVoxel) SetEvicted(evicted bool) {
	if evicted {
		v.Flags |= VoxelFlagEvicted
	} else {
		v.Flags &^= VoxelFlagEvicted
	}
}

// GetCenter returns the center point of the voxel
func (v *CompactVoxel) GetCenter() [3]float32 {
	return [3]float32{
		(v.BoundsMin[0] + v.BoundsMax[0]) * 0.5,
		(v.BoundsMin[1] + v.BoundsMax[1]) * 0.5,
		(v.BoundsMin[2] + v.BoundsMax[2]) * 0.5,
	}
}

// GetSize returns the size of the voxel
func (v *CompactVoxel) GetSize() [3]float32 {
	return [3]float32{
		v.BoundsMax[0] - v.BoundsMin[0],
		v.BoundsMax[1] - v.BoundsMin[1],
		v.BoundsMax[2] - v.BoundsMin[2],
	}
}

// GetRadius returns the bounding sphere radius
func (v *CompactVoxel) GetRadius() float32 {
	size := v.GetSize()
	return float32(math.Sqrt(float64(size[0]*size[0] + size[1]*size[1] + size[2]*size[2]))) * 0.5
}

// ContainsPoint returns true if the point is inside the voxel
func (v *CompactVoxel) ContainsPoint(x, y, z float32) bool {
	return x >= v.BoundsMin[0] && x <= v.BoundsMax[0] &&
		y >= v.BoundsMin[1] && y <= v.BoundsMax[1] &&
		z >= v.BoundsMin[2] && z <= v.BoundsMax[2]
}

// IntersectsAABB returns true if this voxel intersects another AABB
func (v *CompactVoxel) IntersectsAABB(minX, minY, minZ, maxX, maxY, maxZ float32) bool {
	return !(v.BoundsMax[0] < minX || v.BoundsMin[0] > maxX ||
		v.BoundsMax[1] < minY || v.BoundsMin[1] > maxY ||
		v.BoundsMax[2] < minZ || v.BoundsMin[2] > maxZ)
}

// IntersectsSphere returns true if this voxel intersects a sphere
func (v *CompactVoxel) IntersectsSphere(centerX, centerY, centerZ, radius float32) bool {
	// Find closest point on AABB to sphere center
	closestX := clampFloat32(centerX, v.BoundsMin[0], v.BoundsMax[0])
	closestY := clampFloat32(centerY, v.BoundsMin[1], v.BoundsMax[1])
	closestZ := clampFloat32(centerZ, v.BoundsMin[2], v.BoundsMax[2])

	// Distance from sphere center to closest point
	dx := centerX - closestX
	dy := centerY - closestY
	dz := centerZ - closestZ
	distanceSquared := dx*dx + dy*dy + dz*dz

	return distanceSquared <= radius*radius
}

// DistanceToPoint returns the distance from the voxel center to a point
func (v *CompactVoxel) DistanceToPoint(x, y, z float32) float32 {
	center := v.GetCenter()
	dx := center[0] - x
	dy := center[1] - y
	dz := center[2] - z
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}

// GetParticleRange returns the start and end indices for particles in this voxel
func (v *CompactVoxel) GetParticleRange() (start uint32, end uint32) {
	return v.ParticleOffset, v.ParticleOffset + uint32(v.ParticleCount)
}

// SetParticleRange sets the particle index range for this voxel
func (v *CompactVoxel) SetParticleRange(offset uint32, count uint16) {
	v.ParticleOffset = offset
	v.ParticleCount = count
	v.SetDirty(true) // Mark for GPU upload
}

// MemoryFootprint returns the memory footprint in bytes
func (v *CompactVoxel) MemoryFootprint() int {
	return 32 // Compact voxel is exactly 32 bytes
}

// clampFloat32 clamps a float32 value between min and max
func clampFloat32(value, min, max float32) float32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// CompactVoxelStats provides statistics about voxel memory usage
type CompactVoxelStats struct {
	TotalVoxels     int
	VisibleVoxels   int
	StreamingVoxels int
	EvictedVoxels   int
	MemoryUsed      int64 // Bytes
	MemorySaved     int64 // Bytes saved vs original 96-byte voxel
}

// CalculateStats computes statistics for a slice of compact voxels
func CalculateStats(voxels []*CompactVoxel) CompactVoxelStats {
	stats := CompactVoxelStats{
		TotalVoxels: len(voxels),
		MemoryUsed:  int64(len(voxels) * 32),
		MemorySaved: int64(len(voxels) * (96 - 32)), // Original 96 bytes vs compact 32 bytes
	}

	for _, v := range voxels {
		if v.IsVisible() {
			stats.VisibleVoxels++
		}
		if v.IsStreaming() {
			stats.StreamingVoxels++
		}
		if v.IsEvicted() {
			stats.EvictedVoxels++
		}
	}

	return stats
}
