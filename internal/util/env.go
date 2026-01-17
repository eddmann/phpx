package util

import (
	"os"
	"strings"
)

// SafeEnvPrefixes are environment variable prefixes that are safe to pass through.
var SafeEnvPrefixes = []string{
	"LC_",  // Locale settings
	"XDG_", // XDG directories
}

// SafeEnvVars are specific environment variables that are safe to pass through.
var SafeEnvVars = map[string]bool{
	// System essentials
	"PATH":   true,
	"HOME":   true,
	"USER":   true,
	"SHELL":  true,
	"LANG":   true,
	"TERM":   true,
	"TZ":     true,
	"TMPDIR": true,
	"TEMP":   true,
	"TMP":    true,

	// User info
	"LOGNAME": true,
	"UID":     true,

	// Locale
	"LANGUAGE":    true,
	"LC_ALL":      true,
	"LC_COLLATE":  true,
	"LC_CTYPE":    true,
	"LC_MESSAGES": true,
	"LC_MONETARY": true,
	"LC_NUMERIC":  true,
	"LC_TIME":     true,

	// Terminal
	"COLORTERM": true,
	"COLUMNS":   true,
	"LINES":     true,

	// Editor (non-sensitive)
	"EDITOR": true,
	"VISUAL": true,
	"PAGER":  true,
}

// FilterEnv returns a filtered list of environment variables containing only safe vars.
// Additional vars can be explicitly allowed via the allow parameter.
func FilterEnv(allow []string) []string {
	// Build set of explicitly allowed vars
	explicitAllow := make(map[string]bool)
	for _, v := range allow {
		// Handle both "VAR" and "VAR=value" formats
		name := v
		if idx := strings.Index(v, "="); idx != -1 {
			name = v[:idx]
		}
		explicitAllow[name] = true
	}

	var filtered []string

	for _, env := range os.Environ() {
		idx := strings.Index(env, "=")
		if idx == -1 {
			continue
		}

		name := env[:idx]

		// Check if explicitly allowed
		if explicitAllow[name] {
			filtered = append(filtered, env)
			continue
		}

		// Check safelist
		if SafeEnvVars[name] {
			filtered = append(filtered, env)
			continue
		}

		// Check safe prefixes
		for _, prefix := range SafeEnvPrefixes {
			if strings.HasPrefix(name, prefix) {
				filtered = append(filtered, env)
				break
			}
		}
	}

	// Add any explicit "VAR=value" entries that weren't in os.Environ()
	for _, v := range allow {
		if strings.Contains(v, "=") {
			// Check if we already have this var
			name := v[:strings.Index(v, "=")]
			found := false
			for _, f := range filtered {
				if strings.HasPrefix(f, name+"=") {
					found = true
					break
				}
			}
			if !found {
				filtered = append(filtered, v)
			}
		}
	}

	return filtered
}
