package input

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mghoff/oraicwinconfig/internal/errs"
)

// getUserDestPath retrieves the user profile directory for a given endpoint
// and checks if the directory exists
func GetUserDownloadsPath() (string, error) {
	usrProfilePath, err := exec.Command("powershell", "$env:USERPROFILE").Output()
	if err != nil {
		return "", errs.HandleError(err, errs.ErrorTypeUserPath, "getting user profile directory")
	}

	usrDownloadsPath := filepath.Join(strings.TrimSuffix(string(usrProfilePath), "\r\n"), "Downloads")
	if _, err := os.Stat(usrDownloadsPath); os.IsNotExist(err) {
		return "", errs.HandleError(fmt.Errorf("directory does not exist: %s", usrDownloadsPath), errs.ErrorTypeUserPath, "checking user profile directory")
	} else if err != nil {
		return "", errs.HandleError(err, errs.ErrorTypeUserPath, "checking user profile directory")
	}

	return usrDownloadsPath, nil
}

// reqUserConfirmation prompts the user for a yes/no confirmation
// and returns true for 'y' and false for 'n'
func ReqUserConfirmation(label string) bool {
	choices := "y/n"
	r := bufio.NewReader(os.Stdin)
	attempts := 0
	maxAttempts := 3
	for attempts < maxAttempts {
		fmt.Fprintf(os.Stderr, "%s (%s): ", label, choices)
		s, err := r.ReadString('\n')
		if err != nil {
			log.Fatal("error reading input: ", err)
		}
		s = strings.ToLower(strings.TrimSpace(s))
		switch s {
		case "y":
			return true
		case "n":
			return false
		default:
			attempts++
			fmt.Printf("must enter 'y' or 'n' (%d attempts remaining)\n", maxAttempts-attempts)
		}
	}
	log.Fatal("maximum input attempts exceeded")
	return false
}

// reqUserInstallPath prompts the user for a valid installation path
// and validates that it is an existing directory
func ReqUserInstallPath(label string) string {
	r := bufio.NewReader(os.Stdin)
	attempts := 0
	maxAttempts := 3
	for attempts < maxAttempts {
		fmt.Fprintf(os.Stderr, "%s", label)
		path, err := r.ReadString('\n')
		if err != nil || path == "" {
			log.Fatal("error reading input: ", err)
		}
		path = strings.TrimSpace(path)
		if stat, err := os.Stat(path); err == nil && stat.IsDir() {
			return path
		} else {
			fmt.Printf("Invalid path provided: %s (error: %v)\n", path, err)
			fmt.Printf("Please provide a valid existing directory (%d attempts remaining)\n", maxAttempts-attempts)
		}
		attempts++
	}
	log.Fatal("maximum input attempts exceeded, installation aborted")
	return "" // This line will never be reached due to log.Fatal above
}
