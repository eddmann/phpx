package sandbox

import (
	"bytes"
	"context"
	"os"
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
// No environment filtering - inherits full environment from parent.
func (n *None) Execute(ctx context.Context, cfg *Config) (*Result, error) {
	args := BuildPHPArgs(cfg)

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = cfg.WorkDir

	// Setup I/O only (no env filtering for none sandbox)
	var stdout, stderr bytes.Buffer
	if cfg.Stdin != nil {
		cmd.Stdin = cfg.Stdin
	}
	if cfg.Stdout != nil {
		cmd.Stdout = cfg.Stdout
	} else {
		cmd.Stdout = &stdout
	}
	if cfg.Stderr != nil {
		cmd.Stderr = cfg.Stderr
	} else {
		cmd.Stderr = &stderr
	}

	// Inherit full environment from parent
	cmd.Env = os.Environ()

	err := cmd.Run()
	return BuildResult(err, cfg, &stdout, &stderr)
}
