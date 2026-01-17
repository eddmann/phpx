package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Nsjail implements sandbox using nsjail.
type Nsjail struct{}

// Name returns the sandbox name.
func (n *Nsjail) Name() string {
	return "nsjail"
}

// IsSandboxed returns true - this backend applies sandboxing.
func (n *Nsjail) IsSandboxed() bool {
	return true
}

// Available returns true if nsjail is available.
func (n *Nsjail) Available() bool {
	return runtime.GOOS == "linux" && commandExists("nsjail")
}

// Execute runs a command in the nsjail sandbox.
func (n *Nsjail) Execute(ctx context.Context, cfg *Config) (*Result, error) {
	args := n.buildArgs(cfg)

	cmd := exec.CommandContext(ctx, "nsjail", args...)
	cmd.Dir = cfg.WorkDir

	stdout, stderr := SetupCommand(cmd, cfg)

	err := cmd.Run()
	return BuildResult(err, cfg, stdout, stderr)
}

// buildArgs constructs the nsjail command arguments.
// Follows principle of least privilege - minimal mounts for static PHP binary.
func (n *Nsjail) buildArgs(cfg *Config) []string {
	args := []string{
		"--mode", "o",
		"--user", "65534",
		"--group", "65534",
		"--quiet", // Reduce nsjail output noise
	}

	// Resource limits
	if cfg.Timeout > 0 {
		args = append(args, "--time_limit", fmt.Sprintf("%d", int(cfg.Timeout.Seconds())))
	}
	if cfg.MemoryMB > 0 {
		args = append(args, "--rlimit_as", fmt.Sprintf("%d", cfg.MemoryMB))
	}
	if cfg.CPUSeconds > 0 {
		args = append(args, "--rlimit_cpu", fmt.Sprintf("%d", cfg.CPUSeconds))
	}

	// File limits
	args = append(args,
		"--rlimit_fsize", "50",   // 50MB max file size
		"--rlimit_nofile", "128", // Max open files
		"--rlimit_nproc", "10",   // Max processes
	)

	// Network isolation
	// By default, nsjail creates a new network namespace (isolated, no network)
	// Only disable network namespace isolation if network access is needed (for proxy)
	if cfg.Network {
		args = append(args, "--disable_clone_newnet")
	}

	// ============================================================
	// MINIMAL DEVICE ACCESS
	// ============================================================
	args = append(args,
		"--bindmount_ro", "/dev/null:/dev/null",
		"--bindmount_ro", "/dev/urandom:/dev/urandom",
		"--bindmount_ro", "/dev/random:/dev/random",
	)

	// ============================================================
	// TIMEZONE DATA
	// ============================================================
	args = append(args, "--bindmount_ro", "/usr/share/zoneinfo:/usr/share/zoneinfo")
	args = append(args, "--bindmount_ro", "/etc/localtime:/etc/localtime")

	// ============================================================
	// DNS RESOLUTION (only if network enabled)
	// ============================================================
	if cfg.Network {
		args = append(args, "--bindmount_ro", "/etc/resolv.conf:/etc/resolv.conf")
		args = append(args, "--bindmount_ro", "/etc/hosts:/etc/hosts")
		args = append(args, "--bindmount_ro", "/etc/nsswitch.conf:/etc/nsswitch.conf")
	}

	// ============================================================
	// PHP BINARY (exact file only)
	// ============================================================
	if cfg.PHPBinary != "" {
		args = append(args, "--bindmount_ro", cfg.PHPBinary+":"+cfg.PHPBinary)
	}

	// ============================================================
	// SCRIPT FILE (exact file only)
	// ============================================================
	if cfg.ScriptPath != "" {
		args = append(args, "--bindmount_ro", cfg.ScriptPath+":"+cfg.ScriptPath)
	}

	// ============================================================
	// VENDOR DIRECTORY (for dependencies)
	// ============================================================
	if cfg.AutoloadFile != "" {
		vendorDir := filepath.Dir(cfg.AutoloadFile)
		args = append(args, "--bindmount_ro", vendorDir+":"+vendorDir)
	}

	// ============================================================
	// ADDITIONAL READABLE PATHS (--allow-read)
	// ============================================================
	for _, p := range cfg.ReadablePaths {
		args = append(args, "--bindmount_ro", p+":"+p)
	}

	// ============================================================
	// ADDITIONAL WRITABLE PATHS (--allow-write)
	// ============================================================
	for _, p := range cfg.WritablePaths {
		args = append(args, "--bindmount", p+":"+p)
	}

	// Working directory (just set cwd, no mount = no access by default)
	if cfg.WorkDir != "" {
		args = append(args, "--cwd", cfg.WorkDir)
	}

	// PHP command
	args = append(args, "--", cfg.PHPBinary)

	// PHP options
	if cfg.MemoryMB > 0 {
		args = append(args, "-d", fmt.Sprintf("memory_limit=%dM", cfg.MemoryMB))
	}
	if cfg.CPUSeconds > 0 {
		args = append(args, "-d", fmt.Sprintf("max_execution_time=%d", cfg.CPUSeconds))
	}
	if cfg.AutoloadFile != "" {
		args = append(args, "-d", fmt.Sprintf("auto_prepend_file=%s", cfg.AutoloadFile))
	}

	args = append(args, cfg.ScriptPath)
	args = append(args, cfg.ScriptArgs...)

	return args
}
