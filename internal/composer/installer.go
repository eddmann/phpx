package composer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/eddmann/phpx/internal/cache"
	"github.com/eddmann/phpx/internal/util"
)

// composerJSON is the structure for composer.json.
type composerJSON struct {
	Require map[string]string `json:"require"`
	Config  composerConfig    `json:"config"`
}

type composerConfig struct {
	AllowPlugins       bool `json:"allow-plugins"`
	OptimizeAutoloader bool `json:"optimize-autoloader"`
}

// InstallDeps installs packages to a dependency directory.
func InstallDeps(phpPath, composerPath string, packages []string, destDir string, verbose bool) error {
	if err := cache.EnsureDir(destDir); err != nil {
		return err
	}

	// Generate composer.json
	cj := composerJSON{
		Require: make(map[string]string),
		Config: composerConfig{
			AllowPlugins:       false,
			OptimizeAutoloader: true,
		},
	}

	for _, pkg := range packages {
		name, constraint := parsePackage(pkg)
		if constraint == "" {
			constraint = "*"
		}
		cj.Require[name] = constraint
	}

	composerJSONPath := filepath.Join(destDir, "composer.json")
	data, err := json.MarshalIndent(cj, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(composerJSONPath, data, 0644); err != nil {
		return err
	}

	// Run composer install
	args := []string{
		composerPath,
		"install",
		"--no-dev",
		"--no-interaction",
		"--no-scripts",
		"--prefer-dist",
		"--optimize-autoloader",
	}

	if !verbose {
		args = append(args, "--quiet")
	}

	cmd := exec.Command(phpPath, args...)
	cmd.Dir = destDir
	// Use filtered environment to avoid leaking secrets to package install scripts
	cmd.Env = append(util.FilterEnv(nil), "COMPOSER_HOME="+filepath.Join(destDir, ".composer"))

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install packages %v: %w", packages, err)
	}

	return nil
}

// InstallTool installs a tool package to a directory.
func InstallTool(phpPath, composerPath string, pkg, version, destDir string, verbose bool) error {
	if err := cache.EnsureDir(destDir); err != nil {
		return err
	}

	// Generate composer.json
	constraint := version
	if constraint == "" {
		constraint = "*"
	}

	cj := composerJSON{
		Require: map[string]string{
			pkg: constraint,
		},
		Config: composerConfig{
			AllowPlugins:       false,
			OptimizeAutoloader: true,
		},
	}

	composerJSONPath := filepath.Join(destDir, "composer.json")
	data, err := json.MarshalIndent(cj, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(composerJSONPath, data, 0644); err != nil {
		return err
	}

	// Run composer install
	args := []string{
		composerPath,
		"install",
		"--no-dev",
		"--no-interaction",
		"--no-scripts",
		"--prefer-dist",
		"--optimize-autoloader",
	}

	if !verbose {
		args = append(args, "--quiet")
	}

	cmd := exec.Command(phpPath, args...)
	cmd.Dir = destDir
	// Use filtered environment to avoid leaking secrets to package install scripts
	cmd.Env = append(util.FilterEnv(nil), "COMPOSER_HOME="+filepath.Join(destDir, ".composer"))

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install tool %s@%s: %w", pkg, version, err)
	}

	return nil
}

// parsePackage splits "vendor/package:constraint" into name and constraint.
func parsePackage(pkg string) (name, constraint string) {
	if idx := strings.LastIndex(pkg, ":"); idx != -1 {
		return pkg[:idx], pkg[idx+1:]
	}
	return pkg, ""
}

