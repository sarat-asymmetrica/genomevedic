// Package vedic - Vedic mathematics color mapping for genomics
package vedic

import (
	"image/color"
	"math"
	"strings"

	"genomevedic/backend/pkg/types"
)

// GCContentColor computes color based on GC content (golden ratio hue mapping)
//
// Algorithm (from MATHEMATICAL_FOUNDATIONS.md):
//   HUE = (GC% × φ) mod 360°
//   SATURATION = 0.8 + (GC% - 50%) × 0.004
//   LIGHTNESS = 0.5
//
// Rationale:
//   - GC content affects DNA stability (higher GC = more stable)
//   - Golden ratio spacing creates natural color progression
//   - AT-rich regions = cooler hues, GC-rich = warmer hues
func GCContentColor(sequence string) color.RGBA {
	gc := ComputeGCContent(sequence)

	// Hue from golden ratio
	hue := math.Mod(gc.Percent*types.Phi, types.MaxHue)

	// Saturation increases with GC content deviation from 50%
	saturation := 0.8 + (gc.Percent-50.0)*0.004
	saturation = clamp(saturation, 0.0, 1.0)

	// Constant lightness
	lightness := 0.5

	return HSLToRGB(hue, saturation, lightness)
}

// ComputeGCContent calculates GC content statistics for a sequence
func ComputeGCContent(sequence string) types.GCContent {
	sequence = strings.ToUpper(sequence)

	gCount := strings.Count(sequence, "G")
	cCount := strings.Count(sequence, "C")
	totalBases := len(sequence)

	percent := 0.0
	if totalBases > 0 {
		percent = float64(gCount+cCount) / float64(totalBases) * 100.0
	}

	return types.GCContent{
		GCount:     gCount,
		CCount:     cCount,
		TotalBases: totalBases,
		Percent:    percent,
	}
}

// GCContentGradient creates a color gradient based on GC content
// Used for visualization of GC% distribution
func GCContentGradient(gcPercent float64) color.RGBA {
	// Clamp to [0, 100]
	gcPercent = clamp(gcPercent, 0.0, 100.0)

	// Map to hue using golden ratio
	hue := math.Mod(gcPercent*types.Phi, types.MaxHue)

	// Higher saturation for extreme GC%
	deviation := math.Abs(gcPercent - 50.0)
	saturation := 0.6 + deviation/100.0
	saturation = clamp(saturation, 0.0, 1.0)

	// Lightness based on GC%
	lightness := 0.4 + gcPercent/200.0 // Range: 0.4-0.9
	lightness = clamp(lightness, 0.0, 1.0)

	return HSLToRGB(hue, saturation, lightness)
}

// HSLToRGB converts HSL color to RGB
// H: [0, 360), S: [0, 1], L: [0, 1]
func HSLToRGB(h, s, l float64) color.RGBA {
	// Normalize hue to [0, 1)
	h = math.Mod(h, 360.0) / 360.0

	var r, g, b float64

	if s == 0 {
		// Achromatic (gray)
		r, g, b = l, l, l
	} else {
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q

		r = hueToRGB(p, q, h+1.0/3.0)
		g = hueToRGB(p, q, h)
		b = hueToRGB(p, q, h-1.0/3.0)
	}

	return color.RGBA{
		R: uint8(r * 255),
		G: uint8(g * 255),
		B: uint8(b * 255),
		A: 255,
	}
}

// hueToRGB is a helper for HSL to RGB conversion
func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}

	if t < 1.0/6.0 {
		return p + (q-p)*6.0*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6.0
	}

	return p
}

// clamp restricts a value to a range
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// GetGCCategory categorizes GC content into biological regions
func GetGCCategory(gcPercent float64) string {
	if gcPercent < 35.0 {
		return "AT-rich" // Low GC (repetitive regions, introns)
	} else if gcPercent < 45.0 {
		return "Low-GC" // Below average
	} else if gcPercent < 55.0 {
		return "Normal" // Average human genome ~41%
	} else if gcPercent < 65.0 {
		return "High-GC" // Gene-rich regions
	} else {
		return "GC-rich" // CpG islands, promoters
	}
}
