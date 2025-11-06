package memory

import (
	"sync"
	"testing"
)

func TestArenaAlloc(t *testing.T) {
	arena := NewArena(1024)

	// Test allocation
	slice1 := arena.Alloc(100)
	if slice1 == nil {
		t.Fatal("Alloc() returned nil")
	}
	if len(slice1) != 100 {
		t.Errorf("Expected length 100, got %d", len(slice1))
	}

	// Test used
	if arena.Used() == 0 {
		t.Error("Used() should be > 0 after allocation")
	}

	// Allocate more
	slice2 := arena.Alloc(200)
	if slice2 == nil {
		t.Fatal("Second Alloc() returned nil")
	}
	if len(slice2) != 200 {
		t.Errorf("Expected length 200, got %d", len(slice2))
	}

	// Test available
	if arena.Available() == arena.Capacity() {
		t.Error("Available() should be less than capacity after allocations")
	}
}

func TestArenaReset(t *testing.T) {
	arena := NewArena(1024)

	// Allocate some memory
	arena.Alloc(100)
	arena.Alloc(200)

	usedBefore := arena.Used()
	if usedBefore == 0 {
		t.Error("Used() should be > 0 before reset")
	}

	// Reset
	arena.Reset()

	if arena.Used() != 0 {
		t.Errorf("Used() should be 0 after reset, got %d", arena.Used())
	}
	if arena.Available() != arena.Capacity() {
		t.Error("Available() should equal capacity after reset")
	}
}

func TestArenaFull(t *testing.T) {
	arena := NewArena(100)

	// Allocate almost full (90 bytes -> 96 aligned)
	slice1 := arena.Alloc(90)
	if slice1 == nil {
		t.Fatal("First alloc failed")
	}

	// This should fail (only 4 bytes left after alignment)
	slice2 := arena.Alloc(10)
	if slice2 != nil {
		t.Error("Alloc() should return nil when arena is full")
	}

	// Reset and try again
	arena.Reset()
	slice3 := arena.Alloc(50)
	if slice3 == nil {
		t.Error("Alloc() should succeed after reset")
	}
}

func TestParticleArena(t *testing.T) {
	arena := NewParticleArena(1000)

	// Allocate particles
	particleBytes := arena.AllocParticles(100)
	if particleBytes == nil {
		t.Fatal("AllocParticles() returned nil")
	}

	// Check size
	expectedSize := 100 * 32 // 32 bytes per particle
	if len(particleBytes) != expectedSize {
		t.Errorf("Expected %d bytes, got %d", expectedSize, len(particleBytes))
	}

	// Check used
	used := arena.UsedParticles()
	if used != 100 {
		t.Errorf("Expected 100 used particles, got %d", used)
	}

	// Check available
	available := arena.AvailableParticles()
	if available != 900 {
		t.Errorf("Expected 900 available particles, got %d", available)
	}
}

func TestVoxelArena(t *testing.T) {
	arena := NewVoxelArena(1000)

	// Allocate voxels
	voxelBytes := arena.AllocVoxels(50)
	if voxelBytes == nil {
		t.Fatal("AllocVoxels() returned nil")
	}

	// Check size
	expectedSize := 50 * 32 // 32 bytes per voxel
	if len(voxelBytes) != expectedSize {
		t.Errorf("Expected %d bytes, got %d", expectedSize, len(voxelBytes))
	}
}

func TestPooledArena(t *testing.T) {
	pa := NewPooledArena(1024)

	// Get arena
	arena := pa.GetArena()
	if arena == nil {
		t.Fatal("GetArena() returned nil")
	}
	if arena.Capacity() != 1024 {
		t.Errorf("Expected capacity 1024, got %d", arena.Capacity())
	}

	// Use arena
	arena.Alloc(100)

	// Return to pool
	pa.PutArena(arena)

	// Get again - should be reset
	arena2 := pa.GetArena()
	if arena2.Used() != 0 {
		t.Errorf("Expected reset arena with 0 used, got %d", arena2.Used())
	}

	// Check stats
	allocs, reuses := pa.Stats()
	if allocs == 0 {
		t.Error("Expected at least 1 allocation")
	}
	if reuses < 1 {
		t.Error("Expected at least 1 reuse")
	}
}

func TestStreamBuffer(t *testing.T) {
	sb := NewStreamBuffer(10240, 1024)

	// Get chunks
	chunk1, err := sb.GetChunk()
	if err != nil {
		t.Fatalf("GetChunk() error: %v", err)
	}
	if len(chunk1) != 1024 {
		t.Errorf("Expected chunk size 1024, got %d", len(chunk1))
	}

	// Get more chunks
	for i := 0; i < 9; i++ {
		_, err := sb.GetChunk()
		if err != nil {
			t.Fatalf("GetChunk() %d error: %v", i+2, err)
		}
	}

	// Should be full now
	_, err = sb.GetChunk()
	if err == nil {
		t.Error("Expected error when stream buffer full")
	}

	// Reset and try again
	sb.Reset()
	chunk, err := sb.GetChunk()
	if err != nil {
		t.Fatalf("GetChunk() after reset error: %v", err)
	}
	if len(chunk) != 1024 {
		t.Error("Chunk size wrong after reset")
	}
}

func TestConcurrentArenaAccess(t *testing.T) {
	pa := NewPooledArena(10240)
	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				arena := pa.GetArena()
				arena.Alloc(100)
				pa.PutArena(arena)
			}
		}()
	}

	wg.Wait()
	// If we got here without panicking, concurrent access works
}

func BenchmarkArenaAlloc(b *testing.B) {
	arena := NewArena(1024 * 1024) // 1MB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		arena.Alloc(1024)
		if i%100 == 0 {
			arena.Reset() // Reset periodically
		}
	}
}

func BenchmarkArenaAllocWithoutPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = make([]byte, 1024)
	}
}

func BenchmarkPooledArena(b *testing.B) {
	pa := NewPooledArena(10240)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		arena := pa.GetArena()
		arena.Alloc(1024)
		pa.PutArena(arena)
	}
}
