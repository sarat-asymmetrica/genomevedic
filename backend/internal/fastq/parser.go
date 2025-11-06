package fastq

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"genomevedic/internal/memory"
	"genomevedic/internal/navigation"
)

// FASTQRead represents a single sequencing read
type FASTQRead struct {
	Header       string    // Read identifier
	Sequence     string    // DNA sequence
	QualityScore string    // Phred quality scores
	AvgQuality   float64   // Average quality
	GCContent    float64   // GC content ratio
	ReadLength   int       // Length of read
	Position     uint64    // Genomic position (mapped)
}

// FASTQParser parses FASTQ files and converts them to particles
type FASTQParser struct {
	coordSystem   *navigation.CoordinateSystem
	memManager    *memory.MemoryManager
	minQuality    float64  // Minimum average quality threshold
	reads         []FASTQRead
	particleCount uint64
}

// NewFASTQParser creates a new FASTQ parser
func NewFASTQParser(minQuality float64) *FASTQParser {
	return &FASTQParser{
		coordSystem: navigation.NewCoordinateSystem(1.0, 1000.0, 2000.0), // Default scale factors
		memManager:  memory.GetGlobalMemoryManager(),
		minQuality:  minQuality,
		reads:       make([]FASTQRead, 0, 10000),
	}
}

// ParseFile parses a FASTQ file from an io.Reader
func (fp *FASTQParser) ParseFile(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB buffer for large reads

	lineNum := 0
	var currentRead FASTQRead

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		switch lineNum % 4 {
		case 1: // Header line
			if !strings.HasPrefix(line, "@") {
				return fmt.Errorf("invalid FASTQ header at line %d: %s", lineNum, line)
			}
			currentRead = FASTQRead{
				Header: line[1:], // Remove @ prefix
			}

		case 2: // Sequence line
			currentRead.Sequence = line
			currentRead.ReadLength = len(line)
			currentRead.GCContent = calculateGCContent(line)

		case 3: // Plus line (separator)
			if !strings.HasPrefix(line, "+") {
				return fmt.Errorf("invalid FASTQ separator at line %d: %s", lineNum, line)
			}
			// Ignore separator line

		case 0: // Quality line
			currentRead.QualityScore = line
			if len(currentRead.QualityScore) != currentRead.ReadLength {
				return fmt.Errorf("quality length mismatch at line %d: seq=%d, qual=%d",
					lineNum, currentRead.ReadLength, len(currentRead.QualityScore))
			}

			// Calculate average quality
			currentRead.AvgQuality = calculateAvgQuality(currentRead.QualityScore)

			// Apply quality filter
			if currentRead.AvgQuality >= fp.minQuality {
				// Assign genomic position (simple mapping for now)
				currentRead.Position = fp.assignGenomicPosition(len(fp.reads), currentRead.ReadLength)
				fp.reads = append(fp.reads, currentRead)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading FASTQ: %w", err)
	}

	return nil
}

// assignGenomicPosition assigns a genomic position to a read
// In production, this would use an aligner like BWA
// For now, we distribute reads evenly across the genome
func (fp *FASTQParser) assignGenomicPosition(readIndex int, seqLength int) uint64 {
	// Distribute reads evenly across genome
	totalGenome := navigation.TotalGenomeLength
	position := uint64(readIndex) * (totalGenome / uint64(readIndex+1))

	// Add some randomness to avoid uniform distribution
	// In real implementation, this would come from read alignment
	offset := uint64(seqLength * 100)
	position = (position + offset) % totalGenome

	return position
}

// GenerateParticles generates particles from parsed reads
func (fp *FASTQParser) GenerateParticles() *memory.ParticleSlice {
	particleSlice := fp.memManager.GetParticleSlice()

	particleCount := 0
	for i := 0; i < len(fp.reads) && particleCount < len(particleSlice.Data); i++ {
		read := fp.reads[i]

		// Convert genomic position to 3D coordinates
		pos3D := fp.coordSystem.LinearTo3D(read.Position)

		// Color based on GC content
		color := gcContentColor(read.GCContent)

		// Size based on quality (higher quality = larger particles)
		size := qualityToSize(read.AvgQuality)

		// Create particle
		particleSlice.Data[particleCount] = memory.Particle{
			Position: pos3D,
			Color:    color,
			Size:     size,
			Metadata: read.Position,
		}

		particleCount++
	}

	particleSlice.Length = particleCount
	fp.particleCount = uint64(particleCount)

	return particleSlice
}

// GenerateParticleStream generates particles in batches for streaming
func (fp *FASTQParser) GenerateParticleStream(batchSize int, callback func(*memory.ParticleSlice) error) error {
	totalReads := len(fp.reads)
	batches := (totalReads + batchSize - 1) / batchSize

	for batch := 0; batch < batches; batch++ {
		startIdx := batch * batchSize
		endIdx := startIdx + batchSize
		if endIdx > totalReads {
			endIdx = totalReads
		}

		// Get particle slice from pool
		particleSlice := fp.memManager.GetParticleSlice()

		particleCount := 0
		for i := startIdx; i < endIdx && particleCount < len(particleSlice.Data); i++ {
			read := fp.reads[i]

			pos3D := fp.coordSystem.LinearTo3D(read.Position)
			color := gcContentColor(read.GCContent)
			size := qualityToSize(read.AvgQuality)

			particleSlice.Data[particleCount] = memory.Particle{
				Position: pos3D,
				Color:    color,
				Size:     size,
				Metadata: read.Position,
			}

			particleCount++
		}

		particleSlice.Length = particleCount

		// Callback with particle batch
		if err := callback(particleSlice); err != nil {
			fp.memManager.PutParticleSlice(particleSlice)
			return err
		}

		// Return to pool
		fp.memManager.PutParticleSlice(particleSlice)
	}

	return nil
}

// Statistics returns parsing statistics
func (fp *FASTQParser) Statistics() map[string]interface{} {
	if len(fp.reads) == 0 {
		return map[string]interface{}{
			"total_reads":     0,
			"particles":       0,
			"avg_quality":     0.0,
			"avg_gc_content":  0.0,
			"avg_read_length": 0,
		}
	}

	totalQuality := 0.0
	totalGC := 0.0
	totalLength := 0

	for _, read := range fp.reads {
		totalQuality += read.AvgQuality
		totalGC += read.GCContent
		totalLength += read.ReadLength
	}

	numReads := float64(len(fp.reads))

	return map[string]interface{}{
		"total_reads":     len(fp.reads),
		"particles":       fp.particleCount,
		"avg_quality":     totalQuality / numReads,
		"avg_gc_content":  totalGC / numReads,
		"avg_read_length": float64(totalLength) / numReads,
		"min_quality":     fp.minQuality,
	}
}

// Reset resets the parser for reuse
func (fp *FASTQParser) Reset() {
	fp.reads = fp.reads[:0]
	fp.particleCount = 0
}

// Helper functions

// calculateGCContent calculates GC content ratio
func calculateGCContent(sequence string) float64 {
	gcCount := 0
	total := len(sequence)

	if total == 0 {
		return 0.0
	}

	for _, base := range sequence {
		switch base {
		case 'G', 'g', 'C', 'c':
			gcCount++
		}
	}

	return float64(gcCount) / float64(total)
}

// calculateAvgQuality calculates average Phred quality score
// Assumes Phred+33 encoding (Illumina 1.8+)
func calculateAvgQuality(qualityString string) float64 {
	if len(qualityString) == 0 {
		return 0.0
	}

	totalQuality := 0
	for _, char := range qualityString {
		// Phred+33: ASCII value - 33 = quality score
		quality := int(char) - 33
		if quality < 0 {
			quality = 0
		}
		totalQuality += quality
	}

	return float64(totalQuality) / float64(len(qualityString))
}

// gcContentColor returns color based on GC content
// Low GC (AT-rich) = blue
// Medium GC = green
// High GC = red
func gcContentColor(gcContent float64) [4]float32 {
	// GC content typically ranges from 0.0 to 1.0
	// Map to color spectrum: blue → green → red

	if gcContent < 0.4 {
		// Low GC (AT-rich): blue to cyan
		t := gcContent / 0.4
		return [4]float32{
			0.0,
			float32(t * 0.5),
			1.0,
			1.0,
		}
	} else if gcContent < 0.6 {
		// Medium GC: cyan to green to yellow
		t := (gcContent - 0.4) / 0.2
		return [4]float32{
			float32(t),
			1.0,
			float32(1.0 - t),
			1.0,
		}
	} else {
		// High GC: yellow to red
		t := (gcContent - 0.6) / 0.4
		return [4]float32{
			1.0,
			float32(1.0 - t),
			0.0,
			1.0,
		}
	}
}

// qualityToSize maps quality score to particle size
func qualityToSize(avgQuality float64) float32 {
	// Quality typically ranges from 0 to 40
	// Map to size range: 0.5 to 2.0

	minSize := 0.5
	maxSize := 2.0
	minQuality := 0.0
	maxQuality := 40.0

	// Clamp quality to range
	if avgQuality < minQuality {
		avgQuality = minQuality
	}
	if avgQuality > maxQuality {
		avgQuality = maxQuality
	}

	// Linear mapping
	t := (avgQuality - minQuality) / (maxQuality - minQuality)
	size := minSize + t*(maxSize-minSize)

	return float32(size)
}

// FASTQMetadata stores summary information about a FASTQ file
type FASTQMetadata struct {
	FileName        string
	TotalReads      int
	FilteredReads   int
	AvgQuality      float64
	AvgGCContent    float64
	AvgReadLength   float64
	MinQuality      float64
	Format          string // "Illumina", "Sanger", etc.
}

// QuickMetadata quickly reads metadata from first N reads without full parsing
func QuickMetadata(reader io.Reader, numReads int) (*FASTQMetadata, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	metadata := &FASTQMetadata{
		Format: "Unknown",
	}

	lineNum := 0
	readCount := 0
	totalQuality := 0.0
	totalGC := 0.0
	totalLength := 0

	for scanner.Scan() && readCount < numReads {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		switch lineNum % 4 {
		case 1: // Header
			if strings.Contains(line, "Illumina") {
				metadata.Format = "Illumina"
			} else if strings.Contains(line, "SRA") {
				metadata.Format = "SRA"
			}

		case 2: // Sequence
			totalGC += calculateGCContent(line)
			totalLength += len(line)

		case 0: // Quality
			totalQuality += calculateAvgQuality(line)
			readCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading FASTQ metadata: %w", err)
	}

	if readCount > 0 {
		metadata.TotalReads = readCount
		metadata.AvgQuality = totalQuality / float64(readCount)
		metadata.AvgGCContent = totalGC / float64(readCount)
		metadata.AvgReadLength = float64(totalLength) / float64(readCount)
	}

	return metadata, nil
}
