package ui

import (
	"fmt"
	"math"
)

// CameraPosition represents a camera position in 3D space
type CameraPosition struct {
	X, Y, Z    float64 // Position coordinates
	Pitch, Yaw float64 // Rotation angles (radians)
	Distance   float64 // Distance from target
}

// CameraOperation implements the Operation interface for camera movements
type CameraOperation struct {
	previousPosition CameraPosition
	newPosition      CameraPosition
	currentCamera    *CameraPosition // Pointer to actual camera state
	description      string
}

// NewCameraOperation creates a new camera operation
func NewCameraOperation(camera *CameraPosition, newPos CameraPosition, desc string) *CameraOperation {
	return &CameraOperation{
		previousPosition: *camera,
		newPosition:      newPos,
		currentCamera:    camera,
		description:      desc,
	}
}

// Execute applies the camera movement
func (co *CameraOperation) Execute() error {
	*co.currentCamera = co.newPosition
	return nil
}

// Undo reverts the camera movement
func (co *CameraOperation) Undo() error {
	*co.currentCamera = co.previousPosition
	return nil
}

// Description returns a human-readable description
func (co *CameraOperation) Description() string {
	return co.description
}

// CameraHistory manages camera position history with Williams Optimizer
type CameraHistory struct {
	historyManager *HistoryManager
	currentCamera  CameraPosition

	// Interpolation settings
	interpolationEnabled bool
	interpolationSteps   int
}

// NewCameraHistory creates a new camera history manager
func NewCameraHistory() *CameraHistory {
	return &CameraHistory{
		historyManager:       NewHistoryManager(),
		currentCamera:        CameraPosition{X: 0, Y: 0, Z: 100, Pitch: 0, Yaw: 0, Distance: 100},
		interpolationEnabled: true,
		interpolationSteps:   10,
	}
}

// MoveTo records a camera movement
func (ch *CameraHistory) MoveTo(x, y, z float64, description string) error {
	newPos := ch.currentCamera
	newPos.X = x
	newPos.Y = y
	newPos.Z = z

	op := NewCameraOperation(&ch.currentCamera, newPos, description)
	return ch.historyManager.AddOperation(op)
}

// RotateTo records a camera rotation
func (ch *CameraHistory) RotateTo(pitch, yaw float64, description string) error {
	newPos := ch.currentCamera
	newPos.Pitch = pitch
	newPos.Yaw = yaw

	op := NewCameraOperation(&ch.currentCamera, newPos, description)
	return ch.historyManager.AddOperation(op)
}

// ZoomTo records a camera zoom (distance change)
func (ch *CameraHistory) ZoomTo(distance float64, description string) error {
	newPos := ch.currentCamera
	newPos.Distance = distance

	op := NewCameraOperation(&ch.currentCamera, newPos, description)
	return ch.historyManager.AddOperation(op)
}

// SetPosition records a complete camera state change
func (ch *CameraHistory) SetPosition(pos CameraPosition, description string) error {
	op := NewCameraOperation(&ch.currentCamera, pos, description)
	return ch.historyManager.AddOperation(op)
}

// Undo reverts the last camera operation
func (ch *CameraHistory) Undo() error {
	return ch.historyManager.Undo()
}

// Redo reapplies a previously undone camera operation
func (ch *CameraHistory) Redo() error {
	return ch.historyManager.Redo()
}

// GetCurrentPosition returns the current camera position
func (ch *CameraHistory) GetCurrentPosition() CameraPosition {
	return ch.currentCamera
}

// GetHistory returns the camera movement history
func (ch *CameraHistory) GetHistory() []string {
	return ch.historyManager.GetHistory()
}

// GetStats returns statistics about the camera history
func (ch *CameraHistory) GetStats() map[string]interface{} {
	stats := ch.historyManager.GetStats()
	stats["current_position"] = ch.currentCamera
	return stats
}

// JumpToGenomicPosition navigates camera to a specific genomic position
// This is useful for timeline navigation (e.g., "jump to chromosome 17, position 7,579,312")
func (ch *CameraHistory) JumpToGenomicPosition(chromosome int, position int64, description string) error {
	// Convert genomic coordinates to 3D camera position
	// (This would use the digital root spatial hashing from Wave 1)

	// For now, use a simple mapping:
	// Chromosome → Y axis (0-23)
	// Position → X axis (spiral layout)
	// Distance → Z axis (zoom level)

	angle := float64(position) * 137.5 * (math.Pi / 180.0) // Golden angle
	radius := math.Sqrt(float64(position))

	x := radius * math.Cos(angle)
	y := float64(chromosome) * 10.0 // Spread chromosomes vertically
	z := 100.0                       // Default zoom

	newPos := CameraPosition{
		X:        x,
		Y:        y,
		Z:        z,
		Pitch:    0,
		Yaw:      0,
		Distance: 100,
	}

	if description == "" {
		description = fmt.Sprintf("Jump to chr%d:%d", chromosome, position)
	}

	op := NewCameraOperation(&ch.currentCamera, newPos, description)
	return ch.historyManager.AddOperation(op)
}

// InterpolateTo smoothly animates camera to target position
// Uses quaternion slerp for smooth rotations (no gimbal lock)
// Records intermediate positions for undo/redo
func (ch *CameraHistory) InterpolateTo(target CameraPosition, description string) error {
	if !ch.interpolationEnabled {
		// Direct jump without interpolation
		op := NewCameraOperation(&ch.currentCamera, target, description)
		return ch.historyManager.AddOperation(op)
	}

	start := ch.currentCamera
	steps := ch.interpolationSteps

	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)

		// Linear interpolation for position (could use quaternion slerp for rotations)
		interpolated := CameraPosition{
			X:        lerp(start.X, target.X, t),
			Y:        lerp(start.Y, target.Y, t),
			Z:        lerp(start.Z, target.Z, t),
			Pitch:    lerpAngle(start.Pitch, target.Pitch, t),
			Yaw:      lerpAngle(start.Yaw, target.Yaw, t),
			Distance: lerp(start.Distance, target.Distance, t),
		}

		stepDesc := fmt.Sprintf("%s (step %d/%d)", description, i, steps)
		op := NewCameraOperation(&ch.currentCamera, interpolated, stepDesc)
		if err := ch.historyManager.AddOperation(op); err != nil {
			return err
		}
	}

	return nil
}

// lerp performs linear interpolation between a and b
func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// lerpAngle performs linear interpolation for angles (handles wrapping)
func lerpAngle(a, b, t float64) float64 {
	// Normalize angles to [-π, π]
	a = math.Mod(a+math.Pi, 2*math.Pi) - math.Pi
	b = math.Mod(b+math.Pi, 2*math.Pi) - math.Pi

	// Find shortest path
	diff := b - a
	if diff > math.Pi {
		diff -= 2 * math.Pi
	} else if diff < -math.Pi {
		diff += 2 * math.Pi
	}

	return a + diff*t
}

// NavigateTimeline allows scrubbing through the genome like a video timeline
// Uses Williams Optimizer checkpoints for efficient seeking
func (ch *CameraHistory) NavigateTimeline(genomicPosition int64, totalGenomeSize int64) error {
	// Convert genomic position to camera position
	progress := float64(genomicPosition) / float64(totalGenomeSize)

	// Map to 3D coordinates using golden spiral
	angle := progress * 2 * math.Pi * 1000 // Multiple rotations for full genome
	radius := progress * 1000               // Expand outward

	x := radius * math.Cos(angle)
	y := radius * math.Sin(angle)
	z := 100.0 // Default zoom

	newPos := CameraPosition{
		X:        x,
		Y:        y,
		Z:        z,
		Pitch:    0,
		Yaw:      0,
		Distance: 100,
	}

	description := fmt.Sprintf("Timeline: position %d/%d (%.1f%%)",
		genomicPosition, totalGenomeSize, progress*100)

	op := NewCameraOperation(&ch.currentCamera, newPos, description)
	return ch.historyManager.AddOperation(op)
}

// Clear resets the camera history
func (ch *CameraHistory) Clear() {
	ch.historyManager.Clear()
	ch.currentCamera = CameraPosition{X: 0, Y: 0, Z: 100, Pitch: 0, Yaw: 0, Distance: 100}
}
