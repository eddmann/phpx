package sandbox

import (
	"context"
	"os/exec"
	"runtime"
)

// Result holds the result of a sandboxed execution.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Sandbox is the interface for different sandbox implementations.
type Sandbox interface {
	// Name returns the name of this sandbox backend
	Name() string

	// IsSandboxed returns true if this backend actually applies sandboxing
	IsSandboxed() bool

	// Available returns true if this sandbox can be used on the current system
	Available() bool

	// Execute runs a command in the sandbox
	Execute(ctx context.Context, cfg *Config) (*Result, error)
}

// Detect returns the best available sandbox for the current system.
func Detect() Sandbox {
	switch runtime.GOOS {
	case "linux":
		// Prefer bubblewrap, fall back to nsjail, then none
		bwrap := &Bubblewrap{}
		if bwrap.Available() {
			return bwrap
		}
		nsjail := &Nsjail{}
		if nsjail.Available() {
			return nsjail
		}
	case "darwin":
		macos := &MacOS{}
		if macos.Available() {
			return macos
		}
	}

	return &None{}
}

// DetectNetworkOnly returns the best available network-only sandbox.
// Network-only sandboxes are lightweight and only restrict network access,
// not filesystem access. Used for --offline and --allow-host flags.
func DetectNetworkOnly() Sandbox {
	switch runtime.GOOS {
	case "linux":
		// Try unshare-based network isolation
		linux := &LinuxNetwork{}
		if linux.Available() {
			return linux
		}
	case "darwin":
		// macOS always has sandbox-exec
		macos := &MacOSNetwork{}
		if macos.Available() {
			return macos
		}
	}

	return &None{}
}

// commandExists checks if a command is available in PATH.
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
