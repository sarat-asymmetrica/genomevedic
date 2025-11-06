// Package vedic - Digital root color mapping
package vedic

import (
	"image/color"
	"strings"
)

// DigitalRootColor computes color based on digital root of sequence
//
// Algorithm (from MATHEMATICAL_FOUNDATIONS.md):
//   Base_to_number: A=1, C=2, G=3, T=4
//   Digital_root: Reduce sequence to single digit (base-9)
//   Hue: (digital_root × 40°) mod 360°
//
// Rationale:
//   - Digital root reveals modulo-9 patterns in DNA
//   - Each root (1-9) gets a distinct hue
//   - Similar sequences cluster in color space
func DigitalRootColor(sequence string) color.RGBA {
	dr := ComputeSequenceDigitalRoot(sequence)

	// Map digital root to hue (9 distinct colors)
	hue := float64(dr * 40) // 0°, 40°, 80°, ..., 320°

	// Full saturation for vivid colors
	saturation := 0.9

	// Medium lightness
	lightness := 0.5

	return HSLToRGB(hue, saturation, lightness)
}

// ComputeSequenceDigitalRoot computes the digital root of a DNA sequence
func ComputeSequenceDigitalRoot(sequence string) int {
	sequence = strings.ToUpper(sequence)

	sum := 0
	for _, base := range sequence {
		sum += encodeBaseToNumber(byte(base))
	}

	return DigitalRoot(sum)
}

// DigitalRoot computes the Vedic digital root of a number
// Formula: dr(n) = 1 + ((n - 1) mod 9)
func DigitalRoot(n int) int {
	if n == 0 {
		return 0
	}
	if n < 0 {
		n = -n
	}
	result := n % 9
	if result == 0 {
		return 9
	}
	return result
}

// encodeBaseToNumber converts a DNA base to a number
// A=1 (adenine, purine)
// C=2 (cytosine, pyrimidine)
// G=3 (guanine, purine)
// T=4 (thymine, pyrimidine)
func encodeBaseToNumber(base byte) int {
	switch base {
	case 'A':
		return 1
	case 'C':
		return 2
	case 'G':
		return 3
	case 'T':
		return 4
	default:
		return 0 // Unknown base
	}
}

// DigitalRootPalette generates 9 distinct colors (one per digital root)
func DigitalRootPalette() [9]color.RGBA {
	var palette [9]color.RGBA

	for i := 0; i < 9; i++ {
		dr := i + 1 // Digital roots are 1-9
		hue := float64(dr * 40)
		palette[i] = HSLToRGB(hue, 0.9, 0.5)
	}

	return palette
}

// GetDigitalRootPattern returns the Vedic pattern name for a digital root
func GetDigitalRootPattern(dr int) string {
	patterns := []string{
		"Origin",      // 0 (not used in Vedic, but included)
		"Unity",       // 1
		"Duality",     // 2
		"Trinity",     // 3
		"Foundation",  // 4
		"Balance",     // 5
		"Harmony",     // 6
		"Mystical",    // 7
		"Infinity",    // 8
		"Completion",  // 9
	}

	if dr < 0 || dr >= len(patterns) {
		return "Unknown"
	}

	return patterns[dr]
}

// CodonDigitalRoot computes digital root for triplet codons
// This reveals patterns in genetic code
func CodonDigitalRoot(codon string) int {
	if len(codon) != 3 {
		return 0
	}

	codon = strings.ToUpper(codon)

	// Weight each position differently (biological significance)
	sum := encodeBaseToNumber(codon[0])*1 +
		encodeBaseToNumber(codon[1])*2 +
		encodeBaseToNumber(codon[2])*3

	return DigitalRoot(sum)
}

// DigitalRootGradient creates a smooth color gradient based on digital root
func DigitalRootGradient(dr int, intensity float64) color.RGBA {
	// Base hue from digital root
	hue := float64(dr * 40)

	// Vary saturation with intensity
	saturation := 0.5 + intensity*0.5
	saturation = clamp(saturation, 0.0, 1.0)

	// Vary lightness with intensity
	lightness := 0.3 + intensity*0.4
	lightness = clamp(lightness, 0.0, 1.0)

	return HSLToRGB(hue, saturation, lightness)
}

// SequenceColorByBase colors each base individually by digital root
func SequenceColorByBase(sequence string) []color.RGBA {
	sequence = strings.ToUpper(sequence)
	colors := make([]color.RGBA, len(sequence))

	for i, base := range sequence {
		// Color each base by its digital root
		colors[i] = DigitalRootColor(string(base))
	}

	return colors
}
