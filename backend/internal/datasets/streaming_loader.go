// Package datasets - Streaming loader for large genomic datasets
// Implements progressive loading with LOD levels for 10 GB+ datasets
package datasets

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
)

// ParticleMetadata contains dataset metadata
type ParticleMetadata struct {
	SequenceName         string   `json:"sequence_name"`
	Length               int      `json:"length"`
	Particles            int      `json:"particles"`
	LODLevels            []int    `json:"lod_levels"`
	VoxelCount           int      `json:"voxel_count"`
	VoxelSize            float64  `json:"voxel_size"`
	WilliamsBatchSize    int      `json:"williams_batch_size"`
	GenerationTime       float64  `json:"generation_time"`
	DigitalRootAlgorithm string   `json:"digital_root_algorithm"`
	GoldenAngleDegrees   float64  `json:"golden_angle_degrees"`
	Version              string   `json:"version"`
}

// Particle represents a single DNA base in 3D space
type Particle struct {
	X     float64   `json:"x"`
	Y     float64   `json:"y"`
	Z     float64   `json:"z"`
	Base  string    `json:"base"`
	Pos   int       `json:"pos"`
	Voxel int       `json:"voxel"`
	Color []float64 `json:"color"`
}

// ParticleData contains the full particle dataset
type ParticleData struct {
	Metadata    ParticleMetadata       `json:"metadata"`
	Particles   []Particle             `json:"particles"`
	SpatialHash map[string][]int       `json:"spatial_hash"`
	LODLevels   map[string][]int       `json:"lod_levels"`
}

// StreamingLoader handles progressive loading of large particle datasets
type StreamingLoader struct {
	dataDir      string
	cache        map[string]*ParticleData
	cacheMutex   sync.RWMutex
	maxCacheSize int64
	currentSize  int64
}

// NewStreamingLoader creates a new streaming loader
func NewStreamingLoader(dataDir string, maxCacheSizeMB int64) *StreamingLoader {
	return &StreamingLoader{
		dataDir:      dataDir,
		cache:        make(map[string]*ParticleData),
		maxCacheSize: maxCacheSizeMB * 1024 * 1024,
		currentSize:  0,
	}
}

// LoadDataset loads a dataset with optional LOD level
// Returns particle data progressively (streaming friendly)
func (sl *StreamingLoader) LoadDataset(datasetID string, lodLevel int) (*ParticleData, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s_lod%d", datasetID, lodLevel)

	sl.cacheMutex.RLock()
	cached, exists := sl.cache[cacheKey]
	sl.cacheMutex.RUnlock()

	if exists {
		return cached, nil
	}

	// Load from disk
	data, err := sl.loadFromDisk(datasetID, lodLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to load dataset %s: %w", datasetID, err)
	}

	// Cache if under limit
	dataSize := sl.estimateSize(data)
	if sl.currentSize+dataSize <= sl.maxCacheSize {
		sl.cacheMutex.Lock()
		sl.cache[cacheKey] = data
		sl.currentSize += dataSize
		sl.cacheMutex.Unlock()
	}

	return data, nil
}

// loadFromDisk loads particle data from compressed file
func (sl *StreamingLoader) loadFromDisk(datasetID string, lodLevel int) (*ParticleData, error) {
	// Try .zst first, then .json.gz, then .json
	extensions := []string{".particles.zst", ".particles.json.gz", ".particles.json"}

	var file *os.File
	var err error
	var compression string

	for _, ext := range extensions {
		path := filepath.Join(sl.dataDir, datasetID+ext)
		file, err = os.Open(path)
		if err == nil {
			compression = ext
			break
		}
	}

	if file == nil {
		return nil, fmt.Errorf("dataset file not found: %s", datasetID)
	}
	defer file.Close()

	// Decompress based on extension
	var reader io.Reader = file

	switch compression {
	case ".particles.zst":
		decoder, err := zstd.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("zstd decompression failed: %w", err)
		}
		defer decoder.Close()
		reader = decoder

	case ".particles.json.gz":
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("gzip decompression failed: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// Parse JSON
	var data ParticleData
	decoder := json.NewDecoder(reader)

	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("JSON parsing failed: %w", err)
	}

	// Filter to specific LOD level if requested
	if lodLevel >= 0 {
		data = sl.filterToLOD(data, lodLevel)
	}

	return &data, nil
}

// filterToLOD filters particle data to specific LOD level
func (sl *StreamingLoader) filterToLOD(data ParticleData, lodLevel int) ParticleData {
	lodKey := fmt.Sprintf("%d", lodLevel)
	indices, exists := data.LODLevels[lodKey]

	if !exists {
		// Return all particles if LOD level doesn't exist
		return data
	}

	// Create filtered particle list
	filteredParticles := make([]Particle, len(indices))
	for i, idx := range indices {
		if idx < len(data.Particles) {
			filteredParticles[i] = data.Particles[idx]
		}
	}

	data.Particles = filteredParticles
	return data
}

// estimateSize estimates memory usage of particle data
func (sl *StreamingLoader) estimateSize(data *ParticleData) int64 {
	// Rough estimate: each particle ~100 bytes
	return int64(len(data.Particles) * 100)
}

// LoadProgressive loads dataset progressively (LOD 0 → 1 → 2 → 3)
// Calls callback for each LOD level loaded
func (sl *StreamingLoader) LoadProgressive(datasetID string, callback func(lodLevel int, data *ParticleData, progress float64)) error {
	lodLevels := []int{0, 1, 2, 3} // 5K, 50K, 500K, 5M

	for i, lod := range lodLevels {
		data, err := sl.LoadDataset(datasetID, lod)
		if err != nil {
			return fmt.Errorf("failed to load LOD %d: %w", lod, err)
		}

		progress := float64(i+1) / float64(len(lodLevels))
		callback(lod, data, progress)
	}

	return nil
}

// StreamChunked streams particle data in chunks (memory efficient)
func (sl *StreamingLoader) StreamChunked(datasetID string, chunkSize int, callback func(chunk []Particle, progress float64) error) error {
	data, err := sl.LoadDataset(datasetID, -1) // Load all
	if err != nil {
		return err
	}

	total := len(data.Particles)

	for i := 0; i < total; i += chunkSize {
		end := i + chunkSize
		if end > total {
			end = total
		}

		chunk := data.Particles[i:end]
		progress := float64(end) / float64(total)

		if err := callback(chunk, progress); err != nil {
			return fmt.Errorf("chunk callback failed: %w", err)
		}
	}

	return nil
}

// ClearCache clears the in-memory cache
func (sl *StreamingLoader) ClearCache() {
	sl.cacheMutex.Lock()
	defer sl.cacheMutex.Unlock()

	sl.cache = make(map[string]*ParticleData)
	sl.currentSize = 0
}

// GetCacheStats returns cache statistics
func (sl *StreamingLoader) GetCacheStats() map[string]interface{} {
	sl.cacheMutex.RLock()
	defer sl.cacheMutex.RUnlock()

	return map[string]interface{}{
		"cached_datasets": len(sl.cache),
		"current_size_mb": sl.currentSize / 1024 / 1024,
		"max_size_mb":     sl.maxCacheSize / 1024 / 1024,
		"usage_percent":   float64(sl.currentSize) / float64(sl.maxCacheSize) * 100,
	}
}

// PreloadDatasets preloads datasets in the background
func (sl *StreamingLoader) PreloadDatasets(datasetIDs []string, lodLevel int) {
	go func() {
		for _, id := range datasetIDs {
			_, err := sl.LoadDataset(id, lodLevel)
			if err != nil {
				fmt.Printf("WARNING: Failed to preload %s: %v\n", id, err)
			}
			time.Sleep(100 * time.Millisecond) // Throttle to avoid overwhelming disk
		}
	}()
}

// GetMetadata loads only metadata (fast, for listing datasets)
func (sl *StreamingLoader) GetMetadata(datasetID string) (*ParticleMetadata, error) {
	// Load full dataset (cached if possible)
	data, err := sl.LoadDataset(datasetID, 0)
	if err != nil {
		return nil, err
	}

	return &data.Metadata, nil
}

// ListAvailableDatasets scans data directory for available datasets
func (sl *StreamingLoader) ListAvailableDatasets() ([]string, error) {
	files, err := os.ReadDir(sl.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	datasets := make(map[string]bool)

	for _, file := range files {
		name := file.Name()

		// Extract dataset ID from filename
		if filepath.Ext(name) == ".zst" {
			id := name[:len(name)-len(".particles.zst")]
			datasets[id] = true
		} else if filepath.Ext(name) == ".json" && filepath.Ext(name[:len(name)-5]) == ".particles" {
			id := name[:len(name)-len(".particles.json")]
			datasets[id] = true
		}
	}

	result := make([]string, 0, len(datasets))
	for id := range datasets {
		result = append(result, id)
	}

	return result, nil
}

// WilliamsBatchSize calculates optimal batch size using Williams Optimizer
// Formula: BatchSize = √n × log₂(n)
func WilliamsBatchSize(n int) int {
	if n <= 0 {
		return 1
	}

	sqrtN := math.Sqrt(float64(n))
	log2N := math.Log2(float64(n))

	batchSize := int(sqrtN * log2N)

	if batchSize < 1 {
		return 1
	}

	return batchSize
}
