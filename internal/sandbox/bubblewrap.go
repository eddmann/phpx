package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Bubblewrap implements sandbox using bubblewrap (bwrap).
type Bubblewrap struct{}

// Name returns the sandbox name.
func (b *Bubblewrap) Name() string {
	return "bubblewrap"
}

// IsSandboxed returns true - this backend applies sandboxing.
func (b *Bubblewrap) IsSandboxed() bool {
	return true
}

// Available returns true if bubblewrap is available.
func (b *Bubblewrap) Available() bool {
	return runtime.GOOS == "linux" && commandExists("bwrap")
}

// hasSocat checks if socat is available for network bridging.
func hasSocat() bool {
	return commandExists("socat")
}

// Execute runs a command in the bubblewrap sandbox.
func (b *Bubblewrap) Execute(ctx context.Context, cfg *Config) (*Result, error) {
	// Always unshare network for security
	// If network is needed, we use socat to bridge to Unix socket proxy
	args := b.buildArgs(cfg)

	cmd := exec.CommandContext(ctx, "bwrap", args...)
	cmd.Dir = cfg.WorkDir

	stdout, stderr := SetupCommand(cmd, cfg)

	// If network is enabled and we have a proxy socket, set up proxy env vars
	// The socat bridge inside sandbox will forward to the socket
	if cfg.Network && cfg.ProxySocketPath != "" && hasSocat() {
		cmd.Env = append(cmd.Env, ProxyEnvVars()...)
	}

	err := cmd.Run()
	return BuildResult(err, cfg, stdout, stderr)
}

// buildArgs constructs the bwrap command arguments.
// Follows principle of least privilege - minimal mounts for static PHP binary.
func (b *Bubblewrap) buildArgs(cfg *Config) []string {
	args := []string{}

	// ============================================================
	// MINIMAL DEVICE ACCESS
	// Static PHP binary only needs /dev/null, /dev/urandom
	// ============================================================
	args = append(args,
		"--dev-bind", "/dev/null", "/dev/null",
		"--dev-bind", "/dev/urandom", "/dev/urandom",
		"--dev-bind", "/dev/random", "/dev/random",
	)

	// ============================================================
	// TIMEZONE DATA
	// Required for date() functions
	// ============================================================
	if _, err := os.Stat("/usr/share/zoneinfo"); err == nil {
		args = append(args, "--ro-bind", "/usr/share/zoneinfo", "/usr/share/zoneinfo")
	}
	if _, err := os.Stat("/etc/localtime"); err == nil {
		args = append(args, "--ro-bind", "/etc/localtime", "/etc/localtime")
	}

	// ============================================================
	// DNS RESOLUTION (only if network enabled)
	// ============================================================
	if cfg.Network {
		if _, err := os.Stat("/etc/resolv.conf"); err == nil {
			args = append(args, "--ro-bind", "/etc/resolv.conf", "/etc/resolv.conf")
		}
		if _, err := os.Stat("/etc/hosts"); err == nil {
			args = append(args, "--ro-bind", "/etc/hosts", "/etc/hosts")
		}
		// NSS config for DNS resolution
		if _, err := os.Stat("/etc/nsswitch.conf"); err == nil {
			args = append(args, "--ro-bind", "/etc/nsswitch.conf", "/etc/nsswitch.conf")
		}
	}

	// ============================================================
	// PHP BINARY (exact file only)
	// ============================================================
	if cfg.PHPBinary != "" {
		args = append(args, "--ro-bind", cfg.PHPBinary, cfg.PHPBinary)
	}

	// ============================================================
	// SCRIPT FILE (exact file only)
	// ============================================================
	if cfg.ScriptPath != "" {
		args = append(args, "--ro-bind", cfg.ScriptPath, cfg.ScriptPath)
	}

	// ============================================================
	// VENDOR DIRECTORY (for dependencies)
	// ============================================================
	if cfg.AutoloadFile != "" {
		vendorDir := filepath.Dir(cfg.AutoloadFile)
		args = append(args, "--ro-bind", vendorDir, vendorDir)
	}

	// ============================================================
	// ADDITIONAL READABLE PATHS (--allow-read)
	// ============================================================
	for _, p := range cfg.ReadablePaths {
		if _, err := os.Stat(p); err == nil {
			args = append(args, "--ro-bind", p, p)
		}
	}

	// ============================================================
	// ADDITIONAL WRITABLE PATHS (--allow-write)
	// ============================================================
	for _, p := range cfg.WritablePaths {
		if _, err := os.Stat(p); err == nil {
			args = append(args, "--bind", p, p)
		}
	}

	// ============================================================
	// PROXY SOCKET (for network access)
	// ============================================================
	if cfg.Network && cfg.ProxySocketPath != "" {
		args = append(args, "--ro-bind", cfg.ProxySocketPath, "/tmp/proxy.sock")
	}

	// ============================================================
	// ISOLATION OPTIONS
	// ============================================================
	args = append(args,
		"--unshare-user",
		"--unshare-pid",
		"--unshare-uts",
		"--unshare-cgroup",
		"--unshare-net", // Always isolate network
		"--die-with-parent",
		"--new-session",
	)

	// Working directory (if specified, just set chdir - no mount means no access)
	if cfg.WorkDir != "" {
		args = append(args, "--chdir", cfg.WorkDir)
	}

	// If network is enabled and socat is available, use it to bridge to proxy
	if cfg.Network && cfg.ProxySocketPath != "" && hasSocat() {
		phpCmd := BuildPHPCommand(cfg)
		shellCmd := BuildSocatBridgeCommand("/tmp/proxy.sock", phpCmd)
		args = append(args, "--", "sh", "-c", shellCmd)
	} else {
		args = append(args, "--")
		args = append(args, BuildPHPArgs(cfg)...)
	}

	return args
}
