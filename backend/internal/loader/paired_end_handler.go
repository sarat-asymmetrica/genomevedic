package loader

import (
	"fmt"
)

// PairedEndRead represents a pair of reads (R1 and R2)
type PairedEndRead struct {
	ID           string
	Sequence1    string // Forward read (R1)
	Quality1     string
	Sequence2    string // Reverse read (R2)
	Quality2     string
	InsertSize   int    // Distance between R1 and R2
	IsProperPair bool   // True if reads map to expected distance
}

// PairedEndHandler manages paired-end FASTQ reads
type PairedEndHandler struct {
	format         *FormatDetector
	orphanedReads  map[string]*OrphanedRead // Reads waiting for mate
	pairedCount    int64
	orphanedCount  int64
	mismatchCount  int64
}

// OrphanedRead represents a read that hasn't found its mate yet
type OrphanedRead struct {
	Header   string
	Sequence string
	Quality  string
	ReadNum  int // 1 or 2
}

// NewPairedEndHandler creates a new paired-end handler
func NewPairedEndHandler(format *FormatDetector) *PairedEndHandler {
	return &PairedEndHandler{
		format:        format,
		orphanedReads: make(map[string]*OrphanedRead),
		pairedCount:   0,
		orphanedCount: 0,
		mismatchCount: 0,
	}
}

// ProcessRead processes a single FASTQ read and attempts to pair it
// Returns a complete paired read if mate is found, otherwise nil
func (peh *PairedEndHandler) ProcessRead(header, sequence, quality string) (*PairedEndRead, error) {
	// Extract read ID and read number (1 or 2)
	readID, readNum, err := peh.parseReadID(header)
	if err != nil {
		return nil, fmt.Errorf("failed to parse read ID: %w", err)
	}

	// Check if mate is already waiting
	if mate, exists := peh.orphanedReads[readID]; exists {
		// Found mate! Create paired read
		delete(peh.orphanedReads, readID)
		peh.pairedCount++

		if readNum == 1 && mate.ReadNum == 2 {
			// Current is R1, mate is R2
			return &PairedEndRead{
				ID:           readID,
				Sequence1:    sequence,
				Quality1:     quality,
				Sequence2:    mate.Sequence,
				Quality2:     mate.Quality,
				InsertSize:   0, // Would be calculated from alignment
				IsProperPair: true,
			}, nil
		} else if readNum == 2 && mate.ReadNum == 1 {
			// Current is R2, mate is R1
			return &PairedEndRead{
				ID:           readID,
				Sequence1:    mate.Sequence,
				Quality1:     mate.Quality,
				Sequence2:    sequence,
				Quality2:     quality,
				InsertSize:   0,
				IsProperPair: true,
			}, nil
		} else {
			// Read numbers don't match (both R1 or both R2)
			peh.mismatchCount++
			return nil, fmt.Errorf("mismatched read numbers: %d and %d", readNum, mate.ReadNum)
		}
	}

	// Mate not found yet, store as orphaned
	peh.orphanedReads[readID] = &OrphanedRead{
		Header:   header,
		Sequence: sequence,
		Quality:  quality,
		ReadNum:  readNum,
	}
	peh.orphanedCount++

	return nil, nil // No pair yet
}

// parseReadID extracts the read ID and read number from a header
// Supports multiple formats:
// - Illumina 1.8+: @INSTRUMENT:RUN:FLOWCELL:LANE:TILE:X:Y 1:N:0:ATCG
// - Illumina old:  @READID/1 or @READID/2
// - Generic:       @READID_1 or @READID_2
func (peh *PairedEndHandler) parseReadID(header string) (string, int, error) {
	if len(header) < 2 {
		return "", 0, fmt.Errorf("header too short")
	}

	// Remove @ prefix
	header = header[1:]

	// Illumina 1.8+ format: @INSTRUMENT:... 1:N:0:ATCG
	if spaceIdx := findChar(header, ' '); spaceIdx != -1 {
		readID := header[:spaceIdx]
		rest := header[spaceIdx+1:]

		if len(rest) > 0 && (rest[0] == '1' || rest[0] == '2') {
			readNum := int(rest[0] - '0')
			return readID, readNum, nil
		}
	}

	// Illumina old format: @READID/1 or @READID/2
	if slashIdx := findChar(header, '/'); slashIdx != -1 {
		readID := header[:slashIdx]
		rest := header[slashIdx+1:]

		if len(rest) > 0 && (rest[0] == '1' || rest[0] == '2') {
			readNum := int(rest[0] - '0')
			return readID, readNum, nil
		}
	}

	// Generic format: @READID_1 or @READID_2
	if underscoreIdx := findLastChar(header, '_'); underscoreIdx != -1 {
		readID := header[:underscoreIdx]
		rest := header[underscoreIdx+1:]

		if len(rest) > 0 && (rest[0] == '1' || rest[0] == '2') {
			readNum := int(rest[0] - '0')
			return readID, readNum, nil
		}
	}

	// No read number found, treat as single-end
	return header, 1, nil
}

// FlushOrphans returns all orphaned reads that never found their mates
func (peh *PairedEndHandler) FlushOrphans() []*OrphanedRead {
	orphans := make([]*OrphanedRead, 0, len(peh.orphanedReads))
	for _, read := range peh.orphanedReads {
		orphans = append(orphans, read)
	}
	return orphans
}

// GetStats returns pairing statistics
func (peh *PairedEndHandler) GetStats() PairedEndStats {
	return PairedEndStats{
		PairedCount:   peh.pairedCount,
		OrphanedCount: int64(len(peh.orphanedReads)),
		MismatchCount: peh.mismatchCount,
		PairingRate:   peh.calculatePairingRate(),
	}
}

// calculatePairingRate computes the percentage of reads that were successfully paired
func (peh *PairedEndHandler) calculatePairingRate() float64 {
	total := peh.pairedCount*2 + int64(len(peh.orphanedReads))
	if total == 0 {
		return 0.0
	}
	return float64(peh.pairedCount*2) / float64(total) * 100.0
}

// PairedEndStats contains pairing statistics
type PairedEndStats struct {
	PairedCount   int64   // Number of successfully paired reads
	OrphanedCount int64   // Number of reads without mates
	MismatchCount int64   // Number of reads with mismatched pair numbers
	PairingRate   float64 // Percentage of reads successfully paired
}

// PrintStats prints pairing statistics
func (stats PairedEndStats) PrintStats() {
	fmt.Println("\nPaired-End Statistics:")
	fmt.Printf("  Paired reads:     %d\n", stats.PairedCount)
	fmt.Printf("  Orphaned reads:   %d\n", stats.OrphanedCount)
	fmt.Printf("  Mismatched reads: %d\n", stats.MismatchCount)
	fmt.Printf("  Pairing rate:     %.1f%%\n", stats.PairingRate)
}

// MergePairedReads merges R1 and R2 sequences for visualization
// This is useful for short-insert paired-end reads where R1 and R2 overlap
func MergePairedReads(read *PairedEndRead) string {
	// For non-overlapping reads, just concatenate with a gap
	// In real implementation, would check for overlap and merge intelligently
	gap := "NNNNNNNNNN" // 10 N's to represent insert
	return read.Sequence1 + gap + reverseComplement(read.Sequence2)
}

// reverseComplement returns the reverse complement of a DNA sequence
func reverseComplement(sequence string) string {
	complement := map[byte]byte{
		'A': 'T',
		'T': 'A',
		'G': 'C',
		'C': 'G',
		'N': 'N',
	}

	reversed := make([]byte, len(sequence))
	for i := 0; i < len(sequence); i++ {
		base := sequence[len(sequence)-1-i]
		if comp, exists := complement[base]; exists {
			reversed[i] = comp
		} else {
			reversed[i] = 'N' // Unknown base
		}
	}

	return string(reversed)
}

// findChar finds the first occurrence of a character in a string
func findChar(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// findLastChar finds the last occurrence of a character in a string
func findLastChar(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// InterleaveReads combines two separate FASTQ files (R1 and R2) into paired reads
// This is useful when reads are stored in separate files
func InterleaveReads(r1Path, r2Path string) (<-chan *PairedEndRead, error) {
	// Open both files
	// Read from both simultaneously
	// Pair up reads
	// Return channel of paired reads
	//
	// Simplified implementation - full version would handle files properly

	pairedChan := make(chan *PairedEndRead, 1000)

	// In real implementation, would open files and stream pairs
	// For now, return empty channel
	go func() {
		defer close(pairedChan)
		// Would process files here
	}()

	return pairedChan, nil
}
