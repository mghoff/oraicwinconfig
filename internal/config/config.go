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
	DownloadsPath string
	InstallPath   string
	PkgFile       string
	SdkFile       string
	BaseURL       string
}

// NewDefaultConfig creates a new configuration with default values
// and returns a pointer to it
func New() *InstallConfig {
	return &InstallConfig{
		InstallPath: defaultInstallPath,
		PkgFile:     pkgFileName,
		SdkFile:     sdkFileName,
		BaseURL:     baseDownloadURL,
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
