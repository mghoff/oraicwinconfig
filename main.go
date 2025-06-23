package main

import (
	"errors"
	"fmt"
	"log"
	"context"
	"time"
	"path/filepath"

	"github.com/mghoff/oraicwinconfig/internal/config"
	"github.com/mghoff/oraicwinconfig/internal/env"
	"github.com/mghoff/oraicwinconfig/internal/errs"
	"github.com/mghoff/oraicwinconfig/internal/input"
	"github.com/mghoff/oraicwinconfig/internal/install"
	"github.com/mghoff/oraicwinconfig/internal/version"
)

func main() {
	// Display  version information
	fmt.Println(version.Info())
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Initialize configuration with default values
	// and set the DownloadsPath to the user's Downloads directory
	conf := config.New()
	env := env.New()

	downloadsPath, err := env.FetchUserDownloadsPath()
	if err != nil {
		log.Fatal("error getting user Downloads directory: ", err)
	}
	conf.DownloadsPath = downloadsPath

	fmt.Printf("files will be downloaded from '%s' to '%s':\n", conf.BaseURL, conf.DownloadsPath)
	fmt.Printf("- %s\n- %s\n\n", conf.PkgFile, conf.SdkFile)

	// Handle existing installation
	if err := handleCurrentInstall(ctx, conf, env); err != nil {
		log.Fatal("error handling current installation: ", err)
	}

	// Handle installation path selection
	if err := handleInstallLocation(conf); err != nil {
		log.Fatal("error handling install location: ", err)
	}

	// Validate configuration before proceeding
	if err := conf.Validate(); err != nil {
		log.Fatal("invalid configuration: ", err)
	}

	// Perform installation
	if err := install.OracleInstantClient(ctx, conf, env); err != nil {
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
}

// handleInstallLocation handles the user interaction for user-defined installation path
func handleInstallLocation(conf *config.InstallConfig) error {
	if ok := input.Confirmation("\nAccept the following install location?\n - " + conf.InstallPath + "\nSelect"); !ok {
		if change := input.Confirmation("Are you sure you wish to change the default install location?\nSelect"); change {
			newPath := input.InstallPath("Enter desired install path...\n")
			conf.InstallPath = newPath
			fmt.Printf("install path set to: %s\n", conf.InstallPath)
		}

		if cont := input.Confirmation("Continue with install?"); !cont {
			return errs.HandleError(
				fmt.Errorf("installation aborted by user"),
				errs.ErrorTypeValidation,
				"user confirmation",
			)
		}
	}
	return nil
}

// handleCurrentInstall checks for an existing Oracle InstantClient installation
func handleCurrentInstall(ctx context.Context, conf *config.InstallConfig, env *env.EnvVarManager) error {
	if ok, err := install.Exists(ctx, conf, env); !ok {
		return nil
	} else if err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "checking for existing Oracle InstantClient installation")
	}
	
	fmt.Printf("\nThe path of the new installation will be set to the base directory of the existing installation: %s\n", filepath.Dir(conf.InstallPath)) 
	conf.InstallPath = filepath.Dir(conf.InstallPath)

	const promptAsk = "\nDo you wish to overwrite this current installation?"
	if !input.Confirmation(promptAsk+"\nSelect") {
		fmt.Println("Existing installation to be left in place. Resetting default install path to base directory of existing.")
		fmt.Printf("New install location set to base directory of existing: %s\n", conf.InstallPath)
		return nil
	} else {
		// Remove existing Oracle InstantClient at base install path
		if err := install.Remove(ctx, conf, env); err != nil {
			return errs.HandleError(err, errs.ErrorTypeInstall, "uninstalling existing Oracle InstantClient")
		} else {
			fmt.Println("Existing Oracle InstantClient installation removed successfully.")
		}
		return nil
	}
}
