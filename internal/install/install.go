package install

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"context"
	"strings"

	"github.com/mghoff/oraicwinconfig/internal/config"
	"github.com/mghoff/oraicwinconfig/internal/env"
	"github.com/mghoff/oraicwinconfig/internal/errs"
)

// ensureContext returns context.Background() if ctx is nil, otherwise returns ctx.
func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// InstallExists checks if Oracle InstantClient is already installed
func Exists(ctx context.Context, conf *config.InstallConfig, env *env.EnvVarManager) (bool, error) {
	ctx = ensureContext(ctx)
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return false, errs.HandleError(err, errs.ErrorTypeInstall, "context cancellation")
	}

	fmt.Println("Checking for existing Oracle InstantClient installation...")

	// Check if OCI_LIB64 environment variable exists
	ociLibPath, err := env.GetEnvVar("OCI_LIB64")
	if err != nil {
		if errs.IsErrorType(err, errs.ErrorTypeEnvVarNotFound) {
			fmt.Println("No existing installation detected. Coninuing with fresh installation...")
			return false, nil
		}
		return false, errs.HandleError(err, errs.ErrorTypeEnvironment, "getting OCI_LIB64 environment variable")
	}
	if ociLibPath == "" {
		fmt.Println("OCI_LIB64 environment variable is set but empty, indicating a misconfigured existing installation.")
		return false, nil
	}
	fmt.Printf("OCI_LIB64 environment variable found: %s\n", ociLibPath)
	// If OCI_LIB64 exists, check if it points to a valid directory
	// This is the directory where the Oracle Instant Client files are expected to be located
	ociLibPath = filepath.Clean(ociLibPath)
	if _, err := os.Stat(ociLibPath); os.IsNotExist(err) {
		fmt.Println("OCI_LIB64 environment variable is set, but the path does not point to a valid directory. This indicates a misconfigured Oracle InstantClient installation, not just a missing install.")
	} else if err != nil {
		return false, errs.HandleError(err, errs.ErrorTypeEnvironment, "checking OCI_LIB64 path")
	} else {
		fmt.Println("OCI_LIB64 environment variable is set and points to a valid directory: existing installation detected.")
	}

	// Check if TNS_ADMIN environment variable exists
	// This variable should point to the directory containing the Oracle Net configuration files
	tnsAdminPath, err := env.GetEnvVar("TNS_ADMIN")
	if err != nil {
		if errs.IsErrorType(err, errs.ErrorTypeEnvVarNotFound) {
			fmt.Println("TNS_ADMIN environment variable not found, existing installation not properly configured.")
		} else {
			return false, errs.HandleError(err, errs.ErrorTypeEnvironment, "getting TNS_ADMIN environment variable")
		}
	}
	if tnsAdminPath == "" {
		fmt.Println("TNS_ADMIN environment variable is set but empty, indicating a misconfigured existing installation.")
	} 
	fmt.Printf("TNS_ADMIN environment variable found: %s\n", tnsAdminPath)

	// If TNS_ADMIN exists, check if it points to a valid directory
	tnsAdminPath = filepath.Clean(tnsAdminPath)
	if strings.Contains(tnsAdminPath, ociLibPath) && tnsAdminPath != ociLibPath && tnsAdminPath == filepath.Join(ociLibPath, "network", "admin") {
		fmt.Println("TNS_ADMIN environment variable is set and points to a subdirectory of OCI_LIB64, indicating a valid existing installation.")
	}	 else if _, err := os.Stat(tnsAdminPath); os.IsNotExist(err) {
		fmt.Println("TNS_ADMIN environment variable is set, but the path does not point to a valid directory. This indicates a misconfigured Oracle InstantClient installation, not just a missing install.")
		return false, nil
	} else if err != nil {
		return false, errs.HandleError(err, errs.ErrorTypeEnvironment, "checking TNS_ADMIN path")
	} else {
		fmt.Println("TNS_ADMIN environment variable is set and points to a valid directory: existing installation detected.")
	}

	fmt.Printf("\nExisting Oracle InstantClient installation found at %s and is valid and configured correctly.", ociLibPath)

	// Update the config with the installation path
	if ociLibPath != "" && ociLibPath != "/" && ociLibPath != "\\" && ociLibPath != "." {
		conf.InstallPath = ociLibPath
	} else {
		return false, errs.HandleError(
			fmt.Errorf("OCI_LIB64 environment variable is set but does not point to a valid installation path: %s", ociLibPath),
			errs.ErrorTypeInstall,
			"validating OCI_LIB64 path",
		)
	}
	
	return true, nil
}

// UninstallOracleInstantClient removes the Oracle InstantClient installation
// It cleans up the environment variables and removes the installation directory
func Remove(ctx context.Context, conf *config.InstallConfig, env *env.EnvVarManager) error {
	ctx = ensureContext(ctx)
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "context cancellation")
	}

	// Remove OCI_LIB64 from PATH
	envVar, err := env.GetEnvVar("OCI_LIB64")
	if err != nil {
		if errs.IsErrorType(err, errs.ErrorTypeEnvVarNotFound) {
			fmt.Println("OCI_LIB64 environment variable not found, skipping removal from PATH.")
			return nil
		}
		return errs.HandleError(err, errs.ErrorTypeEnvironment, "getting OCI_LIB64 environment variable")
	}
	if err := env.RemoveFromPath(envVar); err != nil {
		return errs.HandleError(err, errs.ErrorTypeEnvironment, "removing OCI_LIB64 from PATH")
	}

	// Remove OCI_LIB64 environment variable
	if err := env.RemoveEnvVar("OCI_LIB64"); err != nil {
		return errs.HandleError(err, errs.ErrorTypeEnvironment, "removing OCI_LIB64")
	}

	// Remove TNS_ADMIN environment variable
	if err := env.RemoveEnvVar("TNS_ADMIN"); err != nil {
		return errs.HandleError(err, errs.ErrorTypeEnvironment, "removing TNS_ADMIN")
	}

	// Remove installation directory with safety checks
	if conf.InstallPath == "" || conf.InstallPath == "/" || conf.InstallPath == "\\" || conf.InstallPath == "." {
		return errs.HandleError(
			fmt.Errorf("refusing to remove invalid or critical system directory: %q", conf.InstallPath),
			errs.ErrorTypeInstall,
			"removing installation directory",
		)
	}
	if err := os.RemoveAll(conf.InstallPath); err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "removing installation directory")
	}

	// Reset the installation path in the config
	conf.InstallPath = filepath.Dir(conf.InstallPath) // Reset install path to base directory
	if conf.InstallPath == "" || conf.InstallPath == "/" || conf.InstallPath == "\\" || conf.InstallPath == "." {
		return errs.HandleError(
			fmt.Errorf("installation path reset to an invalid or critical system directory: %q", conf.InstallPath),
			errs.ErrorTypeInstall,
			"resetting installation path",
		)
	}

	return nil
}

// InstallOracleInstantClient performs the installation and configuration of Oracle Instant Client
func OracleInstantClient(ctx context.Context, conf *config.InstallConfig, env *env.EnvVarManager) error {
	ctx = ensureContext(ctx)
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "context cancellation")
	}


	// INSTALLATION STEPS
	fmt.Println("\nStarting Oracle InstantClient installation...")
	// Set paths for downloads
	pkgZipPath := filepath.Join(conf.DownloadsPath, conf.PkgFile)
	sdkZipPath := filepath.Join(conf.DownloadsPath, conf.SdkFile)

	// Download package files
	fmt.Printf("downloading package: %s...\n", pkgZipPath)
	if err := downloadOracleInstantClient(ctx, conf.BaseURL+conf.PkgFile, pkgZipPath); err != nil {
		return err
	}

	// Download SDK files
	fmt.Printf("downloading SDK: %s...\n", sdkZipPath)
	if err := downloadOracleInstantClient(ctx, conf.BaseURL+conf.SdkFile, sdkZipPath); err != nil {
		return err
	}

	// Unzip package files
	fmt.Printf("extracting: %s to %s\n", pkgZipPath, conf.InstallPath)
	pkgDir, err := unzipOracleInstantClient(pkgZipPath, conf.InstallPath)
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "unzip package")
	}

	// Unzip SDK files
	fmt.Printf("extracting: %s to %s\n", sdkZipPath, filepath.Join(conf.InstallPath, pkgDir, "sdk"))
	sdkDir, err := unzipOracleInstantClient(sdkZipPath, conf.InstallPath)
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "unzip SDK")
	}

	// Verify version match
	if pkgDir != sdkDir {
		return errs.HandleError(
			fmt.Errorf("package version (%s) does not match SDK version (%s)", pkgDir, sdkDir),
			errs.ErrorTypeInstall,
			"version verification",
		)
	}
	fmt.Println("package and SDK versions match, continuing...")

	// CONFIGURATION STEPS
	fmt.Println("\nConfiguring Oracle InstantClient...")

	// Set OCI_LIB64 environment variable
	ociLibPath := filepath.Join(conf.InstallPath, pkgDir)
	fmt.Printf("setting OCI_LIB64=%s\n", ociLibPath)
	if err := env.SetEnvVar("OCI_LIB64", ociLibPath); err != nil {
		return err
	}

	// Add OCI_LIB64 to PATH
	fmt.Printf("updating PATH to include %s\n", ociLibPath)
	if err := env.AppendToPath(ociLibPath); err != nil {
		return err
	}

	// Set TNS_ADMIN environment variable
	tnsAdminPath := filepath.Join(ociLibPath, "network", "admin")
	fmt.Printf("setting TNS_ADMIN=%s\n", tnsAdminPath)
	if err := env.SetEnvVar("TNS_ADMIN", tnsAdminPath); err != nil {
		return err
	}

	fmt.Println("\nOracle InstantClient installation and configuration completed successfully!")
	return nil
}

// downloadOracleInstantClient downloads the Oracle Instant Client zip file from the specified URL
func downloadOracleInstantClient(ctx context.Context, urlPath, downloadsPath string) error {
	ctx = ensureContext(ctx)
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return errs.HandleError(err, errs.ErrorTypeDownload, "context cancellation")
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeDownload, "creating HTTP request")
	}

	// Get zip archive from URL
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeDownload, "downloading from URL")
	}
	if resp.StatusCode != http.StatusOK {
		return errs.HandleError(fmt.Errorf("HTTP status %s", resp.Status), errs.ErrorTypeDownload, "checking response status")
	}
	defer resp.Body.Close()

	// Create file
	out, err := os.Create(downloadsPath)
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeDownload, "creating download file")
	}
	defer out.Close()

	// Write response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeDownload, "writing download to file")
	}
	return nil
}

// unzipOracleInstantClient extracts the Oracle Instant Client zip file to the specified destination path
// and returns the directory name of the extracted files
func unzipOracleInstantClient(downloadsPath, installPath string) (string, error) {
	// Create base install directory
	if err := os.MkdirAll(installPath, 0777); err != nil {
		return "", errs.HandleError(err, errs.ErrorTypeInstall, "creating base installation directory")
	}

	// Open a zip archive for reading.zip files from the Downloads directory
	r, err := zip.OpenReader(downloadsPath)
	if err != nil {
		return "", errs.HandleError(err, errs.ErrorTypeInstall, "opening zip archive")
	}
	defer r.Close()

	// Iterate through the files in the zip archive,
	// and extract contents into the Installation directory
	var outPath string
	for k, f := range r.File {
		re := regexp.MustCompilePOSIX(`^(instantclient_){1}([0-9]{1,2})_([0-9]{1,2})\/$`)
		if re.Match([]byte(f.Name)) {
			outPath = f.Name
		}
		if err := extractFile(f, installPath); err != nil {
			return "", errs.HandleError(err, errs.ErrorTypeInstall, fmt.Sprintf("extracting file %d", k))
		}
	}

	if outPath == "" {
		return "", errs.HandleError(
			fmt.Errorf("no valid instant client directory found in zip"),
			errs.ErrorTypeInstall,
			"validating zip contents",
		)
	}

	return filepath.Clean(outPath), nil
}

// Helper function to extract a single file from zip archive to specified install path
// It creates necessary directories and handles file creation
func extractFile(f *zip.File, installPath string) error {
	outName := filepath.Join(installPath, f.Name)

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
