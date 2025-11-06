package fastq

import (
	"strings"
	"testing"

	"genomevedic/internal/memory"
)

const validFASTQ = `@READ1
ATCGATCG
+
IIIIIIII
@READ2
GCGCGCGC
+
HHHHHHHH
`

const invalidFASTQ = `INVALID
ATCGATCG
+
IIIIIIII
`

const lowQualityFASTQ = `@LOWQ
NNNNNNNN
+
!!!!!!!!
`

func TestParseFASTQ(t *testing.T) {
	parser := NewFASTQParser(20.0)
	reader := strings.NewReader(validFASTQ)

	err := parser.ParseFile(reader)
	if err != nil {
		t.Fatalf("ParseFile() error: %v", err)
	}

	stats := parser.Statistics()
	totalReads := stats["total_reads"].(int)
	if totalReads != 2 {
		t.Errorf("Expected 2 reads, got %d", totalReads)
	}
}

func TestInvalidFASTQ(t *testing.T) {
	parser := NewFASTQParser(20.0)
	reader := strings.NewReader(invalidFASTQ)

	err := parser.ParseFile(reader)
	if err == nil {
		t.Error("Expected error for invalid FASTQ")
	}
}

func TestQualityFiltering(t *testing.T) {
	// High quality threshold
	parser := NewFASTQParser(40.0)
	reader := strings.NewReader(validFASTQ)

	parser.ParseFile(reader)
	stats := parser.Statistics()
	highQReads := stats["total_reads"].(int)

	// Low quality threshold
	parser2 := NewFASTQParser(10.0)
	reader2 := strings.NewReader(validFASTQ)

	parser2.ParseFile(reader2)
	stats2 := parser2.Statistics()
	lowQReads := stats2["total_reads"].(int)

	if lowQReads < highQReads {
		t.Error("Low quality threshold should accept more reads")
	}
}

func TestFilterLowQuality(t *testing.T) {
	parser := NewFASTQParser(20.0)
	reader := strings.NewReader(lowQualityFASTQ)

	parser.ParseFile(reader)
	stats := parser.Statistics()
	totalReads := stats["total_reads"].(int)

	if totalReads != 0 {
		t.Errorf("Expected 0 reads (filtered), got %d", totalReads)
	}
}

func TestGCContentCalculation(t *testing.T) {
	tests := []struct {
		sequence string
		expected float64
	}{
		{"ATATATAT", 0.0},
		{"GCGCGCGC", 1.0},
		{"ATCGATCG", 0.5},
		{"AAGGCCTT", 0.5}, // 4 GC out of 8
	}

	for _, tt := range tests {
		result := calculateGCContent(tt.sequence)
		if result != tt.expected {
			t.Errorf("calculateGCContent(%s) = %.2f, expected %.2f",
				tt.sequence, result, tt.expected)
		}
	}
}

func TestQualityScoreCalculation(t *testing.T) {
	// Phred+33 encoding
	// 'I' = 73, so quality = 73 - 33 = 40
	qual := "IIIIIIII"
	avg := calculateAvgQuality(qual)
	if avg != 40.0 {
		t.Errorf("Expected avg quality 40, got %.2f", avg)
	}

	// '!' = 33, so quality = 33 - 33 = 0
	qual2 := "!!!!!!!!"
	avg2 := calculateAvgQuality(qual2)
	if avg2 != 0.0 {
		t.Errorf("Expected avg quality 0, got %.2f", avg2)
	}
}

func TestGenerateParticles(t *testing.T) {
	parser := NewFASTQParser(20.0)
	reader := strings.NewReader(validFASTQ)

	err := parser.ParseFile(reader)
	if err != nil {
		t.Fatalf("ParseFile() error: %v", err)
	}

	particles := parser.GenerateParticles()
	if particles == nil {
		t.Fatal("GenerateParticles() returned nil")
	}

	stats := parser.Statistics()
	expectedCount := stats["total_reads"].(int)

	if particles.Length != expectedCount {
		t.Errorf("Expected %d particles, got %d", expectedCount, particles.Length)
	}

	// Check first particle
	if particles.Length > 0 {
		p := particles.Data[0]

		// Position should be set
		if p.Position[0] == 0 && p.Position[1] == 0 && p.Position[2] == 0 {
			t.Error("Particle position not set")
		}

		// Color should be set
		if p.Color[3] != 1.0 {
			t.Error("Particle alpha should be 1.0")
		}

		// Size should be positive
		if p.Size <= 0 {
			t.Error("Particle size should be positive")
		}
	}
}

func TestParticleStream(t *testing.T) {
	parser := NewFASTQParser(20.0)
	reader := strings.NewReader(validFASTQ)

	parser.ParseFile(reader)

	particleCount := 0
	err := parser.GenerateParticleStream(10, func(ps *memory.ParticleSlice) error {
		particleCount += ps.Length
		return nil
	})

	if err != nil {
		t.Fatalf("GenerateParticleStream() error: %v", err)
	}

	stats := parser.Statistics()
	expectedCount := stats["total_reads"].(int)

	if particleCount != expectedCount {
		t.Errorf("Expected %d particles streamed, got %d", expectedCount, particleCount)
	}
}

func TestQuickMetadata(t *testing.T) {
	reader := strings.NewReader(validFASTQ)

	metadata, err := QuickMetadata(reader, 5)
	if err != nil {
		t.Fatalf("QuickMetadata() error: %v", err)
	}

	if metadata.TotalReads == 0 {
		t.Error("Expected at least 1 read in metadata")
	}

	if metadata.AvgQuality == 0 {
		t.Error("Expected non-zero average quality")
	}

	if metadata.AvgReadLength == 0 {
		t.Error("Expected non-zero average read length")
	}
}

func TestParserReset(t *testing.T) {
	parser := NewFASTQParser(20.0)
	reader := strings.NewReader(validFASTQ)

	parser.ParseFile(reader)
	stats1 := parser.Statistics()
	reads1 := stats1["total_reads"].(int)

	if reads1 == 0 {
		t.Fatal("Expected reads after first parse")
	}

	// Reset
	parser.Reset()
	stats2 := parser.Statistics()
	reads2 := stats2["total_reads"].(int)

	if reads2 != 0 {
		t.Errorf("Expected 0 reads after reset, got %d", reads2)
	}

	// Parse again
	reader2 := strings.NewReader(validFASTQ)
	parser.ParseFile(reader2)
	stats3 := parser.Statistics()
	reads3 := stats3["total_reads"].(int)

	if reads3 != reads1 {
		t.Errorf("Expected %d reads after re-parse, got %d", reads1, reads3)
	}
}

func TestGCContentColor(t *testing.T) {
	tests := []struct {
		gcContent float64
		checkFunc func([4]float32) bool
		desc      string
	}{
		{0.2, func(c [4]float32) bool { return c[2] > 0.5 }, "Low GC should be blue-ish"},
		{0.5, func(c [4]float32) bool { return c[1] > 0.5 }, "Medium GC should be green-ish"},
		{0.8, func(c [4]float32) bool { return c[0] > 0.5 }, "High GC should be red-ish"},
	}

	for _, tt := range tests {
		color := gcContentColor(tt.gcContent)

		if !tt.checkFunc(color) {
			t.Errorf("%s (GC=%.2f, color=%v)", tt.desc, tt.gcContent, color)
		}

		// Alpha should always be 1.0
		if color[3] != 1.0 {
			t.Errorf("Alpha should be 1.0, got %.2f", color[3])
		}
	}
}

func TestQualityToSize(t *testing.T) {
	tests := []struct {
		quality      float64
		minExpected  float32
		maxExpected  float32
	}{
		{0.0, 0.4, 0.6},    // Low quality -> small
		{20.0, 0.9, 1.3},   // Medium quality -> medium
		{40.0, 1.8, 2.1},   // High quality -> large
	}

	for _, tt := range tests {
		size := qualityToSize(tt.quality)

		if size < tt.minExpected || size > tt.maxExpected {
			t.Errorf("Quality %.1f: expected size between %.2f and %.2f, got %.2f",
				tt.quality, tt.minExpected, tt.maxExpected, size)
		}
	}
}

func BenchmarkParseFASTQ(b *testing.B) {
	parser := NewFASTQParser(20.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(validFASTQ)
		parser.ParseFile(reader)
		parser.Reset()
	}
}

func BenchmarkGenerateParticles(b *testing.B) {
	parser := NewFASTQParser(20.0)
	reader := strings.NewReader(validFASTQ)
	parser.ParseFile(reader)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		particles := parser.GenerateParticles()
		_ = particles
	}
}

func BenchmarkStreamParticles(b *testing.B) {
	parser := NewFASTQParser(20.0)
	reader := strings.NewReader(validFASTQ)
	parser.ParseFile(reader)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.GenerateParticleStream(10, func(ps *memory.ParticleSlice) error {
			return nil
		})
	}
}
