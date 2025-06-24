package env

import (
	"fmt"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mghoff/oraicwinconfig/internal/errs"
)

// EnvVarManager handles environment variable operations
type EnvVarManager struct {
	powershell string
}

// NewEnvVarManager creates a new environment variable manager
func New() *EnvVarManager {
	return &EnvVarManager{
		powershell: "powershell",
	}
}

// FetchUserDownloadsPath retrieves the user profile directory for a given endpoint
// and checks if the directory exists
func (e *EnvVarManager) FetchUserDownloadsPath() (string, error) {
	cmd := "$env:USERPROFILE"
	out, err := exec.Command(e.powershell, cmd).Output()
	if err != nil {
		return "", errs.HandleError(err, errs.ErrorTypeUserPath, "getting user profile directory")
	}
	usrProfilePath := strings.TrimSpace(string(out)) // Convert to string and trim whitespace
	
	usrDownloadsPath := filepath.Join(usrProfilePath, "Downloads")
	if _, err := os.Stat(usrDownloadsPath); errors.Is(err, os.ErrNotExist) {
		return "", errs.HandleError(fmt.Errorf("directory does not exist: %s", usrDownloadsPath), errs.ErrorTypeUserPath, "checking user profile directory")
	} else if err != nil {
		return "", errs.HandleError(err, errs.ErrorTypeUserPath, "checking user profile directory")
	}
	
	return usrDownloadsPath, nil
}

// GetEnvVar retrieves a user environment variable
func (e *EnvVarManager) GetEnvVar(name string) (string, error) {
	cmd := fmt.Sprintf("[System.Environment]::GetEnvironmentVariable('%s', 'User')", name)
	out, err := exec.Command(e.powershell, cmd).Output()
	if err != nil {
		return "", errs.HandleError(err, errs.ErrorTypeEnvVarNotFound, fmt.Sprintf("getting %s environment variable", name))
	}
	path := strings.TrimSpace(string(out)) // Trim whitespace including newlines
	if path == ""  || path == "." || path == ".." || path == "/" || path == "\\" {
		return "", errs.HandleError(
			fmt.Errorf("environment variable %s not found", name),
			errs.ErrorTypeEnvVarNotFound,
			fmt.Sprintf("getting %s environment variable", name))
	}
	
	return path, nil
}

// ValidateEnvVar checks if an environment variable is set and points to a valid directory
func (e *EnvVarManager) ValidateEnvVar(name string) (string, error) {
	path, err := e.GetEnvVar(name)
	if errs.IsErrorType(err, errs.ErrorTypeEnvVarNotFound) {
		return "", err
	}
	
	// If exists, check if it points to a valid directory
	// This is the directory where the Oracle Instant Client files are expected to be located
	path = filepath.Clean(path)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "", errs.HandleError(fmt.Errorf("environment variable %s points to a non-existent directory: %s", name, path),
			errs.ErrorTypeEnvironment,
			fmt.Sprintf("checking %s path", name))
	}
	if err != nil {
		return "", errs.HandleError(err, errs.ErrorTypeEnvironment, fmt.Sprintf("checking %s path", name))
	}

	fmt.Printf("%s environment variable found: %s\n", name, path)
	return path, nil
}

// SetEnvVar sets a user environment variable
func (e *EnvVarManager) SetEnvVar(name, value string) error {
	cmd := fmt.Sprintf("[Environment]::SetEnvironmentVariable('%s', '%s', 'User')", name, value)
	if _, err := exec.Command(e.powershell, cmd).Output(); err != nil {
		return errs.HandleError(err, errs.ErrorTypeEnvironment, fmt.Sprintf("setting %s environment variable", name))
	}
	return nil
}

// RemoveEnvVar removes a user environment variable
func (e *EnvVarManager) RemoveEnvVar(name string) error {
	cmd := fmt.Sprintf("[Environment]::SetEnvironmentVariable('%s', $null, 'User')", name)
	if _, err := exec.Command(e.powershell, cmd).Output(); err != nil {
		return errs.HandleError(err, errs.ErrorTypeEnvironment, fmt.Sprintf("removing %s environment variable", name))
	}
	return nil
}

// AppendToPath adds a new path to the PATH environment variable
func (e *EnvVarManager) AppendToPath(newPath string) error {
	currentPath, err := e.GetEnvVar("PATH")
	if err != nil {
		return err
	}

	// Check if path already exists
	if strings.Contains(currentPath, newPath) {
		fmt.Printf("path %s already exists in PATH\n", newPath)
		return nil
	}

	// Ensure path ends with semicolon
	if !strings.HasSuffix(currentPath, ";") {
		currentPath += ";"
	}

	newFullPath := currentPath + newPath + ";"
	return e.SetEnvVar("PATH", newFullPath)
}

// removeFromPath removes a specified path from the PATH environment variable
func (e *EnvVarManager) RemoveFromPath(pathToRemove string) error {
	currentPath, err := e.GetEnvVar("PATH")
	if err != nil {
		return err
	}

	// Split the current PATH into segments
	segments := strings.Split(currentPath, ";")
	var newSegments []string

	// Filter out the segment to remove
	for _, segment := range segments {
		if segment != pathToRemove {
			newSegments = append(newSegments, segment)
		}
	}

	// Join the remaining segments back into a single string
	newPath := strings.Join(newSegments, ";")
	return e.SetEnvVar("PATH", newPath)
}
