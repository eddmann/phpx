package php

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/schollz/progressbar/v3"
)

const (
	CommonBaseURL = "https://dl.static-php.dev/static-php-cli/common/"
	BulkBaseURL   = "https://dl.static-php.dev/static-php-cli/bulk/"
)

// osName returns the OS name for static-php.dev URLs.
func osName() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return runtime.GOOS
}

// archName returns the architecture name for static-php.dev URLs.
func archName() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	default:
		return runtime.GOARCH
	}
}

// Download fetches and extracts a PHP binary to the specified path.
func Download(version, tier, destPath string, showProgress bool) error {
	baseURL := CommonBaseURL
	if tier == "bulk" {
		baseURL = BulkBaseURL
	}

	filename := fmt.Sprintf("php-%s-cli-%s-%s.tar.gz", version, osName(), archName())
	url := baseURL + filename

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download PHP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download PHP: HTTP %d", resp.StatusCode)
	}

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// Set up reader with optional progress bar
	var reader io.Reader = resp.Body
	if showProgress {
		bar := progressbar.DefaultBytes(
			resp.ContentLength,
			fmt.Sprintf("Downloading PHP %s", version),
		)
		reader = io.TeeReader(resp.Body, bar)
	}

	// Extract tar.gz
	if err := extractTarGz(reader, filepath.Dir(destPath)); err != nil {
		return fmt.Errorf("failed to extract PHP: %w", err)
	}

	// Verify the binary exists and is executable
	if _, err := os.Stat(destPath); err != nil {
		return fmt.Errorf("PHP binary not found after extraction: %w", err)
	}

	return nil
}

func extractTarGz(r io.Reader, destDir string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()

		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		}
	}

	return nil
}

// EnsurePHP ensures a PHP binary is available, downloading if necessary.
func EnsurePHP(res *Resolution, showProgress bool) error {
	if res.Cached {
		return nil
	}

	return Download(res.Version.String(), res.Tier, res.Path, showProgress)
}
