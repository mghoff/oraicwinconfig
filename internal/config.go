package internal

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
