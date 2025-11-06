package memory

import (
	"sync"
	"testing"
)

func TestParticlePool(t *testing.T) {
	pool := NewParticlePool(1000)

	// Test Get
	ps := pool.Get()
	if ps == nil {
		t.Fatal("Get() returned nil")
	}
	if ps.Capacity != 1000 {
		t.Errorf("Expected capacity 1000, got %d", ps.Capacity)
	}
	if ps.Length != 0 {
		t.Errorf("Expected length 0, got %d", ps.Length)
	}

	// Test Put
	ps.Length = 100
	pool.Put(ps)

	// Get again - should be reset
	ps2 := pool.Get()
	if ps2.Length != 0 {
		t.Errorf("Expected reset length 0, got %d", ps2.Length)
	}

	// Should reuse the same slice (can't test identity, but capacity should match)
	if ps2.Capacity != 1000 {
		t.Errorf("Expected reused capacity 1000, got %d", ps2.Capacity)
	}
}

func TestVoxelPool(t *testing.T) {
	pool := NewVoxelPool(100)

	vd := pool.Get()
	if vd == nil {
		t.Fatal("Get() returned nil")
	}
	if vd.Capacity != 100 {
		t.Errorf("Expected capacity 100, got %d", vd.Capacity)
	}

	// Add some data
	vd.ParticleIndices = append(vd.ParticleIndices, 1, 2, 3)
	vd.Count = 3

	// Put back
	pool.Put(vd)

	// Get again - should be reset
	vd2 := pool.Get()
	if len(vd2.ParticleIndices) != 0 {
		t.Errorf("Expected reset indices length 0, got %d", len(vd2.ParticleIndices))
	}
	if vd2.Count != 0 {
		t.Errorf("Expected reset count 0, got %d", vd2.Count)
	}
}

func TestBufferPool(t *testing.T) {
	pool := NewBufferPool(1024)

	buf := pool.Get()
	if buf == nil {
		t.Fatal("Get() returned nil")
	}
	if len(buf) != 1024 {
		t.Errorf("Expected buffer size 1024, got %d", len(buf))
	}

	// Modify buffer
	buf[0] = 42

	// Put back
	pool.Put(buf)

	// Get again
	buf2 := pool.Get()
	if len(buf2) != 1024 {
		t.Errorf("Expected reused buffer size 1024, got %d", len(buf2))
	}
}

func TestMonitoredParticlePool(t *testing.T) {
	pool := NewMonitoredParticlePool(1000)

	// Test stats
	initialStats := pool.Stats()
	if initialStats.Gets != 0 || initialStats.Puts != 0 {
		t.Error("Expected zero stats initially")
	}

	// Get and put
	ps := pool.Get()
	pool.Put(ps)

	// Check stats
	stats := pool.Stats()
	if stats.Gets != 1 {
		t.Errorf("Expected 1 get, got %d", stats.Gets)
	}
	if stats.Puts != 1 {
		t.Errorf("Expected 1 put, got %d", stats.Puts)
	}
	if stats.Reuses != 1 {
		t.Errorf("Expected 1 reuse, got %d", stats.Reuses)
	}

	// Reset stats
	pool.ResetStats()
	stats = pool.Stats()
	if stats.Gets != 0 || stats.Puts != 0 || stats.Reuses != 0 {
		t.Error("Stats not reset properly")
	}
}

func TestConcurrentPoolAccess(t *testing.T) {
	pool := NewParticlePool(1000)
	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				ps := pool.Get()
				// Do some work
				ps.Length = j
				pool.Put(ps)
			}
		}()
	}

	wg.Wait()
	// If we got here without panicking, concurrent access works
}

func BenchmarkParticlePoolGet(b *testing.B) {
	pool := NewParticlePool(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps := pool.Get()
		pool.Put(ps)
	}
}

func BenchmarkParticlePoolWithoutPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps := &ParticleSlice{
			Data:     make([]Particle, 1000),
			Capacity: 1000,
			Length:   0,
		}
		_ = ps
	}
}

func BenchmarkMemoryManager(b *testing.B) {
	mm := NewMemoryManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps := mm.GetParticleSlice()
		vd := mm.GetVoxelData()
		buf := mm.GetBuffer()

		mm.PutParticleSlice(ps)
		mm.PutVoxelData(vd)
		mm.PutBuffer(buf)
	}
}
