package errors

import (
	"fmt"
	"runtime"
	"time"
)

// ErrorCode represents a specific error type
type ErrorCode string

const (
	// Memory errors
	ErrMemoryAllocation  ErrorCode = "MEMORY_ALLOCATION"
	ErrMemoryExhausted   ErrorCode = "MEMORY_EXHAUSTED"
	ErrArenaFull         ErrorCode = "ARENA_FULL"

	// FASTQ parsing errors
	ErrInvalidFASTQ      ErrorCode = "INVALID_FASTQ"
	ErrQualityMismatch   ErrorCode = "QUALITY_MISMATCH"
	ErrFileRead          ErrorCode = "FILE_READ"

	// Coordinate errors
	ErrInvalidCoordinate ErrorCode = "INVALID_COORDINATE"
	ErrOutOfBounds       ErrorCode = "OUT_OF_BOUNDS"

	// Rendering errors
	ErrWebGLContext      ErrorCode = "WEBGL_CONTEXT"
	ErrShaderCompilation ErrorCode = "SHADER_COMPILATION"
	ErrTextureCreation   ErrorCode = "TEXTURE_CREATION"

	// System errors
	ErrSystemResource    ErrorCode = "SYSTEM_RESOURCE"
	ErrTimeout           ErrorCode = "TIMEOUT"
	ErrCancelled         ErrorCode = "CANCELLED"
)

// Severity levels for errors
type Severity string

const (
	SeverityCritical Severity = "CRITICAL" // System cannot continue
	SeverityError    Severity = "ERROR"    // Operation failed
	SeverityWarning  Severity = "WARNING"  // Operation degraded
	SeverityInfo     Severity = "INFO"     // Informational
)

// GenomeVedicError is a custom error type with rich context
type GenomeVedicError struct {
	Code       ErrorCode
	Severity   Severity
	Message    string
	Cause      error
	Timestamp  time.Time
	StackTrace string
	Metadata   map[string]interface{}
	Recoverable bool
}

// Error implements the error interface
func (e *GenomeVedicError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s (caused by: %v)", e.Code, e.Severity, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Severity, e.Message)
}

// Unwrap implements error unwrapping
func (e *GenomeVedicError) Unwrap() error {
	return e.Cause
}

// New creates a new GenomeVedicError
func New(code ErrorCode, severity Severity, message string) *GenomeVedicError {
	return &GenomeVedicError{
		Code:       code,
		Severity:   severity,
		Message:    message,
		Timestamp:  time.Now(),
		StackTrace: captureStackTrace(),
		Metadata:   make(map[string]interface{}),
		Recoverable: severity == SeverityWarning || severity == SeverityInfo,
	}
}

// Wrap wraps an existing error with additional context
func Wrap(code ErrorCode, severity Severity, message string, cause error) *GenomeVedicError {
	return &GenomeVedicError{
		Code:       code,
		Severity:   severity,
		Message:    message,
		Cause:      cause,
		Timestamp:  time.Now(),
		StackTrace: captureStackTrace(),
		Metadata:   make(map[string]interface{}),
		Recoverable: severity == SeverityWarning || severity == SeverityInfo,
	}
}

// WithMetadata adds metadata to the error
func (e *GenomeVedicError) WithMetadata(key string, value interface{}) *GenomeVedicError {
	e.Metadata[key] = value
	return e
}

// WithRecoverable sets whether the error is recoverable
func (e *GenomeVedicError) WithRecoverable(recoverable bool) *GenomeVedicError {
	e.Recoverable = recoverable
	return e
}

// captureStackTrace captures the current stack trace
func captureStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// ErrorHandler handles errors with recovery strategies
type ErrorHandler struct {
	handlers map[ErrorCode]RecoveryFunc
	logger   Logger
}

// RecoveryFunc is a function that attempts to recover from an error
type RecoveryFunc func(*GenomeVedicError) error

// Logger is an interface for logging errors
type Logger interface {
	Log(err *GenomeVedicError)
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger Logger) *ErrorHandler {
	return &ErrorHandler{
		handlers: make(map[ErrorCode]RecoveryFunc),
		logger:   logger,
	}
}

// RegisterHandler registers a recovery handler for an error code
func (h *ErrorHandler) RegisterHandler(code ErrorCode, handler RecoveryFunc) {
	h.handlers[code] = handler
}

// Handle handles an error and attempts recovery
func (h *ErrorHandler) Handle(err error) error {
	// Cast to GenomeVedicError if possible
	gverr, ok := err.(*GenomeVedicError)
	if !ok {
		// Wrap unknown errors
		gverr = Wrap(ErrSystemResource, SeverityError, "Unknown error", err)
	}

	// Log the error
	if h.logger != nil {
		h.logger.Log(gverr)
	}

	// If not recoverable, return immediately
	if !gverr.Recoverable {
		return gverr
	}

	// Try to recover
	if handler, exists := h.handlers[gverr.Code]; exists {
		if recoveryErr := handler(gverr); recoveryErr == nil {
			return nil // Successfully recovered
		}
	}

	return gverr
}

// SimpleLogger is a basic logger implementation
type SimpleLogger struct{}

// Log logs an error to stdout
func (l *SimpleLogger) Log(err *GenomeVedicError) {
	fmt.Printf("[%s] %s [%s]: %s\n",
		err.Timestamp.Format("2006-01-02 15:04:05"),
		err.Severity,
		err.Code,
		err.Message)

	if err.Cause != nil {
		fmt.Printf("  Caused by: %v\n", err.Cause)
	}

	if len(err.Metadata) > 0 {
		fmt.Printf("  Metadata: %v\n", err.Metadata)
	}
}

// RecoveryStrategy defines common recovery strategies
type RecoveryStrategy struct{}

// RetryWithBackoff retries an operation with exponential backoff
func (rs *RecoveryStrategy) RetryWithBackoff(
	operation func() error,
	maxRetries int,
	initialDelay time.Duration,
) error {
	delay := initialDelay

	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := operation(); err == nil {
			return nil
		}

		time.Sleep(delay)
		delay *= 2 // Exponential backoff
	}

	return New(ErrTimeout, SeverityError, "Max retries exceeded")
}

// FallbackValue provides a fallback value when operation fails
func (rs *RecoveryStrategy) FallbackValue(
	operation func() (interface{}, error),
	fallback interface{},
) interface{} {
	if value, err := operation(); err == nil {
		return value
	}
	return fallback
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	maxFailures int
	resetTimeout time.Duration
	failures     int
	lastFailure  time.Time
	isOpen       bool
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		failures:     0,
		isOpen:       false,
	}
}

// Call executes an operation through the circuit breaker
func (cb *CircuitBreaker) Call(operation func() error) error {
	// Check if circuit should reset
	if cb.isOpen && time.Since(cb.lastFailure) > cb.resetTimeout {
		cb.isOpen = false
		cb.failures = 0
	}

	// If circuit is open, fail fast
	if cb.isOpen {
		return New(ErrSystemResource, SeverityError, "Circuit breaker is open")
	}

	// Try operation
	if err := operation(); err != nil {
		cb.failures++
		cb.lastFailure = time.Now()

		if cb.failures >= cb.maxFailures {
			cb.isOpen = true
		}

		return err
	}

	// Success - reset failures
	cb.failures = 0
	return nil
}

// ErrorAggregator collects multiple errors
type ErrorAggregator struct {
	errors []*GenomeVedicError
}

// NewErrorAggregator creates a new error aggregator
func NewErrorAggregator() *ErrorAggregator {
	return &ErrorAggregator{
		errors: make([]*GenomeVedicError, 0),
	}
}

// Add adds an error to the aggregator
func (ea *ErrorAggregator) Add(err error) {
	if err == nil {
		return
	}

	if gverr, ok := err.(*GenomeVedicError); ok {
		ea.errors = append(ea.errors, gverr)
	} else {
		ea.errors = append(ea.errors, Wrap(ErrSystemResource, SeverityError, "Unknown error", err))
	}
}

// HasErrors returns true if there are any errors
func (ea *ErrorAggregator) HasErrors() bool {
	return len(ea.errors) > 0
}

// GetErrors returns all collected errors
func (ea *ErrorAggregator) GetErrors() []*GenomeVedicError {
	return ea.errors
}

// Error returns a combined error message
func (ea *ErrorAggregator) Error() string {
	if len(ea.errors) == 0 {
		return "No errors"
	}

	if len(ea.errors) == 1 {
		return ea.errors[0].Error()
	}

	return fmt.Sprintf("Multiple errors (%d): %s (and %d more)",
		len(ea.errors), ea.errors[0].Message, len(ea.errors)-1)
}

// HighestSeverity returns the highest severity among all errors
func (ea *ErrorAggregator) HighestSeverity() Severity {
	if len(ea.errors) == 0 {
		return SeverityInfo
	}

	highest := SeverityInfo
	for _, err := range ea.errors {
		if err.Severity == SeverityCritical {
			return SeverityCritical
		}
		if err.Severity == SeverityError && highest != SeverityCritical {
			highest = SeverityError
		}
		if err.Severity == SeverityWarning && highest == SeverityInfo {
			highest = SeverityWarning
		}
	}

	return highest
}
