package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eddmann/phpx/internal/cache"
	"github.com/eddmann/phpx/internal/composer"
	"github.com/eddmann/phpx/internal/exec"
	"github.com/eddmann/phpx/internal/index"
	"github.com/eddmann/phpx/internal/php"
	"github.com/spf13/cobra"
)

var (
	toolPHP        string
	toolExtensions string
	toolFrom       string
)

var toolCmd = &cobra.Command{
	Use:   "tool <package[@version]> [-- args...]",
	Short: "Run a Composer package's binary",
	Long: `Run a Composer tool without global installation.

Examples:
    phpx tool phpstan -- analyze src/
    phpx tool phpstan@1.10.0 -- analyze src/
    phpx tool phpstan:^1.10 -- analyze src/

Common aliases are supported:
    phpstan      → phpstan/phpstan
    psalm        → vimeo/psalm
    php-cs-fixer → friendsofphp/php-cs-fixer
    pint         → laravel/pint
    phpunit      → phpunit/phpunit
    pest         → pestphp/pest
    rector       → rector/rector
    phpcs        → squizlabs/php_codesniffer
    laravel      → laravel/installer
    psysh        → psy/psysh`,
	Args: cobra.MinimumNArgs(1),
	RunE: runTool,
}

func init() {
	toolCmd.Flags().StringVar(&toolPHP, "php", "", "PHP version constraint")
	toolCmd.Flags().StringVar(&toolExtensions, "extensions", "", "comma-separated PHP extensions")
	toolCmd.Flags().StringVar(&toolFrom, "from", "", "explicit package name when binary differs")

	rootCmd.AddCommand(toolCmd)
}

func runTool(cmd *cobra.Command, args []string) error {
	toolArg := args[0]
	toolArgs := args[1:]

	// Parse package and version
	pkgName, versionConstraint := composer.ParseToolArg(toolArg)
	pkgName = composer.ResolveAlias(pkgName)

	if verbose {
		fmt.Fprintf(os.Stderr, "[phpx] Tool: %s", pkgName)
		if versionConstraint != "" {
			fmt.Fprintf(os.Stderr, " (%s)", versionConstraint)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Fetch package info
	if verbose {
		fmt.Fprintln(os.Stderr, "[phpx] Fetching package info from Packagist...")
	}

	pkgInfo, err := composer.FetchPackage(pkgName)
	if err != nil {
		return err
	}

	// Resolve version
	version, err := composer.ResolveVersion(pkgInfo, versionConstraint)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[phpx] Resolved version: %s\n", version.Version)
	}

	// Infer binary
	binary, err := composer.InferBinary(pkgName, version.Bin, toolFrom)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[phpx] Binary: %s\n", binary)
	}

	// Parse extensions
	var extensions []string
	if toolExtensions != "" {
		extensions = strings.Split(toolExtensions, ",")
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
	phpConstraint := toolPHP
	if phpConstraint == "" {
		// Use package's PHP requirement if available
		if req, ok := version.Require["php"]; ok {
			phpConstraint = req
		}
	}

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

	// Check if tool is cached
	toolPath, err := cache.ToolPath(pkgName, version.Version)
	if err != nil {
		return err
	}

	binaryPath := filepath.Join(toolPath, "vendor", "bin", binary)

	if !cache.Exists(binaryPath) {
		if verbose {
			fmt.Fprintf(os.Stderr, "[phpx] Installing %s@%s to %s\n", pkgName, version.Version, toolPath)
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
		if err := composer.InstallTool(res.Path, composerPath, pkgName, version.Version, toolPath, verbose); err != nil {
			return err
		}
	} else if verbose {
		fmt.Fprintln(os.Stderr, "[phpx] Tool cached")
	}

	// Execute tool
	if verbose {
		fmt.Fprintf(os.Stderr, "[phpx] Executing: %s %s\n", res.Path, binaryPath)
	}

	exitCode, err := exec.RunTool(res.Path, toolPath, binary, toolArgs)
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
