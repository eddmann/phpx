package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// MacOS implements sandbox using macOS sandbox-exec (Seatbelt).
type MacOS struct{}

// Name returns the sandbox name.
func (m *MacOS) Name() string {
	return "macos"
}

// IsSandboxed returns true - this backend applies sandboxing.
func (m *MacOS) IsSandboxed() bool {
	return true
}

// Available returns true if sandbox-exec is available.
func (m *MacOS) Available() bool {
	return runtime.GOOS == "darwin" && commandExists("sandbox-exec")
}

// Execute runs a command in the macOS sandbox.
func (m *MacOS) Execute(ctx context.Context, cfg *Config) (*Result, error) {
	// Generate sandbox profile
	profile := m.generateProfile(cfg)

	// Write profile to temp file
	profileFile, err := os.CreateTemp("", "phpx-sandbox-*.sb")
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox profile: %w", err)
	}
	defer func() { _ = os.Remove(profileFile.Name()) }()

	if _, err := profileFile.WriteString(profile); err != nil {
		_ = profileFile.Close()
		return nil, fmt.Errorf("failed to write sandbox profile: %w", err)
	}
	_ = profileFile.Close()

	// Build command: sandbox-exec -f profile php [args...]
	args := append([]string{"-f", profileFile.Name()}, BuildPHPArgs(cfg)...)

	cmd := exec.CommandContext(ctx, "sandbox-exec", args...)
	cmd.Dir = cfg.WorkDir

	stdout, stderr := SetupCommand(cmd, cfg)

	err = cmd.Run()
	return BuildResult(err, cfg, stdout, stderr)
}

// resolvePath resolves symlinks in a path for Seatbelt compatibility.
// macOS Seatbelt doesn't follow symlinks in subpath rules, so we need
// to resolve them (e.g., /var/folders -> /private/var/folders).
func resolvePath(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// If we can't resolve (e.g., path doesn't exist yet), return original
		return path
	}
	return resolved
}

// seatbeltEscape escapes a string for use in Seatbelt profile.
// Prevents injection attacks via malicious paths.
func seatbeltEscape(s string) string {
	// Escape backslashes first, then quotes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// generateProfile creates a minimal Seatbelt sandbox profile.
// Follows principle of least privilege - only allows what's strictly necessary
// for a static PHP binary to execute scripts.
func (m *MacOS) generateProfile(cfg *Config) string {
	var profile bytes.Buffer

	profile.WriteString("(version 1)\n")
	profile.WriteString("(deny default)\n\n")

	// ============================================================
	// PROCESS & SYSTEM OPERATIONS
	// ============================================================
	profile.WriteString(";; Process operations (required for PHP to run)\n")
	profile.WriteString("(allow process*)\n")
	profile.WriteString("(allow sysctl-read)\n")
	profile.WriteString("(allow mach-lookup)\n")
	profile.WriteString("(allow signal (target self))\n\n")

	// ============================================================
	// MINIMAL READ ACCESS
	// Static PHP binary needs very little system access
	// ============================================================
	profile.WriteString(";; Minimal device access\n")
	profile.WriteString("(allow file-read* (literal \"/dev/null\"))\n")
	profile.WriteString("(allow file-read* (literal \"/dev/urandom\"))\n")
	profile.WriteString("(allow file-read* (literal \"/dev/random\"))\n\n")

	// Timezone support (date() is very common)
	profile.WriteString(";; Timezone data\n")
	profile.WriteString("(allow file-read* (subpath \"/usr/share/zoneinfo\"))\n")
	profile.WriteString("(allow file-read* (subpath \"/var/db/timezone\"))\n")
	profile.WriteString("(allow file-read* (literal \"/etc/localtime\"))\n")
	profile.WriteString("(allow file-read* (literal \"/private/etc/localtime\"))\n\n")

	// DNS resolution (only if network enabled)
	if cfg.Network {
		profile.WriteString(";; DNS resolution (network enabled)\n")
		profile.WriteString("(allow file-read* (literal \"/etc/resolv.conf\"))\n")
		profile.WriteString("(allow file-read* (literal \"/private/etc/resolv.conf\"))\n")
		profile.WriteString("(allow file-read* (literal \"/etc/hosts\"))\n")
		profile.WriteString("(allow file-read* (literal \"/private/etc/hosts\"))\n\n")
	}

	// PHP binary (exact path only)
	if cfg.PHPBinary != "" {
		profile.WriteString(";; PHP binary\n")
		profile.WriteString(fmt.Sprintf("(allow file-read* (literal \"%s\"))\n\n", seatbeltEscape(resolvePath(cfg.PHPBinary))))
	}

	// Script file (exact path only)
	if cfg.ScriptPath != "" {
		profile.WriteString(";; Script file\n")
		profile.WriteString(fmt.Sprintf("(allow file-read* (literal \"%s\"))\n\n", seatbeltEscape(resolvePath(cfg.ScriptPath))))
	}

	// Vendor directory (Composer dependencies) - required for autoload
	if cfg.AutoloadFile != "" {
		profile.WriteString(";; Vendor directory (dependencies)\n")
		vendorDir := filepath.Dir(cfg.AutoloadFile)
		profile.WriteString(fmt.Sprintf("(allow file-read* (subpath \"%s\"))\n\n", seatbeltEscape(resolvePath(vendorDir))))
	}

	// Additional readable paths from --allow-read flag
	if len(cfg.ReadablePaths) > 0 {
		profile.WriteString(";; Additional readable paths (--allow-read)\n")
		for _, p := range cfg.ReadablePaths {
			profile.WriteString(fmt.Sprintf("(allow file-read* (subpath \"%s\"))\n", seatbeltEscape(resolvePath(p))))
		}
		profile.WriteString("\n")
	}

	// ============================================================
	// MINIMAL WRITE ACCESS
	// By default, only /dev/null is writable
	// ============================================================
	profile.WriteString(";; Minimal write access\n")
	profile.WriteString("(allow file-write* (literal \"/dev/null\"))\n\n")

	// Additional writable paths from --allow-write flag
	if len(cfg.WritablePaths) > 0 {
		profile.WriteString(";; Additional writable paths (--allow-write)\n")
		for _, p := range cfg.WritablePaths {
			profile.WriteString(fmt.Sprintf("(allow file-write* (subpath \"%s\"))\n", seatbeltEscape(resolvePath(p))))
			// Also need read access to write
			profile.WriteString(fmt.Sprintf("(allow file-read* (subpath \"%s\"))\n", seatbeltEscape(resolvePath(p))))
		}
		profile.WriteString("\n")
	}

	// ============================================================
	// NETWORK ACCESS
	// ============================================================
	if cfg.Network {
		// Restrict to proxy port only - no fallback to wildcard localhost
		profile.WriteString(";; Network: proxy connections only\n")
		if cfg.ProxyPort > 0 {
			profile.WriteString(fmt.Sprintf("(allow network-outbound (remote ip \"localhost:%d\"))\n", cfg.ProxyPort))
		}
		// Always allow Unix socket connections for proxy
		profile.WriteString("(allow network-outbound (remote unix-socket))\n")
	}

	return profile.String()
}
