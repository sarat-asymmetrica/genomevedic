package spatial

import (
	"fmt"
	"math"
	"sync"
)

// StreamingGrid manages dynamic voxel loading/unloading based on camera position
// Only keeps voxels in memory that are near the camera (streaming from disk as needed)
type StreamingGrid struct {
	mu sync.RWMutex

	// Voxel storage
	voxels map[VoxelKey]*CompactVoxel // Voxel coordinate → voxel
	pool   *VoxelPool                 // Pool for voxel allocation

	// Grid parameters
	voxelSize     float64 // Size of each voxel cube
	streamRadius  float64 // Distance from camera to load voxels
	unloadRadius  float64 // Distance from camera to unload voxels
	maxLoadedVoxels int   // Maximum number of voxels in memory

	// Camera tracking
	lastCameraPos [3]float64 // Last known camera position
	cameraMovedDistance float64 // Distance camera has moved since last update

	// Statistics
	stats StreamingGridStats
}

// VoxelKey is a unique identifier for a voxel based on its grid coordinates
type VoxelKey struct {
	X, Y, Z int32 // Grid coordinates
}

// StreamingGridStats tracks performance metrics
type StreamingGridStats struct {
	LoadedVoxels    int   // Current number of loaded voxels
	VisibleVoxels   int   // Number of visible voxels (after frustum culling)
	StreamedIn      int64 // Total voxels streamed from disk
	Evicted         int64 // Total voxels evicted from memory
	CacheHits       int64 // Voxel was already in memory
	CacheMisses     int64 // Voxel needed to be loaded from disk
	MemoryUsedBytes int64 // Total memory used by voxels
}

// NewStreamingGrid creates a new streaming grid
func NewStreamingGrid(voxelSize, streamRadius float64, maxLoadedVoxels int) *StreamingGrid {
	return &StreamingGrid{
		voxels:          make(map[VoxelKey]*CompactVoxel),
		pool:            GlobalVoxelPool,
		voxelSize:       voxelSize,
		streamRadius:    streamRadius,
		unloadRadius:    streamRadius * 1.5, // Unload at 1.5× stream radius
		maxLoadedVoxels: maxLoadedVoxels,
		lastCameraPos:   [3]float64{0, 0, 0},
		stats:           StreamingGridStats{},
	}
}

// WorldToVoxelKey converts world coordinates to a voxel key
func (sg *StreamingGrid) WorldToVoxelKey(x, y, z float64) VoxelKey {
	return VoxelKey{
		X: int32(math.Floor(x / sg.voxelSize)),
		Y: int32(math.Floor(y / sg.voxelSize)),
		Z: int32(math.Floor(z / sg.voxelSize)),
	}
}

// VoxelKeyToWorldBounds converts a voxel key to world space bounds
func (sg *StreamingGrid) VoxelKeyToWorldBounds(key VoxelKey) (minX, minY, minZ, maxX, maxY, maxZ float64) {
	minX = float64(key.X) * sg.voxelSize
	minY = float64(key.Y) * sg.voxelSize
	minZ = float64(key.Z) * sg.voxelSize
	maxX = minX + sg.voxelSize
	maxY = minY + sg.voxelSize
	maxZ = minZ + sg.voxelSize
	return
}

// UpdateCamera updates the camera position and streams voxels as needed
func (sg *StreamingGrid) UpdateCamera(cameraX, cameraY, cameraZ float64) error {
	// Calculate distance moved since last update
	dx := cameraX - sg.lastCameraPos[0]
	dy := cameraY - sg.lastCameraPos[1]
	dz := cameraZ - sg.lastCameraPos[2]
	distanceMoved := math.Sqrt(dx*dx + dy*dy + dz*dz)

	// Only update if camera moved significantly (optimization)
	if distanceMoved < sg.voxelSize*0.5 {
		return nil // Camera hasn't moved much, skip update
	}

	sg.mu.Lock()
	defer sg.mu.Unlock()

	// Update camera position
	sg.lastCameraPos = [3]float64{cameraX, cameraY, cameraZ}
	sg.cameraMovedDistance = distanceMoved

	// Step 1: Unload distant voxels
	sg.unloadDistantVoxels(cameraX, cameraY, cameraZ)

	// Step 2: Load nearby voxels
	sg.loadNearbyVoxels(cameraX, cameraY, cameraZ)

	// Step 3: Enforce memory budget
	sg.enforceMemoryBudget(cameraX, cameraY, cameraZ)

	return nil
}

// unloadDistantVoxels evicts voxels that are too far from the camera
func (sg *StreamingGrid) unloadDistantVoxels(cameraX, cameraY, cameraZ float64) {
	unloadRadiusSquared := sg.unloadRadius * sg.unloadRadius

	for key, voxel := range sg.voxels {
		// Get voxel center
		center := voxel.GetCenter()

		// Calculate distance to camera
		dx := float64(center[0]) - cameraX
		dy := float64(center[1]) - cameraY
		dz := float64(center[2]) - cameraZ
		distanceSquared := dx*dx + dy*dy + dz*dz

		// Unload if too far
		if distanceSquared > unloadRadiusSquared {
			sg.pool.Put(voxel)
			delete(sg.voxels, key)
			sg.stats.Evicted++
		}
	}

	sg.stats.LoadedVoxels = len(sg.voxels)
	sg.stats.MemoryUsedBytes = int64(len(sg.voxels) * 32)
}

// loadNearbyVoxels loads voxels that are within stream radius of camera
func (sg *StreamingGrid) loadNearbyVoxels(cameraX, cameraY, cameraZ float64) {
	// Calculate grid extent to load (sphere around camera)
	gridRadius := int32(math.Ceil(sg.streamRadius / sg.voxelSize))

	// Get camera's voxel coordinate
	cameraKey := sg.WorldToVoxelKey(cameraX, cameraY, cameraZ)

	// Iterate over voxels in a cube around camera (will filter to sphere)
	for dx := -gridRadius; dx <= gridRadius; dx++ {
		for dy := -gridRadius; dy <= gridRadius; dy++ {
			for dz := -gridRadius; dz <= gridRadius; dz++ {
				key := VoxelKey{
					X: cameraKey.X + dx,
					Y: cameraKey.Y + dy,
					Z: cameraKey.Z + dz,
				}

				// Check if voxel is already loaded (cache hit)
				if _, exists := sg.voxels[key]; exists {
					sg.stats.CacheHits++
					continue
				}

				// Check if voxel is within stream radius (sphere test)
				minX, minY, minZ, maxX, maxY, maxZ := sg.VoxelKeyToWorldBounds(key)
				centerX := (minX + maxX) * 0.5
				centerY := (minY + maxY) * 0.5
				centerZ := (minZ + maxZ) * 0.5

				dx := centerX - cameraX
				dy := centerY - cameraY
				dz := centerZ - cameraZ
				distanceSquared := dx*dx + dy*dy + dz*dz

				if distanceSquared <= sg.streamRadius*sg.streamRadius {
					// Load voxel from pool (in real implementation, would stream from disk)
					voxel := sg.pool.GetWithBounds(minX, minY, minZ, maxX, maxY, maxZ)
					voxel.SetStreaming(true) // Mark as streaming

					// In real implementation, would load particle data from disk here
					// For now, just create empty voxel

					voxel.SetStreaming(false) // Mark as loaded
					sg.voxels[key] = voxel
					sg.stats.StreamedIn++
					sg.stats.CacheMisses++
				}
			}
		}
	}

	sg.stats.LoadedVoxels = len(sg.voxels)
	sg.stats.MemoryUsedBytes = int64(len(sg.voxels) * 32)
}

// enforceMemoryBudget evicts voxels if over memory limit
func (sg *StreamingGrid) enforceMemoryBudget(cameraX, cameraY, cameraZ float64) {
	if len(sg.voxels) <= sg.maxLoadedVoxels {
		return // Within budget
	}

	// Evict farthest voxels first
	type voxelDistance struct {
		key      VoxelKey
		distance float64
	}

	distances := make([]voxelDistance, 0, len(sg.voxels))
	for key, voxel := range sg.voxels {
		center := voxel.GetCenter()
		dx := float64(center[0]) - cameraX
		dy := float64(center[1]) - cameraY
		dz := float64(center[2]) - cameraZ
		distance := math.Sqrt(dx*dx + dy*dy + dz*dz)

		distances = append(distances, voxelDistance{key, distance})
	}

	// Sort by distance (descending)
	for i := 0; i < len(distances); i++ {
		for j := i + 1; j < len(distances); j++ {
			if distances[i].distance < distances[j].distance {
				distances[i], distances[j] = distances[j], distances[i]
			}
		}
	}

	// Evict until within budget
	toEvict := len(sg.voxels) - sg.maxLoadedVoxels
	for i := 0; i < toEvict && i < len(distances); i++ {
		key := distances[i].key
		voxel := sg.voxels[key]
		voxel.SetEvicted(true)
		sg.pool.Put(voxel)
		delete(sg.voxels, key)
		sg.stats.Evicted++
	}

	sg.stats.LoadedVoxels = len(sg.voxels)
	sg.stats.MemoryUsedBytes = int64(len(sg.voxels) * 32)
}

// GetVoxel returns a voxel at the given world coordinates (or nil if not loaded)
func (sg *StreamingGrid) GetVoxel(x, y, z float64) *CompactVoxel {
	sg.mu.RLock()
	defer sg.mu.RUnlock()

	key := sg.WorldToVoxelKey(x, y, z)
	return sg.voxels[key]
}

// GetVoxelByKey returns a voxel by its key (or nil if not loaded)
func (sg *StreamingGrid) GetVoxelByKey(key VoxelKey) *CompactVoxel {
	sg.mu.RLock()
	defer sg.mu.RUnlock()
	return sg.voxels[key]
}

// GetLoadedVoxels returns all currently loaded voxels
func (sg *StreamingGrid) GetLoadedVoxels() []*CompactVoxel {
	sg.mu.RLock()
	defer sg.mu.RUnlock()

	voxels := make([]*CompactVoxel, 0, len(sg.voxels))
	for _, v := range sg.voxels {
		voxels = append(voxels, v)
	}
	return voxels
}

// GetVisibleVoxels returns voxels marked as visible (after frustum culling)
func (sg *StreamingGrid) GetVisibleVoxels() []*CompactVoxel {
	sg.mu.RLock()
	defer sg.mu.RUnlock()

	visible := make([]*CompactVoxel, 0, len(sg.voxels)/10)
	for _, v := range sg.voxels {
		if v.IsVisible() {
			visible = append(visible, v)
		}
	}

	sg.stats.VisibleVoxels = len(visible)
	return visible
}

// GetStats returns streaming grid statistics
func (sg *StreamingGrid) GetStats() StreamingGridStats {
	sg.mu.RLock()
	defer sg.mu.RUnlock()
	return sg.stats
}

// Clear removes all voxels from the grid
func (sg *StreamingGrid) Clear() {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	// Return all voxels to pool
	for _, v := range sg.voxels {
		sg.pool.Put(v)
	}

	sg.voxels = make(map[VoxelKey]*CompactVoxel)
	sg.stats = StreamingGridStats{}
}

// PrintStats prints streaming grid statistics
func (sg *StreamingGrid) PrintStats() {
	stats := sg.GetStats()
	fmt.Printf("\nStreaming Grid Statistics:\n")
	fmt.Printf("  Loaded voxels:     %d / %d max\n", stats.LoadedVoxels, sg.maxLoadedVoxels)
	fmt.Printf("  Visible voxels:    %d (%.1f%%)\n", stats.VisibleVoxels,
		float64(stats.VisibleVoxels)/float64(stats.LoadedVoxels+1)*100)
	fmt.Printf("  Memory used:       %.2f MB\n", float64(stats.MemoryUsedBytes)/(1024*1024))
	fmt.Printf("  Streamed in:       %d voxels\n", stats.StreamedIn)
	fmt.Printf("  Evicted:           %d voxels\n", stats.Evicted)
	fmt.Printf("  Cache hits:        %d (%.1f%%)\n", stats.CacheHits,
		float64(stats.CacheHits)/float64(stats.CacheHits+stats.CacheMisses+1)*100)
	fmt.Printf("  Cache misses:      %d\n", stats.CacheMisses)
}

// CalculateOptimalVoxelSize calculates the optimal voxel size for a given genome size
// Uses Williams formula: √n × log₂(n) to determine batch size
func CalculateOptimalVoxelSize(genomeSize int64, targetParticlesPerVoxel int) float64 {
	// Estimate number of voxels needed
	estimatedVoxels := genomeSize / int64(targetParticlesPerVoxel)

	// Calculate optimal batch size using Williams formula
	n := float64(estimatedVoxels)
	optimalBatchCount := math.Sqrt(n) * math.Log2(n)

	// Calculate voxel size based on batch count
	// Assuming genome spans a cube of size genomeSpan
	genomeSpan := math.Cbrt(float64(genomeSize))
	voxelSize := genomeSpan / math.Cbrt(optimalBatchCount)

	return voxelSize
}

// EstimateMemoryUsage estimates the memory usage for a given configuration
func EstimateMemoryUsage(totalParticles int64, voxelSize float64, particlesPerVoxel int) (voxelMemoryMB, particleMemoryMB, totalMemoryMB float64) {
	// Calculate number of voxels
	voxelCount := totalParticles / int64(particlesPerVoxel)

	// Voxel memory (32 bytes per compact voxel)
	voxelMemoryBytes := voxelCount * 32
	voxelMemoryMB = float64(voxelMemoryBytes) / (1024 * 1024)

	// Particle memory (24 bytes per particle: pos=12, color=4, size=4, metadata=4)
	particleMemoryBytes := totalParticles * 24
	particleMemoryMB = float64(particleMemoryBytes) / (1024 * 1024)

	totalMemoryMB = voxelMemoryMB + particleMemoryMB
	return
}
