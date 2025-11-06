// Package loader - FASTQ format parsing
package loader

import (
	"fmt"
	"io"
	"strings"

	"genomevedic/backend/pkg/types"
)

// FASTQParser parses FASTQ format files
// FASTQ format:
//   Line 1: @sequence_id description
//   Line 2: ATCGN... (DNA sequence)
//   Line 3: + (optional description, often just "+")
//   Line 4: quality scores (Phred+33 ASCII encoded)
type FASTQParser struct {
	decompressor *Decompressor
	lineNumber   int
	readsProcessed int
}

// NewFASTQParser creates a new FASTQ parser
func NewFASTQParser(filepath string) (*FASTQParser, error) {
	decompressor, err := NewDecompressor(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to create decompressor: %w", err)
	}

	return &FASTQParser{
		decompressor: decompressor,
		lineNumber:   0,
		readsProcessed: 0,
	}, nil
}

// ParseRead parses a single FASTQ read (4 lines)
func (p *FASTQParser) ParseRead() (*types.FASTQRead, error) {
	// Line 1: Header (starts with @)
	header, err := p.decompressor.ReadLine()
	if err != nil {
		return nil, err
	}
	p.lineNumber++

	if !strings.HasPrefix(header, "@") {
		return nil, fmt.Errorf("invalid FASTQ header at line %d: expected '@', got %s", p.lineNumber, header)
	}

	// Line 2: Sequence
	sequence, err := p.decompressor.ReadLine()
	if err != nil {
		return nil, fmt.Errorf("failed to read sequence at line %d: %w", p.lineNumber, err)
	}
	p.lineNumber++

	// Validate sequence (only ATCGN allowed)
	if err := validateSequence(sequence); err != nil {
		return nil, fmt.Errorf("invalid sequence at line %d: %w", p.lineNumber, err)
	}

	// Line 3: Plus (separator)
	plus, err := p.decompressor.ReadLine()
	if err != nil {
		return nil, fmt.Errorf("failed to read separator at line %d: %w", p.lineNumber, err)
	}
	p.lineNumber++

	if !strings.HasPrefix(plus, "+") {
		return nil, fmt.Errorf("invalid FASTQ separator at line %d: expected '+', got %s", p.lineNumber, plus)
	}

	// Line 4: Quality scores
	quality, err := p.decompressor.ReadLine()
	if err != nil {
		return nil, fmt.Errorf("failed to read quality at line %d: %w", p.lineNumber, err)
	}
	p.lineNumber++

	// Validate quality length matches sequence length
	if len(quality) != len(sequence) {
		return nil, fmt.Errorf("quality length (%d) doesn't match sequence length (%d) at line %d",
			len(quality), len(sequence), p.lineNumber)
	}

	p.readsProcessed++

	return &types.FASTQRead{
		Header:   header,
		Sequence: sequence,
		Plus:     plus,
		Quality:  quality,
	}, nil
}

// StreamReads streams all FASTQ reads through a channel
// This enables parallel processing of reads
func (p *FASTQParser) StreamReads() <-chan *types.FASTQRead {
	readChan := make(chan *types.FASTQRead, 1000) // Buffer 1000 reads

	go func() {
		defer close(readChan)

		for {
			read, err := p.ParseRead()
			if err != nil {
				if err == io.EOF {
					break // End of file
				}
				// Log error but continue (skip malformed reads)
				fmt.Printf("Warning: Failed to parse read at line %d: %v\n", p.lineNumber, err)
				continue
			}

			readChan <- read
		}
	}()

	return readChan
}

// Close closes the underlying decompressor
func (p *FASTQParser) Close() error {
	return p.decompressor.Close()
}

// GetReadsProcessed returns the number of reads successfully processed
func (p *FASTQParser) GetReadsProcessed() int {
	return p.readsProcessed
}

// validateSequence checks if sequence contains only valid bases
func validateSequence(sequence string) error {
	for i, base := range sequence {
		switch base {
		case 'A', 'T', 'G', 'C', 'N', 'a', 't', 'g', 'c', 'n':
			// Valid base
		default:
			return fmt.Errorf("invalid base '%c' at position %d", base, i)
		}
	}
	return nil
}

// EncodeBase converts a base character to numeric value
func EncodeBase(base byte) int {
	switch base {
	case 'A', 'a':
		return types.BaseA
	case 'T', 't':
		return types.BaseT
	case 'G', 'g':
		return types.BaseG
	case 'C', 'c':
		return types.BaseC
	default:
		return types.BaseN // Unknown
	}
}

// DecodeQuality converts Phred+33 quality score to numeric value
func DecodeQuality(qualityChar byte) int {
	return int(qualityChar) - 33
}
