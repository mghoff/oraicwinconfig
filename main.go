package main

import (
	"errors"
	"fmt"
	"log"
)

func main() {
	config := NewDefaultConfig()

	downloads, err := getUserDestPath("Downloads")
	if err != nil {
		log.Fatal("error getting user Downloads directory: ", err)
	}
	config.DownloadsPath = downloads

	fmt.Printf("files will be downloaded from '%s' to '%s':\n", config.BaseURL, config.DownloadsPath)
	fmt.Printf("- %s\n- %s\n", config.PkgFile, config.SdkFile)

	// Handle installation path selection
	if err := handleInstallLocation(config); err != nil {
		log.Fatal("error handling install location: ", err)
	}

	// Perform installation
	if err := InstallOracleInstantClient(config); err != nil {
		var installErr *InstallError
		if errors.As(err, &installErr) {
			switch installErr.Type {
			case ErrorTypeDownload:
				log.Fatal("download failed: ", err)
			case ErrorTypeInstall:
				log.Fatal("installation failed: ", err)
			case ErrorTypeEnvironment:
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
func handleInstallLocation(config *InstallConfig) error {
	if ok := reqUserConfirmation("Accept the default install location?\n - " + config.InstallPath + "\nSelect"); !ok {
		if change := reqUserConfirmation("Are you sure you wish to change the default install location?\nSelect"); change {
			newPath := reqUserInstallPath("Enter desired install path...\n")
			config.InstallPath = newPath
			fmt.Printf("install path set to: %s\n", config.InstallPath)
		}

		if cont := reqUserConfirmation("Continue with install?"); !cont {
			return handleError(
				fmt.Errorf("installation aborted by user"),
				ErrorTypeValidation,
				"user confirmation",
			)
		}
	}
	return nil
}
