package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Configuration constants
const (
	OraicDstPath = "C:/OraClient"
	OraicPkgName = "instantclient-basiclite-windows.zip"
	OraicSdkName = "instantclient-sdk-windows.zip"
	OraicBaseUrl = "https://download.oracle.com/otn_software/nt/instantclient/"
)

// Error type definition
type ErrorType int

// Error type definitions
const (
	ErrorTypeDownload ErrorType = iota
	ErrorTypeInstall
	ErrorTypeEnvironment
	ErrorTypeValidation
	ErrorTypeUserPath
)

// InstallError represents a contextual error during installation
type InstallError struct {
	Type      ErrorType
	Operation string
	Err       error
}

func (e *InstallError) Error() string {
	return fmt.Sprintf("%s: %v", e.Operation, e.Err)
}

func (e *InstallError) Unwrap() error {
	return e.Err
}

// handleError creates a new InstallError with context
func handleError(err error, errorType ErrorType, operation string) error {
	if err != nil {
		return &InstallError{
			Type:      errorType,
			Operation: operation,
			Err:       err,
		}
	}
	return nil
}

func main() {
	userDownloads, err := getUserDestPath("Downloads")
	if err != nil {
		log.Fatal("Error getting user Downloads directory: ", err)
	}
	fmt.Println("The following .zip files will be downloaded from", "'"+OraicBaseUrl+"'", "to", "'"+userDownloads+"'")
	fmt.Println("-", OraicPkgName)
	fmt.Println("-", OraicSdkName)

	okDefaultInstall := reqUserConfirmation("Accept the default install location?\n - " + OraicDstPath + "\nSelect")
	if !okDefaultInstall {
		changeDefaultInstall := reqUserConfirmation("Change the default install location from '" + OraicDstPath + "'? Select")
		if !changeDefaultInstall {
			continueInstall := reqUserConfirmation("Continue with install? Select")
			if !continueInstall {
				handleError(fmt.Errorf("installation aborted by user"), ErrorTypeValidation, "user confirmation")
				log.Fatal("installation aborted by user.")
			}
		} else {
			OraicDstPath := reqUserInstallPath("Enter desired install path...\n")
			continueInstall := reqUserConfirmation("Continue with install to '" + OraicDstPath + "'? Select")
			if !continueInstall {
				handleError(fmt.Errorf("installation aborted by user"), ErrorTypeValidation, "user confirmation")
				log.Fatal("installation aborted by user.")
			}
		}
	}
	InstallOracleInstantClient(userDownloads, OraicDstPath)
}

func getUserDestPath(dirEndpoint string) (string, error) {
	usrDir, err := exec.Command("powershell", "$env:USERPROFILE").Output()
	if err != nil {
		return "", handleError(err, ErrorTypeUserPath, "getting user profile directory")
	}

	dir := filepath.Join(strings.TrimSuffix(string(usrDir), "\r\n"), dirEndpoint)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return "", handleError(fmt.Errorf("directory does not exist: %s", dir), ErrorTypeUserPath, "checking user profile directory")
	} else if err != nil {
		log.Fatal("User profile directory does not exist: ", dir)
	}

	return dir, nil
}

func reqUserConfirmation(label string) bool {
	choices := "y/n"
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprintf(os.Stderr, "%s (%s): ", label, choices)
		s, _ := r.ReadString('\n')
		s = strings.ToLower(strings.TrimSpace(s))
		switch s {
		case "y":
			return true
		case "n":
			return false
		default:
			fmt.Println("Must enter 'y' or 'n'")
		}
	}
}

func reqUserInstallPath(label string) string {
	r := bufio.NewReader(os.Stdin)
	var path string
	for {
		fmt.Fprintf(os.Stderr, "%s", label)
		path, _ = r.ReadString('\n')
		path = strings.TrimSpace(path)
		if stat, err := os.Stat(path); stat.IsDir() && err == nil {
			return path
		} else {
			log.Fatal("Invalid path provided: ", path)
			fmt.Println("Please provide a valid directory path.")
			continue
		}
	}
}

func downloadOracleInstantClient(url, dest string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return handleError(err, ErrorTypeDownload, "downloading from URL")
	}
	if resp.StatusCode != http.StatusOK {
		return handleError(fmt.Errorf("HTTP status %s", resp.Status), ErrorTypeDownload, "checking response status")
	}
	defer resp.Body.Close()

	// Create file
	out, err := os.Create(dest)
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

func unzipOracleInstantClient(zipPath, destPath string) string {
	// Create base folder
	err := os.MkdirAll(destPath, 0777)
	if err != nil {
		handleError(err, ErrorTypeInstall, "creating base directory")
		log.Fatalf("error creating base directory: %s", err)
	}

	// Open a zip archive for reading.
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		handleError(err, ErrorTypeInstall, "opening zip archive")
		log.Fatal("error opening zip archive: ", err)
	}
	defer r.Close()

	// Iterate through the files in the archive, printing some of their contents.
	var outDestPath string
	for k, f := range r.File {
		re := regexp.MustCompilePOSIX(`^(instantclient_){1}([0-9]{1,2})_([0-9]{1,2})\/$`)
		matched := re.Match([]byte(f.Name))
		if matched {
			outDestPath = f.Name
		}

		// If current 'f' ends in '/', then it's a dir, and that dir needs created.
		var outPath string
		if f.Name[len(f.Name)-1:] == "/" {
			outPath = filepath.Join(destPath, f.Name)
			fmt.Println("DIR OUTPATH: " + outPath)
			err := os.MkdirAll(outPath, 0777)
			if err != nil {
				log.Fatalf("Impossible to MkdirAll: %s", err)
			}
			continue
		} else {
			re := regexp.MustCompile(`^(.*[\\\/])[^\\\/]*$`)
			outPath = filepath.Join(destPath, re.ReplaceAllString(f.Name, "$1"))
			err := os.MkdirAll(outPath, 0777)
			if err != nil {
				log.Fatalf("Impossible to MkdirAll: %s", err)
			}

			fmt.Printf(" - Unzipping: %s\n", f.Name)
			rc, err := f.Open()
			if err != nil {
				log.Fatalf("Cannot open file n%d in zip: %s", k, err)
			}
			unzippedFile, err := os.Create(filepath.Join(destPath, f.Name))
			if err != nil {
				log.Fatalf("Impossible to unzip: %s", err)
			}
			_, err = io.Copy(unzippedFile, rc)
			if err != nil {
				log.Fatalf("Cannot copy file n%d: %s", k, err)
			}
		}
	}
	return outDestPath
}

func setEnvironmentVariable(usrEnvVar, envVarPath string) {
	// Check existing environment variables
	psGetEnvCmd := "[System.Environment]::GetEnvironmentVariable('" + usrEnvVar + "', 'User')"
	currUsrVar, err := exec.Command("powershell", psGetEnvCmd).Output()
	if err != nil {
		log.Fatal("Error getting current User Environment Variable: ", err)
	}
	currUsrVarStr := strings.TrimSuffix(string(currUsrVar), "\r\n")

	var needToAdd bool
	switch usrEnvVar {
	case "OCI_LIB64", "TNS_ADMIN":
		if currUsrVarStr == envVarPath {
			fmt.Println(usrEnvVar + " already exists in User Environment Variable list. No changes made.")
			needToAdd = false
		} else {
			fmt.Println("Adding " + usrEnvVar + " to User Environment Variable list.")
			needToAdd = true
		}
	case "PATH":
		if strings.Contains(currUsrVarStr, envVarPath) {
			fmt.Println(envVarPath + " already exists in User PATH Variable. No changes made.")
			needToAdd = false
		} else {
			fmt.Println("Adding " + envVarPath + " to User PATH Environment Variable.")
			needToAdd = true
		}
	default:
		log.Fatal("Error: no known handle for " + usrEnvVar)
		needToAdd = false
	}

	// If needed, add new Environment Variable or new path to the Path environment variable
	if needToAdd {
		var envVarDir string
		switch usrEnvVar {
		case "OCI_LIB64", "TNS_ADMIN":
			envVarDir = envVarPath
			fmt.Println("\t" + usrEnvVar + "=" + envVarDir)
		case "PATH":
			psGetEnvCmd := "[System.Environment]::GetEnvironmentVariable('" + usrEnvVar + "', 'User')"
			usrPath, err := exec.Command("powershell", psGetEnvCmd).Output()
			if err != nil {
				log.Fatal("Error getting current User PATH Environment Variable: ", err)
			}
			if string(usrPath)[len(string(usrPath))-1:] == ";" {
				envVarDir = strings.TrimSuffix(string(usrPath), "\r\n") + envVarPath + ";"
			} else {
				envVarDir = strings.TrimSuffix(string(usrPath), "\r\n") + ";" + envVarPath + ";"
			}
			fmt.Println("\t" + usrEnvVar + "=" + envVarDir)
		default:
			log.Fatal("Error: no known handle for " + usrEnvVar)
			return
		}

		psSetEnvCmd := "[Environment]::SetEnvironmentVariable('" + usrEnvVar + "', '" + envVarDir + "' , 'User')"
		_, err := exec.Command("powershell", psSetEnvCmd).Output()
		if err != nil {
			log.Fatal("Error setting User Environment Variable: ", err)
		}
		_, exists := os.LookupEnv(usrEnvVar)
		if exists {
			fmt.Println(usrEnvVar + " successfully added to User Environment Variable list.\n")
		}
	}
}

func InstallOracleInstantClient(downloadPath, installPath string) {
	// Set paths for filename download locations
	oraicPkgZipLoc := filepath.Join(downloadPath, OraicPkgName)
	oraicSdkZipLoc := filepath.Join(downloadPath, OraicSdkName)
	// Set paths for Oracle Instant Client PKG & SDK URLs
	oraicPkgUrl := OraicBaseUrl + OraicPkgName
	oraicSdkUrl := OraicBaseUrl + OraicSdkName

	// Download LATEST Oracle Instant Client PKG & SDK
	fmt.Println("Downloading PKG ZIP: " + oraicPkgZipLoc + "...")
	downloadOracleInstantClient(oraicPkgUrl, oraicPkgZipLoc)
	fmt.Println("Downloaded SDK ZIP: " + oraicSdkZipLoc + "...")
	downloadOracleInstantClient(oraicSdkUrl, oraicSdkZipLoc)

	// Unzip Oracle Instant Client PKG & SDK (NOTE: '*_Tld' short for 'Top Level Directory')
	fmt.Println("Unzipping: " + oraicPkgZipLoc + " to " + installPath)
	oraicPkgTld := unzipOracleInstantClient(oraicPkgZipLoc, installPath)
	fmt.Println("Unzipping:", oraicSdkZipLoc)
	oraicSkdTld := unzipOracleInstantClient(oraicSdkZipLoc, installPath)

	// Verify version match
	if oraicPkgTld == oraicSkdTld {
		fmt.Println("Oracle Instant Client PKG and SDK versions match. Continuing...")
	} else {
		fmt.Println("Oracle Instant Client PKG and SDK versions DO NOT match. Exiting...")
		fmt.Println("    ORAIC_PKG_TLD: " + oraicPkgTld)
		fmt.Println("    ORAIC_SDK_TLD: " + oraicSkdTld)
		os.Exit(1)
	}

	evpOCILib64 := filepath.Join(installPath, oraicPkgTld)
	fmt.Println("OCI_LIB64_ENVARPATH: " + evpOCILib64)
	setEnvironmentVariable("OCI_LIB64", evpOCILib64)
	setEnvironmentVariable("PATH", evpOCILib64)

	evpTNSAdmin := filepath.Join(evpOCILib64, "network", "admin")
	fmt.Println("TNS_ADMIN_ENVARPATH: " + evpTNSAdmin)
	setEnvironmentVariable("TNS_ADMIN", evpTNSAdmin)

	// Wait for user input
	fmt.Println("Oracle InstantClient Installation Complete!\nPress any key to escape...")
	fmt.Scanln()
}
