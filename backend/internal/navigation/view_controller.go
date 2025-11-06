/**
 * View Controller for Multi-Scale Navigation
 *
 * Manages camera movement, zoom transitions, and LOD updates
 * Provides smooth navigation between zoom levels
 *
 * Features:
 * - Smooth zoom transitions (exponential easing)
 * - Automatic LOD adjustment
 * - Particle density culling
 * - Jump-to-gene/exon navigation
 */

package navigation

import (
	"fmt"
	"math"
)

// ViewController manages genomic navigation
type ViewController struct {
	viewport         *GenomicViewport
	cameraDistance   float32
	targetDistance   float32
	transitionSpeed  float32 // 0.0-1.0 (higher = faster)
	isTransitioning  bool
	bookmarks        []*NavigationBookmark
	history          []GenomicViewport
	historyIndex     int
	maxHistory       int
}

// NewViewController creates a new view controller
func NewViewController(chromosome string, startPos, endPos uint64) *ViewController {
	viewport := NewGenomicViewport(chromosome, startPos, endPos)

	return &ViewController{
		viewport:        viewport,
		cameraDistance:  viewport.Config.CameraDistance,
		targetDistance:  viewport.Config.CameraDistance,
		transitionSpeed: 0.1, // 10% per frame
		bookmarks:       make([]*NavigationBookmark, 0, 20),
		history:         make([]GenomicViewport, 0, 100),
		historyIndex:    -1,
		maxHistory:      100,
	}
}

// Update updates the view controller (called every frame)
func (vc *ViewController) Update(deltaTime float32) {
	// Smooth camera transition
	if vc.isTransitioning {
		diff := vc.targetDistance - vc.cameraDistance
		if math.Abs(float64(diff)) < 0.1 {
			// Transition complete
			vc.cameraDistance = vc.targetDistance
			vc.isTransitioning = false
		} else {
			// Exponential easing
			vc.cameraDistance += diff * vc.transitionSpeed
		}

		// Update zoom level based on camera distance
		newLevel := GetZoomLevelFromDistance(vc.cameraDistance)
		if newLevel != vc.viewport.ZoomLevel {
			vc.viewport.SetZoomLevel(newLevel)
		}
	}
}

// GetViewport returns the current viewport
func (vc *ViewController) GetViewport() *GenomicViewport {
	return vc.viewport
}

// GetCameraDistance returns the current camera distance
func (vc *ViewController) GetCameraDistance() float32 {
	return vc.cameraDistance
}

// SetCameraDistance sets the target camera distance
func (vc *ViewController) SetCameraDistance(distance float32) {
	vc.targetDistance = distance
	vc.isTransitioning = true
}

// ZoomIn zooms in to the next level
func (vc *ViewController) ZoomIn() bool {
	if vc.viewport.ZoomIn() {
		vc.SetCameraDistance(vc.viewport.Config.CameraDistance)
		vc.addToHistory()
		return true
	}
	return false
}

// ZoomOut zooms out to the previous level
func (vc *ViewController) ZoomOut() bool {
	if vc.viewport.ZoomOut() {
		vc.SetCameraDistance(vc.viewport.Config.CameraDistance)
		vc.addToHistory()
		return true
	}
	return false
}

// SetZoomLevel sets the zoom level directly
func (vc *ViewController) SetZoomLevel(level ZoomLevel) {
	vc.viewport.SetZoomLevel(level)
	vc.SetCameraDistance(vc.viewport.Config.CameraDistance)
	vc.addToHistory()
}

// NavigateToPosition navigates to a specific genomic position
func (vc *ViewController) NavigateToPosition(chromosome string, position uint64, zoomLevel ZoomLevel) {
	config := GetZoomLevelConfig(zoomLevel)
	halfWidth := config.MaxBasePairs / 2

	startPos := uint64(0)
	if position > halfWidth {
		startPos = position - halfWidth
	}
	endPos := position + halfWidth

	vc.viewport = NewGenomicViewport(chromosome, startPos, endPos)
	vc.viewport.SetZoomLevel(zoomLevel)
	vc.SetCameraDistance(config.CameraDistance)
	vc.addToHistory()
}

// NavigateToGene navigates to a gene (gene zoom level)
func (vc *ViewController) NavigateToGene(chromosome string, geneStart, geneEnd uint64) {
	// Center on gene with some padding
	geneLength := geneEnd - geneStart
	padding := geneLength / 4 // 25% padding on each side

	startPos := uint64(0)
	if geneStart > padding {
		startPos = geneStart - padding
	}
	endPos := geneEnd + padding

	vc.viewport = NewGenomicViewport(chromosome, startPos, endPos)
	vc.viewport.SetZoomLevel(ZoomGene)
	vc.SetCameraDistance(vc.viewport.Config.CameraDistance)
	vc.addToHistory()
}

// NavigateToExon navigates to an exon (exon zoom level)
func (vc *ViewController) NavigateToExon(chromosome string, exonStart, exonEnd uint64) {
	// Center on exon with padding
	exonLength := exonEnd - exonStart
	padding := exonLength * 2 // 200% padding (show context)

	startPos := uint64(0)
	if exonStart > padding {
		startPos = exonStart - padding
	}
	endPos := exonEnd + padding

	vc.viewport = NewGenomicViewport(chromosome, startPos, endPos)
	vc.viewport.SetZoomLevel(ZoomExon)
	vc.SetCameraDistance(vc.viewport.Config.CameraDistance)
	vc.addToHistory()
}

// Pan moves the viewport left/right
func (vc *ViewController) Pan(deltaBasePairs int64) {
	if deltaBasePairs > 0 {
		// Pan right
		vc.viewport.StartPos += uint64(deltaBasePairs)
		vc.viewport.EndPos += uint64(deltaBasePairs)
	} else {
		// Pan left
		delta := uint64(-deltaBasePairs)
		if vc.viewport.StartPos > delta {
			vc.viewport.StartPos -= delta
			vc.viewport.EndPos -= delta
		} else {
			vc.viewport.StartPos = 0
			vc.viewport.EndPos = vc.viewport.Length()
		}
	}
}

// AddBookmark adds a bookmark at the current position
func (vc *ViewController) AddBookmark(name, notes string) *NavigationBookmark {
	bookmark := NewNavigationBookmark(
		name,
		vc.viewport.Chromosome,
		vc.viewport.Center(),
		vc.viewport.ZoomLevel,
	)
	bookmark.Notes = notes
	vc.bookmarks = append(vc.bookmarks, bookmark)
	return bookmark
}

// GetBookmarks returns all bookmarks
func (vc *ViewController) GetBookmarks() []*NavigationBookmark {
	return vc.bookmarks
}

// NavigateToBookmark navigates to a bookmark
func (vc *ViewController) NavigateToBookmark(bookmark *NavigationBookmark) {
	vc.NavigateToPosition(bookmark.Chromosome, bookmark.Position, bookmark.ZoomLevel)
}

// addToHistory adds current viewport to history
func (vc *ViewController) addToHistory() {
	// Remove any history ahead of current index (after going back)
	if vc.historyIndex < len(vc.history)-1 {
		vc.history = vc.history[:vc.historyIndex+1]
	}

	// Add to history
	vc.history = append(vc.history, *vc.viewport)
	vc.historyIndex++

	// Limit history size
	if len(vc.history) > vc.maxHistory {
		vc.history = vc.history[1:]
		vc.historyIndex--
	}
}

// GoBack goes back in navigation history
func (vc *ViewController) GoBack() bool {
	if vc.historyIndex > 0 {
		vc.historyIndex--
		prev := vc.history[vc.historyIndex]
		vc.viewport = &GenomicViewport{
			Chromosome: prev.Chromosome,
			StartPos:   prev.StartPos,
			EndPos:     prev.EndPos,
			ZoomLevel:  prev.ZoomLevel,
			Config:     prev.Config,
		}
		vc.SetCameraDistance(prev.Config.CameraDistance)
		return true
	}
	return false
}

// GoForward goes forward in navigation history
func (vc *ViewController) GoForward() bool {
	if vc.historyIndex < len(vc.history)-1 {
		vc.historyIndex++
		next := vc.history[vc.historyIndex]
		vc.viewport = &GenomicViewport{
			Chromosome: next.Chromosome,
			StartPos:   next.StartPos,
			EndPos:     next.EndPos,
			ZoomLevel:  next.ZoomLevel,
			Config:     next.Config,
		}
		vc.SetCameraDistance(next.Config.CameraDistance)
		return true
	}
	return false
}

// GetParticleDensity returns the particle density for current zoom level
func (vc *ViewController) GetParticleDensity() float32 {
	return vc.viewport.Config.ParticleDensity
}

// GetLODLevel returns the LOD level for current zoom
func (vc *ViewController) GetLODLevel() int {
	return vc.viewport.Config.LODLevel
}

// ShouldShowLabels returns whether labels should be shown
func (vc *ViewController) ShouldShowLabels() bool {
	return vc.viewport.Config.ShowLabels
}

// ShouldShowSequence returns whether sequence should be shown
func (vc *ViewController) ShouldShowSequence() bool {
	return vc.viewport.Config.ShowSequence
}

// ShouldShowAnnotations returns whether annotations should be shown
func (vc *ViewController) ShouldShowAnnotations() bool {
	return vc.viewport.Config.ShowAnnotations
}

// GetStatistics returns view controller statistics
func (vc *ViewController) GetStatistics() map[string]interface{} {
	return map[string]interface{}{
		"current_zoom":       vc.viewport.ZoomLevel.String(),
		"camera_distance":    vc.cameraDistance,
		"target_distance":    vc.targetDistance,
		"is_transitioning":   vc.isTransitioning,
		"visible_bp":         vc.viewport.Length(),
		"particle_density":   vc.viewport.Config.ParticleDensity,
		"lod_level":          vc.viewport.Config.LODLevel,
		"show_labels":        vc.viewport.Config.ShowLabels,
		"show_sequence":      vc.viewport.Config.ShowSequence,
		"show_annotations":   vc.viewport.Config.ShowAnnotations,
		"history_count":      len(vc.history),
		"history_index":      vc.historyIndex,
		"bookmark_count":     len(vc.bookmarks),
	}
}

// String returns a human-readable string representation
func (vc *ViewController) String() string {
	return fmt.Sprintf("View: %s | Camera: %.1f → %.1f | Density: %.0f%% | LOD: %d",
		vc.viewport.String(),
		vc.cameraDistance,
		vc.targetDistance,
		vc.viewport.Config.ParticleDensity*100,
		vc.viewport.Config.LODLevel,
	)
}
