// Package vedic - Mutation frequency coloring
package vedic

import (
	"image/color"

	"genomevedic/backend/pkg/types"
)

// MutationFrequencyColor maps mutation frequency to color
//
// Algorithm (from handoff prompt):
//   High frequency (>10 mutations/kb): Red (urgent)
//   Medium frequency (1-10 mutations/kb): Orange (caution)
//   Low frequency (<1 mutation/kb): Blue (stable)
//
// Rationale:
//   - Mutation hotspots should be visually obvious
//   - Color intensity correlates with mutation density
//   - Aids in identifying cancer driver regions
func MutationFrequencyColor(mutationsPerKb float64) color.RGBA {
	if mutationsPerKb > 10.0 {
		// High frequency - RED (urgent)
		return color.RGBA{R: 255, G: 0, B: 0, A: 255}
	} else if mutationsPerKb > 1.0 {
		// Medium frequency - ORANGE (caution)
		// Interpolate between orange and red
		t := (mutationsPerKb - 1.0) / 9.0 // [0, 1]
		return interpolateColor(
			color.RGBA{R: 255, G: 165, B: 0, A: 255}, // Orange
			color.RGBA{R: 255, G: 0, B: 0, A: 255},   // Red
			t,
		)
	} else {
		// Low frequency - BLUE (stable)
		// Darker blue for lower mutation rates
		intensity := uint8(100 + mutationsPerKb*155) // [100, 255]
		return color.RGBA{R: 0, G: 0, B: intensity, A: 255}
	}
}

// MutationTypeColor colors by mutation type (transition vs transversion)
//
// Transitions (purine↔purine, pyrimidine↔pyrimidine): Smooth gradient
// Transversions (purine↔pyrimidine): Sharp contrast
func MutationTypeColor(from, to byte) color.RGBA {
	isTransition := isMutationTransition(from, to)

	if isTransition {
		// Smooth color transition (yellow-green gradient)
		return color.RGBA{R: 180, G: 200, B: 50, A: 255}
	} else {
		// Transversion (red-purple)
		return color.RGBA{R: 200, G: 50, B: 150, A: 255}
	}
}

// isMutationTransition checks if mutation is a transition
// Transitions: A↔G (purines), C↔T (pyrimidines)
// Transversions: everything else
func isMutationTransition(from, to byte) bool {
	// A ↔ G (purines)
	if (from == 'A' && to == 'G') || (from == 'G' && to == 'A') {
		return true
	}

	// C ↔ T (pyrimidines)
	if (from == 'C' && to == 'T') || (from == 'T' && to == 'C') {
		return true
	}

	return false // Transversion
}

// ComputeMutationDensity calculates mutations per kilobase
func ComputeMutationDensity(mutations []types.Mutation, windowSize int) float64 {
	if windowSize == 0 {
		return 0.0
	}

	mutationCount := len(mutations)
	windowSizeKb := float64(windowSize) / 1000.0

	return float64(mutationCount) / windowSizeKb
}

// GetMutationCategory categorizes mutation frequency
func GetMutationCategory(mutationsPerKb float64) string {
	if mutationsPerKb > 10.0 {
		return "Hypermutated" // Cancer mutation hotspot
	} else if mutationsPerKb > 1.0 {
		return "Elevated" // Above baseline
	} else if mutationsPerKb > 0.1 {
		return "Normal" // Typical mutation rate
	} else {
		return "Conserved" // Highly stable region
	}
}

// interpolateColor linearly interpolates between two colors
func interpolateColor(c1, c2 color.RGBA, t float64) color.RGBA {
	// Clamp t to [0, 1]
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}

	return color.RGBA{
		R: uint8(float64(c1.R)*(1-t) + float64(c2.R)*t),
		G: uint8(float64(c1.G)*(1-t) + float64(c2.G)*t),
		B: uint8(float64(c1.B)*(1-t) + float64(c2.B)*t),
		A: 255,
	}
}

// MutationHeatmap creates a heatmap color based on mutation density
func MutationHeatmap(mutationsPerKb float64) color.RGBA {
	// Map [0, 20] mutations/kb to heatmap
	// Blue (low) → Green (medium) → Yellow → Red (high)

	t := clamp(mutationsPerKb/20.0, 0.0, 1.0)

	if t < 0.33 {
		// Blue → Green
		return interpolateColor(
			color.RGBA{R: 0, G: 0, B: 255, A: 255},
			color.RGBA{R: 0, G: 255, B: 0, A: 255},
			t/0.33,
		)
	} else if t < 0.67 {
		// Green → Yellow
		return interpolateColor(
			color.RGBA{R: 0, G: 255, B: 0, A: 255},
			color.RGBA{R: 255, G: 255, B: 0, A: 255},
			(t-0.33)/0.34,
		)
	} else {
		// Yellow → Red
		return interpolateColor(
			color.RGBA{R: 255, G: 255, B: 0, A: 255},
			color.RGBA{R: 255, G: 0, B: 0, A: 255},
			(t-0.67)/0.33,
		)
	}
}
