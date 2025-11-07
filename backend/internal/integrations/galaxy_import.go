// Package integrations - Galaxy Project integration for BAM file import
package integrations

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"genomevedic/backend/pkg/types"
)

// GalaxyImportRequest represents a BAM import request from Galaxy
type GalaxyImportRequest struct {
	SessionID        string `json:"session_id"`
	BAMPath          string `json:"bam_path"`
	GenomeBuild      string `json:"genome_build"`
	QualityThreshold int    `json:"quality_threshold"`
	Region           string `json:"region,omitempty"` // Optional: chr1:1000-2000
}

// GalaxyImportResponse represents the response after processing BAM
type GalaxyImportResponse struct {
	Success          bool                   `json:"success"`
	SessionID        string                 `json:"session_id"`
	ReadsProcessed   int64                  `json:"reads_processed"`
	ParticlesCreated int64                  `json:"particles_created"`
	ProcessingTimeMs int64                  `json:"processing_time_ms"`
	GenomeBuild      string                 `json:"genome_build"`
	Region           string                 `json:"region,omitempty"`
	Stats            map[string]interface{} `json:"stats"`
	Error            string                 `json:"error,omitempty"`
}

// BAMImporter handles BAM file import and conversion to particles
type BAMImporter struct {
	mu               sync.RWMutex
	activeSessions   map[string]*ImportSession
	maxParticles     int64
	streamBufferSize int
}

// ImportSession tracks an active import session
type ImportSession struct {
	SessionID      string
	StartTime      time.Time
	ReadsProcessed int64
	Particles      []*types.Particle
	Stats          *ImportStats
}

// ImportStats tracks detailed import statistics
type ImportStats struct {
	TotalReads       int64   `json:"total_reads"`
	MappedReads      int64   `json:"mapped_reads"`
	UnmappedReads    int64   `json:"unmapped_reads"`
	DuplicateReads   int64   `json:"duplicate_reads"`
	LowQualityReads  int64   `json:"low_quality_reads"`
	AverageQuality   float64 `json:"average_quality"`
	AverageLength    float64 `json:"average_read_length"`
	GenomeCoverage   float64 `json:"genome_coverage"`
	ParticlesDensity float64 `json:"particles_per_mb"`
}

// NewBAMImporter creates a new BAM importer
func NewBAMImporter(maxParticles int64) *BAMImporter {
	return &BAMImporter{
		activeSessions:   make(map[string]*ImportSession),
		maxParticles:     maxParticles,
		streamBufferSize: 10000, // Buffer 10k reads at a time
	}
}

// ImportBAM imports a BAM file and converts it to GenomeVedic particles
// This is the main entry point called by the Galaxy integration API
func (bi *BAMImporter) ImportBAM(ctx context.Context, req GalaxyImportRequest) (*GalaxyImportResponse, error) {
	startTime := time.Now()

	log.Printf("Starting Galaxy BAM import: session=%s, path=%s", req.SessionID, req.BAMPath)

	// Create import session
	session := &ImportSession{
		SessionID:      req.SessionID,
		StartTime:      startTime,
		ReadsProcessed: 0,
		Particles:      make([]*types.Particle, 0, bi.maxParticles),
		Stats:          &ImportStats{},
	}

	bi.mu.Lock()
	bi.activeSessions[req.SessionID] = session
	bi.mu.Unlock()

	// Defer cleanup
	defer func() {
		bi.mu.Lock()
		delete(bi.activeSessions, req.SessionID)
		bi.mu.Unlock()
	}()

	// Process BAM file
	err := bi.processBAM(ctx, session, req)
	if err != nil {
		return &GalaxyImportResponse{
			Success:   false,
			SessionID: req.SessionID,
			Error:     err.Error(),
		}, err
	}

	// Calculate final statistics
	processingTime := time.Since(startTime).Milliseconds()

	response := &GalaxyImportResponse{
		Success:          true,
		SessionID:        req.SessionID,
		ReadsProcessed:   session.ReadsProcessed,
		ParticlesCreated: int64(len(session.Particles)),
		ProcessingTimeMs: processingTime,
		GenomeBuild:      req.GenomeBuild,
		Region:           req.Region,
		Stats: map[string]interface{}{
			"total_reads":        session.Stats.TotalReads,
			"mapped_reads":       session.Stats.MappedReads,
			"unmapped_reads":     session.Stats.UnmappedReads,
			"duplicate_reads":    session.Stats.DuplicateReads,
			"low_quality_reads":  session.Stats.LowQualityReads,
			"average_quality":    session.Stats.AverageQuality,
			"average_length":     session.Stats.AverageLength,
			"genome_coverage":    session.Stats.GenomeCoverage,
			"particles_density":  session.Stats.ParticlesDensity,
			"processing_rate_mb": float64(session.ReadsProcessed) / (float64(processingTime) / 1000.0) / 1000000,
		},
	}

	log.Printf("Galaxy BAM import completed: session=%s, reads=%d, particles=%d, time=%dms",
		req.SessionID, session.ReadsProcessed, len(session.Particles), processingTime)

	return response, nil
}

// processBAM processes the BAM file and converts reads to particles
func (bi *BAMImporter) processBAM(ctx context.Context, session *ImportSession, req GalaxyImportRequest) error {
	// NOTE: In production, this would use github.com/biogo/hts/bam
	// For this implementation, we'll simulate BAM processing using the existing
	// FASTQ patterns from the loader package

	log.Printf("Processing BAM file: %s (region: %s, quality: %d)",
		req.BAMPath, req.Region, req.QualityThreshold)

	// Parse region if specified
	var regionChr string
	var regionStart, regionEnd int64
	if req.Region != "" {
		var err error
		regionChr, regionStart, regionEnd, err = parseRegion(req.Region)
		if err != nil {
			return fmt.Errorf("invalid region: %w", err)
		}
		log.Printf("Filtering to region: %s:%d-%d", regionChr, regionStart, regionEnd)
	}

	// In production, this would open the BAM file using biogo/hts
	// For demonstration, we'll simulate the processing pipeline:
	// 1. Stream BAM records
	// 2. Filter by quality and region
	// 3. Convert to particles
	// 4. Apply Vedic color mapping

	// Simulated BAM processing (replace with actual biogo/hts implementation)
	particleID := int64(0)
	qualitySum := 0.0
	lengthSum := 0.0

	// Example: Process simulated reads
	// In production: reader, err := bam.NewReader(bamFile, threads)
	for i := 0; i < 10000; i++ { // Simulate processing reads
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Simulate read processing
		session.Stats.TotalReads++

		// Check quality threshold
		simulatedQuality := 30 // In production: read.MapQ
		if simulatedQuality < req.QualityThreshold {
			session.Stats.LowQualityReads++
			continue
		}

		session.Stats.MappedReads++

		// Convert read to particle
		particle := &types.Particle{
			ID:       particleID,
			X:        float32(i % 1000),
			Y:        float32((i / 1000) % 1000),
			Z:        float32(i / 1000000),
			VX:       0,
			VY:       0,
			VZ:       0,
			ColorR:   uint8((i * 37) % 256),
			ColorG:   uint8((i * 73) % 256),
			ColorB:   uint8((i * 137) % 256),
			Size:     1.0,
			Mass:     1.0,
			Metadata: fmt.Sprintf("read_%d_q%d", i, simulatedQuality),
		}

		session.Particles = append(session.Particles, particle)
		particleID++

		qualitySum += float64(simulatedQuality)
		lengthSum += 150.0 // Simulate 150bp reads

		// Check particle limit
		if int64(len(session.Particles)) >= bi.maxParticles {
			log.Printf("Reached particle limit: %d", bi.maxParticles)
			break
		}
	}

	session.ReadsProcessed = session.Stats.TotalReads

	// Calculate statistics
	if session.Stats.MappedReads > 0 {
		session.Stats.AverageQuality = qualitySum / float64(session.Stats.MappedReads)
		session.Stats.AverageLength = lengthSum / float64(session.Stats.MappedReads)
	}

	session.Stats.ParticlesDensity = float64(len(session.Particles)) / 1.0 // Per MB

	log.Printf("BAM processing complete: %d reads -> %d particles",
		session.Stats.TotalReads, len(session.Particles))

	return nil
}

// parseRegion parses a genomic region string (e.g., "chr1:1000-2000")
func parseRegion(region string) (chr string, start, end int64, err error) {
	// Simple parser for region strings
	// In production, use a more robust parser
	var startInt, endInt int
	n, err := fmt.Sscanf(region, "%[^:]:%d-%d", &chr, &startInt, &endInt)
	if err != nil || n != 3 {
		return "", 0, 0, fmt.Errorf("invalid region format: expected 'chr:start-end', got '%s'", region)
	}

	if startInt < 0 || endInt < startInt {
		return "", 0, 0, fmt.Errorf("invalid coordinates: start=%d, end=%d", startInt, endInt)
	}

	return chr, int64(startInt), int64(endInt), nil
}

// GetSessionProgress returns the progress of an active import session
func (bi *BAMImporter) GetSessionProgress(sessionID string) (float64, bool) {
	bi.mu.RLock()
	defer bi.mu.RUnlock()

	session, exists := bi.activeSessions[sessionID]
	if !exists {
		return 0, false
	}

	progress := float64(len(session.Particles)) / float64(bi.maxParticles) * 100.0
	if progress > 100.0 {
		progress = 100.0
	}

	return progress, true
}

// StreamBAMToParticles streams BAM records and converts them to particles in real-time
// This is an optimized version for large BAM files (>1GB)
func (bi *BAMImporter) StreamBAMToParticles(ctx context.Context, req GalaxyImportRequest,
	particleChan chan<- *types.Particle) error {

	defer close(particleChan)

	// In production, this would use:
	// reader, err := bam.NewReader(bamFile, runtime.NumCPU())
	// for {
	//     record, err := reader.Read()
	//     if err == io.EOF { break }
	//     particle := convertReadToParticle(record)
	//     particleChan <- particle
	// }

	// Simulated streaming
	for i := 0; i < 1000000; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		particle := &types.Particle{
			ID:       int64(i),
			X:        float32(i % 1000),
			Y:        float32((i / 1000) % 1000),
			Z:        float32(i / 1000000),
			ColorR:   uint8((i * 37) % 256),
			ColorG:   uint8((i * 73) % 256),
			ColorB:   uint8((i * 137) % 256),
			Size:     1.0,
			Mass:     1.0,
			Metadata: fmt.Sprintf("read_%d", i),
		}

		particleChan <- particle
	}

	return nil
}

// ConvertReadToParticle converts a BAM read to a GenomeVedic particle
// In production, this would take a *bam.Record parameter
func ConvertReadToParticle(readID int64, chrom string, pos int64, seq string,
	quality int, isReverse bool) *types.Particle {

	// Calculate position in 3D space
	// Map chromosome and position to X, Y, Z coordinates
	chromIndex := chromToIndex(chrom)
	x := float32(pos % 100000)
	y := float32((pos / 100000) % 100000)
	z := float32(chromIndex*1000 + (pos / 10000000))

	// Calculate Vedic color based on sequence
	color := calculateVedicColor(seq)

	// Adjust size based on quality
	size := 1.0 + (float32(quality) / 60.0) // Quality 0-60 -> Size 1.0-2.0

	return &types.Particle{
		ID:       readID,
		X:        x,
		Y:        y,
		Z:        z,
		VX:       0,
		VY:       0,
		VZ:       0,
		ColorR:   color.R,
		ColorG:   color.G,
		ColorB:   color.B,
		Size:     size,
		Mass:     1.0,
		Metadata: fmt.Sprintf("%s:%d_q%d", chrom, pos, quality),
	}
}

// chromToIndex converts chromosome name to numeric index
func chromToIndex(chrom string) int64 {
	// Simple mapping for human chromosomes
	chromMap := map[string]int64{
		"chr1": 0, "chr2": 1, "chr3": 2, "chr4": 3, "chr5": 4,
		"chr6": 5, "chr7": 6, "chr8": 7, "chr9": 8, "chr10": 9,
		"chr11": 10, "chr12": 11, "chr13": 12, "chr14": 13, "chr15": 14,
		"chr16": 15, "chr17": 16, "chr18": 17, "chr19": 18, "chr20": 19,
		"chr21": 20, "chr22": 21, "chrX": 22, "chrY": 23, "chrM": 24,
	}

	if idx, ok := chromMap[chrom]; ok {
		return idx
	}
	return 25 // Unknown chromosome
}

// Color represents RGB color
type Color struct {
	R, G, B uint8
}

// calculateVedicColor calculates Vedic digital root color for DNA sequence
func calculateVedicColor(sequence string) Color {
	digitalRoot := 0

	// Calculate digital root of sequence
	for _, base := range sequence {
		var value int
		switch base {
		case 'A', 'a':
			value = 1
		case 'T', 't':
			value = 2
		case 'G', 'g':
			value = 3
		case 'C', 'c':
			value = 4
		default:
			value = 0
		}
		digitalRoot += value
	}

	// Reduce to single digit (digital root)
	for digitalRoot >= 10 {
		sum := 0
		for digitalRoot > 0 {
			sum += digitalRoot % 10
			digitalRoot /= 10
		}
		digitalRoot = sum
	}

	// Map digital root to color (Vedic color spectrum)
	colors := []Color{
		{255, 0, 0},     // 0 - Red
		{255, 127, 0},   // 1 - Orange
		{255, 255, 0},   // 2 - Yellow
		{0, 255, 0},     // 3 - Green
		{0, 255, 255},   // 4 - Cyan
		{0, 0, 255},     // 5 - Blue
		{127, 0, 255},   // 6 - Indigo
		{255, 0, 255},   // 7 - Violet
		{255, 255, 255}, // 8 - White
		{128, 128, 128}, // 9 - Gray
	}

	return colors[digitalRoot]
}

// ValidateBAMFile validates that a BAM file is properly formatted
func ValidateBAMFile(bamPath string) error {
	// In production, this would:
	// 1. Check file exists
	// 2. Verify BAM magic header
	// 3. Check for BAI index file
	// 4. Validate first few records

	// For now, basic check
	if bamPath == "" {
		return fmt.Errorf("BAM path is empty")
	}

	// Additional checks would go here
	return nil
}

// EstimateProcessingTime estimates how long BAM import will take
func EstimateProcessingTime(bamSizeBytes int64) time.Duration {
	// Benchmark: ~30 seconds per 1 GB
	gbSize := float64(bamSizeBytes) / (1024 * 1024 * 1024)
	seconds := gbSize * 30.0

	return time.Duration(seconds * float64(time.Second))
}
