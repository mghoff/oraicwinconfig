package config

import (
	"fmt"

	"github.com/mghoff/oraicwinconfig/internal/errs"
)

const (
	defaultInstallPath = "C:/OraClient"
	pkgFileName        = "instantclient-basiclite-windows.zip"
	sdkFileName        = "instantclient-sdk-windows.zip"
	baseDownloadURL    = "https://download.oracle.com/otn_software/nt/instantclient/"
)

// InstallConfig holds all installation configurations
type InstallConfig struct {
	DownloadsPath string // Path where downloaded files will be stored
	InstallPath   string // Path where Oracle Instant Client will be installed
	PkgFile       string // Name of the package file to be downloaded
	SdkFile       string // Name of the SDK file to be downloaded
	BaseURL       string // Base URL for downloading the files
	Extant				bool   // Indicates if an existing installation was found
}

// NewDefaultConfig creates a new configuration with default values
// and returns a pointer to it
func New() *InstallConfig {
	return &InstallConfig{
		InstallPath: defaultInstallPath,
		PkgFile:     pkgFileName,
		SdkFile:     sdkFileName,
		BaseURL:     baseDownloadURL,
		Extant:      false,
	}
}

// checkPathValidity checks if the provided path is valid
func checkPathValidity(path string) bool {
	if path == "" || path == "." || path == ".." || path == "/" || path == "\\" {
		return false
	}
	return true
}

// SetDownloadsPath sets the path where the downloaded files will be stored
func (c *InstallConfig) SetDownloadsPath(path string) error {
	if !checkPathValidity(path) {
		return errs.HandleError(
			fmt.Errorf("downloads path cannot be empty or invalid"),
			errs.ErrorTypeValidation,
			"setting downloads path")
	}
	c.DownloadsPath = path
	return nil
}

// SetInstallPath sets the path where the Oracle Instant Client will be installed
func (c *InstallConfig) SetInstallPath(path string) error {
	if !checkPathValidity(path) {
		return errs.HandleError(
			fmt.Errorf("install path cannot be empty or invalid"),
			errs.ErrorTypeValidation,
			"setting install path")
	}
	c.InstallPath = path
	return nil
}

// SetExtant sets the extant flag indicating if an existing installation was found
func (c *InstallConfig) SetExtant(extant bool) error{
	if extant != true && extant != false {
		return errs.HandleError(
			fmt.Errorf("extant must be a boolean value"),
			errs.ErrorTypeValidation,
			"setting extant value")
	}
	c.Extant = extant
	return nil
}

// Validate checks if the configuration is valid
// and returns an error if any required fields are missing or invalid.
func (c *InstallConfig) Validate() error {
	if !checkPathValidity(c.InstallPath) {
		return errs.HandleError(
			fmt.Errorf("install path cannot be empty or invalid"),
			errs.ErrorTypeValidation,
			"config validation")
	}
	if !checkPathValidity(c.DownloadsPath) {
		return errs.HandleError(
			fmt.Errorf("downloads path cannot be empty or invalid"),
			errs.ErrorTypeValidation,
			"config validation")
	}
	return nil
}
