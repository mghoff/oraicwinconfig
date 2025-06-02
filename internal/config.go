package internal

import "fmt"

const (
	defaultInstallPath = "C:/OraClient"
	pkgFileName        = "instantclient-basiclite-windows.zip"
	sdkFileName        = "instantclient-sdk-windows.zip"
	baseDownloadURL    = "https://download.oracle.com/otn_software/nt/instantclient/"
)

// InstallConfig holds all installation configurations
type InstallConfig struct {
	InstallPath   string
	DownloadsPath string
	PkgFile       string
	SdkFile       string
	BaseURL       string
}

// NewDefaultConfig creates a new configuration with default values
// and returns a pointer to it
func NewDefaultConfig() *InstallConfig {
	return &InstallConfig{
		InstallPath: defaultInstallPath,
		PkgFile:     pkgFileName,
		SdkFile:     sdkFileName,
		BaseURL:     baseDownloadURL,
	}
}

// Validate checks if the configuration is valid
// and returns an error if any required fields are missing or invalid.
func (c *InstallConfig) Validate() error {
    if c.InstallPath == "" {
        return HandleError(
            fmt.Errorf("install path cannot be empty"),
            ErrorTypeValidation,
            "config validation",
        )
    }
    if c.DownloadsPath == "" {
        return HandleError(
            fmt.Errorf("downloads path cannot be empty"),
            ErrorTypeValidation,
            "config validation",
        )
    }
    return nil
}