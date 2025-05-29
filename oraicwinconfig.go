package main

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	config := NewDefaultConfig()

	downloads, err := getUserDestPath("Downloads")
	if err != nil {
		log.Fatal("error getting user Downloads directory: ", err)
	}
	config.DownloadsPath = downloads

	fmt.Printf("files will be downloaded from '%s' to '%s':\n", config.BaseURL, config.DownloadsPath)
	fmt.Printf("- %s\n- %s\n", config.PkgFile, config.SdkFile)

	// Handle installation path selection
	if err := handleInstallLocation(config); err != nil {
		log.Fatal("error handling install location: ", err)
	}

	// Perform installation
	if err := InstallOracleInstantClient(config); err != nil {
		var installErr *InstallError
		if errors.As(err, &installErr) {
			switch installErr.Type {
			case ErrorTypeDownload:
				log.Fatal("download failed: ", err)
			case ErrorTypeInstall:
				log.Fatal("installation failed: ", err)
			case ErrorTypeEnvironment:
				log.Fatal("environment setup failed: ", err)
			default:
				log.Fatal("unknown error: ", err)
			}
		}
		log.Fatal("installation failed: ", err)
	}

	fmt.Println("installation completed successfully")
}

// Configuration constants
const (
	defaultInstallPath = "C:/OraClient"
	pkgFileName        = "instantclient-basiclite-windows.zip"
	sdkFileName        = "instantclient-sdk-windows.zip"
	baseDownloadURL    = "https://download.oracle.com/otn_software/nt/instantclient/"
)

// InstallConfig holds all installation configuration
type InstallConfig struct {
	InstallPath   string
	DownloadsPath string
	PkgFile       string
	SdkFile       string
	BaseURL       string
}

// NewDefaultConfig creates a new configuration with default values
func NewDefaultConfig() *InstallConfig {
	return &InstallConfig{
		InstallPath: defaultInstallPath,
		PkgFile:     pkgFileName,
		SdkFile:     sdkFileName,
		BaseURL:     baseDownloadURL,
	}
}

// Error type definition
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

func (e *InstallError) Error() string {
	return fmt.Sprintf("%s: %v", e.Operation, e.Err)
}

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

// EnvVarManager handles environment variable operations
type EnvVarManager struct {
	powershell string
}

// NewEnvVarManager creates a new environment variable manager
func NewEnvVarManager() *EnvVarManager {
	return &EnvVarManager{
		powershell: "powershell",
	}
}

// GetEnvVar retrieves a user environment variable
func (e *EnvVarManager) GetEnvVar(name string) (string, error) {
	cmd := fmt.Sprintf("[System.Environment]::GetEnvironmentVariable('%s', 'User')", name)
	out, err := exec.Command(e.powershell, cmd).Output()
	if err != nil {
		return "", handleError(err, ErrorTypeEnvironment, fmt.Sprintf("getting %s environment variable", name))
	}
	return strings.TrimSuffix(string(out), "\r\n"), nil
}

// SetEnvVar sets a user environment variable
func (e *EnvVarManager) SetEnvVar(name, value string) error {
	cmd := fmt.Sprintf("[Environment]::SetEnvironmentVariable('%s', '%s', 'User')", name, value)
	if _, err := exec.Command(e.powershell, cmd).Output(); err != nil {
		return handleError(err, ErrorTypeEnvironment, fmt.Sprintf("setting %s environment variable", name))
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

// handleInstallLocation handles the user interaction for installation path
func handleInstallLocation(config *InstallConfig) error {
	if ok := reqUserConfirmation("Accept the default install location?\n - " + config.InstallPath + "\nSelect"); !ok {
		if change := reqUserConfirmation("Change the default install location?"); change {
			newPath := reqUserInstallPath("Enter desired install path...\n")
			config.InstallPath = newPath
		}

		if cont := reqUserConfirmation("Continue with install?"); !cont {
			return handleError(
				fmt.Errorf("installation aborted by user"),
				ErrorTypeValidation,
				"user confirmation",
			)
		}
	}
	return nil
}

func getUserDestPath(dirEndpoint string) (string, error) {
	usrDir, err := exec.Command("powershell", "$env:USERPROFILE").Output()
	if err != nil {
		return "", handleError(err, ErrorTypeUserPath, "getting user profile directory")
	}

	dir := filepath.Join(strings.TrimSuffix(string(usrDir), "\r\n"), dirEndpoint)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return "", handleError(fmt.Errorf("directory does not exist: %s", dir), ErrorTypeUserPath, "checking user profile directory")
	} else if err != nil {
		return "", handleError(err, ErrorTypeUserPath, "checking user profile directory")
	}

	return dir, nil
}

func reqUserConfirmation(label string) bool {
	choices := "y/n"
	r := bufio.NewReader(os.Stdin)
	attempts := 0
	maxAttempts := 3
	for attempts < maxAttempts {
		fmt.Fprintf(os.Stderr, "%s (%s): ", label, choices)
		s, err := r.ReadString('\n')
		if err != nil {
			log.Fatal("error reading input: ", err)
		}
		s = strings.ToLower(strings.TrimSpace(s))
		switch s {
		case "y":
			return true
		case "n":
			return false
		default:
			attempts++
			fmt.Printf("must enter 'y' or 'n' (%d attempts remaining)\n", maxAttempts-attempts)
		}
	}
	log.Fatal("maximum input attempts exceeded")
	return false
}

func reqUserInstallPath(label string) string {
	r := bufio.NewReader(os.Stdin)
	var path string
	for {
		fmt.Fprintf(os.Stderr, "%s", label)
		path, _ = r.ReadString('\n')
		path = strings.TrimSpace(path)
		if stat, err := os.Stat(path); err == nil && stat.IsDir() {
			return path
		} else {
			fmt.Printf("Invalid path provided: %s (error: %v)\n", path, err)
			continue
		}
	}
}

func downloadOracleInstantClient(url, dest string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return handleError(err, ErrorTypeDownload, "downloading from URL")
	}
	if resp.StatusCode != http.StatusOK {
		return handleError(fmt.Errorf("HTTP status %s", resp.Status), ErrorTypeDownload, "checking response status")
	}
	defer resp.Body.Close()

	// Create file
	out, err := os.Create(dest)
	if err != nil {
		return handleError(err, ErrorTypeDownload, "creating download file")
	}
	defer out.Close()

	// Write response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return handleError(err, ErrorTypeDownload, "writing download to file")
	}
	return nil
}

func unzipOracleInstantClient(zipPath, destPath string) (string, error) {
	// Create base folder
	if err := os.MkdirAll(destPath, 0777); err != nil {
		return "", handleError(err, ErrorTypeInstall, "creating base directory")
	}

	// Open a zip archive for reading.
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", handleError(err, ErrorTypeInstall, "opening zip archive")
	}
	defer r.Close()

	// Iterate through the files in the archive, printing some of their contents.
	var outDestPath string
	for k, f := range r.File {
		re := regexp.MustCompilePOSIX(`^(instantclient_){1}([0-9]{1,2})_([0-9]{1,2})\/$`)
		if re.Match([]byte(f.Name)) {
			outDestPath = f.Name
		}
		if err := extractFile(f, destPath); err != nil {
			return "", handleError(err, ErrorTypeInstall, fmt.Sprintf("extracting file %d", k))
		}
	}

	if outDestPath == "" {
		return "", handleError(
			fmt.Errorf("no valid instant client directory found in zip"),
			ErrorTypeInstall,
			"validating zip contents",
		)
	}

	return outDestPath, nil
}

// Helper function to extract a single file from zip
func extractFile(f *zip.File, destPath string) error {
	outPath := filepath.Join(destPath, f.Name)

	if f.FileInfo().IsDir() {
		return os.MkdirAll(outPath, 0777)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0777); err != nil {
		return fmt.Errorf("creating directories: %w", err)
	}

	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("opening zip file: %w", err)
	}
	defer rc.Close()

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	if err != nil {
		return fmt.Errorf("writing file contents: %w", err)
	}

	return nil
}

func InstallOracleInstantClient(config *InstallConfig) error {
	// Set paths for downloads
	pkgZipPath := filepath.Join(config.DownloadsPath, config.PkgFile)
	sdkZipPath := filepath.Join(config.DownloadsPath, config.SdkFile)

	// Download files
	fmt.Printf("downloading package: %s...\n", pkgZipPath)
	if err := downloadOracleInstantClient(config.BaseURL+config.PkgFile, pkgZipPath); err != nil {
		return err
	}

	fmt.Printf("downloading SDK: %s...\n", sdkZipPath)
	if err := downloadOracleInstantClient(config.BaseURL+config.SdkFile, sdkZipPath); err != nil {
		return err
	}

	// Unzip files
	fmt.Printf("extracting: %s to %s\n", pkgZipPath, config.InstallPath)
	pkgDir, err := unzipOracleInstantClient(pkgZipPath, config.InstallPath)
	if err != nil {
		return handleError(err, ErrorTypeInstall, "unzip package")
	}

	fmt.Printf("extracting: %s\n", sdkZipPath)
	sdkDir, err := unzipOracleInstantClient(sdkZipPath, config.InstallPath)
	if err != nil {
		return handleError(err, ErrorTypeInstall, "unzip SDK")
	}

	// Verify version match
	if pkgDir != sdkDir {
		return handleError(
			fmt.Errorf("package version (%s) does not match SDK version (%s)", pkgDir, sdkDir),
			ErrorTypeInstall,
			"version verification",
		)
	}
	fmt.Println("package and SDK versions match, continuing...")

	// Setup environment variables
	envManager := NewEnvVarManager()

	ociLibPath := filepath.Join(config.InstallPath, pkgDir)
	fmt.Printf("setting OCI_LIB64=%s\n", ociLibPath)
	if err := envManager.SetEnvVar("OCI_LIB64", ociLibPath); err != nil {
		return err
	}

	fmt.Printf("updating PATH to include %s\n", ociLibPath)
	if err := envManager.AppendToPath(ociLibPath); err != nil {
		return err
	}

	tnsAdminPath := filepath.Join(ociLibPath, "network", "admin")
	fmt.Printf("setting TNS_ADMIN=%s\n", tnsAdminPath)
	if err := envManager.SetEnvVar("TNS_ADMIN", tnsAdminPath); err != nil {
		return err
	}

	fmt.Println("Oracle InstantClient installation completed successfully!")
	return nil
}
