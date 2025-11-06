package loader

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// SRADownloader downloads FASTQ files from NCBI SRA
type SRADownloader struct {
	CacheDir      string
	PrefetchPath  string // Path to SRA prefetch binary
	FastqDumpPath string // Path to fastq-dump binary
}

// NewSRADownloader creates a new SRA downloader
func NewSRADownloader(cacheDir string) *SRADownloader {
	return &SRADownloader{
		CacheDir:      cacheDir,
		PrefetchPath:  "prefetch",   // Assumes SRA Toolkit in PATH
		FastqDumpPath: "fastq-dump", // Assumes SRA Toolkit in PATH
	}
}

// Download downloads a FASTQ file from SRA by accession
// Example accession: SRR292678 (human genome, Illumina HiSeq)
func (sd *SRADownloader) Download(accession string) (string, error) {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(sd.CacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Check if file already exists in cache
	cachedFile := filepath.Join(sd.CacheDir, accession+".fastq.gz")
	if _, err := os.Stat(cachedFile); err == nil {
		fmt.Printf("Using cached file: %s\n", cachedFile)
		return cachedFile, nil
	}

	// Step 1: Prefetch SRA file (downloads .sra format)
	fmt.Printf("Downloading %s from NCBI SRA...\n", accession)
	prefetchCmd := exec.Command(sd.PrefetchPath, accession, "-O", sd.CacheDir)
	if err := prefetchCmd.Run(); err != nil {
		return "", fmt.Errorf("prefetch failed: %w\nHint: Install SRA Toolkit: https://github.com/ncbi/sra-tools/wiki/02.-Installing-SRA-Toolkit", err)
	}

	// Step 2: Convert SRA to FASTQ (compressed)
	fmt.Printf("Converting %s to FASTQ format...\n", accession)
	sraFile := filepath.Join(sd.CacheDir, accession, accession+".sra")
	fastqDumpCmd := exec.Command(
		sd.FastqDumpPath,
		"--split-files",    // Split paired-end reads into separate files
		"--gzip",           // Compress output
		"--outdir", sd.CacheDir,
		sraFile,
	)

	if err := fastqDumpCmd.Run(); err != nil {
		return "", fmt.Errorf("fastq-dump failed: %w", err)
	}

	// Return path to first FASTQ file
	// For paired-end, this will be *_1.fastq.gz
	// For single-end, this will be *.fastq.gz
	fastqFile := filepath.Join(sd.CacheDir, accession+"_1.fastq.gz")
	if _, err := os.Stat(fastqFile); err != nil {
		// Try single-end filename
		fastqFile = filepath.Join(sd.CacheDir, accession+".fastq.gz")
		if _, err := os.Stat(fastqFile); err != nil {
			return "", fmt.Errorf("FASTQ file not found after conversion")
		}
	}

	fmt.Printf("Downloaded and converted: %s\n", fastqFile)
	return fastqFile, nil
}

// DownloadMock simulates SRA download by creating a dummy file
// Used for testing when SRA Toolkit is not available
func (sd *SRADownloader) DownloadMock(accession string, numReads int) (string, error) {
	// Create cache directory
	if err := os.MkdirAll(sd.CacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create dummy FASTQ file
	fastqFile := filepath.Join(sd.CacheDir, accession+"_mock.fastq")
	file, err := os.Create(fastqFile)
	if err != nil {
		return "", fmt.Errorf("failed to create mock file: %w", err)
	}
	defer file.Close()

	// Generate synthetic FASTQ data
	fmt.Printf("Generating mock FASTQ with %d reads...\n", numReads)
	for i := 0; i < numReads; i++ {
		// Header
		fmt.Fprintf(file, "@SRR%s.%d\n", accession, i+1)

		// Sequence (150 bp Illumina read)
		sequence := generateRandomSequence(150)
		fmt.Fprintf(file, "%s\n", sequence)

		// Plus line
		fmt.Fprintf(file, "+\n")

		// Quality scores (Phred+33, Q30 = ASCII 63 = '?')
		quality := generateHighQuality(150)
		fmt.Fprintf(file, "%s\n", quality)
	}

	fmt.Printf("Generated mock FASTQ: %s\n", fastqFile)
	return fastqFile, nil
}

// ListPopularAccessions returns a list of popular genomic datasets
func (sd *SRADownloader) ListPopularAccessions() []SRAAccession {
	return []SRAAccession{
		{
			Accession:   "SRR292678",
			Description: "Human genome (HG19), Illumina HiSeq 2000, paired-end 100bp",
			Organism:    "Homo sapiens",
			Platform:    "Illumina HiSeq 2000",
			Size:        "3.2 GB",
		},
		{
			Accession:   "SRR1777291",
			Description: "Human whole genome sequencing, Illumina HiSeq 2500",
			Organism:    "Homo sapiens",
			Platform:    "Illumina HiSeq 2500",
			Size:        "8.5 GB",
		},
		{
			Accession:   "SRR7890936",
			Description: "E. coli genome, Illumina MiSeq, paired-end 250bp",
			Organism:    "Escherichia coli",
			Platform:    "Illumina MiSeq",
			Size:        "180 MB",
		},
		{
			Accession:   "SRR6052133",
			Description: "Human cancer genome (TCGA), PacBio Sequel",
			Organism:    "Homo sapiens",
			Platform:    "PacBio Sequel",
			Size:        "15 GB",
		},
	}
}

// SRAAccession represents an SRA dataset
type SRAAccession struct {
	Accession   string
	Description string
	Organism    string
	Platform    string
	Size        string
}

// generateRandomSequence generates a random DNA sequence
func generateRandomSequence(length int) string {
	bases := []byte{'A', 'T', 'G', 'C'}
	sequence := make([]byte, length)
	for i := 0; i < length; i++ {
		sequence[i] = bases[i%4]
	}
	return string(sequence)
}

// generateHighQuality generates high-quality Phred+33 quality scores
func generateHighQuality(length int) string {
	// Q30 = 99.9% accuracy = ASCII 63 ('?')
	// Q40 = 99.99% accuracy = ASCII 73 ('I')
	quality := make([]byte, length)
	for i := 0; i < length; i++ {
		quality[i] = '?'  // Q30
	}
	return string(quality)
}

// CheckToolsInstalled verifies that SRA Toolkit is installed
func (sd *SRADownloader) CheckToolsInstalled() error {
	// Check prefetch
	if _, err := exec.LookPath(sd.PrefetchPath); err != nil {
		return fmt.Errorf("prefetch not found: %w\nInstall SRA Toolkit from: https://github.com/ncbi/sra-tools/wiki/02.-Installing-SRA-Toolkit", err)
	}

	// Check fastq-dump
	if _, err := exec.LookPath(sd.FastqDumpPath); err != nil {
		return fmt.Errorf("fastq-dump not found: %w\nInstall SRA Toolkit from: https://github.com/ncbi/sra-tools/wiki/02.-Installing-SRA-Toolkit", err)
	}

	return nil
}
