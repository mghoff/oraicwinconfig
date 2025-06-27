package oic

import (
	"fmt"
	"os"
	"path/filepath"
	"context"
	"strings"
	"errors"

	"github.com/mghoff/oraicwinconfig/internal/config"
	"github.com/mghoff/oraicwinconfig/internal/env"
	"github.com/mghoff/oraicwinconfig/internal/errs"
	"github.com/mghoff/oraicwinconfig/internal/utils"
)

// InstallExists checks if Oracle InstantClient is already installed
func Exists(ctx context.Context, conf *config.InstallConfig, env *env.EnvVarManager) (bool, error) {
	ctx = utils.EnsureContext(ctx)
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
	ctx = utils.EnsureContext(ctx)
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
	ctx = utils.EnsureContext(ctx)
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
	if err := utils.DownloadZip(ctx, conf.BaseURL+conf.PkgFile, pkgZipPath); err != nil {
		return err
	}

	// Download SDK files
	fmt.Printf("downloading SDK: %s...\n", sdkZipPath)
	if err := utils.DownloadZip(ctx, conf.BaseURL+conf.SdkFile, sdkZipPath); err != nil {
		return err
	}

	// Unzip package files
	fmt.Printf("extracting: %s to %s\n", pkgZipPath, conf.InstallPath)
	pkgDir, err := utils.UnZip(pkgZipPath, conf.InstallPath)
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "unzip package")
	}

	// Unzip SDK files
	fmt.Printf("extracting: %s to %s\n", sdkZipPath, filepath.Join(conf.InstallPath, pkgDir, "sdk"))
	sdkDir, err := utils.UnZip(sdkZipPath, conf.InstallPath)
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
	if conf.Extant {
		fmt.Printf("moving tnsnames.ora from %s to %s\n", filepath.Join(conf.DownloadsPath, "tnsnames.ora"), tnsAdminPath)
		if err := utils.MigrateFile(
			filepath.Join(conf.DownloadsPath, "tnsnames.ora"),
			filepath.Join(tnsAdminPath, "tnsnames.ora"),
			false,
		); err != nil {
			return err
		}
	}

	fmt.Println("\nOracle InstantClient installation and configuration completed successfully!")
	return nil
}
