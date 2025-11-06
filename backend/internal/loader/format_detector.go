package loader

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// FASTQFormat represents the type of FASTQ file
type FASTQFormat int

const (
	FormatUnknown  FASTQFormat = 0
	FormatIllumina FASTQFormat = 1
	FormatPacBio   FASTQFormat = 2
	FormatNanopore FASTQFormat = 3
)

// String returns the format name
func (f FASTQFormat) String() string {
	switch f {
	case FormatIllumina:
		return "Illumina"
	case FormatPacBio:
		return "PacBio"
	case FormatNanopore:
		return "Nanopore (ONT)"
	default:
		return "Unknown"
	}
}

// QualityEncoding represents the quality score encoding scheme
type QualityEncoding int

const (
	QualityPhred33 QualityEncoding = 0 // Illumina 1.8+, PacBio, Nanopore (offset 33)
	QualityPhred64 QualityEncoding = 1 // Illumina 1.3-1.7 (offset 64)
	QualitySolexa  QualityEncoding = 2 // Early Solexa (different formula)
)

// FormatDetector auto-detects FASTQ format and quality encoding
type FormatDetector struct {
	Format          FASTQFormat
	QualityEncoding QualityEncoding
	AverageReadLength int
	IsPairedEnd     bool
}

// DetectFromFile analyzes a FASTQ file and detects its format
func DetectFromFile(filepath string) (*FormatDetector, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	detector := &FormatDetector{
		Format:          FormatUnknown,
		QualityEncoding: QualityPhred33,
		AverageReadLength: 0,
		IsPairedEnd:     false,
	}

	readCount := 0
	totalLength := 0

	// Analyze first 1000 reads
	for readCount < 1000 && scanner.Scan() {
		// Line 1: Header
		header := scanner.Text()
		if !strings.HasPrefix(header, "@") {
			continue // Skip malformed reads
		}

		// Detect format from header
		detector.Format = detector.detectFormatFromHeader(header)

		// Line 2: Sequence
		if !scanner.Scan() {
			break
		}
		sequence := scanner.Text()
		totalLength += len(sequence)

		// Line 3: Plus line
		if !scanner.Scan() {
			break
		}

		// Line 4: Quality scores
		if !scanner.Scan() {
			break
		}
		quality := scanner.Text()

		// Detect quality encoding
		if readCount == 0 {
			detector.QualityEncoding = detector.detectQualityEncoding(quality)
		}

		// Detect paired-end reads
		if strings.Contains(header, "/1") || strings.Contains(header, "/2") ||
			strings.Contains(header, " 1:") || strings.Contains(header, " 2:") {
			detector.IsPairedEnd = true
		}

		readCount++
	}

	if readCount > 0 {
		detector.AverageReadLength = totalLength / readCount
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return detector, nil
}

// detectFormatFromHeader detects FASTQ format from the header line
func (fd *FormatDetector) detectFormatFromHeader(header string) FASTQFormat {
	// Illumina format: @INSTRUMENT:RUNID:FLOWCELL:LANE:TILE:X:Y
	if strings.Contains(header, "Illumina") || strings.Count(header, ":") >= 6 {
		return FormatIllumina
	}

	// PacBio format: @m64xxx_xxx or contains "PacBio"
	if strings.HasPrefix(header, "@m64") || strings.HasPrefix(header, "@m54") ||
		strings.Contains(header, "PacBio") {
		return FormatPacBio
	}

	// Nanopore format: Contains "ONT" or "Nanopore" or starts with specific prefixes
	if strings.Contains(header, "ONT") || strings.Contains(header, "Nanopore") ||
		strings.Contains(header, "runid=") {
		return FormatNanopore
	}

	// SRA format: @SRR or @ERR or @DRR
	if strings.HasPrefix(header, "@SRR") || strings.HasPrefix(header, "@ERR") ||
		strings.HasPrefix(header, "@DRR") {
		// SRA files are usually Illumina
		return FormatIllumina
	}

	return FormatUnknown
}

// detectQualityEncoding detects the quality score encoding from a quality string
func (fd *FormatDetector) detectQualityEncoding(quality string) QualityEncoding {
	if len(quality) == 0 {
		return QualityPhred33
	}

	// Check ASCII values
	minChar := quality[0]
	maxChar := quality[0]

	for i := 1; i < len(quality); i++ {
		if quality[i] < minChar {
			minChar = quality[i]
		}
		if quality[i] > maxChar {
			maxChar = quality[i]
		}
	}

	// Phred+33: ASCII 33-126 (!"#$%...~)
	// Phred+64: ASCII 64-126 (@ABC...~)
	//
	// If minimum is < 64, must be Phred+33
	// If minimum is >= 64, likely Phred+64 (old Illumina)

	if minChar < 64 {
		return QualityPhred33
	} else if minChar >= 64 && maxChar < 105 {
		return QualityPhred64
	}

	return QualityPhred33 // Default to Phred+33 (modern standard)
}

// ParseQualityScore converts a quality character to a Phred score
func (fd *FormatDetector) ParseQualityScore(qual byte) int {
	switch fd.QualityEncoding {
	case QualityPhred33:
		return int(qual) - 33
	case QualityPhred64:
		return int(qual) - 64
	case QualitySolexa:
		// Solexa uses a different formula: Q = 10 × log₁₀(p/(1-p))
		// For simplicity, treat as Phred+64
		return int(qual) - 64
	default:
		return 0
	}
}

// GetAverageQuality calculates the average quality score for a quality string
func (fd *FormatDetector) GetAverageQuality(quality string) float64 {
	if len(quality) == 0 {
		return 0.0
	}

	sum := 0
	for i := 0; i < len(quality); i++ {
		sum += fd.ParseQualityScore(quality[i])
	}

	return float64(sum) / float64(len(quality))
}

// IsHighQuality returns true if the average quality is above a threshold
// Illumina: Q30 = 99.9% accuracy (good)
// PacBio: Q20 = 99% accuracy (acceptable for long reads)
// Nanopore: Q15 = 96.8% accuracy (acceptable for ultra-long reads)
func (fd *FormatDetector) IsHighQuality(quality string) bool {
	avgQual := fd.GetAverageQuality(quality)

	switch fd.Format {
	case FormatIllumina:
		return avgQual >= 30 // Q30 threshold for Illumina
	case FormatPacBio:
		return avgQual >= 20 // Q20 threshold for PacBio
	case FormatNanopore:
		return avgQual >= 15 // Q15 threshold for Nanopore
	default:
		return avgQual >= 20 // Default Q20 threshold
	}
}

// GetReadTypeName returns a human-readable description of the read type
func (fd *FormatDetector) GetReadTypeName() string {
	switch fd.Format {
	case FormatIllumina:
		if fd.IsPairedEnd {
			return fmt.Sprintf("Illumina paired-end (~%d bp)", fd.AverageReadLength)
		}
		return fmt.Sprintf("Illumina single-end (~%d bp)", fd.AverageReadLength)
	case FormatPacBio:
		return fmt.Sprintf("PacBio long reads (~%d bp)", fd.AverageReadLength)
	case FormatNanopore:
		return fmt.Sprintf("Nanopore ultra-long reads (~%d bp)", fd.AverageReadLength)
	default:
		return fmt.Sprintf("Unknown format (~%d bp)", fd.AverageReadLength)
	}
}

// Validate validates a FASTQ read
func (fd *FormatDetector) Validate(header, sequence, plus, quality string) error {
	// Check header
	if !strings.HasPrefix(header, "@") {
		return fmt.Errorf("invalid header: must start with @")
	}

	// Check plus line
	if plus != "+" && !strings.HasPrefix(plus, "+") {
		return fmt.Errorf("invalid plus line: must be + or +<description>")
	}

	// Check sequence and quality length match
	if len(sequence) != len(quality) {
		return fmt.Errorf("sequence length (%d) != quality length (%d)",
			len(sequence), len(quality))
	}

	// Check for valid bases
	for i := 0; i < len(sequence); i++ {
		base := sequence[i]
		if base != 'A' && base != 'T' && base != 'G' && base != 'C' && base != 'N' {
			return fmt.Errorf("invalid base '%c' at position %d", base, i)
		}
	}

	return nil
}

// PrintSummary prints a summary of the detected format
func (fd *FormatDetector) PrintSummary() {
	fmt.Println("FASTQ Format Detection Summary:")
	fmt.Printf("  Format:            %s\n", fd.Format)
	fmt.Printf("  Read type:         %s\n", fd.GetReadTypeName())
	fmt.Printf("  Quality encoding:  Phred+%d\n", fd.getQualityOffset())
	fmt.Printf("  Avg read length:   %d bp\n", fd.AverageReadLength)
	fmt.Printf("  Paired-end:        %v\n", fd.IsPairedEnd)
}

func (fd *FormatDetector) getQualityOffset() int {
	switch fd.QualityEncoding {
	case QualityPhred33:
		return 33
	case QualityPhred64:
		return 64
	default:
		return 33
	}
}
