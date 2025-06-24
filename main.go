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
	"github.com/mghoff/oraicwinconfig/internal/oic"
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
	if err := conf.SetDownloadsPath(downloadsPath); errs.IsErrorType(err, errs.ErrorTypeValidation) {
		log.Fatal("error setting Downloads path: ", err)
	}

	fmt.Printf("The following files will be downloaded from '%s' to '%s':\n", conf.BaseURL, conf.DownloadsPath)
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
	if err := oic.Install(ctx, conf, env); err != nil {
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
	if ok := input.Confirmation("\nAccept the suggested install location?\n - " + conf.InstallPath + "\nSelect"); !ok {
		if change := input.Confirmation("Are you sure you wish to change the suggested install location?\nSelect"); change {
			newPath := input.InstallPath("Enter desired install path below... Note: this path must be an existing valid directory\n")
			if err := conf.SetInstallPath(newPath); errs.IsErrorType(err, errs.ErrorTypeValidation) {
				return errs.HandleError(err, errs.ErrorTypeValidation, "setting user-defined install path")
			}
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
	if ok, err := oic.Exists(ctx, conf, env); !ok {
		return nil
	} else if err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "checking for existing Oracle InstantClient installation")
	}
	
	fmt.Printf("\nThe path of the new installation will be set to the base directory of the existing installation; e.g. %s\n", filepath.Dir(conf.InstallPath))

	if !input.Confirmation("\nDo you wish to overwrite the existing installation?\nSelect") {
		fmt.Printf("\nExisting installation will be left in place.\nSetting install path to base directory of existing installation: %s\n", filepath.Dir(conf.InstallPath))
		if err := conf.SetInstallPath(filepath.Dir(conf.InstallPath)); err != nil {
			return errs.HandleError(err, errs.ErrorTypeValidation, "setting install path to existing installation base directory")
		}
		fmt.Printf("Installation path set to: %s\n", conf.InstallPath)
		return nil
	} else {
		fmt.Println("Uninstalling existing Oracle InstantClient installation...")
		if err := oic.Uninstall(ctx, conf, env); err != nil {
			return errs.HandleError(err, errs.ErrorTypeInstall, "uninstalling existing Oracle InstantClient")
		} else {
			fmt.Println("Existing Oracle InstantClient installation successfully removed.")
			fmt.Printf("Installation path reset to: %s\n", conf.InstallPath)
		}
		return nil
	}
}
