package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// MacOSNetwork implements a lightweight network-only sandbox using macOS sandbox-exec.
// Unlike the full MacOS sandbox, this only restricts network access while allowing
// full filesystem access. Used for --offline and --allow-host flags.
type MacOSNetwork struct{}

// Name returns the sandbox name.
func (m *MacOSNetwork) Name() string {
	return "macos-network"
}

// IsSandboxed returns true - this backend applies sandboxing.
func (m *MacOSNetwork) IsSandboxed() bool {
	return true
}

// Available returns true if sandbox-exec is available.
func (m *MacOSNetwork) Available() bool {
	return runtime.GOOS == "darwin" && commandExists("sandbox-exec")
}

// Execute runs a command with network-only restrictions.
func (m *MacOSNetwork) Execute(ctx context.Context, cfg *Config) (*Result, error) {
	// Generate minimal network-only sandbox profile
	profile := m.generateProfile(cfg)

	// Write profile to temp file
	profileFile, err := os.CreateTemp("", "phpx-network-*.sb")
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

// generateProfile creates a minimal Seatbelt profile that only restricts network.
// Allows full filesystem access - only network is controlled.
func (m *MacOSNetwork) generateProfile(cfg *Config) string {
	var profile bytes.Buffer

	profile.WriteString("(version 1)\n")
	profile.WriteString("(allow default)\n\n") // Allow everything by default

	// ============================================================
	// NETWORK RESTRICTIONS
	// This is the only thing we restrict
	// ============================================================
	profile.WriteString(";; Block all network except proxy\n")
	profile.WriteString("(deny network*)\n\n")

	if cfg.Network {
		profile.WriteString(";; Allow connections to proxy only\n")
		if cfg.ProxyPort > 0 {
			profile.WriteString(fmt.Sprintf("(allow network-outbound (remote ip \"localhost:%d\"))\n", cfg.ProxyPort))
		}
		// Allow Unix socket connections for proxy
		profile.WriteString("(allow network-outbound (remote unix-socket))\n")
	}
	// If !cfg.Network (--offline), no network rules are added, so all network is blocked

	return profile.String()
}
