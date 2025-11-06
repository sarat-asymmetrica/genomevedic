package profiling

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// MemorySnapshot represents memory usage at a point in time
type MemorySnapshot struct {
	Timestamp    time.Time
	HeapAlloc    uint64 // Bytes allocated on heap
	HeapSys      uint64 // Bytes obtained from OS for heap
	StackInUse   uint64 // Bytes used by stack allocator
	NumGC        uint32 // Number of completed GC cycles
	GCPauseNs    uint64 // Cumulative GC pause time (nanoseconds)
}

// MemoryTracker monitors memory usage over time
type MemoryTracker struct {
	snapshots []MemorySnapshot
	startTime time.Time
	interval  time.Duration
	stopChan  chan bool
	running   bool
}

// NewMemoryTracker creates a new memory tracker
func NewMemoryTracker(sampleInterval time.Duration) *MemoryTracker {
	return &MemoryTracker{
		snapshots: make([]MemorySnapshot, 0, 1000),
		startTime: time.Now(),
		interval:  sampleInterval,
		stopChan:  make(chan bool),
		running:   false,
	}
}

// Start begins memory tracking in the background
func (mt *MemoryTracker) Start() {
	if mt.running {
		return
	}

	mt.running = true
	mt.startTime = time.Now()

	go func() {
		ticker := time.NewTicker(mt.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				mt.TakeSnapshot()
			case <-mt.stopChan:
				return
			}
		}
	}()
}

// Stop stops memory tracking
func (mt *MemoryTracker) Stop() {
	if !mt.running {
		return
	}

	mt.stopChan <- true
	mt.running = false
}

// TakeSnapshot records current memory usage
func (mt *MemoryTracker) TakeSnapshot() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	snapshot := MemorySnapshot{
		Timestamp:  time.Now(),
		HeapAlloc:  m.Alloc,
		HeapSys:    m.HeapSys,
		StackInUse: m.StackInuse,
		NumGC:      m.NumGC,
		GCPauseNs:  m.PauseTotalNs,
	}

	mt.snapshots = append(mt.snapshots, snapshot)
}

// GetCurrentMemory returns current memory usage
func (mt *MemoryTracker) GetCurrentMemory() MemorySnapshot {
	mt.TakeSnapshot()
	return mt.snapshots[len(mt.snapshots)-1]
}

// GetPeakMemory returns the peak memory usage
func (mt *MemoryTracker) GetPeakMemory() MemorySnapshot {
	if len(mt.snapshots) == 0 {
		return MemorySnapshot{}
	}

	peak := mt.snapshots[0]
	for _, snapshot := range mt.snapshots {
		if snapshot.HeapAlloc > peak.HeapAlloc {
			peak = snapshot
		}
	}
	return peak
}

// GetAverageMemory returns the average memory usage
func (mt *MemoryTracker) GetAverageMemory() float64 {
	if len(mt.snapshots) == 0 {
		return 0
	}

	total := uint64(0)
	for _, snapshot := range mt.snapshots {
		total += snapshot.HeapAlloc
	}

	return float64(total) / float64(len(mt.snapshots))
}

// GetMemoryGrowth returns memory growth rate (bytes per second)
func (mt *MemoryTracker) GetMemoryGrowth() float64 {
	if len(mt.snapshots) < 2 {
		return 0
	}

	first := mt.snapshots[0]
	last := mt.snapshots[len(mt.snapshots)-1]

	memoryDiff := int64(last.HeapAlloc) - int64(first.HeapAlloc)
	timeDiff := last.Timestamp.Sub(first.Timestamp).Seconds()

	if timeDiff == 0 {
		return 0
	}

	return float64(memoryDiff) / timeDiff
}

// GetGCStats returns garbage collection statistics
func (mt *MemoryTracker) GetGCStats() GCStats {
	if len(mt.snapshots) == 0 {
		return GCStats{}
	}

	first := mt.snapshots[0]
	last := mt.snapshots[len(mt.snapshots)-1]

	gcCount := last.NumGC - first.NumGC
	gcPauseTotal := last.GCPauseNs - first.GCPauseNs

	avgPause := uint64(0)
	if gcCount > 0 {
		avgPause = gcPauseTotal / uint64(gcCount)
	}

	return GCStats{
		TotalGCs:    gcCount,
		TotalPause:  gcPauseTotal,
		AveragePause: avgPause,
	}
}

// GCStats contains garbage collection statistics
type GCStats struct {
	TotalGCs     uint32
	TotalPause   uint64 // Nanoseconds
	AveragePause uint64 // Nanoseconds
}

// Report generates a memory usage report
func (mt *MemoryTracker) Report() string {
	var sb strings.Builder

	sb.WriteString("Memory Usage Report\n")
	sb.WriteString("===================\n\n")

	if len(mt.snapshots) == 0 {
		sb.WriteString("No snapshots recorded\n")
		return sb.String()
	}

	current := mt.GetCurrentMemory()
	peak := mt.GetPeakMemory()
	average := mt.GetAverageMemory()
	growth := mt.GetMemoryGrowth()
	gcStats := mt.GetGCStats()

	sb.WriteString(fmt.Sprintf("Duration:        %.1fs\n",
		time.Since(mt.startTime).Seconds()))
	sb.WriteString(fmt.Sprintf("Snapshots:       %d\n", len(mt.snapshots)))
	sb.WriteString(fmt.Sprintf("Sample interval: %v\n\n", mt.interval))

	sb.WriteString("Memory Usage:\n")
	sb.WriteString("-------------\n")
	sb.WriteString(fmt.Sprintf("  Current:       %.2f MB\n",
		float64(current.HeapAlloc)/(1024*1024)))
	sb.WriteString(fmt.Sprintf("  Peak:          %.2f MB\n",
		float64(peak.HeapAlloc)/(1024*1024)))
	sb.WriteString(fmt.Sprintf("  Average:       %.2f MB\n",
		average/(1024*1024)))
	sb.WriteString(fmt.Sprintf("  Growth rate:   %.2f KB/s\n\n",
		growth/1024))

	sb.WriteString("Garbage Collection:\n")
	sb.WriteString("-------------------\n")
	sb.WriteString(fmt.Sprintf("  Total GCs:     %d\n", gcStats.TotalGCs))
	sb.WriteString(fmt.Sprintf("  Total pause:   %.2fms\n",
		float64(gcStats.TotalPause)/1e6))
	sb.WriteString(fmt.Sprintf("  Average pause: %.2fms\n",
		float64(gcStats.AveragePause)/1e6))

	return sb.String()
}

// FormatBytes converts bytes to human-readable format
func FormatBytes(bytes uint64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(bytes)
	unitIndex := 0

	for value >= 1024 && unitIndex < len(units)-1 {
		value /= 1024
		unitIndex++
	}

	return fmt.Sprintf("%.2f %s", value, units[unitIndex])
}
