package php

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/phpx-dev/phpx/internal/cache"
	"github.com/phpx-dev/phpx/internal/index"
)

// Resolution contains the result of resolving a PHP requirement.
type Resolution struct {
	Version *semver.Version
	Tier    string
	Path    string
	Cached  bool
}

// Resolve determines the PHP version and tier needed for the given constraint and extensions.
func Resolve(idx *index.Index, constraint string, extensions []string) (*Resolution, error) {
	// Determine required tier
	tier, err := idx.RequiredTier(extensions)
	if err != nil {
		return nil, err
	}

	// Select version list based on tier
	versions := idx.CommonVersions
	if tier == "bulk" {
		versions = idx.BulkVersions
	}

	// Resolve version
	var version *semver.Version
	if constraint == "" {
		version = index.LatestVersion(versions)
		if version == nil {
			return nil, fmt.Errorf("no PHP versions available")
		}
	} else {
		version, err = index.MatchingVersion(versions, constraint)
		if err != nil {
			return nil, err
		}
	}

	// Check cache
	path, err := cache.PHPPath(version.String(), tier)
	if err != nil {
		return nil, err
	}

	return &Resolution{
		Version: version,
		Tier:    tier,
		Path:    path,
		Cached:  cache.Exists(path),
	}, nil
}
