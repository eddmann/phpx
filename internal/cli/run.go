package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eddmann/phpx/internal/cache"
	"github.com/eddmann/phpx/internal/composer"
	"github.com/eddmann/phpx/internal/executor"
	"github.com/eddmann/phpx/internal/index"
	"github.com/eddmann/phpx/internal/metadata"
	"github.com/eddmann/phpx/internal/php"
	"github.com/eddmann/phpx/internal/sandbox"
	"github.com/spf13/cobra"
)

var (
	runPHP        string
	runPackages   string
	runExtensions string

	// Security flags
	runSandbox   bool
	runOffline   bool
	runAllowHost string
	runAllowRead string
	runAllowWrite string
	runAllowEnv  string
	runMemory    int
	runTimeout   int
	runCPU       int
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

Use "-" to read from stdin.

Security options:
    --sandbox          Enable sandboxing (restricts filesystem access)
    --offline          Block all network access
    --allow-host       Allow network to specific hosts (comma-separated)
    --allow-read       Allow reading additional paths (comma-separated)
    --allow-write      Allow writing to additional paths (comma-separated)
    --allow-env        Pass through environment variables (comma-separated)`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: false,
	RunE:               runScript,
}

// addScriptFlags registers script execution flags on the given command.
// Called for both the root command and the run subcommand.
func addScriptFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&runPHP, "php", "", "PHP version constraint (overrides script)")
	cmd.Flags().StringVar(&runPackages, "packages", "", "comma-separated packages to add")
	cmd.Flags().StringVar(&runExtensions, "extensions", "", "comma-separated PHP extensions")

	// Security flags
	cmd.Flags().BoolVar(&runSandbox, "sandbox", false, "enable sandboxing")
	cmd.Flags().BoolVar(&runOffline, "offline", false, "block all network access")
	cmd.Flags().StringVar(&runAllowHost, "allow-host", "", "allowed hosts (comma-separated)")
	cmd.Flags().StringVar(&runAllowRead, "allow-read", "", "additional readable paths (comma-separated)")
	cmd.Flags().StringVar(&runAllowWrite, "allow-write", "", "additional writable paths (comma-separated)")
	cmd.Flags().StringVar(&runAllowEnv, "allow-env", "", "environment variables to pass (comma-separated)")
	cmd.Flags().IntVar(&runMemory, "memory", 128, "memory limit in MB")
	cmd.Flags().IntVar(&runTimeout, "timeout", 30, "execution timeout in seconds")
	cmd.Flags().IntVar(&runCPU, "cpu", 30, "CPU time limit in seconds")
}

func init() {
	addScriptFlags(runCmd)
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
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		if _, err := io.Copy(tmpFile, os.Stdin); err != nil {
			_ = tmpFile.Close()
			return fmt.Errorf("failed to read stdin: %w", err)
		}
		_ = tmpFile.Close()
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
		if phpConstraint != "" {
			return fmt.Errorf("failed to resolve PHP for constraint %q: %w", phpConstraint, err)
		}
		return fmt.Errorf("failed to resolve PHP: %w", err)
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

	// Determine sandbox
	var sb sandbox.Sandbox = &sandbox.None{}
	if runSandbox {
		sb = sandbox.Detect()
		if !sb.IsSandboxed() {
			return fmt.Errorf("--sandbox requested but no sandbox is available on this system")
		}
	} else if runOffline || runAllowHost != "" {
		sb = sandbox.DetectNetworkOnly()
		if !sb.IsSandboxed() {
			return fmt.Errorf("--offline/--allow-host requires network sandboxing, but no sandbox is available on this system")
		}
	}

	// Parse security options
	var allowedHosts []string
	if runAllowHost != "" {
		allowedHosts = splitCSV(runAllowHost)
	}

	var readPaths []string
	if runAllowRead != "" {
		readPaths = splitCSV(runAllowRead)
	}

	var writePaths []string
	if runAllowWrite != "" {
		writePaths = splitCSV(runAllowWrite)
	}

	var allowedEnvVars []string
	if runAllowEnv != "" {
		allowedEnvVars = splitCSV(runAllowEnv)
	}

	// Determine network access
	network := !runOffline

	// Build executor options with real-time I/O streaming
	opts := &executor.ScriptOptions{
		ScriptPath:     scriptPath,
		PHPBinary:      res.Path,
		AutoloadFile:   autoloadPath,
		Sandbox:        sb,
		Network:        network,
		AllowedHosts:   allowedHosts,
		AllowedEnvVars: allowedEnvVars,
		ReadPaths:      readPaths,
		WritePaths:     writePaths,
		MemoryMB:       runMemory,
		Timeout:        time.Duration(runTimeout) * time.Second,
		CPUSeconds:     runCPU,
		Args:           scriptArgs,
		Stdin:          os.Stdin,
		Stdout:         os.Stdout,
		Stderr:         os.Stderr,
		Verbose:        verbose,
	}

	// Execute script using executor
	runner := executor.NewScriptRunner(opts)
	result, err := runner.Run(context.Background())
	if err != nil {
		return err
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[phpx] Exit code: %d\n", result.ExitCode)
	}

	if result.ExitCode != 0 {
		os.Exit(result.ExitCode)
	}

	return nil
}

// splitCSV splits a comma-separated string into a slice, trimming whitespace.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
