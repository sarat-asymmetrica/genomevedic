// Package loader - FASTQ file loading and decompression
package loader

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"

	"genomevedic/backend/pkg/types"
)

// Decompressor handles streaming decompression of gzipped FASTQ files
type Decompressor struct {
	file       *os.File
	gzipReader *gzip.Reader
	bufReader  *bufio.Reader
	isGzipped  bool
}

// NewDecompressor creates a new decompressor for a FASTQ file
// Automatically detects if file is gzipped based on extension
func NewDecompressor(filepath string) (*Decompressor, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	isGzipped := strings.HasSuffix(filepath, ".gz")

	var reader io.Reader = file

	// If gzipped, wrap with gzip reader
	var gzipReader *gzip.Reader
	if isGzipped {
		gzipReader, err = gzip.NewReader(file)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		reader = gzipReader
	}

	// Buffered reader for efficient reading
	bufReader := bufio.NewReaderSize(reader, types.DefaultBufferSize)

	return &Decompressor{
		file:       file,
		gzipReader: gzipReader,
		bufReader:  bufReader,
		isGzipped:  isGzipped,
	}, nil
}

// ReadLine reads a single line from the decompressed stream
func (d *Decompressor) ReadLine() (string, error) {
	line, err := d.bufReader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// Trim newline characters
	return strings.TrimRight(line, "\r\n"), nil
}

// Close closes all underlying readers and files
func (d *Decompressor) Close() error {
	if d.gzipReader != nil {
		if err := d.gzipReader.Close(); err != nil {
			return fmt.Errorf("failed to close gzip reader: %w", err)
		}
	}

	if d.file != nil {
		if err := d.file.Close(); err != nil {
			return fmt.Errorf("failed to close file: %w", err)
		}
	}

	return nil
}

// Reset resets the decompressor to the beginning of the file
func (d *Decompressor) Reset() error {
	// Seek to beginning of file
	if _, err := d.file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek to beginning: %w", err)
	}

	// If gzipped, recreate gzip reader
	if d.isGzipped {
		if d.gzipReader != nil {
			d.gzipReader.Close()
		}

		var err error
		d.gzipReader, err = gzip.NewReader(d.file)
		if err != nil {
			return fmt.Errorf("failed to recreate gzip reader: %w", err)
		}

		d.bufReader = bufio.NewReaderSize(d.gzipReader, types.DefaultBufferSize)
	} else {
		d.bufReader = bufio.NewReaderSize(d.file, types.DefaultBufferSize)
	}

	return nil
}

// GetReader returns the underlying buffered reader
func (d *Decompressor) GetReader() *bufio.Reader {
	return d.bufReader
}
