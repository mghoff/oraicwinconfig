package internal

import "fmt"

type ErrorType int

// Error type definitions
const (
	ErrorTypeDownload ErrorType = iota
	ErrorTypeInstall
	ErrorTypeEnvironment
	ErrorTypeValidation
	ErrorTypeUserPath
)

// InstallError represents a contextual error during installation
type InstallError struct {
	Type      ErrorType
	Operation string
	Err       error
}

// Error implements the error interface for InstallError
func (e *InstallError) Error() string {
	return fmt.Sprintf("%s: %v", e.Operation, e.Err)
}

// Unwrap implements the Unwrap method for InstallError
func (e *InstallError) Unwrap() error {
	return e.Err
}

// handleError creates a new InstallError with context
func handleError(err error, errorType ErrorType, operation string) error {
	if err != nil {
		return &InstallError{
			Type:      errorType,
			Operation: operation,
			Err:       err,
		}
	}
	return nil
}
