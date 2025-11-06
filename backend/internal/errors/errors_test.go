package errors

import (
	"fmt"
	"testing"
	"time"
)

func TestNewError(t *testing.T) {
	err := New(ErrMemoryAllocation, SeverityError, "Memory allocation failed")

	if err.Code != ErrMemoryAllocation {
		t.Errorf("Expected code %s, got %s", ErrMemoryAllocation, err.Code)
	}
	if err.Severity != SeverityError {
		t.Errorf("Expected severity %s, got %s", SeverityError, err.Severity)
	}
	if err.Message != "Memory allocation failed" {
		t.Errorf("Unexpected message: %s", err.Message)
	}
	if err.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
	if len(err.StackTrace) == 0 {
		t.Error("Stack trace should be captured")
	}
}

func TestWrapError(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := Wrap(ErrFileRead, SeverityError, "Failed to read file", cause)

	if err.Cause != cause {
		t.Error("Cause should be set")
	}
	if err.Unwrap() != cause {
		t.Error("Unwrap should return cause")
	}
}

func TestErrorWithMetadata(t *testing.T) {
	err := New(ErrInvalidFASTQ, SeverityError, "Invalid FASTQ format").
		WithMetadata("line", 42).
		WithMetadata("file", "test.fastq")

	if len(err.Metadata) != 2 {
		t.Errorf("Expected 2 metadata entries, got %d", len(err.Metadata))
	}

	line, ok := err.Metadata["line"].(int)
	if !ok || line != 42 {
		t.Error("Metadata 'line' not set correctly")
	}
}

func TestRecoverable(t *testing.T) {
	// Warning errors should be recoverable by default
	err := New(ErrQualityMismatch, SeverityWarning, "Quality mismatch")
	if !err.Recoverable {
		t.Error("Warning errors should be recoverable by default")
	}

	// Critical errors should not be recoverable by default
	err2 := New(ErrMemoryExhausted, SeverityCritical, "Out of memory")
	if err2.Recoverable {
		t.Error("Critical errors should not be recoverable by default")
	}

	// Can override
	err3 := New(ErrFileRead, SeverityError, "File read error").WithRecoverable(true)
	if !err3.Recoverable {
		t.Error("Should be able to mark error as recoverable")
	}
}

func TestErrorHandler(t *testing.T) {
	logger := &SimpleLogger{}
	handler := NewErrorHandler(logger)

	// Register recovery handler
	recoveryAttempted := false
	handler.RegisterHandler(ErrMemoryAllocation, func(err *GenomeVedicError) error {
		recoveryAttempted = true
		return nil // Successfully recovered
	})

	// Create recoverable error
	err := New(ErrMemoryAllocation, SeverityWarning, "Memory allocation failed")
	result := handler.Handle(err)

	if result != nil {
		t.Errorf("Expected successful recovery, got error: %v", result)
	}
	if !recoveryAttempted {
		t.Error("Recovery handler should have been called")
	}
}

func TestErrorHandlerNonRecoverable(t *testing.T) {
	logger := &SimpleLogger{}
	handler := NewErrorHandler(logger)

	// Create non-recoverable error
	err := New(ErrMemoryExhausted, SeverityCritical, "Out of memory")
	result := handler.Handle(err)

	if result == nil {
		t.Error("Critical errors should not be recovered automatically")
	}
}

func TestRetryWithBackoff(t *testing.T) {
	strategy := &RecoveryStrategy{}

	attempts := 0
	operation := func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("failure %d", attempts)
		}
		return nil
	}

	err := strategy.RetryWithBackoff(operation, 5, 1*time.Millisecond)
	if err != nil {
		t.Errorf("Expected successful retry, got: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryWithBackoffExhausted(t *testing.T) {
	strategy := &RecoveryStrategy{}

	operation := func() error {
		return fmt.Errorf("always fails")
	}

	err := strategy.RetryWithBackoff(operation, 3, 1*time.Millisecond)
	if err == nil {
		t.Error("Expected error after max retries")
	}
}

func TestFallbackValue(t *testing.T) {
	strategy := &RecoveryStrategy{}

	// Successful operation
	result := strategy.FallbackValue(func() (interface{}, error) {
		return 42, nil
	}, 0)

	if result != 42 {
		t.Errorf("Expected 42, got %v", result)
	}

	// Failed operation
	result2 := strategy.FallbackValue(func() (interface{}, error) {
		return nil, fmt.Errorf("failure")
	}, "fallback")

	if result2 != "fallback" {
		t.Errorf("Expected fallback value, got %v", result2)
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)

	// Fail 3 times
	for i := 0; i < 3; i++ {
		err := cb.Call(func() error {
			return fmt.Errorf("failure %d", i)
		})
		if err == nil {
			t.Error("Expected error")
		}
	}

	// Circuit should be open now
	err := cb.Call(func() error {
		return nil
	})

	if err == nil {
		t.Error("Circuit breaker should be open")
	}

	// Wait for reset
	time.Sleep(150 * time.Millisecond)

	// Should work now
	err = cb.Call(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Circuit breaker should have reset, got error: %v", err)
	}
}

func TestErrorAggregator(t *testing.T) {
	agg := NewErrorAggregator()

	if agg.HasErrors() {
		t.Error("Should not have errors initially")
	}

	// Add errors
	agg.Add(New(ErrMemoryAllocation, SeverityWarning, "Warning 1"))
	agg.Add(New(ErrInvalidFASTQ, SeverityError, "Error 1"))
	agg.Add(New(ErrMemoryExhausted, SeverityCritical, "Critical 1"))

	if !agg.HasErrors() {
		t.Error("Should have errors after adding")
	}

	if len(agg.GetErrors()) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(agg.GetErrors()))
	}

	// Check highest severity
	severity := agg.HighestSeverity()
	if severity != SeverityCritical {
		t.Errorf("Expected CRITICAL severity, got %s", severity)
	}
}

func TestErrorAggregatorEmpty(t *testing.T) {
	agg := NewErrorAggregator()

	if agg.Error() != "No errors" {
		t.Errorf("Expected 'No errors', got %s", agg.Error())
	}

	if agg.HighestSeverity() != SeverityInfo {
		t.Error("Empty aggregator should return INFO severity")
	}
}

func TestErrorString(t *testing.T) {
	err := New(ErrMemoryAllocation, SeverityError, "Test error")
	str := err.Error()

	if str == "" {
		t.Error("Error string should not be empty")
	}

	// Should contain code and message
	if err.Code != ErrMemoryAllocation {
		t.Error("Error string should contain error code")
	}
}

func BenchmarkNewError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New(ErrMemoryAllocation, SeverityError, "Test error")
	}
}

func BenchmarkWrapError(b *testing.B) {
	cause := fmt.Errorf("underlying")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Wrap(ErrFileRead, SeverityError, "Test error", cause)
	}
}

func BenchmarkCircuitBreaker(b *testing.B) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Call(func() error {
			return nil
		})
	}
}
