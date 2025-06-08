package main

import (
	"errors"
	"fmt"
	"log"
	"context"
	"time"

	"github.com/mghoff/oraicwinconfig/internal"
)

func main() {
	// Display  version information
	fmt.Println(internal.GetVersionInfo())
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Initialize configuration with default values
	// and set the DownloadsPath to the user's Downloads directory
	config := internal.NewDefaultConfig()

	downloadsPath, err := internal.GetUserDownloadsPath()
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
	if err := internal.InstallOracleInstantClient(ctx, config); err != nil {
		var installErr *internal.InstallError
		if errors.As(err, &installErr) {
			switch installErr.Type {
			case internal.ErrorTypeDownload:
				log.Fatal("download failed: ", err)
			case internal.ErrorTypeInstall:
				log.Fatal("installation failed: ", err)
			case internal.ErrorTypeEnvironment:
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
func handleInstallLocation(config *internal.InstallConfig) error {
	if ok := internal.ReqUserConfirmation("Accept the default install location?\n - " + config.InstallPath + "\nSelect"); !ok {
		if change := internal.ReqUserConfirmation("Are you sure you wish to change the default install location?\nSelect"); change {
			newPath := internal.ReqUserInstallPath("Enter desired install path...\n")
			config.InstallPath = newPath
			fmt.Printf("install path set to: %s\n", config.InstallPath)
		}

		if cont := internal.ReqUserConfirmation("Continue with install?"); !cont {
			return internal.HandleError(
				fmt.Errorf("installation aborted by user"),
				internal.ErrorTypeValidation,
				"user confirmation",
			)
		}
	}
	return nil
}
