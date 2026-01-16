package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/phpx-dev/phpx/internal/cache"
	"github.com/phpx-dev/phpx/internal/composer"
	"github.com/phpx-dev/phpx/internal/exec"
	"github.com/phpx-dev/phpx/internal/index"
	"github.com/phpx-dev/phpx/internal/metadata"
	"github.com/phpx-dev/phpx/internal/php"
	"github.com/spf13/cobra"
)

var (
	runPHP        string
	runPackages   string
	runExtensions string
)

var runCmd = &cobra.Command{
	Use:   "run <script.php> [-- args...]",
	Short: "Run a PHP script with inline dependencies",
	Long: `Run a PHP script, automatically installing any declared dependencies.

The script can declare dependencies in a // phpx comment block:

    <?php
    // phpx
    // php = ">=8.2"
    // packages = ["guzzlehttp/guzzle:^7.0"]
    // extensions = ["redis"]

    // Script code here...

Use "-" to read from stdin.`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: false,
	RunE:               runScript,
}

func init() {
	runCmd.Flags().StringVar(&runPHP, "php", "", "PHP version constraint (overrides script)")
	runCmd.Flags().StringVar(&runPackages, "packages", "", "comma-separated packages to add")
	runCmd.Flags().StringVar(&runExtensions, "extensions", "", "comma-separated PHP extensions")

	rootCmd.AddCommand(runCmd)
}

func runScript(cmd *cobra.Command, args []string) error {
	scriptPath := args[0]
	scriptArgs := args[1:]

	// Handle stdin
	if scriptPath == "-" {
		tmpFile, err := os.CreateTemp("", "phpx-*.php")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := io.Copy(tmpFile, os.Stdin); err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to read stdin: %w", err)
		}
		tmpFile.Close()
		scriptPath = tmpFile.Name()
	} else {
		// Verify script exists
		if _, err := os.Stat(scriptPath); err != nil {
			return fmt.Errorf("script not found: %s", scriptPath)
		}
		scriptPath, _ = filepath.Abs(scriptPath)
	}

	// Read and parse script
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read script: %w", err)
	}

	meta, err := metadata.Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Merge CLI flags with metadata
	phpConstraint := runPHP
	if phpConstraint == "" {
		phpConstraint = meta.PHP
	}

	packages := meta.Packages
	if runPackages != "" {
		packages = append(packages, strings.Split(runPackages, ",")...)
	}

	extensions := meta.Extensions
	if runExtensions != "" {
		extensions = append(extensions, strings.Split(runExtensions, ",")...)
	}

	// Load index
	if verbose {
		fmt.Fprintln(os.Stderr, "[phpx] Loading index...")
	}

	idx, err := index.Load()
	if err != nil {
		return fmt.Errorf("failed to load index: %w", err)
	}

	// Resolve PHP
	if verbose {
		if phpConstraint != "" {
			fmt.Fprintf(os.Stderr, "[phpx] Resolving PHP version for constraint '%s'\n", phpConstraint)
		} else {
			fmt.Fprintln(os.Stderr, "[phpx] Resolving latest PHP version")
		}
	}

	res, err := php.Resolve(idx, phpConstraint, extensions)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[phpx] Matched: %s (%s tier)\n", res.Version, res.Tier)
	}

	// Ensure PHP is available
	showProgress := !quiet && !verbose
	if err := php.EnsurePHP(res, showProgress); err != nil {
		return err
	}

	if verbose && !res.Cached {
		fmt.Fprintf(os.Stderr, "[phpx] PHP binary downloaded to %s\n", res.Path)
	}

	var autoloadPath string

	// Install dependencies if any
	if len(packages) > 0 {
		hash := cache.DepsHash(packages)
		depsPath, err := cache.DepsPath(hash)
		if err != nil {
			return err
		}

		autoloadPath = filepath.Join(depsPath, "vendor", "autoload.php")

		if !cache.Exists(autoloadPath) {
			if verbose {
				fmt.Fprintf(os.Stderr, "[phpx] Installing dependencies to %s\n", depsPath)
			}

			// Get Composer
			cv, err := idx.SelectComposer(res.Version.String())
			if err != nil {
				return err
			}

			composerPath, err := index.DownloadComposer(cv)
			if err != nil {
				return fmt.Errorf("failed to download Composer: %w", err)
			}

			if verbose {
				fmt.Fprintf(os.Stderr, "[phpx] Using Composer %s\n", cv.Version)
			}

			// Install
			if err := composer.InstallDeps(res.Path, composerPath, packages, depsPath, verbose); err != nil {
				return err
			}
		} else if verbose {
			fmt.Fprintln(os.Stderr, "[phpx] Dependencies cached")
		}
	}

	// Execute script
	if verbose {
		fmt.Fprintf(os.Stderr, "[phpx] Executing: %s %s\n", res.Path, scriptPath)
	}

	exitCode, err := exec.RunScript(res.Path, scriptPath, autoloadPath, scriptArgs)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[phpx] Exit code: %d\n", exitCode)
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}
