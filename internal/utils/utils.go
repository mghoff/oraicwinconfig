package utils

import (
	"archive/zip"
	"context"
	"fmt"
	"path/filepath"
	"io"
	"net/http"
	"os"
	"regexp"

	"github.com/mghoff/oraicwinconfig/internal/errs"
)

// ensureContext returns context.Background() if ctx is nil, otherwise returns ctx.
func EnsureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// downloadZip downloads the Oracle Instant Client zip file from the specified URL
func DownloadZip(ctx context.Context, urlPath, downloadsPath string) error {
	ctx = EnsureContext(ctx)
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return errs.HandleError(err, errs.ErrorTypeDownload, "context cancellation")
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeDownload, "creating HTTP request")
	}

	// Get zip archive from URL
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeDownload, "downloading from URL")
	}
	if resp.StatusCode != http.StatusOK {
		return errs.HandleError(fmt.Errorf("HTTP status %s", resp.Status), errs.ErrorTypeDownload, "checking response status")
	}
	defer resp.Body.Close()

	// Create file
	out, err := os.Create(downloadsPath)
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeDownload, "creating download file")
	}
	defer out.Close()

	// Write response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeDownload, "writing download to file")
	}
	return nil
}

// unZip extracts the Oracle Instant Client zip file to the specified destination path
// and returns the directory name of the extracted files
func UnZip(downloadsPath, installPath string) (string, error) {
	// Create base install directory
	if err := os.MkdirAll(installPath, 0777); err != nil {
		return "", errs.HandleError(err, errs.ErrorTypeInstall, "creating base installation directory")
	}

	// Open a zip archive for reading.zip files from the Downloads directory
	r, err := zip.OpenReader(downloadsPath)
	if err != nil {
		return "", errs.HandleError(err, errs.ErrorTypeInstall, "opening zip archive")
	}
	defer r.Close()

	// Iterate through the files in the zip archive,
	// and extract contents into the Installation directory
	var outPath string
	for k, f := range r.File {
		re := regexp.MustCompilePOSIX(`^(instantclient_){1}([0-9]{1,2})_([0-9]{1,2})\/$`)
		if re.Match([]byte(f.Name)) {
			outPath = f.Name
		}
		if err := extractFile(f, installPath); err != nil {
			return "", errs.HandleError(err, errs.ErrorTypeInstall, fmt.Sprintf("extracting file %d", k))
		}
	}

	if outPath == "" {
		return "", errs.HandleError(
			fmt.Errorf("no valid instant client directory found in zip"),
			errs.ErrorTypeInstall,
			"validating zip contents",
		)
	}

	return filepath.Clean(outPath), nil
}

// Helper function to extract a single file from zip archive to specified install path
// It creates necessary directories and handles file creation
func extractFile(f *zip.File, installPath string) error {
	outName := filepath.Join(installPath, f.Name)

	if f.FileInfo().IsDir() {
		return os.MkdirAll(outName, 0777)
	}

	if err := os.MkdirAll(filepath.Dir(outName), 0777); err != nil {
		return fmt.Errorf("creating directories: %w", err)
	}

	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("opening zip file: %w", err)
	}
	defer rc.Close()

	out, err := os.Create(outName)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	if err != nil {
		return fmt.Errorf("writing file contents: %w", err)
	}

	return nil
}

// migrate (move or copy file from source to destination)
func MigrateFile(from, to string, copy bool) error {
	if copy {
		if err := copyFile(from, to); err != nil {
			return err
		}
	} else {
		if err := moveFile(from, to); err != nil {
			return err
		}
	}
	return nil
}

// move file from source to destination
func moveFile(src, dst string) error {
	// Check if the source file exists
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return errs.HandleError(err, errs.ErrorTypeInstall, "source tnsnames.ora file does not exist")
	}

	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dst), 0777); err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "creating destination directory for tnsnames.ora")
	}

	// Move the tnsnames.ora file to the destination
	if err := os.Rename(src, dst); err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "moving tnsnames.ora file")
	}

	return nil
}

// copy file from source to destination
func copyFile(src, dst string) error {
	// open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "opening tnsnames.ora file")
	}
	defer srcFile.Close()

	// create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "creating tnsnames.ora copy file")
	}
	defer dstFile.Close()

	// copy contents
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "copying tnsnames.ora contents")
	}
	// sync
	err = dstFile.Sync()
	if  err != nil {
		return errs.HandleError(err, errs.ErrorTypeInstall, "syncing destination tnsnames.ora file")
	} 

	return nil
}