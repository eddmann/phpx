package sandbox

import (
	"context"
	"os/exec"
)

// None implements execution without sandboxing.
type None struct{}

// Name returns the sandbox name.
func (n *None) Name() string {
	return "none"
}

// IsSandboxed returns false - this backend applies no sandboxing.
func (n *None) IsSandboxed() bool {
	return false
}

// Available always returns true - no sandbox is always available.
func (n *None) Available() bool {
	return true
}

// Execute runs a command without sandboxing.
func (n *None) Execute(ctx context.Context, cfg *Config) (*Result, error) {
	args := BuildPHPArgs(cfg)

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = cfg.WorkDir

	stdout, stderr := SetupCommand(cmd, cfg)

	err := cmd.Run()
	return BuildResult(err, cfg, stdout, stderr)
}
