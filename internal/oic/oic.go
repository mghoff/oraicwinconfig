package oic

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
	"errors"

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
	// This variable should point to the directory where the Oracle Instant Client files are located
	// If it exists and points to a valid directory, it indicates an existing installation
	ociLibPath, err := env.ValidateEnvVar("OCI_LIB64")
	if err != nil {
		fmt.Println("OCI_LIB64 environment variable not found or invalid, indicating no existing installation.")
		return false, err
	}
	fmt.Println("OCI_LIB64 environment variable is set and is valid, indicating an existing installation.")

	// Update the config with the existing installation path
	if err := conf.SetInstallPath(ociLibPath); errs.IsErrorType(err, errs.ErrorTypeValidation) {
		return false, err
	}

	// Check if TNS_ADMIN environment variable exists
	// This variable should point to the directory containing the Oracle Net configuration files
	// If it exists and points to a valid subdirectory of OCI_LIB64, it indicates a valid existing installation
	tnsAdminPath, err := env.ValidateEnvVar("TNS_ADMIN")
	if err != nil || !strings.Contains(tnsAdminPath, ociLibPath) || tnsAdminPath == ociLibPath || tnsAdminPath != filepath.Join(ociLibPath, "network", "admin"){
		fmt.Println("TNS_ADMIN environment variable not found or invalid, indicating a misconfigured existing installation.")
		fmt.Println("\nAn existing Oracle InstantClient installation was found, but appears misconfigured.")
		return true, nil
	}
	fmt.Println("TNS_ADMIN environment variable is set and points to a subdirectory of OCI_LIB64, indicating a valid existing installation.")

	// Check if the TNS_ADMIN directory contains tnsnames.ora file
	// This file is essential for Oracle Net configuration and should exist in the TNS_ADMIN directory
	if _, err := os.Stat(filepath.Join(tnsAdminPath, "tnsnames.ora")); err != nil || errors.Is(err, os.ErrNotExist) {
		fmt.Println("TNS_ADMIN directory does not contain a tnsnames.ora file, indicating a misconfigured existing installation.")
		fmt.Println("\nAn existing Oracle InstantClient installation was found, but appears misconfigured.")
		return true, nil
	}
	fmt.Println("TNS_ADMIN directory contains a tnsnames.ora file, indicating a valid existing installation.")

	// If all checks passed, we have a valid existing installation
	if err := conf.SetExtant(true); err != nil {
		return false, err
	}
	
	fmt.Printf("\nExisting Oracle InstantClient installation found at %s and is valid and configured correctly.", ociLibPath)
	return true, nil
}

// UninstallOracleInstantClient removes the Oracle InstantClient installation
// It cleans up the environment variables and removes the installation directory
func Uninstall(ctx context.Context, conf *config.InstallConfig, env *env.EnvVarManager) error {
	ctx = ensureContext(ctx)
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
		return err
	}
	if err := env.RemoveFromPath(envVar); err != nil {
		return err
	}

	// Remove OCI_LIB64 environment variable
	if err := env.RemoveEnvVar("OCI_LIB64"); err != nil {
		return err
	}

	// Remove TNS_ADMIN environment variable
	if err := env.RemoveEnvVar("TNS_ADMIN"); err != nil {
		return err
	}

	// Save tnsnames.ora file to user Downloads, if it exists
	// This is useful for restoring the configuration later during reinstallation
	if conf.Extant && conf.Overwrite {
		fmt.Printf("saving tnsnames.ora file to %s...\n", conf.DownloadsPath)
		moveTNSNamesFile(
			filepath.Join(conf.InstallPath, "network", "admin", "tnsnames.ora"),
			filepath.Join(conf.DownloadsPath, "tnsnames.ora"),
		)
	} else if conf.Extant && !conf.Overwrite {
			fmt.Println("Skipping saving tnsnames.ora file as overwrite is not set.")
	}

	// Remove installation directory with safety checks
	if err := os.RemoveAll(conf.InstallPath); err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "removing installation directory")
	}

	// Reset the installation path in the config to the base directory of existing installation
	if err := conf.SetInstallPath(filepath.Dir(conf.InstallPath)); err != nil {
		return err
	}

	return nil
}

// InstallOracleInstantClient performs the installation and configuration of Oracle Instant Client
func Install(ctx context.Context, conf *config.InstallConfig, env *env.EnvVarManager) error {
	ctx = ensureContext(ctx)
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
	if err := downloadZip(ctx, conf.BaseURL+conf.PkgFile, pkgZipPath); err != nil {
		return err
	}

	// Download SDK files
	fmt.Printf("downloading SDK: %s...\n", sdkZipPath)
	if err := downloadZip(ctx, conf.BaseURL+conf.SdkFile, sdkZipPath); err != nil {
		return err
	}

	// Unzip package files
	fmt.Printf("extracting: %s to %s\n", pkgZipPath, conf.InstallPath)
	pkgDir, err := unZip(pkgZipPath, conf.InstallPath)
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "unzip package")
	}

	// Unzip SDK files
	fmt.Printf("extracting: %s to %s\n", sdkZipPath, filepath.Join(conf.InstallPath, pkgDir, "sdk"))
	sdkDir, err := unZip(sdkZipPath, conf.InstallPath)
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

	// Move tnsnames.ora file to TNS_ADMIN directory
	if conf.Extant && conf.Overwrite {
		fmt.Printf("moving tnsnames.ora from %s to %s\n", filepath.Join(conf.DownloadsPath, "tnsnames.ora"), tnsAdminPath)
		if err := moveTNSNamesFile(
			filepath.Join(conf.DownloadsPath, "tnsnames.ora"),
			filepath.Join(tnsAdminPath, "tnsnames.ora"),
		); err != nil {
			return err
		}
	} else if conf.Extant && !conf.Overwrite {
		fmt.Println("Skipping moving tnsnames.ora file as overwrite is not set.")
	}

	fmt.Println("\nOracle InstantClient installation and configuration completed successfully!")
	return nil
}

// downloadOracleInstantClient downloads the Oracle Instant Client zip file from the specified URL
func downloadZip(ctx context.Context, urlPath, downloadsPath string) error {
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

// unZip extracts the Oracle Instant Client zip file to the specified destination path
// and returns the directory name of the extracted files
func unZip(downloadsPath, installPath string) (string, error) {
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

// moveTNSNamesFile moves the tnsnames.ora file from the source to the destination path
// It checks if the source file exists, creates the destination directory if it doesn't exist,
// and then moves the file. If any error occurs, it returns an appropriate error.
// This function is used to ensure that the tnsnames.ora file is correctly placed in
// the TNS_ADMIN directory after installation.
// It is typically called after the installation process to ensure that the Oracle Net configuration
// file is in the correct location for the Oracle Instant Client to function properly.
func moveTNSNamesFile(from, to string) error {
	// Check if the source file exists
	if _, err := os.Stat(from); os.IsNotExist(err) {
		return errs.HandleError(err, errs.ErrorTypeInstall, "source tnsnames.ora file does not exist")
	}

	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(to), 0777); err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "creating destination directory for tnsnames.ora")
	}

	// Move the tnsnames.ora file to the destination
	if err := os.Rename(from, to); err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "moving tnsnames.ora file")
	}

	return nil
}