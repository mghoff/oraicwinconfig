package internal

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

// InstallOracleInstantClient performs the installation and configuration of Oracle Instant Client
func InstallOracleInstantClient(config *InstallConfig) error {
	// INSTALLATION STEPS
	fmt.Println("Starting Oracle InstantClient installation...")
	// Set paths for downloads
	pkgZipPath := filepath.Join(config.DownloadsPath, config.PkgFile)
	sdkZipPath := filepath.Join(config.DownloadsPath, config.SdkFile)

	// Download package files
	fmt.Printf("downloading package: %s...\n", pkgZipPath)
	if err := downloadOracleInstantClient(config.BaseURL+config.PkgFile, pkgZipPath); err != nil {
		return err
	}

	// Download SDK files
	fmt.Printf("downloading SDK: %s...\n", sdkZipPath)
	if err := downloadOracleInstantClient(config.BaseURL+config.SdkFile, sdkZipPath); err != nil {
		return err
	}

	// Unzip package files
	fmt.Printf("extracting: %s to %s\n", pkgZipPath, config.InstallPath)
	pkgDir, err := unzipOracleInstantClient(pkgZipPath, config.InstallPath)
	if err != nil {
		return handleError(err, ErrorTypeInstall, "unzip package")
	}

	// Unzip SDK files
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

	// CONFIGURATION STEPS
	fmt.Println("Configuring Oracle InstantClient...")
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

// downloadOracleInstantClient downloads the Oracle Instant Client zip file from the specified URL
func downloadOracleInstantClient(urlPath, destPath string) error {
	// Get zip archive from URL
	resp, err := http.Get(urlPath)
	if err != nil {
		return handleError(err, ErrorTypeDownload, "downloading from URL")
	}
	if resp.StatusCode != http.StatusOK {
		return handleError(fmt.Errorf("HTTP status %s", resp.Status), ErrorTypeDownload, "checking response status")
	}
	defer resp.Body.Close()

	// Create file
	out, err := os.Create(destPath)
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

// unzipOracleInstantClient extracts the Oracle Instant Client zip file to the specified destination path
// and returns the directory name of the extracted files
func unzipOracleInstantClient(zipPath, destPath string) (string, error) {
	// Create base directory
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
	var outPath string
	for k, f := range r.File {
		re := regexp.MustCompilePOSIX(`^(instantclient_){1}([0-9]{1,2})_([0-9]{1,2})\/$`)
		if re.Match([]byte(f.Name)) {
			outPath = f.Name
		}
		if err := extractFile(f, destPath); err != nil {
			return "", handleError(err, ErrorTypeInstall, fmt.Sprintf("extracting file %d", k))
		}
	}

	if outPath == "" {
		return "", handleError(
			fmt.Errorf("no valid instant client directory found in zip"),
			ErrorTypeInstall,
			"validating zip contents",
		)
	}

	return outPath, nil
}

// Helper function to extract a single file from zip
func extractFile(f *zip.File, destPath string) error {
	outName := filepath.Join(destPath, f.Name)

	if f.FileInfo().IsDir() {
		return os.MkdirAll(outName, 0777)
	}

	if err := os.MkdirAll(filepath.Dir(outName), 0777); err != nil {
		return fmt.Errorf("creating directories: %w", err)
	}

	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("opening zip file: %w", err)
	}
	defer rc.Close()

	out, err := os.Create(outName)
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
