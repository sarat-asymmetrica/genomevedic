/**
 * Mutation Overlay System
 *
 * Overlays COSMIC mutations on genomic particles
 * Colors particles based on mutation significance and type
 *
 * Color Scheme:
 * - Red (Pathogenic hotspots)
 * - Orange (Likely Pathogenic)
 * - Yellow (Uncertain significance)
 * - Green (Benign)
 * - Default (No mutation)
 */

package mutations

import (
	"fmt"
	"math"
	"sync"
)

// MutationColor represents RGBA color for mutation visualization
type MutationColor struct {
	R, G, B, A float32
}

// Predefined colors for mutation significance
var (
	ColorPathogenic        = MutationColor{R: 1.0, G: 0.0, B: 0.0, A: 1.0} // Red
	ColorLikelyPathogenic  = MutationColor{R: 1.0, G: 0.5, B: 0.0, A: 1.0} // Orange
	ColorUncertain         = MutationColor{R: 1.0, G: 1.0, B: 0.0, A: 0.8} // Yellow
	ColorLikelyBenign      = MutationColor{R: 0.5, G: 1.0, B: 0.5, A: 0.6} // Light Green
	ColorBenign            = MutationColor{R: 0.0, G: 1.0, B: 0.0, A: 0.5} // Green
	ColorHotspot           = MutationColor{R: 1.0, G: 0.0, B: 0.0, A: 1.0} // Bright Red (hotspot)
)

// ParticleMutation represents mutation data for a single particle
type ParticleMutation struct {
	Position     uint64
	Chromosome   string
	HasMutation  bool
	Mutations    []*Mutation
	Color        MutationColor
	IsHotspot    bool
	HotspotScore float32 // 0.0-1.0, higher = more significant
}

// MutationOverlay overlays mutations on particles
type MutationOverlay struct {
	parser           *COSMICParser
	particleMutations map[string]*ParticleMutation // Key: "chr:position"
	hotspotRadius    uint64                        // Radius around hotspot to color
	mu               sync.RWMutex
}

// NewMutationOverlay creates a new mutation overlay
func NewMutationOverlay(parser *COSMICParser, hotspotRadius uint64) *MutationOverlay {
	return &MutationOverlay{
		parser:            parser,
		particleMutations: make(map[string]*ParticleMutation),
		hotspotRadius:     hotspotRadius,
	}
}

// BuildOverlay builds the mutation overlay for all particles
func (mo *MutationOverlay) BuildOverlay() error {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	// Clear existing overlay
	mo.particleMutations = make(map[string]*ParticleMutation)

	// Add all mutations
	for _, mut := range mo.parser.GetMutations() {
		key := fmt.Sprintf("%s:%d", mut.Chromosome, mut.Position)

		pm, exists := mo.particleMutations[key]
		if !exists {
			pm = &ParticleMutation{
				Position:   mut.Position,
				Chromosome: mut.Chromosome,
				Mutations:  make([]*Mutation, 0, 4),
			}
			mo.particleMutations[key] = pm
		}

		pm.Mutations = append(pm.Mutations, mut)
		pm.HasMutation = true
	}

	// Mark hotspots
	for _, hotspot := range mo.parser.GetHotspots() {
		key := fmt.Sprintf("%s:%d", hotspot.Chromosome, hotspot.Position)
		if pm, exists := mo.particleMutations[key]; exists {
			pm.IsHotspot = true
		}
	}

	// Compute colors and hotspot scores
	for _, pm := range mo.particleMutations {
		mo.computeParticleColor(pm)
	}

	// Propagate hotspot colors to nearby particles
	mo.propagateHotspotColors()

	return nil
}

// GetParticleMutation returns mutation data for a particle
func (mo *MutationOverlay) GetParticleMutation(chromosome string, position uint64) *ParticleMutation {
	mo.mu.RLock()
	defer mo.mu.RUnlock()

	key := fmt.Sprintf("%s:%d", chromosome, position)
	return mo.particleMutations[key]
}

// GetParticleColor returns the color for a particle based on mutations
func (mo *MutationOverlay) GetParticleColor(chromosome string, position uint64) (MutationColor, bool) {
	pm := mo.GetParticleMutation(chromosome, position)
	if pm == nil || !pm.HasMutation {
		return MutationColor{}, false
	}
	return pm.Color, true
}

// computeParticleColor computes the color for a particle based on its mutations
func (mo *MutationOverlay) computeParticleColor(pm *ParticleMutation) {
	if len(pm.Mutations) == 0 {
		pm.Color = MutationColor{R: 1.0, G: 1.0, B: 1.0, A: 1.0} // Default white
		return
	}

	// Find most significant mutation
	mostSig := SignificanceUnknown
	maxSamples := 0

	for _, mut := range pm.Mutations {
		if mut.Significance > mostSig {
			mostSig = mut.Significance
		}
		if mut.SampleCount > maxSamples {
			maxSamples = mut.SampleCount
		}
	}

	// Compute hotspot score (0.0-1.0)
	pm.HotspotScore = float32(math.Min(float64(maxSamples)/1000.0, 1.0))

	// Assign color based on significance
	if pm.IsHotspot {
		// Hotspots get bright red
		pm.Color = ColorHotspot
	} else {
		switch mostSig {
		case SignificancePathogenic:
			pm.Color = ColorPathogenic
		case SignificanceLikelyPathogenic:
			pm.Color = ColorLikelyPathogenic
		case SignificanceUncertain:
			pm.Color = ColorUncertain
		case SignificanceLikelyBenign:
			pm.Color = ColorLikelyBenign
		case SignificanceBenign:
			pm.Color = ColorBenign
		default:
			pm.Color = ColorUncertain
		}
	}

	// Adjust alpha based on sample count (higher sample count = more opaque)
	if maxSamples > 0 {
		alpha := float32(math.Min(0.5+float64(maxSamples)/200.0, 1.0))
		pm.Color.A = alpha
	}
}

// propagateHotspotColors propagates hotspot colors to nearby particles
func (mo *MutationOverlay) propagateHotspotColors() {
	hotspots := mo.parser.GetHotspots()

	for _, hotspot := range hotspots {
		// Color particles within hotspotRadius of this hotspot
		startPos := uint64(0)
		if hotspot.Position > mo.hotspotRadius {
			startPos = hotspot.Position - mo.hotspotRadius
		}
		endPos := hotspot.Position + mo.hotspotRadius

		for pos := startPos; pos <= endPos; pos++ {
			key := fmt.Sprintf("%s:%d", hotspot.Chromosome, pos)

			pm, exists := mo.particleMutations[key]
			if !exists {
				// Create particle mutation for this position
				pm = &ParticleMutation{
					Position:   pos,
					Chromosome: hotspot.Chromosome,
					HasMutation: true,
					Mutations:  []*Mutation{hotspot},
					IsHotspot:  true,
				}
				mo.particleMutations[key] = pm
			}

			// Compute falloff based on distance
			distance := float64(0)
			if pos > hotspot.Position {
				distance = float64(pos - hotspot.Position)
			} else {
				distance = float64(hotspot.Position - pos)
			}

			falloff := 1.0 - (distance / float64(mo.hotspotRadius))
			falloff = math.Max(0.0, falloff)

			// Blend with hotspot color
			pm.Color.R = ColorHotspot.R * float32(falloff)
			pm.Color.G = ColorHotspot.G * float32(falloff)
			pm.Color.B = ColorHotspot.B * float32(falloff)
			pm.Color.A = ColorHotspot.A * float32(falloff) * 0.8
			pm.HotspotScore = float32(falloff)
		}
	}
}

// GetStatistics returns overlay statistics
func (mo *MutationOverlay) GetStatistics() map[string]interface{} {
	mo.mu.RLock()
	defer mo.mu.RUnlock()

	totalParticles := len(mo.particleMutations)
	hotspotParticles := 0
	pathogenicParticles := 0
	uncertainParticles := 0
	benignParticles := 0

	for _, pm := range mo.particleMutations {
		if pm.IsHotspot {
			hotspotParticles++
		}

		for _, mut := range pm.Mutations {
			switch mut.Significance {
			case SignificancePathogenic, SignificanceLikelyPathogenic:
				pathogenicParticles++
			case SignificanceUncertain:
				uncertainParticles++
			case SignificanceBenign, SignificanceLikelyBenign:
				benignParticles++
			}
		}
	}

	return map[string]interface{}{
		"total_particles":      totalParticles,
		"hotspot_particles":    hotspotParticles,
		"pathogenic_particles": pathogenicParticles,
		"uncertain_particles":  uncertainParticles,
		"benign_particles":     benignParticles,
		"hotspot_radius":       mo.hotspotRadius,
	}
}

// FilterBySignificance filters mutations by significance level
func (mo *MutationOverlay) FilterBySignificance(minSig Significance) []*ParticleMutation {
	mo.mu.RLock()
	defer mo.mu.RUnlock()

	filtered := make([]*ParticleMutation, 0, len(mo.particleMutations))

	for _, pm := range mo.particleMutations {
		for _, mut := range pm.Mutations {
			if mut.Significance >= minSig {
				filtered = append(filtered, pm)
				break
			}
		}
	}

	return filtered
}

// FilterByType filters mutations by type
func (mo *MutationOverlay) FilterByType(mutType MutationType) []*ParticleMutation {
	mo.mu.RLock()
	defer mo.mu.RUnlock()

	filtered := make([]*ParticleMutation, 0, len(mo.particleMutations))

	for _, pm := range mo.particleMutations {
		for _, mut := range pm.Mutations {
			if mut.MutationType == mutType {
				filtered = append(filtered, pm)
				break
			}
		}
	}

	return filtered
}
