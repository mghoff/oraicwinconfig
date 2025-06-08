package main

import (
	"errors"
	"fmt"
	"log"
	"context"
	"time"

	"github.com/mghoff/oraicwinconfig/internal/config"
	"github.com/mghoff/oraicwinconfig/internal/errs"
	"github.com/mghoff/oraicwinconfig/internal/input"
	"github.com/mghoff/oraicwinconfig/internal/install"
	"github.com/mghoff/oraicwinconfig/internal/version"
)

func main() {
	// Display  version information
	fmt.Println(version.GetVersionInfo())
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Initialize configuration with default values
	// and set the DownloadsPath to the user's Downloads directory
	config := config.NewDefaultConfig()

	downloadsPath, err := input.GetUserDownloadsPath()
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

	// Perform installation
	if err := install.InstallOracleInstantClient(ctx, config); err != nil {
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
