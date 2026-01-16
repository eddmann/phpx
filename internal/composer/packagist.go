package composer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Masterminds/semver/v3"
)

const PackagistURL = "https://repo.packagist.org/p2/"

// PackageInfo contains information about a Composer package.
type PackageInfo struct {
	Name     string
	Versions []PackageVersion
}

// PackageVersion represents a single version of a package.
type PackageVersion struct {
	Version           string            `json:"version"`
	VersionNormalized string            `json:"version_normalized"`
	Require           map[string]string `json:"require"`
	Bin               []string          `json:"bin"`
	Type              string            `json:"type"`
}

// packagistResponse is the raw API response structure.
type packagistResponse struct {
	Packages map[string][]PackageVersion `json:"packages"`
}

// FetchPackage retrieves package information from Packagist.
func FetchPackage(name string) (*PackageInfo, error) {
	url := PackagistURL + name + ".json"

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package not found: %s", name)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("packagist returned HTTP %d", resp.StatusCode)
	}

	var data packagistResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	versions, ok := data.Packages[name]
	if !ok {
		return nil, fmt.Errorf("package not found: %s", name)
	}

	return &PackageInfo{
		Name:     name,
		Versions: versions,
	}, nil
}

// ResolveVersion finds the best matching version for a constraint.
// If constraint is empty, returns the latest stable version.
func ResolveVersion(pkg *PackageInfo, constraint string) (*PackageVersion, error) {
	if constraint == "" {
		return latestStable(pkg.Versions)
	}

	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return nil, fmt.Errorf("invalid constraint %q: %w", constraint, err)
	}

	// Sort versions descending and find first match
	var candidates []*PackageVersion
	for i := range pkg.Versions {
		v := &pkg.Versions[i]
		if isPrerelease(v.Version) {
			continue
		}

		sv, err := semver.NewVersion(v.Version)
		if err != nil {
			continue
		}

		if c.Check(sv) {
			candidates = append(candidates, v)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no version satisfies constraint %q", constraint)
	}

	// Return highest matching version
	return highestVersion(candidates)
}

// latestStable returns the highest non-prerelease, non-dev version.
func latestStable(versions []PackageVersion) (*PackageVersion, error) {
	var stable []*PackageVersion

	for i := range versions {
		v := &versions[i]
		if isPrerelease(v.Version) || isDev(v.Version) {
			continue
		}

		if _, err := semver.NewVersion(v.Version); err != nil {
			continue
		}

		stable = append(stable, v)
	}

	if len(stable) == 0 {
		return nil, fmt.Errorf("no stable version found")
	}

	return highestVersion(stable)
}

func highestVersion(versions []*PackageVersion) (*PackageVersion, error) {
	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions provided")
	}

	highest := versions[0]
	highestSV, _ := semver.NewVersion(highest.Version)

	for _, v := range versions[1:] {
		sv, err := semver.NewVersion(v.Version)
		if err != nil {
			continue
		}
		if sv.GreaterThan(highestSV) {
			highest = v
			highestSV = sv
		}
	}

	return highest, nil
}

func isPrerelease(version string) bool {
	lower := strings.ToLower(version)
	return strings.Contains(lower, "-alpha") ||
		strings.Contains(lower, "-beta") ||
		strings.Contains(lower, "-rc") ||
		strings.Contains(lower, "-dev")
}

func isDev(version string) bool {
	return strings.HasPrefix(strings.ToLower(version), "dev-")
}
