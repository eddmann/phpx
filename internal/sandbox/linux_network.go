package sandbox

import (
	"context"
	"os/exec"
	"runtime"
)

// LinuxNetwork implements a lightweight network-only sandbox using unshare.
// Unlike the full Bubblewrap/nsjail sandboxes, this only isolates the network
// namespace while allowing full filesystem access. Used for --offline and --allow-host.
type LinuxNetwork struct{}

// Name returns the sandbox name.
func (l *LinuxNetwork) Name() string {
	return "linux-network"
}

// IsSandboxed returns true - this backend applies sandboxing.
func (l *LinuxNetwork) IsSandboxed() bool {
	return true
}

// Available returns true if unshare is available.
func (l *LinuxNetwork) Available() bool {
	return runtime.GOOS == "linux" && commandExists("unshare")
}

// Execute runs a command with network-only isolation.
func (l *LinuxNetwork) Execute(ctx context.Context, cfg *Config) (*Result, error) {
	cmd := l.buildCommand(ctx, cfg)
	cmd.Dir = cfg.WorkDir

	stdout, stderr := SetupCommand(cmd, cfg)

	// If network is enabled and we have a proxy socket, set up proxy env vars
	if cfg.Network && cfg.ProxySocketPath != "" && hasSocat() {
		cmd.Env = append(cmd.Env, ProxyEnvVars()...)
	}

	err := cmd.Run()
	return BuildResult(err, cfg, stdout, stderr)
}

// buildCommand creates the appropriate exec.Cmd based on network requirements.
func (l *LinuxNetwork) buildCommand(ctx context.Context, cfg *Config) *exec.Cmd {
	// Network access via proxy - use unshare with socat bridge
	if cfg.Network && cfg.ProxySocketPath != "" && hasSocat() {
		phpCmd := BuildPHPCommand(cfg)
		shellCmd := BuildSocatBridgeCommand(cfg.ProxySocketPath, phpCmd)
		return exec.CommandContext(ctx, "unshare", "--net", "--map-root-user", "sh", "-c", shellCmd)
	}

	// Network access requested but no proxy socket - can't enforce filtering
	// Fall back to running without network isolation but with proxy env vars
	// This is a degraded mode - proxy vars are hints only
	if cfg.Network {
		args := BuildPHPArgs(cfg)
		return exec.CommandContext(ctx, args[0], args[1:]...)
	}

	// Offline mode - full network isolation
	phpArgs := BuildPHPArgs(cfg)
	args := append([]string{"--net", "--map-root-user", "--"}, phpArgs...)
	return exec.CommandContext(ctx, "unshare", args...)
}
