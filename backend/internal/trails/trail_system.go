/**
 * Particle Trail System
 *
 * Creates trails behind particles to show temporal evolution
 * Used for visualizing cancer evolution, phylogenetic relationships, etc.
 *
 * Trail features:
 * - Fade-out over time (alpha decay)
 * - Color interpolation (mutation history)
 * - Position interpolation (smooth animation)
 * - Configurable trail length and density
 */

package trails

import (
	"math"
	"sync"
)

// TrailPoint represents a single point in a particle trail
type TrailPoint struct {
	Position [3]float32
	Color    [4]float32
	Size     float32
	Age      float32 // Age in seconds (0.0 = newest)
	MaxAge   float32 // Maximum age before removal
}

// ParticleTrail represents a trail behind a single particle
type ParticleTrail struct {
	ParticleID uint64
	Points     []TrailPoint
	MaxPoints  int
	FadeTime   float32 // Time for alpha to fade to 0
	Enabled    bool
}

// NewParticleTrail creates a new particle trail
func NewParticleTrail(particleID uint64, maxPoints int, fadeTime float32) *ParticleTrail {
	return &ParticleTrail{
		ParticleID: particleID,
		Points:     make([]TrailPoint, 0, maxPoints),
		MaxPoints:  maxPoints,
		FadeTime:   fadeTime,
		Enabled:    true,
	}
}

// AddPoint adds a new point to the trail
func (pt *ParticleTrail) AddPoint(position [3]float32, color [4]float32, size float32) {
	if !pt.Enabled {
		return
	}

	point := TrailPoint{
		Position: position,
		Color:    color,
		Size:     size,
		Age:      0.0,
		MaxAge:   pt.FadeTime,
	}

	// Add to front of trail
	pt.Points = append([]TrailPoint{point}, pt.Points...)

	// Limit trail length
	if len(pt.Points) > pt.MaxPoints {
		pt.Points = pt.Points[:pt.MaxPoints]
	}
}

// Update updates all trail points (called every frame)
func (pt *ParticleTrail) Update(deltaTime float32) {
	if !pt.Enabled {
		return
	}

	// Age all points
	toRemove := 0
	for i := range pt.Points {
		pt.Points[i].Age += deltaTime

		// Mark for removal if too old
		if pt.Points[i].Age >= pt.Points[i].MaxAge {
			toRemove++
		} else {
			break // Points are ordered by age, so stop here
		}
	}

	// Remove old points (from end)
	if toRemove > 0 {
		pt.Points = pt.Points[:len(pt.Points)-toRemove]
	}

	// Update alpha based on age (fade out)
	for i := range pt.Points {
		t := pt.Points[i].Age / pt.Points[i].MaxAge
		pt.Points[i].Color[3] = (1.0 - t) * pt.Points[i].Color[3]
	}
}

// Clear clears all trail points
func (pt *ParticleTrail) Clear() {
	pt.Points = pt.Points[:0]
}

// GetPoints returns all trail points
func (pt *ParticleTrail) GetPoints() []TrailPoint {
	return pt.Points
}

// TrailSystem manages trails for all particles
type TrailSystem struct {
	trails       map[uint64]*ParticleTrail
	maxPoints    int
	fadeTime     float32
	emissionRate float32 // Points per second
	timeSinceLastEmission float32
	mu           sync.RWMutex
}

// NewTrailSystem creates a new trail system
func NewTrailSystem(maxPoints int, fadeTime float32, emissionRate float32) *TrailSystem {
	return &TrailSystem{
		trails:       make(map[uint64]*ParticleTrail),
		maxPoints:    maxPoints,
		fadeTime:     fadeTime,
		emissionRate: emissionRate,
	}
}

// AddTrail adds a trail for a particle
func (ts *TrailSystem) AddTrail(particleID uint64) *ParticleTrail {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	trail := NewParticleTrail(particleID, ts.maxPoints, ts.fadeTime)
	ts.trails[particleID] = trail
	return trail
}

// GetTrail returns a trail for a particle (creates if not exists)
func (ts *TrailSystem) GetTrail(particleID uint64) *ParticleTrail {
	ts.mu.RLock()
	trail, exists := ts.trails[particleID]
	ts.mu.RUnlock()

	if !exists {
		return ts.AddTrail(particleID)
	}

	return trail
}

// RemoveTrail removes a trail for a particle
func (ts *TrailSystem) RemoveTrail(particleID uint64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.trails, particleID)
}

// Update updates all trails (called every frame)
func (ts *TrailSystem) Update(deltaTime float32) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.timeSinceLastEmission += deltaTime

	for _, trail := range ts.trails {
		trail.Update(deltaTime)
	}
}

// EmitTrailPoint emits a trail point for a particle (if emission rate allows)
func (ts *TrailSystem) EmitTrailPoint(particleID uint64, position [3]float32, color [4]float32, size float32) {
	emissionInterval := 1.0 / ts.emissionRate
	if ts.timeSinceLastEmission < emissionInterval {
		return
	}

	ts.timeSinceLastEmission = 0.0

	trail := ts.GetTrail(particleID)
	trail.AddPoint(position, color, size)
}

// GetAllTrails returns all trails
func (ts *TrailSystem) GetAllTrails() []*ParticleTrail {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	trails := make([]*ParticleTrail, 0, len(ts.trails))
	for _, trail := range ts.trails {
		trails = append(trails, trail)
	}
	return trails
}

// GetTotalTrailPoints returns the total number of trail points across all trails
func (ts *TrailSystem) GetTotalTrailPoints() int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	total := 0
	for _, trail := range ts.trails {
		total += len(trail.Points)
	}
	return total
}

// ClearAll clears all trails
func (ts *TrailSystem) ClearAll() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	for _, trail := range ts.trails {
		trail.Clear()
	}
}

// SetEmissionRate sets the emission rate (points per second)
func (ts *TrailSystem) SetEmissionRate(rate float32) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.emissionRate = rate
}

// SetFadeTime sets the fade time for all trails
func (ts *TrailSystem) SetFadeTime(fadeTime float32) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.fadeTime = fadeTime
	for _, trail := range ts.trails {
		trail.FadeTime = fadeTime
	}
}

// GetStatistics returns trail system statistics
func (ts *TrailSystem) GetStatistics() map[string]interface{} {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	totalPoints := 0
	activeTrails := 0

	for _, trail := range ts.trails {
		if len(trail.Points) > 0 {
			activeTrails++
		}
		totalPoints += len(trail.Points)
	}

	avgPointsPerTrail := 0.0
	if activeTrails > 0 {
		avgPointsPerTrail = float64(totalPoints) / float64(activeTrails)
	}

	return map[string]interface{}{
		"total_trails":         len(ts.trails),
		"active_trails":        activeTrails,
		"total_trail_points":   totalPoints,
		"avg_points_per_trail": avgPointsPerTrail,
		"max_points_per_trail": ts.maxPoints,
		"fade_time":            ts.fadeTime,
		"emission_rate":        ts.emissionRate,
	}
}

// InterpolatePosition interpolates between two positions using smooth Hermite
func InterpolatePosition(p1, p2 [3]float32, t float32) [3]float32 {
	// Smooth Hermite interpolation
	t = smoothstep(t)

	return [3]float32{
		p1[0] + (p2[0]-p1[0])*t,
		p1[1] + (p2[1]-p1[1])*t,
		p1[2] + (p2[2]-p1[2])*t,
	}
}

// InterpolateColor interpolates between two colors
func InterpolateColor(c1, c2 [4]float32, t float32) [4]float32 {
	return [4]float32{
		c1[0] + (c2[0]-c1[0])*t,
		c1[1] + (c2[1]-c1[1])*t,
		c1[2] + (c2[2]-c1[2])*t,
		c1[3] + (c2[3]-c1[3])*t,
	}
}

// smoothstep provides smooth Hermite interpolation
func smoothstep(t float32) float32 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3.0 - 2.0*t)
}

// catmullRom provides Catmull-Rom spline interpolation for smooth curves
func catmullRom(p0, p1, p2, p3 [3]float32, t float32) [3]float32 {
	t2 := t * t
	t3 := t2 * t

	return [3]float32{
		0.5 * ((2*p1[0]) + (-p0[0]+p2[0])*t + (2*p0[0]-5*p1[0]+4*p2[0]-p3[0])*t2 + (-p0[0]+3*p1[0]-3*p2[0]+p3[0])*t3),
		0.5 * ((2*p1[1]) + (-p0[1]+p2[1])*t + (2*p0[1]-5*p1[1]+4*p2[1]-p3[1])*t2 + (-p0[1]+3*p1[1]-3*p2[1]+p3[1])*t3),
		0.5 * ((2*p1[2]) + (-p0[2]+p2[2])*t + (2*p0[2]-5*p1[2]+4*p2[2]-p3[2])*t2 + (-p0[2]+3*p1[2]-3*p2[2]+p3[2])*t3),
	}
}

// distance3D calculates Euclidean distance between two 3D points
func distance3D(p1, p2 [3]float32) float32 {
	dx := p2[0] - p1[0]
	dy := p2[1] - p1[1]
	dz := p2[2] - p1[2]
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}
