package index

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/phpx-dev/phpx/internal/cache"
	"github.com/phpx-dev/phpx/internal/composer"
)

const (
	CommonListURL    = "https://dl.static-php.dev/static-php-cli/common/?format=json"
	BulkListURL      = "https://dl.static-php.dev/static-php-cli/bulk/?format=json"
	CommonExtURL     = "https://dl.static-php.dev/static-php-cli/common/build-extensions.json"
	BulkExtURL       = "https://dl.static-php.dev/static-php-cli/bulk/build-extensions.json"
	ComposerVersions = "https://getcomposer.org/versions"

	CacheTTL = 24 * time.Hour
)

// Index holds cached version and extension information.
type Index struct {
	CommonVersions    []*semver.Version
	BulkVersions      []*semver.Version
	CommonExtensions  []string
	BulkExtensions    []string
	ComposerVersions  []ComposerVersion
	FetchedAt         time.Time
}

// ComposerVersion represents a Composer release.
type ComposerVersion struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	MinPHP  int    `json:"min-php"`
}

// FileEntry represents a file in the static-php.dev listing.
type fileEntry struct {
	Name string `json:"name"`
}

var versionRegex = regexp.MustCompile(`php-(\d+\.\d+\.\d+)-cli-`)

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

// Load retrieves the index, using cache if fresh or fetching if stale.
func Load() (*Index, error) {
	indexDir, err := cache.IndexDir()
	if err != nil {
		return nil, err
	}

	// Check if cache exists and is fresh
	fetchedAtPath := filepath.Join(indexDir, "fetched_at")
	if data, err := os.ReadFile(fetchedAtPath); err == nil {
		if t, err := time.Parse(time.RFC3339, string(data)); err == nil {
			if time.Since(t) < CacheTTL {
				return loadFromCache(indexDir)
			}
		}
	}

	// Fetch fresh data
	return Refresh()
}

// Refresh fetches fresh index data from remote sources.
func Refresh() (*Index, error) {
	indexDir, err := cache.IndexDir()
	if err != nil {
		return nil, err
	}

	if err := cache.EnsureDir(indexDir); err != nil {
		return nil, err
	}

	idx := &Index{FetchedAt: time.Now()}

	// Fetch PHP versions
	idx.CommonVersions, err = fetchVersions(CommonListURL)
	if err != nil {
		return nil, fmt.Errorf("fetch common versions: %w", err)
	}

	idx.BulkVersions, err = fetchVersions(BulkListURL)
	if err != nil {
		return nil, fmt.Errorf("fetch bulk versions: %w", err)
	}

	// Fetch extensions
	idx.CommonExtensions, err = fetchExtensions(CommonExtURL)
	if err != nil {
		return nil, fmt.Errorf("fetch common extensions: %w", err)
	}

	idx.BulkExtensions, err = fetchExtensions(BulkExtURL)
	if err != nil {
		return nil, fmt.Errorf("fetch bulk extensions: %w", err)
	}

	// Fetch Composer versions
	idx.ComposerVersions, err = fetchComposerVersions()
	if err != nil {
		return nil, fmt.Errorf("fetch composer versions: %w", err)
	}

	// Save to cache
	if err := saveToCache(indexDir, idx); err != nil {
		return nil, fmt.Errorf("save cache: %w", err)
	}

	return idx, nil
}

func fetchVersions(url string) ([]*semver.Version, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var entries []fileEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}

	// Filter for current platform CLI binaries
	suffix := fmt.Sprintf("-cli-%s-%s.tar.gz", osName(), archName())
	seen := make(map[string]bool)
	var versions []*semver.Version

	for _, e := range entries {
		if !strings.HasSuffix(e.Name, suffix) {
			continue
		}

		matches := versionRegex.FindStringSubmatch(e.Name)
		if len(matches) < 2 {
			continue
		}

		vStr := matches[1]
		if seen[vStr] {
			continue
		}
		seen[vStr] = true

		v, err := semver.NewVersion(vStr)
		if err != nil {
			continue
		}
		versions = append(versions, v)
	}

	// Sort descending
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].GreaterThan(versions[j])
	})

	return versions, nil
}

func fetchExtensions(url string) ([]string, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // Follow redirects
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var extensions []string
	if err := json.NewDecoder(resp.Body).Decode(&extensions); err != nil {
		return nil, err
	}

	return extensions, nil
}

func fetchComposerVersions() ([]ComposerVersion, error) {
	resp, err := http.Get(ComposerVersions)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var data struct {
		Stable []ComposerVersion `json:"stable"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return data.Stable, nil
}

func loadFromCache(indexDir string) (*Index, error) {
	idx := &Index{}

	// Load common versions
	data, err := os.ReadFile(filepath.Join(indexDir, "common-versions.json"))
	if err != nil {
		return nil, err
	}
	var commonStrs []string
	if err := json.Unmarshal(data, &commonStrs); err != nil {
		return nil, err
	}
	for _, s := range commonStrs {
		v, _ := semver.NewVersion(s)
		if v != nil {
			idx.CommonVersions = append(idx.CommonVersions, v)
		}
	}

	// Load bulk versions
	data, err = os.ReadFile(filepath.Join(indexDir, "bulk-versions.json"))
	if err != nil {
		return nil, err
	}
	var bulkStrs []string
	if err := json.Unmarshal(data, &bulkStrs); err != nil {
		return nil, err
	}
	for _, s := range bulkStrs {
		v, _ := semver.NewVersion(s)
		if v != nil {
			idx.BulkVersions = append(idx.BulkVersions, v)
		}
	}

	// Load extensions
	data, err = os.ReadFile(filepath.Join(indexDir, "common-extensions.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &idx.CommonExtensions); err != nil {
		return nil, err
	}

	data, err = os.ReadFile(filepath.Join(indexDir, "bulk-extensions.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &idx.BulkExtensions); err != nil {
		return nil, err
	}

	// Load Composer versions
	data, err = os.ReadFile(filepath.Join(indexDir, "composer-versions.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &idx.ComposerVersions); err != nil {
		return nil, err
	}

	// Load fetched_at
	data, err = os.ReadFile(filepath.Join(indexDir, "fetched_at"))
	if err != nil {
		return nil, err
	}
	idx.FetchedAt, _ = time.Parse(time.RFC3339, string(data))

	return idx, nil
}

func saveToCache(indexDir string, idx *Index) error {
	// Save common versions
	commonStrs := make([]string, len(idx.CommonVersions))
	for i, v := range idx.CommonVersions {
		commonStrs[i] = v.String()
	}
	data, _ := json.Marshal(commonStrs)
	if err := os.WriteFile(filepath.Join(indexDir, "common-versions.json"), data, 0644); err != nil {
		return err
	}

	// Save bulk versions
	bulkStrs := make([]string, len(idx.BulkVersions))
	for i, v := range idx.BulkVersions {
		bulkStrs[i] = v.String()
	}
	data, _ = json.Marshal(bulkStrs)
	if err := os.WriteFile(filepath.Join(indexDir, "bulk-versions.json"), data, 0644); err != nil {
		return err
	}

	// Save extensions
	data, _ = json.Marshal(idx.CommonExtensions)
	if err := os.WriteFile(filepath.Join(indexDir, "common-extensions.json"), data, 0644); err != nil {
		return err
	}

	data, _ = json.Marshal(idx.BulkExtensions)
	if err := os.WriteFile(filepath.Join(indexDir, "bulk-extensions.json"), data, 0644); err != nil {
		return err
	}

	// Save Composer versions
	data, _ = json.Marshal(idx.ComposerVersions)
	if err := os.WriteFile(filepath.Join(indexDir, "composer-versions.json"), data, 0644); err != nil {
		return err
	}

	// Save fetched_at
	return os.WriteFile(filepath.Join(indexDir, "fetched_at"), []byte(idx.FetchedAt.Format(time.RFC3339)), 0644)
}

// LatestVersion returns the highest version from a list.
func LatestVersion(versions []*semver.Version) *semver.Version {
	if len(versions) == 0 {
		return nil
	}
	return versions[0] // Already sorted descending
}

// MatchingVersion returns the highest version satisfying a constraint.
func MatchingVersion(versions []*semver.Version, constraint string) (*semver.Version, error) {
	normalized := composer.NormalizeConstraint(constraint)
	c, err := semver.NewConstraint(normalized)
	if err != nil {
		return nil, fmt.Errorf("invalid constraint %q: %w", constraint, err)
	}

	for _, v := range versions {
		if c.Check(v) {
			return v, nil
		}
	}

	return nil, fmt.Errorf("no PHP version satisfies '%s'", constraint)
}

// SelectComposer returns the highest Composer version compatible with the given PHP version.
func (idx *Index) SelectComposer(phpVersion string) (*ComposerVersion, error) {
	phpVer, err := semver.NewVersion(phpVersion)
	if err != nil {
		return nil, err
	}

	// Convert PHP version to int: "8.4.17" → 80417
	phpInt := int(phpVer.Major())*10000 + int(phpVer.Minor())*100 + int(phpVer.Patch())

	for _, cv := range idx.ComposerVersions {
		if cv.MinPHP <= phpInt {
			return &cv, nil
		}
	}

	return nil, fmt.Errorf("no Composer version compatible with PHP %s", phpVersion)
}

// HasExtension checks if an extension is available in the given tier.
func (idx *Index) HasExtension(ext, tier string) bool {
	var extensions []string
	if tier == "common" {
		extensions = idx.CommonExtensions
	} else {
		extensions = idx.BulkExtensions
	}

	for _, e := range extensions {
		if e == ext {
			return true
		}
	}
	return false
}

// RequiredTier determines which tier is needed for the given extensions.
// Returns "common", "bulk", or an error if an extension is unavailable.
func (idx *Index) RequiredTier(extensions []string) (string, error) {
	if len(extensions) == 0 {
		return "common", nil
	}

	needsBulk := false
	for _, ext := range extensions {
		if idx.HasExtension(ext, "common") {
			continue
		}
		if idx.HasExtension(ext, "bulk") {
			needsBulk = true
			continue
		}
		return "", fmt.Errorf("extension '%s' not available in static PHP builds", ext)
	}

	if needsBulk {
		return "bulk", nil
	}
	return "common", nil
}

// DownloadComposer downloads a Composer phar to the cache.
func DownloadComposer(cv *ComposerVersion) (string, error) {
	cachePath, err := cache.ComposerPath(cv.Version)
	if err != nil {
		return "", err
	}

	if cache.Exists(cachePath) {
		return cachePath, nil
	}

	if err := cache.EnsureDir(filepath.Dir(cachePath)); err != nil {
		return "", err
	}

	url := "https://getcomposer.org" + cv.Path
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(cachePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(cachePath)
		return "", err
	}

	return cachePath, nil
}
