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

const (
	ORAIC_DST_PATH = "C:/OraClient"
	ORAIC_PKG_NAME = "instantclient-basiclite-windows.zip"
	ORAIC_SDK_NAME = "instantclient-sdk-windows.zip"
	ORAIC_BASE_URL = "https://download.oracle.com/otn_software/nt/instantclient/"
)

func main() {
	USER_DOWNLOADS := getUserDestPath("Downloads")
	fmt.Println("The following .zip files will be downloaded from", "'"+ORAIC_BASE_URL+"'", "to", "'"+USER_DOWNLOADS+"'")
	fmt.Println("-", ORAIC_PKG_NAME)
	fmt.Println("-", ORAIC_SDK_NAME)

	OK_DEFAULT_INSTALL := askInstallOK("Accept the default install location?\n - " + ORAIC_DST_PATH + "\nSelect")
	if !OK_DEFAULT_INSTALL {
		CHANGE_DEFAULT_INSTALL := askChangeDefaultInstall("Change the default install location from '" + ORAIC_DST_PATH + "'? Select")
		if !CHANGE_DEFAULT_INSTALL {
			CONT_DEFAULT_INSTALL := askChangeDefaultInstall("Continue with install? Select")
			if !CONT_DEFAULT_INSTALL {
				os.Exit(1)
			}
		} else {
			ORAIC_DST_PATH := askNewInstallPath("Enter desired install path...\n")
			OK_INSTALL := askInstallOK("Continue with install to '" + ORAIC_DST_PATH + "'? Select")
			if !OK_INSTALL {
				os.Exit(1)
			}
		}
	}
	InstallOracleInstantClient(USER_DOWNLOADS, ORAIC_DST_PATH)
}

func getUserDestPath(dirEndpoint string) string {
	usrDir, err := exec.Command("powershell", "$env:USERPROFILE").Output()
	if err != nil {
		fmt.Println(err.Error())
	}
	dir := filepath.Join(strings.TrimSuffix(string(usrDir), "\r\n"), dirEndpoint)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Println(dir, "does not exist.")
		os.Exit(1)
	}
	return dir
}

func askInstallOK(label string) bool {
	choices := "y/n"
	r := bufio.NewReader(os.Stdin)
	var s string
	for {
		fmt.Fprintf(os.Stderr, "%s (%s): ", label, choices)
		s, _ = r.ReadString('\n')
		s = strings.ToLower(strings.TrimSpace(s))
		if s != "y" && s != "n" {
			panic("Must enter 'y' or 'n'.")
		} else if s == "y" {
			return true
		} else {
			return false
		}
	}
}

func askChangeDefaultInstall(label string) bool {
	choices := "y/n"
	r := bufio.NewReader(os.Stdin)
	var s string
	for {
		fmt.Fprintf(os.Stderr, "%s (%s): ", label, choices)
		s, _ = r.ReadString('\n')
		s = strings.ToLower(strings.TrimSpace(s))
		if s != "y" && s != "n" {
			panic("Must enter 'y' or 'n'.")
		} else if s == "y" {
			return true
		} else {
			return false
		}
	}
}

func askNewInstallPath(label string) string {
	r := bufio.NewReader(os.Stdin)
	var path string
	for {
		fmt.Fprintf(os.Stderr, "%s", label)
		path, _ = r.ReadString('\n')
		path = strings.TrimSpace(path)
		if stat, err := os.Stat(path); err == nil && stat.IsDir() {
			return path
		} else {
			panic("Path provided either does not exist or is not a directory!")
		}
	}
}

func downloadOracleInstantClient(url, dest string) (err error) {
	// Create file
	out, err := os.Create(dest)
	if err != nil {
		panic(err)
	}
	defer out.Close()
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error while downloading: %s", resp.Status)
	}
	defer resp.Body.Close()
	// Write response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		panic(err)
	}
	return nil
}

func unzipOracleInstantClient(zipPath, destPath string) string {
	// Create base folder
	err := os.MkdirAll(destPath, 0777)
	if err != nil {
		log.Fatalf("Impossible to make base dir: %s", err)
	}

	// Open a zip archive for reading.
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	// Iterate through the files in the archive, printing some of their contents.
	var DEST_PATH string
	for k, f := range r.File {
		re := regexp.MustCompilePOSIX(`^(instantclient_){1}([0-9]{1,2})_([0-9]{1,2})\/$`)
		matched := re.Match([]byte(f.Name))
		if matched {
			DEST_PATH = f.Name
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
	return DEST_PATH
}

func setEnvironmentVariable(usrEnvVar, envVarPath string) {
	// Check existing environment variables
	psGetEnvCmd := "[System.Environment]::GetEnvironmentVariable('" + usrEnvVar + "', 'User')"
	currUsrVar, err := exec.Command("powershell", psGetEnvCmd).Output()
	if err != nil {
		panic(err)
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
		fmt.Println("Error: no known handle for " + usrEnvVar)
		os.Exit(1)
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
				panic(err)
			}
			if string(usrPath)[len(string(usrPath))-1:] == ";" {
				envVarDir = strings.TrimSuffix(string(usrPath), "\r\n") + envVarPath + ";"
			} else {
				envVarDir = strings.TrimSuffix(string(usrPath), "\r\n") + ";" + envVarPath + ";"
			}
			fmt.Println("\t" + usrEnvVar + "=" + envVarDir)
		default:
			fmt.Println("Error: no known handle for " + usrEnvVar)
			os.Exit(1)
		}

		psSetEnvCmd := "[Environment]::SetEnvironmentVariable('" + usrEnvVar + "', '" + envVarDir + "' , 'User')"
		_, err := exec.Command("powershell", psSetEnvCmd).Output()
		if err != nil {
			panic(err)
		}
		_, exists := os.LookupEnv(usrEnvVar)
		if exists {
			fmt.Println(usrEnvVar + " successfully added to User Environment Variable list.\n")
		}
	}
}

func InstallOracleInstantClient(downloadPath, installPath string) {
	// Set paths for filename download locations
	ORAIC_PKG_ZIP_LOCATION := filepath.Join(downloadPath, ORAIC_PKG_NAME)
	ORAIC_SDK_ZIP_LOCATION := filepath.Join(downloadPath, ORAIC_SDK_NAME)
	ORAIC_PKG_URL := ORAIC_BASE_URL + ORAIC_PKG_NAME
	ORAIC_SDK_URL := ORAIC_BASE_URL + ORAIC_SDK_NAME

	// Download LATEST Oracle Instant Client PKG & SDK
	fmt.Println("Downloading PKG ZIP: " + ORAIC_PKG_ZIP_LOCATION + "...")
	downloadOracleInstantClient(ORAIC_PKG_URL, ORAIC_PKG_ZIP_LOCATION)
	fmt.Println("Downloaded SDK ZIP: " + ORAIC_SDK_ZIP_LOCATION + "...")
	downloadOracleInstantClient(ORAIC_SDK_URL, ORAIC_SDK_ZIP_LOCATION)

	// Unzip Oracle Instant Client PKG & SDK (NOTE: '*_TLD' short for 'Top Level Directory')
	fmt.Println("Unzipping: " + ORAIC_PKG_ZIP_LOCATION + " to " + installPath)
	ORAIC_PKG_TLD := unzipOracleInstantClient(ORAIC_PKG_ZIP_LOCATION, installPath)
	fmt.Println("Unzipping:", ORAIC_SDK_ZIP_LOCATION)
	ORAIC_SDK_TLD := unzipOracleInstantClient(ORAIC_SDK_ZIP_LOCATION, installPath)

	// Verify version match
	if ORAIC_PKG_TLD == ORAIC_SDK_TLD {
		fmt.Println("Oracle Instant Client PKG and SDK versions match. Continuing...")
	} else {
		fmt.Println("Oracle Instant Client PKG and SDK versions DO NOT match. Exiting...")
		fmt.Println("    ORAIC_PKG_TLD: " + ORAIC_PKG_TLD)
		fmt.Println("    ORAIC_SDK_TLD: " + ORAIC_SDK_TLD)
		os.Exit(1)
	}

	ENVARPATH_OCI_LIB64 := filepath.Join(installPath, ORAIC_PKG_TLD)
	fmt.Println("OCI_LIB64_ENVARPATH: " + ENVARPATH_OCI_LIB64)
	setEnvironmentVariable("OCI_LIB64", ENVARPATH_OCI_LIB64)
	setEnvironmentVariable("PATH", ENVARPATH_OCI_LIB64)

	ENVARPATH_TNS_ADMIN := filepath.Join(ENVARPATH_OCI_LIB64, "network", "admin")
	fmt.Println("TNS_ADMIN_ENVARPATH: " + ENVARPATH_TNS_ADMIN)
	setEnvironmentVariable("TNS_ADMIN", ENVARPATH_TNS_ADMIN)

	// Wait for user input
	fmt.Println("Oracle InstantClient Installation Complete!\nPress any key to escape...")
	fmt.Scanln()
}
