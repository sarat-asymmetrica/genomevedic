package memory

import (
	"fmt"
	"sync"
)

// Arena is a memory arena allocator
// Allocates a large chunk of memory upfront and hands out slices
// Reduces GC pressure by avoiding many small allocations
type Arena struct {
	buffer   []byte
	offset   int
	capacity int
	mu       sync.Mutex
}

// NewArena creates a new memory arena with specified capacity in bytes
func NewArena(capacity int) *Arena {
	return &Arena{
		buffer:   make([]byte, capacity),
		offset:   0,
		capacity: capacity,
	}
}

// Alloc allocates a byte slice of specified size from the arena
// Returns nil if arena is full
func (a *Arena) Alloc(size int) []byte {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Align to 8-byte boundary for better performance
	alignedSize := (size + 7) &^ 7

	if a.offset+alignedSize > a.capacity {
		return nil // Arena is full
	}

	slice := a.buffer[a.offset : a.offset+size]
	a.offset += alignedSize
	return slice
}

// Reset resets the arena for reuse
// Does not free memory, just resets the offset
func (a *Arena) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.offset = 0
}

// Used returns the number of bytes currently allocated
func (a *Arena) Used() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.offset
}

// Available returns the number of bytes still available
func (a *Arena) Available() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.capacity - a.offset
}

// Capacity returns the total capacity of the arena
func (a *Arena) Capacity() int {
	return a.capacity
}

// ParticleArena is a specialized arena for particle allocations
type ParticleArena struct {
	arena      *Arena
	particleSize int
}

// NewParticleArena creates a new particle arena
// Capacity is the number of particles it can hold
func NewParticleArena(capacity int) *ParticleArena {
	particleSize := 32 // 3*4 (position) + 4*4 (color) + 4 (size) + 8 (metadata) = 32 bytes
	return &ParticleArena{
		arena:      NewArena(capacity * particleSize),
		particleSize: particleSize,
	}
}

// AllocParticles allocates space for n particles
// Returns a slice of bytes that can be cast to particles
func (pa *ParticleArena) AllocParticles(n int) []byte {
	return pa.arena.Alloc(n * pa.particleSize)
}

// Reset resets the particle arena
func (pa *ParticleArena) Reset() {
	pa.arena.Reset()
}

// UsedParticles returns the number of particles currently allocated
func (pa *ParticleArena) UsedParticles() int {
	return pa.arena.Used() / pa.particleSize
}

// AvailableParticles returns the number of particles that can still be allocated
func (pa *ParticleArena) AvailableParticles() int {
	return pa.arena.Available() / pa.particleSize
}

// VoxelArena is a specialized arena for voxel index allocations
type VoxelArena struct {
	arena    *Arena
	voxelSize int
}

// NewVoxelArena creates a new voxel arena
func NewVoxelArena(capacity int) *VoxelArena {
	voxelSize := 32 // Size of voxel index entry (position + bounds + particle count)
	return &VoxelArena{
		arena:    NewArena(capacity * voxelSize),
		voxelSize: voxelSize,
	}
}

// AllocVoxels allocates space for n voxels
func (va *VoxelArena) AllocVoxels(n int) []byte {
	return va.arena.Alloc(n * va.voxelSize)
}

// Reset resets the voxel arena
func (va *VoxelArena) Reset() {
	va.arena.Reset()
}

// PooledArena combines arena allocation with pooling
// Multiple arenas are pooled for concurrent use
type PooledArena struct {
	pool         sync.Pool
	arenaSize    int
	allocations  uint64
	reuses       uint64
	mu           sync.Mutex
}

// NewPooledArena creates a new pooled arena system
func NewPooledArena(arenaSize int) *PooledArena {
	pa := &PooledArena{
		arenaSize: arenaSize,
	}
	pa.pool = sync.Pool{
		New: func() interface{} {
			pa.mu.Lock()
			pa.allocations++
			pa.mu.Unlock()
			return NewArena(arenaSize)
		},
	}
	return pa
}

// GetArena retrieves an arena from the pool
func (pa *PooledArena) GetArena() *Arena {
	arena := pa.pool.Get().(*Arena)
	arena.Reset()
	pa.mu.Lock()
	pa.reuses++
	pa.mu.Unlock()
	return arena
}

// PutArena returns an arena to the pool
func (pa *PooledArena) PutArena(arena *Arena) {
	if arena.Capacity() != pa.arenaSize {
		return // Don't pool arenas with wrong size
	}
	pa.pool.Put(arena)
}

// Stats returns pooled arena statistics
func (pa *PooledArena) Stats() (allocations, reuses uint64) {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	return pa.allocations, pa.reuses
}

// StreamBuffer is a specialized buffer for streaming genomic data
// Uses arena allocation to avoid GC pressure
type StreamBuffer struct {
	arena       *Arena
	chunkSize   int
	activeChunk []byte
}

// NewStreamBuffer creates a new stream buffer
func NewStreamBuffer(totalSize, chunkSize int) *StreamBuffer {
	return &StreamBuffer{
		arena:     NewArena(totalSize),
		chunkSize: chunkSize,
	}
}

// GetChunk returns the next chunk of data
func (sb *StreamBuffer) GetChunk() ([]byte, error) {
	chunk := sb.arena.Alloc(sb.chunkSize)
	if chunk == nil {
		return nil, fmt.Errorf("stream buffer full")
	}
	sb.activeChunk = chunk
	return chunk, nil
}

// Reset resets the stream buffer
func (sb *StreamBuffer) Reset() {
	sb.arena.Reset()
	sb.activeChunk = nil
}

// Used returns bytes used
func (sb *StreamBuffer) Used() int {
	return sb.arena.Used()
}

// Available returns bytes available
func (sb *StreamBuffer) Available() int {
	return sb.arena.Available()
}

// MemoryManager combines all memory management strategies
type MemoryManager struct {
	particlePool  *MonitoredParticlePool
	voxelPool     *VoxelPool
	bufferPool    *BufferPool
	coordPool     *CoordinatePool
	streamArena   *PooledArena
	particleArena *PooledArena
}

// NewMemoryManager creates a new memory manager with sensible defaults
func NewMemoryManager() *MemoryManager {
	return &MemoryManager{
		particlePool:  NewMonitoredParticlePool(50000),     // 50K particles per slice
		voxelPool:     NewVoxelPool(1000),                  // 1K indices per voxel
		bufferPool:    NewBufferPool(1024 * 1024),          // 1MB buffers
		coordPool:     NewCoordinatePool(50000),            // 50K coordinates
		streamArena:   NewPooledArena(100 * 1024 * 1024),   // 100MB stream arenas
		particleArena: NewPooledArena(50000 * 32),          // 50K particles per arena
	}
}

// GetParticleSlice gets a particle slice from the pool
func (mm *MemoryManager) GetParticleSlice() *ParticleSlice {
	return mm.particlePool.Get()
}

// PutParticleSlice returns a particle slice to the pool
func (mm *MemoryManager) PutParticleSlice(ps *ParticleSlice) {
	mm.particlePool.Put(ps)
}

// GetVoxelData gets voxel data from the pool
func (mm *MemoryManager) GetVoxelData() *VoxelData {
	return mm.voxelPool.Get()
}

// PutVoxelData returns voxel data to the pool
func (mm *MemoryManager) PutVoxelData(vd *VoxelData) {
	mm.voxelPool.Put(vd)
}

// GetBuffer gets a buffer from the pool
func (mm *MemoryManager) GetBuffer() []byte {
	return mm.bufferPool.Get()
}

// PutBuffer returns a buffer to the pool
func (mm *MemoryManager) PutBuffer(buf []byte) {
	mm.bufferPool.Put(buf)
}

// GetCoordinates gets a coordinate slice from the pool
func (mm *MemoryManager) GetCoordinates() [][3]float32 {
	return mm.coordPool.Get()
}

// PutCoordinates returns a coordinate slice to the pool
func (mm *MemoryManager) PutCoordinates(coords [][3]float32) {
	mm.coordPool.Put(coords)
}

// GetStreamArena gets a stream arena from the pool
func (mm *MemoryManager) GetStreamArena() *Arena {
	return mm.streamArena.GetArena()
}

// PutStreamArena returns a stream arena to the pool
func (mm *MemoryManager) PutStreamArena(arena *Arena) {
	mm.streamArena.PutArena(arena)
}

// GetParticleArena gets a particle arena from the pool
func (mm *MemoryManager) GetParticleArena() *Arena {
	return mm.particleArena.GetArena()
}

// PutParticleArena returns a particle arena to the pool
func (mm *MemoryManager) PutParticleArena(arena *Arena) {
	mm.particleArena.PutArena(arena)
}

// Stats returns comprehensive memory statistics
func (mm *MemoryManager) Stats() map[string]interface{} {
	poolStats := mm.particlePool.Stats()
	streamAlloc, streamReuse := mm.streamArena.Stats()
	particleAlloc, particleReuse := mm.particleArena.Stats()

	return map[string]interface{}{
		"particle_pool": map[string]uint64{
			"gets":   poolStats.Gets,
			"puts":   poolStats.Puts,
			"reuses": poolStats.Reuses,
		},
		"stream_arena": map[string]uint64{
			"allocations": streamAlloc,
			"reuses":      streamReuse,
		},
		"particle_arena": map[string]uint64{
			"allocations": particleAlloc,
			"reuses":      particleReuse,
		},
	}
}

// Global memory manager instance
var globalMemoryManager *MemoryManager
var once sync.Once

// GetGlobalMemoryManager returns the global memory manager singleton
func GetGlobalMemoryManager() *MemoryManager {
	once.Do(func() {
		globalMemoryManager = NewMemoryManager()
	})
	return globalMemoryManager
}
