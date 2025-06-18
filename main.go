package main

import (
	"errors"
	"fmt"
	"log"
	"context"
	"time"

	"github.com/mghoff/oraicwinconfig/internal/config"
	"github.com/mghoff/oraicwinconfig/internal/env"
	"github.com/mghoff/oraicwinconfig/internal/errs"
	"github.com/mghoff/oraicwinconfig/internal/input"
	"github.com/mghoff/oraicwinconfig/internal/install"
	"github.com/mghoff/oraicwinconfig/internal/version"
)

func main() {
	// Display  version information
	fmt.Println(version.FetchVersionInfo())
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Initialize configuration with default values
	// and set the DownloadsPath to the user's Downloads directory
	config := config.New()
	env := env.New()

	downloadsPath, err := env.FetchUserDownloadsPath()
	if err != nil {
		log.Fatal("error getting user Downloads directory: ", err)
	}
	config.DownloadsPath = downloadsPath

	fmt.Printf("files will be downloaded from '%s' to '%s':\n", config.BaseURL, config.DownloadsPath)
	fmt.Printf("- %s\n- %s\n\n", config.PkgFile, config.SdkFile)

	// Handle installation path selection
	if err := handleInstallLocation(config); err != nil {
		log.Fatal("error handling install location: ", err)
	}

	// Validate configuration before proceeding
	if err := config.Validate(); err != nil {
		log.Fatal("invalid configuration: ", err)
	}

	// Handle existing installation
	if err := handleCurrentInstall(config, env); err != nil {
		log.Fatal("error handling current installation: ", err)
	}

	// Perform installation
	if err := install.InstallOracleInstantClient(ctx, config, env); err != nil {
		var installErr *errs.InstallError
		if errors.As(err, &installErr) {
			switch installErr.Type {
			case errs.ErrorTypeDownload:
				log.Fatal("download failed: ", err)
			case errs.ErrorTypeInstall:
				log.Fatal("installation failed: ", err)
			case errs.ErrorTypeEnvironment:
				log.Fatal("environment setup failed: ", err)
			default:
				log.Fatal("unknown error: ", err)
			}
		}
		log.Fatal("installation failed: ", err)
	}

	fmt.Println("installation completed successfully")
}

// handleInstallLocation handles the user interaction for user-defined installation path
func handleInstallLocation(config *config.InstallConfig) error {
	if ok := input.ReqUserConfirmation("Accept the default install location?\n - " + config.InstallPath + "\nSelect"); !ok {
		if change := input.ReqUserConfirmation("Are you sure you wish to change the default install location?\nSelect"); change {
			newPath := input.ReqUserInstallPath("Enter desired install path...\n")
			config.InstallPath = newPath
			fmt.Printf("install path set to: %s\n", config.InstallPath)
		}

		if cont := input.ReqUserConfirmation("Continue with install?"); !cont {
			return errs.HandleError(
				fmt.Errorf("installation aborted by user"),
				errs.ErrorTypeValidation,
				"user confirmation",
			)
		}
	}
	return nil
}

func handleCurrentInstall(config *config.InstallConfig, env *env.EnvVarManager) error {
	// Check for existing environment variables that indicate an Oracle InstantClient installation
	// Check if OCI_LIB64 environment variable exists
	ociEnvVarExists, err := env.CheckEnvVarExists("OCI_LIB64")
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeEnvironment, "checking OCI_LIB64 environment variable")
	}
	fmt.Printf("OCI_LIB64 environment variable exists: %t\n", ociEnvVarExists)

	// Check if TNS_ADMIN environment variable exists
	tnsEnvVarExists, err := env.CheckEnvVarExists("TNS_ADMIN")
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeEnvironment, "checking TNS_ADMIN environment variable")
	}
	fmt.Printf("TNS_ADMIN environment variable exists: %t\n", tnsEnvVarExists)

	// If neither OCI_LIB64 nor TNS_ADMIN exists, no existing installation is found
	if !ociEnvVarExists && !tnsEnvVarExists {
		fmt.Println("No existing Oracle InstantClient installation found.")
		return nil
	}
	
	// If either OCI_LIB64 or TNS_ADMIN exist, prompt user for confirmation to uninstall	
	const foundInstallNotice = "An existing installation of Oracle InstantClient has been found..."
	fmt.Println(foundInstallNotice)
	const uninstallPrompt = "Do you wish to uninstall the existing Oracle InstantClient installation?\nSelect"
	if ok := input.ReqUserConfirmation(uninstallPrompt); ok {
		if err := install.UninstallOracleInstantClient(context.Background(), config, env); err != nil {
			return errs.HandleError(err, errs.ErrorTypeInstall, "uninstalling existing Oracle Instant Client")
		}
		fmt.Println("Existing Oracle InstantClient uninstalled successfully.")
	} else {
		fmt.Println("Existing Oracle InstantClient installation will remain in place.")
		fmt.Println("The following environment variables will be set or overwritten:")
		if ociEnvVarExists && tnsEnvVarExists {
			fmt.Println("\tOCI_LIB64")
			fmt.Println("\tTNS_ADMIN")
		} else if ociEnvVarExists {
			fmt.Println("\tOCI_LIB64 environment variable will be overwritten.")
			fmt.Println("\tTNS_ADMIN environment variable will be set.")
		} else if tnsEnvVarExists {
			fmt.Println("\tOCI_LIB64 environment variable will be set.")
			fmt.Println("\tTNS_ADMIN environment variable will be overwritten.")
		} else {
			fmt.Println("\tOCI_LIB64 and TNS_ADMIN environment variables will be set.")
		}
		
		if !input.ReqUserConfirmation("Do you wish to continue with the new installation?\nSelect") {
			return errs.HandleError(
				fmt.Errorf("installation aborted by user"),
				errs.ErrorTypeValidation,
				"user confirmation",
			)
		}
		fmt.Println("Continuing with the new installation...")
		fmt.Println("Note: The existing installation will not be removed, and the new installation will overwrite the existing environment variables.")
		fmt.Println("You may need to manually remove the existing installation if it causes conflicts.")
		fmt.Println("Proceeding with the new installation...")
		// Remove OCI_LIB64 from PATH if it exists
		ociLib64Path, err := env.GetEnvVar("OCI_LIB64")
		if err != nil && !errs.IsErrorType(err, errs.ErrorTypeEnvVarNotFound) {
			return errs.HandleError(err, errs.ErrorTypeEnvironment, "getting OCI_LIB64 environment variable")
		}
		if ociLib64Path != "" {
			if err := env.RemoveFromPath(ociLib64Path); err != nil {
				return errs.HandleError(err, errs.ErrorTypeEnvironment, "removing OCI_LIB64 from PATH")
			}
			fmt.Printf("Removed OCI_LIB64 path '%s' from PATH environment variable.\n", ociLib64Path)
		} else {
			fmt.Println("OCI_LIB64 environment variable not found in PATH, no action taken.")
		}
	}
	return nil
}
