package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Dir returns the base cache directory (~/.phpx).
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".phpx"), nil
}

// IndexDir returns the path to the index cache directory.
func IndexDir() (string, error) {
	base, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "index"), nil
}

// PHPDir returns the path to PHP binaries directory.
func PHPDir() (string, error) {
	base, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "php"), nil
}

// PHPPath returns the path to a specific PHP binary.
func PHPPath(version, tier string) (string, error) {
	dir, err := PHPDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, version+"-"+tier, "bin", "php"), nil
}

// DepsDir returns the path to the dependencies cache directory.
func DepsDir() (string, error) {
	base, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "deps"), nil
}

// DepsPath returns the path to a specific dependency installation.
func DepsPath(hash string) (string, error) {
	dir, err := DepsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, hash), nil
}

// ToolsDir returns the path to the tools cache directory.
func ToolsDir() (string, error) {
	base, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "tools"), nil
}

// ToolPath returns the path to a specific tool installation.
func ToolPath(pkg, version string) (string, error) {
	dir, err := ToolsDir()
	if err != nil {
		return "", err
	}
	// Replace / with - for directory name
	safePkg := strings.ReplaceAll(pkg, "/", "-")
	return filepath.Join(dir, safePkg+"-"+version), nil
}

// ComposerDir returns the path to the Composer cache directory.
func ComposerDir() (string, error) {
	base, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "composer"), nil
}

// ComposerPath returns the path to a specific composer.phar.
func ComposerPath(version string) (string, error) {
	dir, err := ComposerDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, version, "composer.phar"), nil
}

// DepsHash computes a cache key from a list of packages.
// Packages are sorted and lowercased before hashing.
func DepsHash(packages []string) string {
	// Copy and normalize
	normalized := make([]string, len(packages))
	for i, pkg := range packages {
		normalized[i] = strings.ToLower(pkg)
	}
	sort.Strings(normalized)

	// Hash
	h := sha256.New()
	h.Write([]byte(strings.Join(normalized, "\n")))
	return hex.EncodeToString(h.Sum(nil))
}

// Exists checks if a path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// Clean removes cache items based on the specified target.
// Valid targets: "php", "deps", "tools", "index", "composer", "all"
func Clean(target string) error {
	base, err := Dir()
	if err != nil {
		return err
	}

	switch target {
	case "php":
		return os.RemoveAll(filepath.Join(base, "php"))
	case "deps":
		return os.RemoveAll(filepath.Join(base, "deps"))
	case "tools":
		return os.RemoveAll(filepath.Join(base, "tools"))
	case "index":
		return os.RemoveAll(filepath.Join(base, "index"))
	case "composer":
		return os.RemoveAll(filepath.Join(base, "composer"))
	case "all":
		return os.RemoveAll(base)
	default:
		// Default to tools only
		return os.RemoveAll(filepath.Join(base, "tools"))
	}
}
