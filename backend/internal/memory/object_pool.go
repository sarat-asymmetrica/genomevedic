package memory

import (
	"sync"
)

// Particle represents a single genomic data point in 3D space
type Particle struct {
	Position [3]float32 // X, Y, Z coordinates
	Color    [4]float32 // R, G, B, A
	Size     float32    // Particle size
	Metadata uint64     // Genomic position or other metadata
}

// ParticleSlice wraps a slice of particles for pooling
type ParticleSlice struct {
	Data     []Particle
	Capacity int
	Length   int
}

// ParticlePool is a sync.Pool wrapper for particle slices
// Reduces GC pressure by reusing particle slice allocations
type ParticlePool struct {
	pool     sync.Pool
	capacity int
}

// NewParticlePool creates a new particle pool with specified capacity
func NewParticlePool(capacity int) *ParticlePool {
	return &ParticlePool{
		capacity: capacity,
		pool: sync.Pool{
			New: func() interface{} {
				return &ParticleSlice{
					Data:     make([]Particle, capacity),
					Capacity: capacity,
					Length:   0,
				}
			},
		},
	}
}

// Get retrieves a particle slice from the pool
func (pp *ParticlePool) Get() *ParticleSlice {
	ps := pp.pool.Get().(*ParticleSlice)
	ps.Length = 0 // Reset length but keep capacity
	return ps
}

// Put returns a particle slice to the pool
func (pp *ParticlePool) Put(ps *ParticleSlice) {
	if ps.Capacity != pp.capacity {
		// Don't pool slices with wrong capacity
		return
	}
	ps.Length = 0 // Reset for reuse
	pp.pool.Put(ps)
}

// VoxelData represents spatial indexing data for a voxel
type VoxelData struct {
	ParticleIndices []uint32 // Indices of particles in this voxel
	Count           int      // Number of particles
	Capacity        int      // Slice capacity
}

// VoxelPool is a sync.Pool for voxel data structures
type VoxelPool struct {
	pool     sync.Pool
	capacity int
}

// NewVoxelPool creates a new voxel pool with specified capacity
func NewVoxelPool(capacity int) *VoxelPool {
	return &VoxelPool{
		capacity: capacity,
		pool: sync.Pool{
			New: func() interface{} {
				return &VoxelData{
					ParticleIndices: make([]uint32, 0, capacity),
					Count:           0,
					Capacity:        capacity,
				}
			},
		},
	}
}

// Get retrieves voxel data from the pool
func (vp *VoxelPool) Get() *VoxelData {
	vd := vp.pool.Get().(*VoxelData)
	vd.ParticleIndices = vd.ParticleIndices[:0] // Reset length
	vd.Count = 0
	return vd
}

// Put returns voxel data to the pool
func (vp *VoxelPool) Put(vd *VoxelData) {
	if vd.Capacity != vp.capacity {
		return
	}
	vd.ParticleIndices = vd.ParticleIndices[:0]
	vd.Count = 0
	vp.pool.Put(vd)
}

// BufferPool is a sync.Pool for byte buffers
// Used for streaming data from disk to GPU
type BufferPool struct {
	pool sync.Pool
	size int
}

// NewBufferPool creates a new buffer pool with specified buffer size
func NewBufferPool(size int) *BufferPool {
	return &BufferPool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (bp *BufferPool) Get() []byte {
	return bp.pool.Get().([]byte)
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf []byte) {
	if len(buf) != bp.size {
		return // Don't pool buffers with wrong size
	}
	bp.pool.Put(buf)
}

// CoordinatePool is a sync.Pool for 3D coordinate slices
type CoordinatePool struct {
	pool     sync.Pool
	capacity int
}

// NewCoordinatePool creates a new coordinate pool
func NewCoordinatePool(capacity int) *CoordinatePool {
	return &CoordinatePool{
		capacity: capacity,
		pool: sync.Pool{
			New: func() interface{} {
				return make([][3]float32, capacity)
			},
		},
	}
}

// Get retrieves a coordinate slice from the pool
func (cp *CoordinatePool) Get() [][3]float32 {
	return cp.pool.Get().([][3]float32)
}

// Put returns a coordinate slice to the pool
func (cp *CoordinatePool) Put(coords [][3]float32) {
	if len(coords) != cp.capacity {
		return
	}
	cp.pool.Put(coords)
}

// Statistics tracks pool usage for monitoring
type Statistics struct {
	Gets       uint64
	Puts       uint64
	Allocations uint64
	Reuses     uint64
}

// MonitoredParticlePool is a particle pool with statistics
type MonitoredParticlePool struct {
	pool  *ParticlePool
	stats Statistics
	mu    sync.Mutex
}

// NewMonitoredParticlePool creates a monitored particle pool
func NewMonitoredParticlePool(capacity int) *MonitoredParticlePool {
	return &MonitoredParticlePool{
		pool: NewParticlePool(capacity),
	}
}

// Get retrieves a particle slice and updates stats
func (mpp *MonitoredParticlePool) Get() *ParticleSlice {
	mpp.mu.Lock()
	mpp.stats.Gets++
	mpp.mu.Unlock()
	return mpp.pool.Get()
}

// Put returns a particle slice and updates stats
func (mpp *MonitoredParticlePool) Put(ps *ParticleSlice) {
	mpp.mu.Lock()
	mpp.stats.Puts++
	mpp.stats.Reuses++
	mpp.mu.Unlock()
	mpp.pool.Put(ps)
}

// Stats returns current pool statistics
func (mpp *MonitoredParticlePool) Stats() Statistics {
	mpp.mu.Lock()
	defer mpp.mu.Unlock()
	return mpp.stats
}

// ResetStats resets pool statistics
func (mpp *MonitoredParticlePool) ResetStats() {
	mpp.mu.Lock()
	defer mpp.mu.Unlock()
	mpp.stats = Statistics{}
}
