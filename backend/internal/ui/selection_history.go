package ui

import (
	"fmt"
)

// ParticleSelection represents a set of selected particles
type ParticleSelection struct {
	ParticleIDs map[int64]bool // Set of selected particle IDs
	Count       int             // Number of selected particles
}

// NewParticleSelection creates a new empty selection
func NewParticleSelection() *ParticleSelection {
	return &ParticleSelection{
		ParticleIDs: make(map[int64]bool),
		Count:       0,
	}
}

// Clone creates a deep copy of the selection
func (ps *ParticleSelection) Clone() *ParticleSelection {
	clone := NewParticleSelection()
	for id := range ps.ParticleIDs {
		clone.ParticleIDs[id] = true
	}
	clone.Count = ps.Count
	return clone
}

// SelectionOperation implements the Operation interface for particle selection
type SelectionOperation struct {
	previousSelection *ParticleSelection
	newSelection      *ParticleSelection
	currentSelection  *ParticleSelection // Pointer to actual selection state
	description       string
	operationType     string // "add", "remove", "replace", "clear"
}

// NewSelectionOperation creates a new selection operation
func NewSelectionOperation(current *ParticleSelection, new *ParticleSelection, opType, desc string) *SelectionOperation {
	return &SelectionOperation{
		previousSelection: current.Clone(),
		newSelection:      new.Clone(),
		currentSelection:  current,
		description:       desc,
		operationType:     opType,
	}
}

// Execute applies the selection change
func (so *SelectionOperation) Execute() error {
	so.currentSelection.ParticleIDs = make(map[int64]bool)
	for id := range so.newSelection.ParticleIDs {
		so.currentSelection.ParticleIDs[id] = true
	}
	so.currentSelection.Count = so.newSelection.Count
	return nil
}

// Undo reverts the selection change
func (so *SelectionOperation) Undo() error {
	so.currentSelection.ParticleIDs = make(map[int64]bool)
	for id := range so.previousSelection.ParticleIDs {
		so.currentSelection.ParticleIDs[id] = true
	}
	so.currentSelection.Count = so.previousSelection.Count
	return nil
}

// Description returns a human-readable description
func (so *SelectionOperation) Description() string {
	return so.description
}

// SelectionHistory manages particle selection history with Williams Optimizer
type SelectionHistory struct {
	historyManager   *HistoryManager
	currentSelection *ParticleSelection

	// Batching for bulk operations
	batchMode        bool
	batchOperations  []int64
	batchDescription string
}

// NewSelectionHistory creates a new selection history manager
func NewSelectionHistory() *SelectionHistory {
	return &SelectionHistory{
		historyManager:   NewHistoryManager(),
		currentSelection: NewParticleSelection(),
		batchMode:        false,
		batchOperations:  make([]int64, 0),
	}
}

// SelectParticle adds a single particle to the selection
func (sh *SelectionHistory) SelectParticle(particleID int64) error {
	if sh.batchMode {
		sh.batchOperations = append(sh.batchOperations, particleID)
		return nil
	}

	newSelection := sh.currentSelection.Clone()
	newSelection.ParticleIDs[particleID] = true
	newSelection.Count = len(newSelection.ParticleIDs)

	description := fmt.Sprintf("Select particle %d", particleID)
	op := NewSelectionOperation(sh.currentSelection, newSelection, "add", description)

	if err := sh.historyManager.AddOperation(op); err != nil {
		return err
	}

	sh.currentSelection = newSelection
	return nil
}

// DeselectParticle removes a single particle from the selection
func (sh *SelectionHistory) DeselectParticle(particleID int64) error {
	newSelection := sh.currentSelection.Clone()
	delete(newSelection.ParticleIDs, particleID)
	newSelection.Count = len(newSelection.ParticleIDs)

	description := fmt.Sprintf("Deselect particle %d", particleID)
	op := NewSelectionOperation(sh.currentSelection, newSelection, "remove", description)

	if err := sh.historyManager.AddOperation(op); err != nil {
		return err
	}

	sh.currentSelection = newSelection
	return nil
}

// SelectMultiple adds multiple particles to the selection
func (sh *SelectionHistory) SelectMultiple(particleIDs []int64) error {
	newSelection := sh.currentSelection.Clone()
	for _, id := range particleIDs {
		newSelection.ParticleIDs[id] = true
	}
	newSelection.Count = len(newSelection.ParticleIDs)

	description := fmt.Sprintf("Select %d particles", len(particleIDs))
	op := NewSelectionOperation(sh.currentSelection, newSelection, "add", description)

	if err := sh.historyManager.AddOperation(op); err != nil {
		return err
	}

	sh.currentSelection = newSelection
	return nil
}

// ReplaceSelection replaces the entire selection
func (sh *SelectionHistory) ReplaceSelection(particleIDs []int64) error {
	newSelection := NewParticleSelection()
	for _, id := range particleIDs {
		newSelection.ParticleIDs[id] = true
	}
	newSelection.Count = len(newSelection.ParticleIDs)

	description := fmt.Sprintf("Replace selection with %d particles", len(particleIDs))
	op := NewSelectionOperation(sh.currentSelection, newSelection, "replace", description)

	if err := sh.historyManager.AddOperation(op); err != nil {
		return err
	}

	sh.currentSelection = newSelection
	return nil
}

// ClearSelection clears all selected particles
func (sh *SelectionHistory) ClearSelection() error {
	if sh.currentSelection.Count == 0 {
		return nil // Nothing to clear
	}

	newSelection := NewParticleSelection()

	description := fmt.Sprintf("Clear %d selected particles", sh.currentSelection.Count)
	op := NewSelectionOperation(sh.currentSelection, newSelection, "clear", description)

	if err := sh.historyManager.AddOperation(op); err != nil {
		return err
	}

	sh.currentSelection = newSelection
	return nil
}

// SelectByRegion selects all particles within a 3D bounding box
func (sh *SelectionHistory) SelectByRegion(minX, minY, minZ, maxX, maxY, maxZ float64) error {
	// This would query the spatial voxel grid from Wave 1
	// For now, simulate selecting particles in region
	particleIDs := []int64{} // Would be populated by voxel grid query

	newSelection := sh.currentSelection.Clone()
	for _, id := range particleIDs {
		newSelection.ParticleIDs[id] = true
	}
	newSelection.Count = len(newSelection.ParticleIDs)

	description := fmt.Sprintf("Select region (%.1f,%.1f,%.1f) to (%.1f,%.1f,%.1f)",
		minX, minY, minZ, maxX, maxY, maxZ)
	op := NewSelectionOperation(sh.currentSelection, newSelection, "add", description)

	if err := sh.historyManager.AddOperation(op); err != nil {
		return err
	}

	sh.currentSelection = newSelection
	return nil
}

// SelectByMutationType selects all particles with a specific mutation type
func (sh *SelectionHistory) SelectByMutationType(mutationType string) error {
	// This would query the mutation frequency analyzer from Wave 1
	particleIDs := []int64{} // Would be populated by mutation query

	newSelection := sh.currentSelection.Clone()
	for _, id := range particleIDs {
		newSelection.ParticleIDs[id] = true
	}
	newSelection.Count = len(newSelection.ParticleIDs)

	description := fmt.Sprintf("Select particles with mutation type: %s", mutationType)
	op := NewSelectionOperation(sh.currentSelection, newSelection, "add", description)

	if err := sh.historyManager.AddOperation(op); err != nil {
		return err
	}

	sh.currentSelection = newSelection
	return nil
}

// SelectByGCContent selects all particles within a GC content range
func (sh *SelectionHistory) SelectByGCContent(minGC, maxGC float64) error {
	// This would query the GC content analyzer from Wave 1
	particleIDs := []int64{} // Would be populated by GC query

	newSelection := sh.currentSelection.Clone()
	for _, id := range particleIDs {
		newSelection.ParticleIDs[id] = true
	}
	newSelection.Count = len(newSelection.ParticleIDs)

	description := fmt.Sprintf("Select particles with GC content %.1f%% - %.1f%%", minGC*100, maxGC*100)
	op := NewSelectionOperation(sh.currentSelection, newSelection, "add", description)

	if err := sh.historyManager.AddOperation(op); err != nil {
		return err
	}

	sh.currentSelection = newSelection
	return nil
}

// BeginBatch starts batch mode for bulk selection operations
// This is useful for selecting thousands of particles at once
// Williams Optimizer will create a single checkpoint for the entire batch
func (sh *SelectionHistory) BeginBatch(description string) {
	sh.batchMode = true
	sh.batchOperations = make([]int64, 0, 1000)
	sh.batchDescription = description
}

// EndBatch commits all batched selection operations as a single operation
func (sh *SelectionHistory) EndBatch() error {
	if !sh.batchMode {
		return fmt.Errorf("not in batch mode")
	}

	defer func() {
		sh.batchMode = false
		sh.batchOperations = nil
		sh.batchDescription = ""
	}()

	if len(sh.batchOperations) == 0 {
		return nil // No operations to commit
	}

	return sh.SelectMultiple(sh.batchOperations)
}

// CancelBatch cancels batch mode without committing operations
func (sh *SelectionHistory) CancelBatch() {
	sh.batchMode = false
	sh.batchOperations = nil
	sh.batchDescription = ""
}

// Undo reverts the last selection operation
func (sh *SelectionHistory) Undo() error {
	return sh.historyManager.Undo()
}

// Redo reapplies a previously undone selection operation
func (sh *SelectionHistory) Redo() error {
	return sh.historyManager.Redo()
}

// GetCurrentSelection returns the current selection
func (sh *SelectionHistory) GetCurrentSelection() *ParticleSelection {
	return sh.currentSelection
}

// IsSelected returns true if a particle is selected
func (sh *SelectionHistory) IsSelected(particleID int64) bool {
	return sh.currentSelection.ParticleIDs[particleID]
}

// GetSelectedCount returns the number of selected particles
func (sh *SelectionHistory) GetSelectedCount() int {
	return sh.currentSelection.Count
}

// GetSelectedIDs returns a slice of all selected particle IDs
func (sh *SelectionHistory) GetSelectedIDs() []int64 {
	ids := make([]int64, 0, sh.currentSelection.Count)
	for id := range sh.currentSelection.ParticleIDs {
		ids = append(ids, id)
	}
	return ids
}

// GetHistory returns the selection history
func (sh *SelectionHistory) GetHistory() []string {
	return sh.historyManager.GetHistory()
}

// GetStats returns statistics about the selection history
func (sh *SelectionHistory) GetStats() map[string]interface{} {
	stats := sh.historyManager.GetStats()
	stats["selected_count"] = sh.currentSelection.Count
	stats["batch_mode"] = sh.batchMode
	if sh.batchMode {
		stats["batch_size"] = len(sh.batchOperations)
	}
	return stats
}

// Clear resets the selection history
func (sh *SelectionHistory) Clear() {
	sh.historyManager.Clear()
	sh.currentSelection = NewParticleSelection()
	sh.batchMode = false
	sh.batchOperations = nil
}
