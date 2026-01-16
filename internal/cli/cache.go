package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/phpx-dev/phpx/internal/cache"
	"github.com/spf13/cobra"
)

var (
	cleanPHP     bool
	cleanDeps    bool
	cleanIndex   bool
	cleanAll     bool
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the phpx cache",
	Long:  `View and manage cached PHP binaries, dependencies, and tools.`,
}

var cacheListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show cached items",
	RunE:  cacheList,
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove cached items",
	Long: `Remove cached items. By default, removes tool cache only.

Flags:
    --php     Remove PHP binaries
    --deps    Remove dependencies
    --index   Remove index cache (forces re-fetch)
    --all     Remove everything`,
	RunE: cacheClean,
}

var cacheDirCmd = &cobra.Command{
	Use:   "dir",
	Short: "Print cache directory path",
	RunE:  cacheDir,
}

var cacheRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Force re-fetch of version/extension index",
	RunE:  cacheRefresh,
}

func init() {
	cacheCleanCmd.Flags().BoolVar(&cleanPHP, "php", false, "remove PHP binaries")
	cacheCleanCmd.Flags().BoolVar(&cleanDeps, "deps", false, "remove dependencies")
	cacheCleanCmd.Flags().BoolVar(&cleanIndex, "index", false, "remove index cache")
	cacheCleanCmd.Flags().BoolVar(&cleanAll, "all", false, "remove everything")

	cacheCmd.AddCommand(cacheListCmd)
	cacheCmd.AddCommand(cacheCleanCmd)
	cacheCmd.AddCommand(cacheDirCmd)
	cacheCmd.AddCommand(cacheRefreshCmd)

	rootCmd.AddCommand(cacheCmd)
}

func cacheList(cmd *cobra.Command, args []string) error {
	baseDir, err := cache.Dir()
	if err != nil {
		return err
	}

	if !cache.Exists(baseDir) {
		fmt.Println("Cache is empty")
		return nil
	}

	// PHP binaries
	phpDir, _ := cache.PHPDir()
	if cache.Exists(phpDir) {
		fmt.Println("PHP Binaries:")
		entries, _ := os.ReadDir(phpDir)
		for _, e := range entries {
			if e.IsDir() {
				size := dirSize(filepath.Join(phpDir, e.Name()))
				fmt.Printf("  %s (%s)\n", e.Name(), formatSize(size))
			}
		}
		fmt.Println()
	}

	// Dependencies
	depsDir, _ := cache.DepsDir()
	if cache.Exists(depsDir) {
		fmt.Println("Dependencies:")
		entries, _ := os.ReadDir(depsDir)
		for _, e := range entries {
			if e.IsDir() {
				size := dirSize(filepath.Join(depsDir, e.Name()))
				fmt.Printf("  %s (%s)\n", e.Name()[:12]+"...", formatSize(size))
			}
		}
		fmt.Println()
	}

	// Tools
	toolsDir, _ := cache.ToolsDir()
	if cache.Exists(toolsDir) {
		fmt.Println("Tools:")
		entries, _ := os.ReadDir(toolsDir)
		for _, e := range entries {
			if e.IsDir() {
				size := dirSize(filepath.Join(toolsDir, e.Name()))
				fmt.Printf("  %s (%s)\n", e.Name(), formatSize(size))
			}
		}
		fmt.Println()
	}

	// Composer
	composerDir, _ := cache.ComposerDir()
	if cache.Exists(composerDir) {
		fmt.Println("Composer:")
		entries, _ := os.ReadDir(composerDir)
		for _, e := range entries {
			if e.IsDir() {
				size := dirSize(filepath.Join(composerDir, e.Name()))
				fmt.Printf("  %s (%s)\n", e.Name(), formatSize(size))
			}
		}
		fmt.Println()
	}

	// Index
	indexDir, _ := cache.IndexDir()
	if cache.Exists(indexDir) {
		fetchedAtPath := filepath.Join(indexDir, "fetched_at")
		if data, err := os.ReadFile(fetchedAtPath); err == nil {
			if t, err := time.Parse(time.RFC3339, string(data)); err == nil {
				fmt.Printf("Index: fetched %s ago\n", formatDuration(time.Since(t)))
			}
		}
	}

	return nil
}

func cacheClean(cmd *cobra.Command, args []string) error {
	if cleanAll {
		if err := cache.Clean("all"); err != nil {
			return err
		}
		fmt.Println("Removed all cache")
		return nil
	}

	cleaned := false

	if cleanPHP {
		if err := cache.Clean("php"); err != nil {
			return err
		}
		fmt.Println("Removed PHP binaries")
		cleaned = true
	}

	if cleanDeps {
		if err := cache.Clean("deps"); err != nil {
			return err
		}
		fmt.Println("Removed dependencies")
		cleaned = true
	}

	if cleanIndex {
		if err := cache.Clean("index"); err != nil {
			return err
		}
		fmt.Println("Removed index cache")
		cleaned = true
	}

	// Default: clean tools only
	if !cleaned {
		if err := cache.Clean("tools"); err != nil {
			return err
		}
		fmt.Println("Removed tools")
	}

	return nil
}

func cacheDir(cmd *cobra.Command, args []string) error {
	dir, err := cache.Dir()
	if err != nil {
		return err
	}
	fmt.Println(dir)
	return nil
}

func cacheRefresh(cmd *cobra.Command, args []string) error {
	// Remove index cache
	if err := cache.Clean("index"); err != nil {
		return err
	}

	fmt.Println("Index cache cleared. Will be re-fetched on next run.")
	return nil
}

func dirSize(path string) int64 {
	var size int64
	filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				size += info.Size()
			}
		}
		return nil
	})
	return size
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	}
	return fmt.Sprintf("%d days", int(d.Hours()/24))
}
