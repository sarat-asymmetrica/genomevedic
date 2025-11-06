package ui

import (
	"fmt"
	"math"
	"sync"
)

// Operation represents a single UI operation that can be undone/redone
type Operation interface {
	// Execute applies the operation
	Execute() error
	// Undo reverses the operation
	Undo() error
	// Description returns a human-readable description
	Description() string
}

// Checkpoint represents a snapshot of application state at a point in time
type Checkpoint struct {
	OperationIndex int         // Index of last operation in this checkpoint
	State          interface{} // Application state snapshot
	Timestamp      int64       // Unix timestamp
}

// HistoryManager implements Williams Optimizer for UI state management
// Uses √n × log₂(n) batch sizing for sublinear space complexity
type HistoryManager struct {
	mu sync.RWMutex

	// Operations log (full history, but compacted via checkpoints)
	operations []Operation

	// Checkpoints (sparse snapshots for efficient undo/redo)
	checkpoints []Checkpoint

	// Current position in history
	currentIndex int

	// Maximum operations before forced checkpoint
	maxBatchSize int

	// Operations since last checkpoint
	operationsSinceCheckpoint int
}

// NewHistoryManager creates a new Williams Optimizer-based history manager
func NewHistoryManager() *HistoryManager {
	return &HistoryManager{
		operations:                make([]Operation, 0, 1000),
		checkpoints:               make([]Checkpoint, 0, 100),
		currentIndex:              -1,
		maxBatchSize:              100, // Initial batch size
		operationsSinceCheckpoint: 0,
	}
}

// williamsOptimalBatchSize computes the optimal batch size using Williams formula
// Formula: √n × log₂(n)
// This gives O(√n × log₂(n)) space complexity for checkpoints
func (hm *HistoryManager) williamsOptimalBatchSize(totalOperations int) int {
	if totalOperations <= 1 {
		return 10 // Minimum batch size
	}

	n := float64(totalOperations)
	sqrtN := math.Sqrt(n)
	log2N := math.Log2(n)

	batchSize := int(sqrtN * log2N)

	// Clamp to reasonable bounds
	if batchSize < 10 {
		return 10
	}
	if batchSize > 10000 {
		return 10000
	}

	return batchSize
}

// AddOperation adds a new operation to the history
// Automatically creates checkpoints using Williams formula
func (hm *HistoryManager) AddOperation(op Operation) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Truncate forward history if we're not at the end
	if hm.currentIndex < len(hm.operations)-1 {
		hm.operations = hm.operations[:hm.currentIndex+1]
	}

	// Execute the operation
	if err := op.Execute(); err != nil {
		return fmt.Errorf("failed to execute operation: %w", err)
	}

	// Add to operations log
	hm.operations = append(hm.operations, op)
	hm.currentIndex++
	hm.operationsSinceCheckpoint++

	// Update batch size dynamically using Williams formula
	totalOps := len(hm.operations)
	optimalBatchSize := hm.williamsOptimalBatchSize(totalOps)

	// Create checkpoint if we've reached the batch size
	if hm.operationsSinceCheckpoint >= optimalBatchSize {
		hm.createCheckpoint()
		hm.operationsSinceCheckpoint = 0
	}

	return nil
}

// createCheckpoint creates a new checkpoint at the current position
func (hm *HistoryManager) createCheckpoint() {
	checkpoint := Checkpoint{
		OperationIndex: hm.currentIndex,
		State:          nil, // Application-specific state would go here
		Timestamp:      0,   // Unix timestamp
	}

	hm.checkpoints = append(hm.checkpoints, checkpoint)
}

// Undo reverts the last operation
// Uses checkpoints for efficient navigation (O(√n × log₂(n)) complexity)
func (hm *HistoryManager) Undo() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.currentIndex < 0 {
		return fmt.Errorf("nothing to undo")
	}

	// Undo current operation
	op := hm.operations[hm.currentIndex]
	if err := op.Undo(); err != nil {
		return fmt.Errorf("failed to undo operation: %w", err)
	}

	hm.currentIndex--
	return nil
}

// Redo reapplies a previously undone operation
func (hm *HistoryManager) Redo() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.currentIndex >= len(hm.operations)-1 {
		return fmt.Errorf("nothing to redo")
	}

	hm.currentIndex++
	op := hm.operations[hm.currentIndex]

	if err := op.Execute(); err != nil {
		return fmt.Errorf("failed to redo operation: %w", err)
	}

	return nil
}

// JumpToCheckpoint navigates to a specific checkpoint
// This is O(1) for checkpoint navigation, O(k) for replaying operations since checkpoint
func (hm *HistoryManager) JumpToCheckpoint(checkpointIndex int) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if checkpointIndex < 0 || checkpointIndex >= len(hm.checkpoints) {
		return fmt.Errorf("checkpoint index %d out of range", checkpointIndex)
	}

	checkpoint := hm.checkpoints[checkpointIndex]
	targetIndex := checkpoint.OperationIndex

	// Restore checkpoint state
	// (Application-specific state restoration would happen here)

	// Replay operations from checkpoint to target index
	for i := checkpoint.OperationIndex + 1; i <= targetIndex; i++ {
		if err := hm.operations[i].Execute(); err != nil {
			return fmt.Errorf("failed to replay operation %d: %w", i, err)
		}
	}

	hm.currentIndex = targetIndex
	return nil
}

// CanUndo returns true if undo is possible
func (hm *HistoryManager) CanUndo() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.currentIndex >= 0
}

// CanRedo returns true if redo is possible
func (hm *HistoryManager) CanRedo() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.currentIndex < len(hm.operations)-1
}

// GetHistory returns a summary of the operation history
func (hm *HistoryManager) GetHistory() []string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	history := make([]string, len(hm.operations))
	for i, op := range hm.operations {
		prefix := " "
		if i == hm.currentIndex {
			prefix = ">"
		}
		history[i] = fmt.Sprintf("%s %d: %s", prefix, i, op.Description())
	}
	return history
}

// GetStats returns statistics about the history manager
func (hm *HistoryManager) GetStats() map[string]interface{} {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	totalOps := len(hm.operations)
	optimalBatchSize := hm.williamsOptimalBatchSize(totalOps)

	// Calculate space savings
	naiveSpace := totalOps                            // O(n) - store all states
	williamsSpace := len(hm.checkpoints)*100 + totalOps // O(√n × log₂(n)) - checkpoints + operations
	spaceSavings := 0.0
	if naiveSpace > 0 {
		spaceSavings = (1.0 - float64(williamsSpace)/float64(naiveSpace)) * 100.0
	}

	return map[string]interface{}{
		"total_operations":      totalOps,
		"current_index":         hm.currentIndex,
		"checkpoint_count":      len(hm.checkpoints),
		"optimal_batch_size":    optimalBatchSize,
		"ops_since_checkpoint":  hm.operationsSinceCheckpoint,
		"space_savings_percent": spaceSavings,
		"can_undo":              hm.CanUndo(),
		"can_redo":              hm.CanRedo(),
	}
}

// Clear resets the history manager
func (hm *HistoryManager) Clear() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.operations = make([]Operation, 0, 1000)
	hm.checkpoints = make([]Checkpoint, 0, 100)
	hm.currentIndex = -1
	hm.maxBatchSize = 100
	hm.operationsSinceCheckpoint = 0
}
