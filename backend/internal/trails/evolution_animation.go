/**
 * Evolution Animation System
 *
 * Animates temporal mutations showing cancer evolution over time
 * Visualizes:
 * - Primary tumor → metastasis progression
 * - Driver mutation → passenger mutation accumulation
 * - Clonal expansion (branching evolution)
 * - Phylogenetic tree traversal
 */

package trails

import (
	"math"
	"sync"
)

// EvolutionStage represents a stage in cancer evolution
type EvolutionStage int

const (
	StageNormal EvolutionStage = iota
	StagePrimary
	StageMetastasis
	StageResistance
)

func (es EvolutionStage) String() string {
	switch es {
	case StageNormal:
		return "Normal"
	case StagePrimary:
		return "Primary"
	case StageMetastasis:
		return "Metastasis"
	case StageResistance:
		return "Resistance"
	default:
		return "Unknown"
	}
}

// TemporalMutation represents a mutation at a specific time point
type TemporalMutation struct {
	Position   uint64
	Chromosome string
	Timepoint  float32 // Time in arbitrary units (0.0 = earliest)
	Stage      EvolutionStage
	Color      [4]float32
	IsDriver   bool // Driver vs passenger mutation
	Clone      int  // Clone ID for branching evolution
}

// EvolutionAnimation controls temporal mutation animation
type EvolutionAnimation struct {
	mutations       []*TemporalMutation
	currentTime     float32
	timeScale       float32 // Time units per second
	isPlaying       bool
	loop            bool
	trailSystem     *TrailSystem
	coordinateFunc  func(chromosome string, position uint64) ([3]float32, error)
	mu              sync.RWMutex
}

// NewEvolutionAnimation creates a new evolution animation
func NewEvolutionAnimation(trailSystem *TrailSystem, timeScale float32) *EvolutionAnimation {
	return &EvolutionAnimation{
		mutations:   make([]*TemporalMutation, 0, 1000),
		currentTime: 0.0,
		timeScale:   timeScale,
		loop:        true,
		trailSystem: trailSystem,
	}
}

// SetCoordinateFunction sets the function to convert genomic position to 3D
func (ea *EvolutionAnimation) SetCoordinateFunction(fn func(string, uint64) ([3]float32, error)) {
	ea.coordinateFunc = fn
}

// AddMutation adds a mutation to the timeline
func (ea *EvolutionAnimation) AddMutation(mutation *TemporalMutation) {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	ea.mutations = append(ea.mutations, mutation)

	// Sort by timepoint
	ea.sortMutations()
}

// sortMutations sorts mutations by timepoint (earliest first)
func (ea *EvolutionAnimation) sortMutations() {
	// Simple bubble sort (small arrays)
	for i := 0; i < len(ea.mutations); i++ {
		for j := i + 1; j < len(ea.mutations); j++ {
			if ea.mutations[j].Timepoint < ea.mutations[i].Timepoint {
				ea.mutations[i], ea.mutations[j] = ea.mutations[j], ea.mutations[i]
			}
		}
	}
}

// Play starts the animation
func (ea *EvolutionAnimation) Play() {
	ea.isPlaying = true
}

// Pause pauses the animation
func (ea *EvolutionAnimation) Pause() {
	ea.isPlaying = false
}

// Stop stops and resets the animation
func (ea *EvolutionAnimation) Stop() {
	ea.isPlaying = false
	ea.currentTime = 0.0
	ea.trailSystem.ClearAll()
}

// SetTime sets the current time
func (ea *EvolutionAnimation) SetTime(time float32) {
	ea.currentTime = time
}

// Update updates the animation (called every frame)
func (ea *EvolutionAnimation) Update(deltaTime float32) {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	if !ea.isPlaying {
		return
	}

	// Advance time
	ea.currentTime += deltaTime * ea.timeScale

	// Handle looping
	if ea.currentTime > ea.getMaxTime() {
		if ea.loop {
			ea.currentTime = 0.0
			ea.trailSystem.ClearAll()
		} else {
			ea.isPlaying = false
		}
	}

	// Emit trail points for mutations that have occurred
	ea.updateTrails()
}

// getMaxTime returns the maximum timepoint in the timeline
func (ea *EvolutionAnimation) getMaxTime() float32 {
	if len(ea.mutations) == 0 {
		return 0.0
	}
	return ea.mutations[len(ea.mutations)-1].Timepoint
}

// updateTrails updates trails for mutations that have occurred
func (ea *EvolutionAnimation) updateTrails() {
	if ea.coordinateFunc == nil {
		return
	}

	for _, mut := range ea.mutations {
		// Only show mutations that have occurred
		if mut.Timepoint > ea.currentTime {
			break
		}

		// Get 3D position
		pos3d, err := ea.coordinateFunc(mut.Chromosome, mut.Position)
		if err != nil {
			continue
		}

		// Emit trail point
		particleID := mut.Position // Use position as ID (simplified)
		size := float32(3.0)
		if mut.IsDriver {
			size = 6.0 // Driver mutations are larger
		}

		ea.trailSystem.EmitTrailPoint(particleID, pos3d, mut.Color, size)
	}
}

// GetActiveMutations returns mutations that have occurred by current time
func (ea *EvolutionAnimation) GetActiveMutations() []*TemporalMutation {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	active := make([]*TemporalMutation, 0, len(ea.mutations))
	for _, mut := range ea.mutations {
		if mut.Timepoint <= ea.currentTime {
			active = append(active, mut)
		} else {
			break
		}
	}
	return active
}

// GetMutationsByStage returns mutations for a specific stage
func (ea *EvolutionAnimation) GetMutationsByStage(stage EvolutionStage) []*TemporalMutation {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	filtered := make([]*TemporalMutation, 0, len(ea.mutations))
	for _, mut := range ea.mutations {
		if mut.Stage == stage {
			filtered = append(filtered, mut)
		}
	}
	return filtered
}

// GetDriverMutations returns all driver mutations
func (ea *EvolutionAnimation) GetDriverMutations() []*TemporalMutation {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	drivers := make([]*TemporalMutation, 0, len(ea.mutations))
	for _, mut := range ea.mutations {
		if mut.IsDriver {
			drivers = append(drivers, mut)
		}
	}
	return drivers
}

// GetStatistics returns animation statistics
func (ea *EvolutionAnimation) GetStatistics() map[string]interface{} {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	driverCount := 0
	passengerCount := 0
	stageCounts := make(map[EvolutionStage]int)

	for _, mut := range ea.mutations {
		if mut.IsDriver {
			driverCount++
		} else {
			passengerCount++
		}
		stageCounts[mut.Stage]++
	}

	progress := float32(0.0)
	maxTime := ea.getMaxTime()
	if maxTime > 0 {
		progress = ea.currentTime / maxTime
	}

	activeMutations := 0
	for _, mut := range ea.mutations {
		if mut.Timepoint <= ea.currentTime {
			activeMutations++
		}
	}

	return map[string]interface{}{
		"total_mutations":   len(ea.mutations),
		"driver_mutations":  driverCount,
		"passenger_mutations": passengerCount,
		"stage_counts":      stageCounts,
		"current_time":      ea.currentTime,
		"max_time":          maxTime,
		"progress":          progress,
		"active_mutations":  activeMutations,
		"is_playing":        ea.isPlaying,
		"time_scale":        ea.timeScale,
	}
}

// GetCurrentTime returns the current animation time
func (ea *EvolutionAnimation) GetCurrentTime() float32 {
	ea.mu.RLock()
	defer ea.mu.RUnlock()
	return ea.currentTime
}

// GetProgress returns the animation progress (0.0-1.0)
func (ea *EvolutionAnimation) GetProgress() float32 {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	maxTime := ea.getMaxTime()
	if maxTime == 0 {
		return 0.0
	}
	return ea.currentTime / maxTime
}

// SetTimeScale sets the time scale (time units per second)
func (ea *EvolutionAnimation) SetTimeScale(scale float32) {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	ea.timeScale = scale
}

// SetLoop sets whether the animation loops
func (ea *EvolutionAnimation) SetLoop(loop bool) {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	ea.loop = loop
}

// GenerateCancerEvolution generates a simulated cancer evolution timeline
func GenerateCancerEvolution(numMutations int) []*TemporalMutation {
	mutations := make([]*TemporalMutation, 0, numMutations)

	// Timeline:
	// 0.0-1.0: Normal → Primary tumor (driver mutations)
	// 1.0-2.0: Primary tumor growth (passenger mutations)
	// 2.0-3.0: Metastasis (new driver mutations)
	// 3.0-4.0: Treatment resistance (additional drivers)

	// Phase 1: Primary tumor initiation (driver mutations)
	driverGenes := []struct {
		chrom string
		pos   uint64
		name  string
	}{
		{"chr17", 7577534, "TP53"},
		{"chr12", 25398284, "KRAS"},
		{"chr7", 55241707, "EGFR"},
	}

	for i, driver := range driverGenes {
		mut := &TemporalMutation{
			Chromosome: driver.chrom,
			Position:   driver.pos,
			Timepoint:  float32(i) * 0.3,
			Stage:      StagePrimary,
			Color:      [4]float32{1.0, 0.0, 0.0, 1.0}, // Red (driver)
			IsDriver:   true,
			Clone:      0,
		}
		mutations = append(mutations, mut)
	}

	// Phase 2: Passenger mutations (random accumulation)
	for i := 0; i < 20; i++ {
		mut := &TemporalMutation{
			Chromosome: "chr1",
			Position:   uint64(1000000 + i*1000000),
			Timepoint:  1.0 + float32(i)*0.05,
			Stage:      StagePrimary,
			Color:      [4]float32{0.5, 0.5, 1.0, 0.6}, // Blue (passenger)
			IsDriver:   false,
			Clone:      0,
		}
		mutations = append(mutations, mut)
	}

	// Phase 3: Metastasis (new drivers)
	metastasisDrivers := []struct {
		chrom string
		pos   uint64
	}{
		{"chr3", 178936091}, // PIK3CA
		{"chr7", 140453136}, // BRAF
	}

	for i, driver := range metastasisDrivers {
		mut := &TemporalMutation{
			Chromosome: driver.chrom,
			Position:   driver.pos,
			Timepoint:  2.0 + float32(i)*0.5,
			Stage:      StageMetastasis,
			Color:      [4]float32{1.0, 0.5, 0.0, 1.0}, // Orange (metastasis driver)
			IsDriver:   true,
			Clone:      1,
		}
		mutations = append(mutations, mut)
	}

	// Phase 4: Treatment resistance
	mut := &TemporalMutation{
		Chromosome: "chr5",
		Position:   112175770, // APC
		Timepoint:  3.0,
		Stage:      StageResistance,
		Color:      [4]float32{1.0, 0.0, 1.0, 1.0}, // Magenta (resistance)
		IsDriver:   true,
		Clone:      2,
	}
	mutations = append(mutations, mut)

	return mutations
}

// interpolateFloat32 interpolates between two float32 values
func interpolateFloat32(a, b, t float32) float32 {
	return a + (b-a)*t
}

// easeInOut provides smooth easing function
func easeInOut(t float32) float32 {
	if t < 0.5 {
		return 2 * t * t
	}
	return float32(-1.0 + (4.0-2.0*float64(t))*float64(t))
}

// exponentialDecay provides exponential decay function
func exponentialDecay(t, halfLife float32) float32 {
	return float32(math.Exp(-0.693147 * float64(t) / float64(halfLife)))
}
