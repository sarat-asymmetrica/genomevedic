package spatial

import (
	"sync"
	"sync/atomic"
)

// VoxelPool manages a pool of reusable CompactVoxel instances
// Uses sync.Pool for zero-GC-pressure allocation
type VoxelPool struct {
	pool sync.Pool

	// Statistics
	allocations int64 // Total allocations (new voxels created)
	reuses      int64 // Total reuses (voxels from pool)
	puts        int64 // Total returns to pool
}

// NewVoxelPool creates a new voxel pool
func NewVoxelPool() *VoxelPool {
	return &VoxelPool{
		pool: sync.Pool{
			New: func() interface{} {
				// Allocate new voxel when pool is empty
				return &CompactVoxel{}
			},
		},
		allocations: 0,
		reuses:      0,
		puts:        0,
	}
}

// Get retrieves a voxel from the pool (or creates a new one if pool is empty)
func (vp *VoxelPool) Get() *CompactVoxel {
	v := vp.pool.Get().(*CompactVoxel)

	// Reset voxel state
	v.BoundsMin = [3]float32{0, 0, 0}
	v.BoundsMax = [3]float32{0, 0, 0}
	v.ParticleOffset = 0
	v.ParticleCount = 0
	v.Flags = 0
	v.Padding = 0

	// Track statistics
	if v == nil {
		atomic.AddInt64(&vp.allocations, 1)
	} else {
		atomic.AddInt64(&vp.reuses, 1)
	}

	return v
}

// GetWithBounds retrieves a voxel from the pool and initializes its bounds
func (vp *VoxelPool) GetWithBounds(minX, minY, minZ, maxX, maxY, maxZ float64) *CompactVoxel {
	v := vp.Get()
	v.BoundsMin = [3]float32{float32(minX), float32(minY), float32(minZ)}
	v.BoundsMax = [3]float32{float32(maxX), float32(maxY), float32(maxZ)}
	return v
}

// Put returns a voxel to the pool for reuse
func (vp *VoxelPool) Put(v *CompactVoxel) {
	if v == nil {
		return
	}

	// Clear sensitive data before returning to pool
	v.ParticleOffset = 0
	v.ParticleCount = 0
	v.Flags = 0

	vp.pool.Put(v)
	atomic.AddInt64(&vp.puts, 1)
}

// PutBatch returns multiple voxels to the pool
func (vp *VoxelPool) PutBatch(voxels []*CompactVoxel) {
	for _, v := range voxels {
		vp.Put(v)
	}
}

// GetStats returns pool usage statistics
func (vp *VoxelPool) GetStats() VoxelPoolStats {
	return VoxelPoolStats{
		Allocations: atomic.LoadInt64(&vp.allocations),
		Reuses:      atomic.LoadInt64(&vp.reuses),
		Puts:        atomic.LoadInt64(&vp.puts),
		ReuseRate:   vp.calculateReuseRate(),
	}
}

// calculateReuseRate computes the percentage of Gets that reused pooled voxels
func (vp *VoxelPool) calculateReuseRate() float64 {
	reuses := atomic.LoadInt64(&vp.reuses)
	allocations := atomic.LoadInt64(&vp.allocations)
	total := reuses + allocations

	if total == 0 {
		return 0.0
	}

	return float64(reuses) / float64(total) * 100.0
}

// VoxelPoolStats contains statistics about pool usage
type VoxelPoolStats struct {
	Allocations int64   // Number of new voxels created
	Reuses      int64   // Number of voxels reused from pool
	Puts        int64   // Number of voxels returned to pool
	ReuseRate   float64 // Percentage of Gets that reused pooled voxels
}

// GlobalVoxelPool is a singleton voxel pool for the entire application
var GlobalVoxelPool = NewVoxelPool()

// VoxelBatch represents a batch of voxels for bulk operations
type VoxelBatch struct {
	voxels []*CompactVoxel
	pool   *VoxelPool
}

// NewVoxelBatch creates a new batch of voxels
func NewVoxelBatch(capacity int, pool *VoxelPool) *VoxelBatch {
	return &VoxelBatch{
		voxels: make([]*CompactVoxel, 0, capacity),
		pool:   pool,
	}
}

// Add adds a voxel to the batch
func (vb *VoxelBatch) Add(v *CompactVoxel) {
	vb.voxels = append(vb.voxels, v)
}

// Len returns the number of voxels in the batch
func (vb *VoxelBatch) Len() int {
	return len(vb.voxels)
}

// Get returns the voxel at index i
func (vb *VoxelBatch) Get(i int) *CompactVoxel {
	return vb.voxels[i]
}

// GetAll returns all voxels in the batch
func (vb *VoxelBatch) GetAll() []*CompactVoxel {
	return vb.voxels
}

// Clear returns all voxels to the pool and resets the batch
func (vb *VoxelBatch) Clear() {
	if vb.pool != nil {
		vb.pool.PutBatch(vb.voxels)
	}
	vb.voxels = vb.voxels[:0]
}

// FilterVisible returns a new batch containing only visible voxels
func (vb *VoxelBatch) FilterVisible() *VoxelBatch {
	filtered := NewVoxelBatch(vb.Len()/10, vb.pool) // Estimate 10% visible
	for _, v := range vb.voxels {
		if v.IsVisible() {
			filtered.Add(v)
		}
	}
	return filtered
}

// FilterByLOD returns a new batch containing only voxels at the specified LOD level
func (vb *VoxelBatch) FilterByLOD(level int) *VoxelBatch {
	filtered := NewVoxelBatch(vb.Len()/4, vb.pool)
	for _, v := range vb.voxels {
		if v.GetLODLevel() == level {
			filtered.Add(v)
		}
	}
	return filtered
}

// SortByDistance sorts voxels by distance to a camera position (for LOD)
// Uses quicksort with in-place partitioning (no extra allocations)
func (vb *VoxelBatch) SortByDistance(cameraX, cameraY, cameraZ float32) {
	vb.quicksortByDistance(0, len(vb.voxels)-1, cameraX, cameraY, cameraZ)
}

func (vb *VoxelBatch) quicksortByDistance(low, high int, cx, cy, cz float32) {
	if low < high {
		pivot := vb.partitionByDistance(low, high, cx, cy, cz)
		vb.quicksortByDistance(low, pivot-1, cx, cy, cz)
		vb.quicksortByDistance(pivot+1, high, cx, cy, cz)
	}
}

func (vb *VoxelBatch) partitionByDistance(low, high int, cx, cy, cz float32) int {
	pivot := vb.voxels[high].DistanceToPoint(cx, cy, cz)
	i := low - 1

	for j := low; j < high; j++ {
		if vb.voxels[j].DistanceToPoint(cx, cy, cz) <= pivot {
			i++
			vb.voxels[i], vb.voxels[j] = vb.voxels[j], vb.voxels[i]
		}
	}

	vb.voxels[i+1], vb.voxels[high] = vb.voxels[high], vb.voxels[i+1]
	return i + 1
}

// CalculateMemoryFootprint returns the total memory used by this batch
func (vb *VoxelBatch) CalculateMemoryFootprint() int64 {
	return int64(len(vb.voxels) * 32) // 32 bytes per compact voxel
}

// VoxelAllocator manages voxel allocation with memory budgets
type VoxelAllocator struct {
	pool        *VoxelPool
	maxVoxels   int   // Maximum number of voxels allowed
	allocatedID int64 // Monotonically increasing voxel ID

	mu             sync.RWMutex
	activeVoxels   map[int64]*CompactVoxel // Voxel ID → voxel
	evictionPolicy EvictionPolicy          // Policy for evicting voxels under memory pressure
}

// EvictionPolicy defines how voxels are evicted under memory pressure
type EvictionPolicy int

const (
	EvictLRU      EvictionPolicy = 0 // Least Recently Used
	EvictFarthest EvictionPolicy = 1 // Farthest from camera
	EvictLOD      EvictionPolicy = 2 // Evict higher LOD levels first
)

// NewVoxelAllocator creates a new voxel allocator with a memory budget
func NewVoxelAllocator(maxVoxels int, policy EvictionPolicy) *VoxelAllocator {
	return &VoxelAllocator{
		pool:           GlobalVoxelPool,
		maxVoxels:      maxVoxels,
		allocatedID:    0,
		activeVoxels:   make(map[int64]*CompactVoxel),
		evictionPolicy: policy,
	}
}

// Allocate allocates a new voxel (or evicts an old one if over budget)
func (va *VoxelAllocator) Allocate(minX, minY, minZ, maxX, maxY, maxZ float64) (int64, *CompactVoxel) {
	va.mu.Lock()
	defer va.mu.Unlock()

	// Check if we need to evict
	if len(va.activeVoxels) >= va.maxVoxels {
		va.evictOne()
	}

	// Allocate new voxel
	id := atomic.AddInt64(&va.allocatedID, 1)
	v := va.pool.GetWithBounds(minX, minY, minZ, maxX, maxY, maxZ)
	va.activeVoxels[id] = v

	return id, v
}

// Free frees a voxel by ID
func (va *VoxelAllocator) Free(id int64) {
	va.mu.Lock()
	defer va.mu.Unlock()

	if v, exists := va.activeVoxels[id]; exists {
		va.pool.Put(v)
		delete(va.activeVoxels, id)
	}
}

// evictOne evicts a single voxel based on the eviction policy
func (va *VoxelAllocator) evictOne() {
	// For now, evict the first voxel (LRU would track access times)
	for id, v := range va.activeVoxels {
		v.SetEvicted(true)
		va.pool.Put(v)
		delete(va.activeVoxels, id)
		return
	}
}

// GetMemoryUsage returns the current memory usage in bytes
func (va *VoxelAllocator) GetMemoryUsage() int64 {
	va.mu.RLock()
	defer va.mu.RUnlock()
	return int64(len(va.activeVoxels) * 32)
}

// GetActiveCount returns the number of active voxels
func (va *VoxelAllocator) GetActiveCount() int {
	va.mu.RLock()
	defer va.mu.RUnlock()
	return len(va.activeVoxels)
}
