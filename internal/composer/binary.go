package composer

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Aliases maps common tool names to their full package names.
var Aliases = map[string]string{
	"phpstan":      "phpstan/phpstan",
	"psalm":        "vimeo/psalm",
	"php-cs-fixer": "friendsofphp/php-cs-fixer",
	"pint":         "laravel/pint",
	"phpunit":      "phpunit/phpunit",
	"pest":         "pestphp/pest",
	"rector":       "rector/rector",
	"phpcs":        "squizlabs/php_codesniffer",
	"laravel":      "laravel/installer",
	"psysh":        "psy/psysh",
}

// ResolveAlias expands a tool name alias to its full package name.
// If the name is not an alias, it's returned unchanged.
func ResolveAlias(name string) string {
	if full, ok := Aliases[name]; ok {
		return full
	}
	return name
}

// InferBinary determines the binary to execute for a package.
// Priority: fromFlag > match short name > first binary
func InferBinary(pkg string, bins []string, fromFlag string) (string, error) {
	if fromFlag != "" {
		return fromFlag, nil
	}

	if len(bins) == 0 {
		return "", fmt.Errorf("binary not found in package: %s", pkg)
	}

	if len(bins) == 1 {
		return filepath.Base(bins[0]), nil
	}

	// Try to match package short name
	shortName := packageShortName(pkg)
	for _, bin := range bins {
		base := filepath.Base(bin)
		// Remove .phar suffix if present
		base = strings.TrimSuffix(base, ".phar")
		if base == shortName {
			return filepath.Base(bin), nil
		}
	}

	// Default to first binary
	return filepath.Base(bins[0]), nil
}

// packageShortName returns the part after the vendor slash.
// e.g., "phpstan/phpstan" -> "phpstan"
func packageShortName(pkg string) string {
	parts := strings.Split(pkg, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return pkg
}

// ParseToolArg parses a tool argument like "phpstan@1.10.0" or "phpstan:^1.10".
// Returns package name and version constraint.
func ParseToolArg(arg string) (pkg, version string) {
	// Check for @ (exact version)
	if idx := strings.Index(arg, "@"); idx != -1 {
		return arg[:idx], arg[idx+1:]
	}

	// Check for : (constraint)
	if idx := strings.Index(arg, ":"); idx != -1 {
		return arg[:idx], arg[idx+1:]
	}

	return arg, ""
}
